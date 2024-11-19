/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"encoding/base64"
	"encoding/json"

	"grafana-esp-plugin/internal/esp/client/messagedto"
	"grafana-esp-plugin/internal/esp/field"
	"grafana-esp-plugin/internal/esp/windowevent"

	"github.com/google/uuid"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sacOO7/gowebsocket"

	"github.com/fxamacker/cbor"
)

type EspWsClient struct {
	socket                 *gowebsocket.Socket
	isConnected            bool
	subscriptions          map[string]*subscription
	Errors                 chan error
	OnConnected            func()
	OnEventMessageReceived func(windowevent.WindowEvent)
	OnProjectLoaded        func(string)
	OnProjectRemoved       func(string)
}

type subscription struct {
	schema         map[string]field.SchemaType
	format         string
	includedFields []string
}

type messageType int

const (
	MessageTypeUnknown messageType = iota
	MessageTypeSchema
	MessageTypeEvent
	MessageTypeError
	MessageTypeProjectLoaded
	MessageTypeProjectRemoved
	MessageTypeBulk
	MessageTypeInfoDiscard
)

const jsonFormat string = "json"
const cborFormat string = "cbor"

func New(wsConnectionUrl url.URL, authorizationHeader *string) *EspWsClient {
	socket := gowebsocket.New(wsConnectionUrl.String())
	if authorizationHeader != nil {
		socket.RequestHeader.Set("Authorization", *authorizationHeader)
	}

	espWsClient := EspWsClient{
		socket:        &socket,
		isConnected:   false,
		subscriptions: make(map[string]*subscription),
		Errors:        make(chan error),
	}

	socket.OnConnected = handleConnect
	socket.OnConnectError = getConnectionErrorHandler(&espWsClient)
	socket.OnTextMessage = getTextMessageHandler(&espWsClient)
	socket.OnBinaryMessage = getBinaryMessageHandler(&espWsClient)
	socket.OnDisconnected = getDisconnectHandler(&espWsClient)

	return &espWsClient
}

func handleConnect(socket gowebsocket.Socket) {
	log.DefaultLogger.Debug(fmt.Sprintf("Opened WebSocket: %s", socket.Url))
}

func getConnectionErrorHandler(espWsClient *EspWsClient) func(err error, socket gowebsocket.Socket) {
	return func(err error, socket gowebsocket.Socket) {
		log.DefaultLogger.Error(fmt.Sprintf("WebSocket error: %s, %s", socket.Url, err))
		espWsClient.handleConnectionError()
	}
}

func getTextMessageHandler(espWsClient *EspWsClient) func(messageString string, socket gowebsocket.Socket) {
	return func(messageString string, socket gowebsocket.Socket) {
		if !espWsClient.isConnected {
			isHandshakeMessage := !espWsClient.isConnected && strings.HasPrefix(messageString, "status: 200\n")
			if isHandshakeMessage {
				espWsClient.handleHandshakeSuccessful()
				return
			}
		}

		messageBytes := []byte(messageString)
		var message messagedto.MessageDTO
		err := json.Unmarshal(messageBytes, &message)
		if err != nil {
			log.DefaultLogger.Error(fmt.Sprintf("Cannot unmarshal messageString: %s", messageString))
			return
		}

		espWsClient.handleMessage(message, messageBytes)
	}
}

func getBinaryMessageHandler(espWsClient *EspWsClient) func(data []byte, socket gowebsocket.Socket) {
	return func(data []byte, socket gowebsocket.Socket) {
		if !espWsClient.isConnected {
			messageString := string(data)[:]
			isHandshakeMessage := !espWsClient.isConnected && strings.HasPrefix(messageString, "status: 200\n")
			if isHandshakeMessage {
				espWsClient.handleHandshakeSuccessful()
				return
			}
		}

		var message *messagedto.MessageDTO
		if isBinaryMessageCborEncoded(&data) {
			var err error
			message, err = decodeCborMessage(&data)
			if err != nil {
				log.DefaultLogger.Error(fmt.Sprintf("Cannot unmarshal CBOR message: %v", data))
				return
			}
		} else {
			message = new(messagedto.MessageDTO)
			err := json.Unmarshal(data, message)
			if err != nil {
				log.DefaultLogger.Error(fmt.Sprintf("Cannot unmarshal message: %v", data))
				return
			}
		}

		espWsClient.handleMessage(*message, data)
	}
}

