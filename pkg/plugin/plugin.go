/*
   Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0
*/

package plugin

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"grafana-esp-plugin/internal/plugin/syncmap"
	"io"
	"net/http"
	"net/url"
	"time"

	"grafana-esp-plugin/internal/esp/client"
	"grafana-esp-plugin/internal/esp/windowevent"
	"grafana-esp-plugin/internal/framefactory"
	"grafana-esp-plugin/internal/plugin/query"
	"grafana-esp-plugin/internal/plugin/querydto"
	"grafana-esp-plugin/internal/plugin/server"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/live"
)

// Make sure SampleDatasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler, backend.StreamHandler interfaces. Plugin should not
// implement all these interfaces - only those which are required for a particular task.
// For example if plugin does not need streaming functionality then you are free to remove
// methods that implement backend.StreamHandler. Implementing instancemgmt.InstanceDisposer
// is useful to clean up resources used by previous datasource instance when a new datasource
// instance created upon datasource settings changed.
var (
	_ backend.QueryDataHandler      = (*SampleDatasource)(nil)
	_ backend.CheckHealthHandler    = (*SampleDatasource)(nil)
	_ backend.StreamHandler         = (*SampleDatasource)(nil)
	_ instancemgmt.InstanceDisposer = (*SampleDatasource)(nil)
)

// NewSampleDatasource creates a new datasource instance.
func NewSampleDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	url, err := url.Parse(settings.URL)
	if err != nil {
		return nil, err
	}

	opts, err := settings.HTTPClientOptions(ctx)
	if err != nil {
		return nil, err
	}
	opts.ForwardHTTPHeaders = true

	cl, err := httpclient.New(opts)
	if err != nil {
		return nil, err
	}

	var jsonData datasourceJsonData
	err = json.Unmarshal(settings.JSONData, &jsonData)
	if err != nil {
		return nil, err
	}

	log.DefaultLogger.Debug(fmt.Sprintf("created data source with ForwardHTTPHeaders option set to: %v", opts.ForwardHTTPHeaders))

	return &SampleDatasource{
		httpClient:           cl,
		url: *url,
		jsonData:             jsonData,
		channelQueryMap:      syncmap.New[string, query.Query](),
		serverUrlTrustedMap:  syncmap.New[string, bool](),
	}, nil
}

// SampleDatasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type SampleDatasource struct {
	channelQueryMap      *syncmap.SyncMap[string, query.Query]
	httpClient           *http.Client
	jsonData             datasourceJsonData
	serverUrlTrustedMap  *syncmap.SyncMap[string, bool]
	url                  url.URL
}

type datasourceJsonData struct {
	UseExternalEspUrl bool `json:"useExternalEspUrl"`
	OauthPassThru     bool `json:"oauthPassThru"`
	TlsSkipVerify     bool `json:"tlsSkipVerify"`
	DirectToEsp     bool `json:"DirectToEsp"`
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *SampleDatasource) Dispose() {
	// Clean up datasource instance resources.
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *SampleDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()

	var dJsonData datasourceJsonData
	err := json.Unmarshal(req.PluginContext.DataSourceInstanceSettings.JSONData, &dJsonData)
	if err != nil {
		return nil, err
	}
	var authorizationHeaderPtr *string = nil
	if dJsonData.OauthPassThru {
		authorizationHeader := req.GetHTTPHeader(backend.OAuthIdentityTokenHeaderName)
		authorizationHeaderPtr = &authorizationHeader
	}

	for _, q := range req.Queries {
		var qdto querydto.QueryDTO
		err := json.Unmarshal(q.JSON, &qdto)
		if err != nil {
			response.Responses[q.RefID] = handleQueryError("invalid query", err)
			continue
		}

		var serverUrl string
		if d.jsonData.UseExternalEspUrl {
			serverUrl = qdto.ExternalServerUrl
		} else {
			serverUrl = qdto.InternalServerUrl
		}

		var authHeaderToBePassed *string = nil
		if authorizationHeaderPtr != nil && d.isServerUrlTrusted(serverUrl, !d.jsonData.DirectToEsp, authorizationHeaderPtr) {
			authHeaderToBePassed = authorizationHeaderPtr
		}

		response.Responses[q.RefID] = d.query(ctx, req.PluginContext.DataSourceInstanceSettings.UID, qdto, authHeaderToBePassed)
	}

	return response, nil
}

