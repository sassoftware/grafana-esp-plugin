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
	"io"

	"net/http"
	"net/url"

	"grafana-esp-plugin/internal/esp/client"
	"grafana-esp-plugin/internal/esp/windowevent"
	"grafana-esp-plugin/internal/framefactory"
	"grafana-esp-plugin/internal/plugin/channelquerystore"
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
func NewSampleDatasource(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	opts, err := settings.HTTPClientOptions()
	if err != nil {
		return nil, fmt.Errorf("http client options: %w", err)
	}
	opts.ForwardHTTPHeaders = true

	cl, err := httpclient.New(opts)
	if err != nil {
		return nil, fmt.Errorf("httpclient new: %w", err)
	}

	log.DefaultLogger.Debug(fmt.Sprintf("created data source with ForwardHTTPHeaders option set to: %v", opts.ForwardHTTPHeaders))

	return &SampleDatasource{
		channelQueryStore: channelquerystore.New(),
		httpClient:        cl,
	}, nil
}

// SampleDatasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type SampleDatasource struct {
	channelQueryStore *channelquerystore.ChannelQueryStore
	httpClient        *http.Client
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
	// create response struct
	response := backend.NewQueryDataResponse()

	for _, q := range req.Queries {
		res := d.query(ctx, req.PluginContext.DataSourceInstanceSettings.UID, q.JSON)

		response.Responses[q.RefID] = res
	}

	return response, nil
}

func (d *SampleDatasource) query(_ context.Context, datasourceUid string, queryJson json.RawMessage) backend.DataResponse {
	var qdto querydto.QueryDTO
	err := json.Unmarshal(queryJson, &qdto)
	if err != nil {
		return handleQueryError("invalid query", err)
	}
	s, err := server.FromUrlString(qdto.ServerUrl)
	if err != nil {
		return handleQueryError("invalid server URL", err)
	}

	q := query.New(*s, qdto.ProjectName, qdto.CqName, qdto.WindowName, qdto.Interval, qdto.MaxDataPoints, qdto.Fields)

	channelPath, err := q.ToChannelPath()
	if err != nil {
		return handleQueryError("invalid channel path", err)
	}

	d.channelQueryStore.Store(*channelPath, q)

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
			Message: "The backend got a success response from the discovery service",
		}, nil
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

	if _, err := d.channelQueryStore.Load(req.Path); err == nil {
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

	q, err := d.channelQueryStore.Load(queryKey)
	if err != nil {
		// The channel refers to an unknown query.
		// Avoid returning the error, to prevent continuous attempts from Grafana to re-establish the stream.
		log.DefaultLogger.Error(fmt.Sprintf("query not found for channel %v", req.Path), "error", err)
		return nil
	}

	espWsClient := client.New(q.ServerUrl)
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
			d.channelQueryStore.Delete(queryKey)

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

func (d *SampleDatasource) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	var response = &backend.CallResourceResponse{
		Status:  http.StatusNotFound,
		Headers: map[string][]string{},
	}

	switch req.Path {
	case "servers":
		discoveryServiceUrl, err := url.Parse(req.PluginContext.DataSourceInstanceSettings.URL)
		if err != nil {
			return err
		}

		var discoveryResponse *http.Response
		discoveryResponse, err = callDiscoveryEndpoint(ctx, d.httpClient, *discoveryServiceUrl)
		if err != nil {
			errorMessage := "Unable to obtain ESP server schema information."
			response.Body = []byte(errorMessage)
			response.Status = http.StatusInternalServerError
			log.DefaultLogger.Error(errorMessage, "error", err)
			return sender.Send(response)
		}

		responseBody, err := io.ReadAll(discoveryResponse.Body)
		if err != nil {
			errorMessage := "Unable to read discovery response."
			response.Body = []byte(errorMessage)
			response.Status = http.StatusInternalServerError
			log.DefaultLogger.Error(errorMessage, "error", err)
			return sender.Send(response)
		}

		response.Status = http.StatusOK
		response.Body = responseBody
		response.Headers["Content-Encoding"] = []string{discoveryResponse.Header.Get("Content-Encoding")}
		response.Headers["Content-Type"] = []string{discoveryResponse.Header.Get("Content-Type")}
	default:
		break
	}

	return sender.Send(response)
}

func callDiscoveryEndpoint(ctx context.Context, httpClient *http.Client, discoveryServiceUrl url.URL) (*http.Response, error) {
	var discoveryEndpointUrl = discoveryServiceUrl.String() + "/grafana/discovery"
	log.DefaultLogger.Debug("Calling discovery endpoint", "discoveryEndpointUrl", discoveryEndpointUrl)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryEndpointUrl, nil)
	if err != nil {
		log.DefaultLogger.Error("Unable to create discovery request.", "error", err)
		return nil, err
	}

	resp, err := httpClient.Do(request)
	if err != nil {
		log.DefaultLogger.Error("Unable to receive discovery response.", "error", err)
		return nil, err
	}

	if resp.StatusCode != 200 {
		var err = fmt.Errorf("the discovery service sent an unexpected HTTP status code: %d", resp.StatusCode)
		return nil, err
	}

	return resp, nil
}