func getDisconnectHandler(espWsClient *EspWsClient) func(err error, socket gowebsocket.Socket) {
	return func(err error, socket gowebsocket.Socket) {
		log.DefaultLogger.Debug(fmt.Sprintf("WebSocket closed: %v, %v", socket.Url, err))
		espWsClient.handleConnectionClosed()

		if err != nil {
			espWsClient.handleConnectionError()
		}
	}
}

func (espWsClient *EspWsClient) handleDisconnect() {
	espWsClient.socket.Connect()
}

func (espWsClient *EspWsClient) Connect() {
	espWsClient.socket.Connect()
}

func (espWsClient *EspWsClient) Close() {
	if espWsClient.socket.Conn != nil {
		espWsClient.socket.Close()
		espWsClient.handleConnectionClosed()
	}
}

func (espWsClient *EspWsClient) Subscribe(projectName string, cqName string, windowName string, interval uint64, maxEvents uint64, fields []string) error {
	subscriptionFormat := cborFormat
	windowPath := fmt.Sprintf("%s/%s/%s", projectName, cqName, windowName)
	subscriptionId := fmt.Sprintf("%s/%s", windowPath, uuid.New().String())
	eventStream := messagedto.StreamMessageDTO{
		Action:        "set",
		Id:            subscriptionId,
		Window:        windowPath,
		Schema:        true,
		UpdateDeletes: true,
		Format:        subscriptionFormat,
		Interval:      interval,
		MaxEvents:     maxEvents,
		IncludeFields: fields,
	}

	subscriptionMessage := messagedto.SubscriptionMessageDTO{
		EventStream: eventStream,
	}

	subscriptionMessageBytes, err := json.Marshal(subscriptionMessage)
	if err != nil {
		return err
	}

	sub := new(subscription)
	sub.format = subscriptionFormat
	sub.includedFields = fields
	espWsClient.subscriptions[subscriptionId] = sub

	espWsClient.socket.SendText(string(subscriptionMessageBytes))
	log.DefaultLogger.Debug(fmt.Sprintf("Subscribed to: %s", subscriptionMessageBytes))

	return nil
}

func (espWsClient *EspWsClient) handleBulkMessage(encodedMessages *[]string) {
	if encodedMessages == nil {
		return
	}

	for _, encodedString := range *encodedMessages {
		decodedMessage, err := decodeBulkMessageString(encodedString)
		if err != nil {
			log.DefaultLogger.Error(fmt.Sprintf("cannot decode base64 message: %s", encodedString))
			continue
		}

		var message messagedto.MessageDTO
		err = json.Unmarshal(*decodedMessage, &message)
		if err != nil {
			log.DefaultLogger.Error(fmt.Sprintf("Cannot unmarshal message: %s", *decodedMessage))
			return
		}

		espWsClient.handleMessage(message, *decodedMessage)
	}
}

func (espWsClient *EspWsClient) determineMessageType(message *messagedto.MessageDTO) messageType {
	if message.Schema != nil {
		return MessageTypeSchema
	}

	if message.Events != nil {
		return MessageTypeEvent
	}

	if message.Error != nil {
		return MessageTypeError
	}

	if message.ProjectLoaded != nil {
		return MessageTypeProjectLoaded
	}

	if message.ProjectRemoved != nil {
		return MessageTypeProjectRemoved
	}

	if message.Bulk != nil {
		return MessageTypeBulk
	}

	if message.Info != nil && message.Info.Type == "event_source_discard" {
		return MessageTypeInfoDiscard
	}

	return MessageTypeUnknown
}