func (d *SampleDatasource) query(_ context.Context, datasourceUid string, qdto querydto.QueryDTO, authorizationHeader *string) backend.DataResponse {
	var qServerUrl string
	if d.jsonData.UseExternalEspUrl {
		qServerUrl = qdto.ExternalServerUrl
	} else {
		qServerUrl = qdto.InternalServerUrl
		log.DefaultLogger.Debug("Using internal ESP server URL from query", "query", qdto)
	}

	s, err := server.FromUrlString(qServerUrl)
	if err != nil {
		return handleQueryError("invalid server URL", err)
	}
	serverUrl := s.GetUrl()

	q := query.New(serverUrl, qdto.ProjectName, qdto.CqName, qdto.WindowName, qdto.Interval, qdto.MaxDataPoints, qdto.Fields, authorizationHeader)

	channelPath := q.ToChannelPath()

	d.channelQueryMap.Set(channelPath, q)

	log.DefaultLogger.Debug("Received query", "path", channelPath, "query", q)

	// If query called with streaming on then return a channel
	// to subscribe on a client-side and consume updates from a plugin.
	// Feel free to remove this if you don't need streaming for your datasource.

	channel := live.Channel{
		Scope:     live.ScopeDatasource,
		Namespace: datasourceUid,
		Path:      channelPath,
	}

	frame := data.NewFrame("response")
	frame.SetMeta(&data.FrameMeta{Channel: channel.String()})

	response := backend.DataResponse{}
	response.Frames = append(response.Frames, frame)

	return response
}

func handleQueryError(errorMessage string, err error) backend.DataResponse {
	log.DefaultLogger.Error(errorMessage, "error", err)
	response := backend.DataResponse{
		Error: errors.New(errorMessage),
	}

	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *SampleDatasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	var dJsonData datasourceJsonData
	err := json.Unmarshal(req.PluginContext.DataSourceInstanceSettings.JSONData, &dJsonData)
	if err != nil {
		return nil, err
	}

	var url = req.PluginContext.DataSourceInstanceSettings.URL

	var request *http.Request
	if dJsonData.DirectToEsp {
		request, err = createEspHostHealthCheckRequest(ctx, url)
		if err != nil {
			var message = "The application was unable to create a valid datasource HTTP request"
			log.DefaultLogger.Error(message, "error", err)
			return newCheckHealthErrorResponse(message), nil
		}
	} else {
		request, err = createDiscoveryHostHealthCheckRequest(ctx, url)
		if err != nil {
			var message = "The application was unable to create a valid datasource HTTP request"
			log.DefaultLogger.Error(message, "error", err)
			return newCheckHealthErrorResponse(message), nil
		}
	}

	resp, err := d.httpClient.Do(request)

	if err != nil {
		var message = "Failed to connect to datasource"
		log.DefaultLogger.Error(message, "error", err)
		return newCheckHealthErrorResponse(message), nil
	}

	log.DefaultLogger.Debug("Datasource response", "status", resp.Status)

	switch resp.StatusCode {
	case 200:
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusOk,
			Message: "Connection successful",
		}, nil
	case 401:
		var message = fmt.Sprintf("Connection rejected due to unauthorized credentials")
		var hasAuthHeader = len(resp.Request.Header.Get("Authorization")) > 0
		log.DefaultLogger.Debug("endpoint authorization failure",
			"authorizationHeaderPresent", hasAuthHeader,
			"oauthPassThru", d.jsonData.OauthPassThru,
		)
		return newCheckHealthErrorResponse(message), nil
	default:
		var message = fmt.Sprintf("The datasource sent an unexpected HTTP status code: %d", resp.StatusCode)
		return newCheckHealthErrorResponse(message), nil
	}
}

