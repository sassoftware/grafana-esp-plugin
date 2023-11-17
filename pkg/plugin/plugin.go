/*
   Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0
*/

package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"grafana-esp-plugin/internal/plugin/syncmap"
	"io"
	"net/http"
	"net/url"

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
	discoveryUrl, err := url.Parse(settings.URL)
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
		discoveryEndpointUrl: *discoveryUrl,
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
	discoveryEndpointUrl url.URL
}

type datasourceJsonData struct {
	IsViya        bool `json:"isViya"`
	OauthPassThru bool `json:"oauthPassThru"`
	TlsSkipVerify bool `json:"tlsSkipVerify"`
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

	var jsonData datasourceJsonData
	err := json.Unmarshal(req.PluginContext.DataSourceInstanceSettings.JSONData, &jsonData)
	if err != nil {
		return nil, err
	}
	var authorizationHeaderPtr *string = nil
	if jsonData.OauthPassThru && jsonData.IsViya {
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

		var authHeaderToBePassed *string = nil
		if authorizationHeaderPtr != nil && d.isServerUrlTrusted(qdto.ServerUrl, true, authorizationHeaderPtr) {
			authHeaderToBePassed = authorizationHeaderPtr
		}

		response.Responses[q.RefID] = d.query(ctx, req.PluginContext.DataSourceInstanceSettings.UID, qdto, authHeaderToBePassed)
	}

	return response, nil
}

func (d *SampleDatasource) query(_ context.Context, datasourceUid string, qdto querydto.QueryDTO, authorizationHeader *string) backend.DataResponse {
	s, err := server.FromUrlString(qdto.ServerUrl)
	if err != nil {
		return handleQueryError("invalid server URL", err)
	}

	q := query.New(*s, qdto.ProjectName, qdto.CqName, qdto.WindowName, qdto.Interval, qdto.MaxDataPoints, qdto.Fields, authorizationHeader)

	channelPath, err := q.ToChannelPath()
	if err != nil {
		return handleQueryError("invalid channel path", err)
	}

	d.channelQueryMap.Set(*channelPath, q)

	log.DefaultLogger.Debug("Received query", "path", *channelPath)

	// If query called with streaming on then return a channel
	// to subscribe on a client-side and consume updates from a plugin.
	// Feel free to remove this if you don't need streaming for your datasource.

	channel := live.Channel{
		Scope:     live.ScopeDatasource,
		Namespace: datasourceUid,
		Path:      *channelPath,
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
	var discoveryServiceUrl = req.PluginContext.DataSourceInstanceSettings.URL
	var endpointUrl = discoveryServiceUrl + "/apiMeta"
	log.DefaultLogger.Debug("Checking connection to discovery service", "endpointUrl", endpointUrl)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointUrl, nil)
	if err != nil {
		var message = "The application was unable to create a valid HTTP request"
		log.DefaultLogger.Error(message, "error", err)
		return newCheckHealthErrorResponse(message), nil
	}

	request.Header.Set("Accept", "application/json")
	resp, err := d.httpClient.Do(request)

	if err != nil {
		var message = "Failed to connect to the discovery service"
		log.DefaultLogger.Error(message, "error", err)
		return newCheckHealthErrorResponse(message), nil
	}

	log.DefaultLogger.Debug("Studio response", "status", resp.Status)

	switch resp.StatusCode {
	case 200:
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusOk,
			Message: "Connection successful",
		}, nil
	case 401:
		var message = fmt.Sprintf("Connection rejected due to unauthorized credentials")
		var hasAuthHeader = len(resp.Request.Header.Get("Authorization")) > 0
		log.DefaultLogger.Debug("Discovery service authorization failure",
			"authorizationHeaderPresent", hasAuthHeader,
			"oauthPassThru", d.jsonData.OauthPassThru,
		)
		return newCheckHealthErrorResponse(message), nil
	default:
		var message = fmt.Sprintf("The discovery service sent an unexpected HTTP status code: %d", resp.StatusCode)
		return newCheckHealthErrorResponse(message), nil
	}
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

type discoveredServer struct {
	Url     string `json:"url"`
	Trusted bool   `json:"trusted"`
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
		serversData, discoveredServers, err := d.fetchDiscoverableServers(authHeaderPtr)
		if err != nil {
			log.DefaultLogger.Error(err.Error())
			body := newSerializedCallResourceResponseErrorBody("Unable to fetch discoverable ESP servers.")
			response := &backend.CallResourceResponse{
				Status: http.StatusInternalServerError,
				Body:   body,
			}
			return sender.Send(response)
		}

		for _, discoveredServer := range *discoveredServers {
			d.serverUrlTrustedMap.Set(discoveredServer.Url, &discoveredServer.Trusted)
		}

		responseBody, err := json.Marshal(callResourceResponseBody{Data: *serversData})
		if err != nil {
			errorMessage := "Unable to serialize discovery response."
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

func (d *SampleDatasource) fetchDiscoverableServers(authHeader *string) (*[]byte, *[]discoveredServer, error) {
	var discoveryEndpointUrl = d.discoveryEndpointUrl.String() + "/grafana/discovery"
	log.DefaultLogger.Debug("Calling discovery endpoint", "discoveryEndpointUrl", discoveryEndpointUrl)

	request, err := http.NewRequest(http.MethodGet, discoveryEndpointUrl, nil)
	if err != nil {
		log.DefaultLogger.Error("Unable to create discovery request.", "error", err)
		return nil, nil, err
	}

	if authHeader != nil {
		request.Header.Set(backend.OAuthIdentityTokenHeaderName, *authHeader)
	}

	resp, err := d.httpClient.Do(request)
	if err != nil {
		log.DefaultLogger.Error(err.Error())
		return nil, nil, fmt.Errorf("unable to receive discovery response")
	}

	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("received unexpected HTTP status code: %d", resp.StatusCode)
	}

	serversData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.DefaultLogger.Error(err.Error())
		return nil, nil, fmt.Errorf("unable to read discovery response")
	}

	var discoveredServers []discoveredServer
	err = json.Unmarshal(serversData, &discoveredServers)
	if err != nil {
		log.DefaultLogger.Error(err.Error())
		return &serversData, nil, fmt.Errorf("unable to unmarshal discovery response")
	}

	return &serversData, &discoveredServers, nil
}

func (d *SampleDatasource) isServerUrlTrusted(url string, fetchIfMissing bool, authHeader *string) bool {
	isServerUrlTrusted, err := d.serverUrlTrustedMap.Get(url)
	if err == nil {
		return *isServerUrlTrusted
	}

	if fetchIfMissing {
		_, discoveredServers, fetchErr := d.fetchDiscoverableServers(authHeader)
		if fetchErr != nil {
			log.DefaultLogger.Error("Unable to fetch trusted status of server URL", "url", url, "error", err)
			return false
		}

		for _, discoveredServer := range *discoveredServers {
			d.serverUrlTrustedMap.Set(discoveredServer.Url, &discoveredServer.Trusted)
		}

		return d.isServerUrlTrusted(url, false, nil)
	}

	log.DefaultLogger.Error("Unable to determine trusted status of server URL", "url", url, "error", err)
	return false
}