func (espWsClient *EspWsClient) handleMessage(message messagedto.MessageDTO, messageBytes []byte) {
	messageType := espWsClient.determineMessageType(&message)

	switch messageType {
	case MessageTypeBulk:
		espWsClient.handleBulkMessage(message.Bulk)
	case MessageTypeSchema:
		espWsClient.handleSchemaMessage(message.Schema)
	case MessageTypeEvent:
		espWsClient.handleEventMessage(message.Events)
	case MessageTypeError:
		espWsClient.handleErrorMessage(message.Error)
	case MessageTypeProjectLoaded:
		espWsClient.handleProjectLoadedMessage(message.ProjectLoaded)
	case MessageTypeProjectRemoved:
		espWsClient.handleProjectRemovedMessage(message.ProjectRemoved)
	case MessageTypeInfoDiscard:
		messageData := message.Info.Data
		formattedMessage := fmt.Sprintf("Events discarded: %d out of %d", messageData.Discarded, messageData.Total)
		log.DefaultLogger.Info(fmt.Sprintf("Received 'info' message: %s", formattedMessage))
	default:
		log.DefaultLogger.Error(fmt.Sprintf("Unknown message type received. Message: %s", messageBytes))
	}
}

func (espWsClient *EspWsClient) handleSchemaMessage(message *messagedto.SchemaMessageDTO) {
	fieldTypeMap := make(map[string]field.SchemaType)
	for _, f := range message.Fields {
		var ft field.SchemaType
		ft, err := field.ParseFieldTypeFromString(f.Type)
		if err != nil {
			espWsClient.Errors <- err
			return
		}

		fieldTypeMap[f.Name] = ft
	}

	espWsClient.subscriptions[message.SubscriptionId].schema = fieldTypeMap
}

func (espWsClient *EspWsClient) handleErrorMessage(message *messagedto.ErrorMessageDTO) {
	log.DefaultLogger.Error(fmt.Sprintf("Received error message: %v", message))

	espWsClient.Errors <- fmt.Errorf(message.Text)
}

func (espWsClient *EspWsClient) handleEventMessage(message *messagedto.EventMessageDTO) {
	subscriptionId := message.SubscriptionId

	log.DefaultLogger.Debug(fmt.Sprintf("Received event message, entries: %d", len(message.Entries)))

	for _, value := range message.Entries {
		espWsClient.handleEvent(subscriptionId, value)
	}
}

func (espWsClient *EspWsClient) handleEvent(subscriptionId string, event messagedto.EventEntryDTO) {
	if event == nil {
		log.DefaultLogger.Warn("received nil event", "subscriptionId", subscriptionId)
		return
	}

	sub, ok := espWsClient.subscriptions[subscriptionId]
	if !ok {
		log.DefaultLogger.Error("received event with unknown subscription id", "subscriptionId", subscriptionId)
		return
	}

	//JSON API spec inconsistency #1: event structure is unnecessarily nested inside an extra event field, unlike CBOR.
	if sub.format == jsonFormat {
		event = event["event"].(messagedto.EventEntryDTO)
	}

	windowEvent, err := espWsClient.parseWindowEvent(event, sub)
	if err != nil {
		log.DefaultLogger.Error("error while parsing window event", "error", err)
		return
	}

	if espWsClient.OnEventMessageReceived != nil {
		espWsClient.OnEventMessageReceived(*windowEvent)
	}
}

func (espWsClient *EspWsClient) parseWindowEvent(event messagedto.EventEntryDTO, sub *subscription) (*windowevent.WindowEvent, error) {
	//JSON API spec inconsistency #2: unlike CBOR structure, all field values are returned as a string regardless of schema type
	eventTimestampRaw := event["@timestamp"]
	eventTime, err := windowevent.ParseWindowEventTime(eventTimestampRaw)
	if err != nil {
		err := fmt.Errorf("error while parsing window event timestamp (%v): %s", eventTimestampRaw, err.Error())
		return nil, err
	}

	eventOpcodeRaw := event["@opcode"]
	eventOpcode, ok := eventOpcodeRaw.(string)
	if !ok {
		err := fmt.Errorf("unexpected value type %T for event opcode: %v", eventOpcodeRaw, eventOpcodeRaw)
		return nil, err
	}

	fields, err := espWsClient.parseEventFields(event, sub)
	if err != nil {
		err := fmt.Errorf("error while parsing window event fields: %s", err.Error())
		return nil, err
	}

	windowEvent := windowevent.New(*eventTime, eventOpcode, *fields)
	return &windowEvent, nil
}