func createDiscoveryHostHealthCheckRequest(ctx context.Context, discoveryServiceUrl string) (*http.Request, error) {
	var endpointUrl = discoveryServiceUrl + "/apiMeta"

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointUrl, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Accept", "application/json")

	return request, nil
}

func createEspHostHealthCheckRequest(ctx context.Context, espUrl string) (*http.Request, error) {
	var endpointUrl = espUrl + "/runningProjects"

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointUrl, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Accept", "application/xml")

	return request, nil
}

func newCheckHealthErrorResponse(message string) *backend.CheckHealthResult {
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusError,
		Message: message,
	}
}

// SubscribeStream is called when a client wants to connect to a stream. This callback
// allows sending the first message.
func (d *SampleDatasource) SubscribeStream(_ context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	log.DefaultLogger.Debug("Received stream subscription", "path", req.Path)

	status := backend.SubscribeStreamStatusPermissionDenied

	if _, err := d.channelQueryMap.Get(req.Path); err == nil {
		// Allow subscribing only on expected path.
		status = backend.SubscribeStreamStatusOK
	}

	return &backend.SubscribeStreamResponse{
		Status: status,
	}, nil
}

// RunStream is called once for any open channel.  Results are shared with everyone
// subscribed to the same channel.
func (d *SampleDatasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	log.DefaultLogger.Debug("initiating stream", "path", req.Path)

	queryKey := req.Path

	q, err := d.channelQueryMap.Get(queryKey)
	if err != nil {
		// The channel refers to an unknown query.
		// Avoid returning the error, to prevent continuous attempts from Grafana to re-establish the stream.
		log.DefaultLogger.Error(fmt.Sprintf("query not found for channel %v", req.Path), "error", err)
		return nil
	}

	log.DefaultLogger.Debug("Instantiating new ESP websocket client from query", "query", q)
	espWsClient := client.New(q.ServerUrl, q.AuthorizationHeader)
	defer espWsClient.Close()

	subscribeToQuery := func() {
		// Clear any preceding errors prior to subscribing
		sendErrorClearFrame(sender)

		err := espWsClient.Subscribe(q.ProjectName, q.CqName, q.WindowName, q.EventInterval, q.MaxEvents, q.Fields)
		if err != nil {
			log.DefaultLogger.Error(fmt.Sprintf("error while subscribing to events on channel %v", req.Path), "error", err)
		}
	}

	espWsClient.OnConnected = subscribeToQuery

	espWsClient.OnProjectLoaded = func(projectName string) {
		if q.ProjectName != projectName {
			return
		}

		subscribeToQuery()
	}

	espWsClient.OnProjectRemoved = func(projectName string) {
		if q.ProjectName != projectName {
			return
		}

		projectRemovedMessage := fmt.Sprintf("Project '%s' is not running", projectName)
		sendErrorFrame(projectRemovedMessage, sender)
	}

	espWsClient.OnEventMessageReceived = func(we windowevent.WindowEvent) {
		frame := framefactory.NewWindowEventFrame(we)

		err := sender.SendFrame(frame, data.IncludeAll)
		if err != nil {
			log.DefaultLogger.Error("Error sending data frame", "error", err)
		}
	}

	go espWsClient.Connect()

	// Stream data frames periodically till stream closed by Grafana.
	for {
		select {
		case <-ctx.Done():
			log.DefaultLogger.Debug("Context done, finish streaming", "path", req.Path)

			// Free the stored query if present.
			d.channelQueryMap.Delete(queryKey)

			return nil
		case err := <-espWsClient.Errors:
			errorMessage := err.Error()
			log.DefaultLogger.Error(errorMessage, "err", err.Error())
			sendErrorFrame(errorMessage, sender)
			return err
		}
	}
}

func sendErrorFrame(errorMessage string, sender *backend.StreamSender) {
	frame := framefactory.NewErrorFrame(errorMessage)

	sendError := sender.SendFrame(frame, data.IncludeAll)
	if sendError != nil {
		log.DefaultLogger.Error("Error sending error frame", "error", sendError)
	}
}

func sendErrorClearFrame(sender *backend.StreamSender) {
	frame := framefactory.NewErrorClearFrame()

	sendError := sender.SendFrame(frame, data.IncludeAll)
	if sendError != nil {
		log.DefaultLogger.Error("Error sending error-clear frame", "error", sendError)
	}
}