func (espWsClient *EspWsClient) parseEventFields(event messagedto.EventEntryDTO, sub *subscription) (*[]field.Field, error) {
	fieldNames := make([]string, 0, len(event))
	for key := range event {
		if field.IsFieldNameInternal(key) {
			continue
		}

		fieldNames = append(fieldNames, key)
	}

	sort.Strings(fieldNames)

	fields := make([]field.Field, 0, len(fieldNames))
	for _, fieldName := range fieldNames {
		fieldValueRaw, ok := event[fieldName]
		if !ok {
			err := fmt.Errorf("field '%s' not found in event", fieldName)
			return nil, err
		}

		schemaType, schemaError := espWsClient.getSchemaFieldType(sub, fieldName)
		if schemaError != nil {
			return nil, schemaError
		}

		validationError := espWsClient.validateField(fieldName, fieldValueRaw, sub, *schemaType)
		if validationError != nil {
			err := fmt.Errorf("invalid field: %s", validationError.Error())
			return nil, err
		}

		fieldValue, fieldValueError := parseFieldValue(fieldValueRaw, sub, *schemaType)
		if fieldValueError != nil {
			err := fmt.Errorf("invalid field value: %s", fieldValueError)
			return nil, err
		}

		f := field.New(fieldName, fieldValue)
		fields = append(fields, f)
	}

	return &fields, nil
}

func parseFieldValue(rawValue any, sub *subscription, schemaType field.SchemaType) (any, error) {
	var fieldValue any
	//JSON API spec inconsistency #2: unlike CBOR structure, all field values are returned as a string regardless of schema type
	if sub.format == jsonFormat {
		value := parseJsonFieldValue(rawValue, schemaType)
		return &value, nil
	}

	switch schemaType {
	case field.Array:
		rawArray := rawValue.([]any)
		ptrArray := make([]*any, 0, len(rawArray))
		for i := range rawArray {
			c := rawArray[i]
			var ptr *any
			if floatValue, isFloat := c.(float64); isFloat && math.IsNaN(floatValue) || c == nil {
				ptr = nil
			} else {
				ptr = &rawArray[i]
			}

			ptrArray = append(ptrArray, ptr)
		}

		var err error
		fieldValue, err = json.Marshal(ptrArray)
		if err != nil {
			return nil, err
		}
		fieldValue = json.RawMessage(fieldValue.([]byte))
	case field.Blob:
		fieldValue = base64.StdEncoding.EncodeToString(rawValue.([]byte))
	case field.Timestamp:
		signed := int64(rawValue.(uint64))
		fieldValue = time.UnixMicro(signed)
	case field.Date:
		signed := int64(rawValue.(uint64))
		fieldValue = time.Unix(signed, 0)
	default:
		fieldValue = rawValue
	}
	return fieldValue, nil
}

func parseJsonFieldValue(rawValue any, schemaType field.SchemaType) any {
	var fieldValueString string
	if schemaType == field.Blob {
		fieldValueString = rawValue.(map[string]any)["*value"].(string)
	} else {
		fieldValueString = rawValue.(string)
	}

	var err error
	var fieldValue any

	switch schemaType {
	case field.Blob:
		fieldValue = fieldValueString
	case field.Double:
		fieldValue, err = strconv.ParseFloat(fieldValueString, 64)
		if err != nil {
			log.DefaultLogger.Error(fmt.Sprintf("Cannot convert field value to type double: %s", fieldValueString))
			panic(err)
		}
	case field.Int:
		fieldValue, err = strconv.ParseInt(fieldValueString, 10, 64)
		if err != nil {
			log.DefaultLogger.Error(fmt.Sprintf("Cannot convert field value to type int: %s", fieldValueString))
			panic(err)
		}
	case field.String:
		fieldValue = fieldValueString
	case field.Timestamp:
		fieldValueInt, err := strconv.ParseInt(fieldValueString, 10, 64)
		if err != nil {
			log.DefaultLogger.Error(fmt.Sprintf("Cannot convert field value to type timestamp: %s", fieldValueString))
			panic(err)
		}

		fieldValue = time.UnixMicro(fieldValueInt)
	case field.Date:
		fieldValueInt, err := strconv.ParseInt(fieldValueString, 10, 64)
		if err != nil {
			log.DefaultLogger.Error(fmt.Sprintf("Cannot convert field value to type timestamp: %s", fieldValueString))
			panic(err)
		}

		fieldValue = time.Unix(fieldValueInt, 0)
	default:
		err := fmt.Errorf("unsupported field type %d", schemaType)
		log.DefaultLogger.Error(err.Error())
		panic(err)
	}

	return fieldValue
}

func (espWsClient *EspWsClient) validateField(name string, value any, sub *subscription, fieldType field.SchemaType) error {
	//JSON API spec inconsistency #2: unlike CBOR structure, all field values are returned as a string regardless of schema type
	if sub.format == jsonFormat {
		return espWsClient.validateFieldJson(name, value)
	}

	switch fieldType {
	case field.Array:
		switch value.(type) {
		case []any:
			return nil
		}
	case field.Blob:
		switch value.(type) {
		case []byte:
			return nil
		}
	case field.Double:
		switch value.(type) {
		case float32, float64:
			return nil
		}
	case field.Int:
		switch value.(type) {
		case int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
			return nil
		}
	case field.String:
		switch value.(type) {
		case string:
			return nil
		}
	case field.Timestamp:
		switch value.(type) {
		case uint64:
			return nil
		}
	case field.Date:
		switch value.(type) {
		case uint64:
			return nil
		}
	default:
		err := fmt.Errorf("unexpected schema field type %v", fieldType)
		log.DefaultLogger.Error(err.Error())
		panic(err)
	}

	return fmt.Errorf("unexpected value type %T for field %s", value, name)
}

func (espWsClient *EspWsClient) validateFieldJson(name string, value any) error {
	switch value.(type) {
	//JSON API spec inconsistency #3: unlike CBOR, blob values are contained within a string map. The map has two keys:
	// - @type, providing the file signature (which is unnecessary since it's already contained and can be inferred from the body)
	// - value, holding a base64-encoded string of the actual blob data
	case map[string]any:
		actualValue := value.(map[string]any)["*value"]
		if actualValue == nil {
			return fmt.Errorf("blob value for field %s is nil", name)
		}

		switch actualValue.(type) {
		case string:
			return nil
		}
	case string:
		return nil
	}

	return fmt.Errorf("unexpected value type %T for JSON field %s", value, name)
}

func (espWsClient *EspWsClient) handleProjectLoadedMessage(message *messagedto.ProjectLoadedMessageDTO) {
	log.DefaultLogger.Debug(fmt.Sprintf("Received 'project-loaded' message: %v", message))

	if espWsClient.OnProjectLoaded != nil {
		espWsClient.OnProjectLoaded(message.Name)
	}
}

func (espWsClient *EspWsClient) handleProjectRemovedMessage(message *messagedto.ProjectRemovedMessageDTO) {
	log.DefaultLogger.Debug(fmt.Sprintf("Received 'project-removed' message: %v", message))

	if espWsClient.OnProjectRemoved != nil {
		espWsClient.OnProjectRemoved(message.Name)
	}
}

func (espWsClient *EspWsClient) getSchemaFieldType(sub *subscription, fieldName string) (*field.SchemaType, error) {
	if sub.schema == nil {
		return nil, fmt.Errorf("no schema received for field: %s", fieldName)
	}

	fieldType, fieldExists := sub.schema[fieldName]
	if !fieldExists {
		return nil, fmt.Errorf("no schema type found for field: %s", fieldName)
	}

	return &fieldType, nil
}

func (espWsClient *EspWsClient) handleHandshakeSuccessful() {
	espWsClient.isConnected = true

	if espWsClient.OnConnected != nil {
		espWsClient.OnConnected()
	}
}

func (espWsClient *EspWsClient) handleConnectionClosed() {
	espWsClient.isConnected = false
}

func (espWsClient *EspWsClient) handleConnectionError() {
	espWsClient.Errors <- fmt.Errorf("websocket connection error")
}

func decodeBulkMessageString(message string) (*[]byte, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(message)
	if err != nil {
		return nil, err
	}

	return &decodedBytes, nil
}

func decodeCborMessage(data *[]byte) (*messagedto.MessageDTO, error) {
	var message messagedto.MessageDTO

	if err := cbor.Unmarshal(*data, &message); err != nil {
		return nil, err
	}

	return &message, nil
}

func isBinaryMessageCborEncoded(binaryMessage *[]byte) bool {
	_, err := cbor.Valid(*binaryMessage)

	return err == nil
}