// PublishStream is called when a client sends a message to the stream.
func (d *SampleDatasource) PublishStream(_ context.Context, _ *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	// Do not allow publishing at all.
	return &backend.PublishStreamResponse{
		Status: backend.PublishStreamStatusPermissionDenied,
	}, nil
}

type callResourceResponseBody struct {
	Error *string         `json:"error,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

func newSerializedCallResourceResponseErrorBody(errorMessage string) []byte {
	errorResponseBody, err := json.Marshal(callResourceResponseBody{
		Error: &errorMessage,
	})
	if err != nil {
		errorResponseBody = []byte(`{error:"Internal server error"}`)
	}

	return errorResponseBody
}

func (d *SampleDatasource) CallResource(_ context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	var response backend.CallResourceResponse
	switch req.Path {
	case "servers":
		var authHeaderPtr *string
		if d.jsonData.OauthPassThru == true {
			authHeader := req.GetHTTPHeader(backend.OAuthIdentityTokenHeaderName)
			authHeaderPtr = &authHeader
		}

		espServerInfoList, err := d.fetchServerInfo(authHeaderPtr)
		if err != nil {
			log.DefaultLogger.Error(err.Error())
			body := newSerializedCallResourceResponseErrorBody("Unable to fetch ESP server information: " + err.Error())
			response := &backend.CallResourceResponse{
				Status: http.StatusBadGateway,
				Body:   body,
			}
			return sender.Send(response)
		}

		espServerInfoListJson, err := json.Marshal(espServerInfoList)
		if err != nil {
			errorMessage := "Unable to serialize ESP server information."
			response := &backend.CallResourceResponse{
				Status: http.StatusInternalServerError,
				Body:   newSerializedCallResourceResponseErrorBody(errorMessage),
			}
			log.DefaultLogger.Error(errorMessage, "error", err)
			return sender.Send(response)
		}

		responseBody, err := json.Marshal(callResourceResponseBody{Data: espServerInfoListJson})
		if err != nil {
			errorMessage := "Unable to serialize ESP server information response."
			response := &backend.CallResourceResponse{
				Status: http.StatusInternalServerError,
				Body:   newSerializedCallResourceResponseErrorBody(errorMessage),
			}
			log.DefaultLogger.Error(errorMessage, "error", err)
			return sender.Send(response)
		}

		response = backend.CallResourceResponse{
			Status: http.StatusOK,
			Body:   responseBody,
		}
		return sender.Send(&response)
	default:
		response = backend.CallResourceResponse{
			Status: http.StatusNotFound,
		}
		break
	}

	return sender.Send(&response)
}

func (d *SampleDatasource) fetchServerInfoFromDiscoveryEndpoint(authHeader *string) (*[]espServerInfo, error) {
	var discoveryEndpointUrl = d.url.String() + "/grafana/discovery"
	log.DefaultLogger.Debug("Calling discovery endpoint", "discoveryEndpointUrl", discoveryEndpointUrl)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryEndpointUrl, nil)
	if err != nil {
		log.DefaultLogger.Error("Unable to create discovery request.", "error", err)
		return nil, err
	}

	if authHeader != nil {
		request.Header.Set(backend.OAuthIdentityTokenHeaderName, *authHeader)
	}

	resp, err := d.httpClient.Do(request)
	if err != nil {
		log.DefaultLogger.Error("Unable to receive discovery response.", "error", err)
		return nil, err
	}

	if resp.StatusCode != 200 {
		switch resp.StatusCode {
		case 401:
			var message = fmt.Sprintf("Connection to discovery endpoint rejected due to unauthorized credentials.")
			var hasAuthHeader = len(resp.Request.Header.Get("Authorization")) > 0
			log.DefaultLogger.Debug("Discovery service authorization failure",
				"authorizationHeaderPresent", hasAuthHeader,
				"oauthPassThru", d.jsonData.OauthPassThru,
			)
			return nil, fmt.Errorf(message)
		default:
			var message = fmt.Sprintf("The discovery service sent an unexpected HTTP status code: %d", resp.StatusCode)
			return nil, fmt.Errorf(message)
		}
	}

	serversData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.DefaultLogger.Error(err.Error())
		return nil, fmt.Errorf("unable to read discovery response")
	}

	var espServerInfo []espServerInfo
	err = json.Unmarshal(serversData, &espServerInfo)
	if err != nil {
		log.DefaultLogger.Error(err.Error())
		return nil, fmt.Errorf("unable to unmarshal discovery response")
	}

	return &espServerInfo, nil
}

func (d *SampleDatasource) isServerUrlTrusted(url string, fetchIfMissing bool, authHeader *string) bool {
	if d.jsonData.DirectToEsp {
		return true
	}

	isServerUrlTrusted, err := d.serverUrlTrustedMap.Get(url)
	if err == nil {
		return *isServerUrlTrusted
	}

	if fetchIfMissing {
		discoveredServers, fetchErr := d.fetchServerInfoFromDiscoveryEndpoint(authHeader)
		if fetchErr != nil {
			log.DefaultLogger.Error("Unable to fetch trusted status of server URL", "url", url, "error", err)
			return false
		}

		d.updateServerTrust(*discoveredServers)

		return d.isServerUrlTrusted(url, false, nil)
	}

	log.DefaultLogger.Error("Unable to determine trusted status of server URL", "url", url, "error", err)
	return false
}

func (d *SampleDatasource) updateServerTrust(discoveredServers []espServerInfo) {
	for _, discoveredServer := range discoveredServers {
		d.serverUrlTrustedMap.Set(discoveredServer.Url.String(), &discoveredServer.Trusted)
		d.serverUrlTrustedMap.Set(discoveredServer.ExternalUrl.String(), &discoveredServer.Trusted)
	}
}

type espRunningProjectsFromXml struct {
	Projects []struct {
		Name              string `xml:"name,attr"`
		ContinuousQueries []struct {
			Name    string `xml:"name,attr"`
			Windows struct {
				Windows []struct {
					Name   string `xml:"name,attr"`
					Fields []struct {
						Name string `xml:"name,attr"`
					} `xml:"schema>fields>field"`
				} `xml:",any"`
			} `xml:"windows"`
		} `xml:"contqueries>contquery"`
	} `xml:"project"`
}

type espServerInfo struct {
	Projects    []project `json:"projects"`
	Name        string    `json:"name"`
	Url         url.URL   `json:"url,string"`
	ExternalUrl url.URL   `json:"externalUrl,string"`
	Trusted     bool      `json:"trusted"`
}

type project struct {
	Name              string            `json:"name"`
	ContinuousQueries []continuousQuery `json:"continuousQueries"`
}

type continuousQuery struct {
	Name    string   `json:"name"`
	Windows []window `json:"windows"`
}

type window struct {
	Name   string  `json:"name"`
	Fields []field `json:"fields"`
}

type field struct {
	Name string `json:"name"`
}

func (d *SampleDatasource) fetchServerInfo(authHeader *string) (*[]espServerInfo, error) {
	var espServerInfoList *[]espServerInfo

	if d.jsonData.DirectToEsp {
		returnedEspServerInfo, err := d.fetchServerInfoFromEspInstance(authHeader)
		if err != nil {
			return nil, err
		}
		espServerInfoList = &[]espServerInfo{*returnedEspServerInfo}
	} else {
		var err error
		espServerInfoList, err = d.fetchServerInfoFromDiscoveryEndpoint(authHeader)
		if err != nil {
			return nil, err
		}

		d.updateServerTrust(*espServerInfoList)
	}

	return espServerInfoList, nil
}

func (d *SampleDatasource) fetchServerInfoFromEspInstance(authHeader *string) (*espServerInfo, error) {
	var espRunningProjectsEndpoint = d.url.String() + "/runningProjects?schema=true"
	log.DefaultLogger.Debug("Calling ESP server endpoint", "espRunningProjectsEndpoint", espRunningProjectsEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, espRunningProjectsEndpoint, nil)
	if err != nil {
		log.DefaultLogger.Error("Unable to create ESP server running projects request.", "error", err)
		return nil, err
	}

	if authHeader != nil {
		request.Header.Set(backend.OAuthIdentityTokenHeaderName, *authHeader)
	}

	resp, err := d.httpClient.Do(request)
	if err != nil {
		log.DefaultLogger.Error("Unable to receive running projects response.", "error", err)
		return nil, err
	}

	if resp.StatusCode != 200 {
		switch resp.StatusCode {
		case 401:
			var message = fmt.Sprintf("Connection to ESP server endpoint rejected due to unauthorized credentials.")
			var hasAuthHeader = len(resp.Request.Header.Get("Authorization")) > 0
			log.DefaultLogger.Debug("ESP server authorization failure",
				"authorizationHeaderPresent", hasAuthHeader,
				"oauthPassThru", d.jsonData.OauthPassThru,
			)
			return nil, fmt.Errorf(message)
		default:
			var message = fmt.Sprintf("The ESP server sent an unexpected HTTP status code: %d", resp.StatusCode)

			responseBody, err := io.ReadAll(resp.Body)
			if err == nil {
				log.DefaultLogger.Debug("Unexpected ESP server response", "responseBody", string(responseBody))
			} else {
				log.DefaultLogger.Debug("Unexpected ESP server response. Unable to serialize response body.")
			}

			return nil, fmt.Errorf(message)
		}
	}

	projectsData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.DefaultLogger.Error(err.Error())
		return nil, fmt.Errorf("unable to read discovery response")
	}

	log.DefaultLogger.Debug("Received ESP server running projects response", "response", string(projectsData))

	var projects espRunningProjectsFromXml
	err = xml.Unmarshal(projectsData, &projects)
	if err != nil {
		log.DefaultLogger.Error(err.Error())
		return nil, fmt.Errorf("unable to unmarshal ESP running projects response")
	}

	espServerInfo := projects.toEspServerInfo(d.url)

	return &espServerInfo, nil
}

func (t *espRunningProjectsFromXml) toEspServerInfo(url url.URL) espServerInfo {
	var espServerName string
	if len(t.Projects) < 1 {
		espServerName = "(Unknown)"
	} else {
		espServerName = t.Projects[0].Name
	}

	var projects []project
	for _, p := range t.Projects {
		var continuousQueries []continuousQuery
		for _, cq := range p.ContinuousQueries {
			var windows []window
			for _, w := range cq.Windows.Windows {
				var fields []field
				for _, f := range w.Fields {
					fields = append(fields, field{Name: f.Name})
				}
				windows = append(windows, window{Name: w.Name, Fields: fields})
			}
			continuousQueries = append(continuousQueries, continuousQuery{Name: cq.Name, Windows: windows})
		}
		projects = append(projects, project{Name: p.Name, ContinuousQueries: continuousQueries})
	}

	switch url.Scheme {
	case "http":
		url.Scheme = "ws"
	case "https":
		url.Scheme = "wss"
	}

	return espServerInfo{projects, espServerName, url, url, true}
}

func (t *espServerInfo) MarshalJSON() ([]byte, error) {
	type Alias espServerInfo
	return json.Marshal(&struct {
		Url         string `json:"url"`
		ExternalUrl string `json:"externalUrl"`
		*Alias
	}{
		Url:         t.Url.String(),
		ExternalUrl: t.ExternalUrl.String(),
		Alias:       (*Alias)(t),
	})
}

func (t *espServerInfo) UnmarshalJSON(data []byte) error {
	type Alias espServerInfo
	serializedJson := &struct {
		Url         string `json:"url"`
		ExternalUrl string `json:"externalUrl"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}

	err := json.Unmarshal(data, &serializedJson)
	if err != nil {
		return err
	}

	deserializedUrl, err := url.Parse(serializedJson.Url)
	if err != nil {
		return err
	}
	t.Url = *deserializedUrl

	deserializedExternalUrl, err := url.Parse(serializedJson.ExternalUrl)
	if err != nil {
		return err
	}
	t.ExternalUrl = *deserializedExternalUrl

	return nil
}