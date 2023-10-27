/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package framefactory

import (
	"encoding/json"
	"fmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"grafana-esp-plugin/internal/esp/windowevent"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

const OpcodeFieldName = "@opcode"

func NewWindowEventFrame(windowEvent windowevent.WindowEvent) *data.Frame {
	frame := data.NewFrame("response")
	frame = populateFrameWithField(frame, "@timestamp", windowEvent.Time)
	frame = populateFrameWithField(frame, OpcodeFieldName, windowEvent.Opcode)

	for _, field := range windowEvent.Fields {
		fieldName := field.Name
		fieldValue := field.Value

		switch fieldValue.(type) {
		case int8:
			populateFrameWithField(frame, fieldName, fieldValue.(int8))
		case *int8:
			populateFrameWithField(frame, fieldName, fieldValue.(*int8))
		case int16:
			populateFrameWithField(frame, fieldName, fieldValue.(int16))
		case *int16:
			populateFrameWithField(frame, fieldName, fieldValue.(*int16))
		case int32:
			populateFrameWithField(frame, fieldName, fieldValue.(int32))
		case *int32:
			populateFrameWithField(frame, fieldName, fieldValue.(*int32))
		case int64:
			populateFrameWithField(frame, fieldName, fieldValue.(int64))
		case *int64:
			populateFrameWithField(frame, fieldName, fieldValue.(*int64))
		case uint8:
			populateFrameWithField(frame, fieldName, fieldValue.(uint8))
		case *uint8:
			populateFrameWithField(frame, fieldName, fieldValue.(*uint8))
		case uint16:
			populateFrameWithField(frame, fieldName, fieldValue.(uint16))
		case *uint16:
			populateFrameWithField(frame, fieldName, fieldValue.(*uint16))
		case uint32:
			populateFrameWithField(frame, fieldName, fieldValue.(uint32))
		case *uint32:
			populateFrameWithField(frame, fieldName, fieldValue.(*uint32))
		case uint64:
			populateFrameWithField(frame, fieldName, fieldValue.(uint64))
		case *uint64:
			populateFrameWithField(frame, fieldName, fieldValue.(*uint64))
		case float32:
			populateFrameWithField(frame, fieldName, fieldValue.(float32))
		case *float32:
			populateFrameWithField(frame, fieldName, fieldValue.(*float32))
		case float64:
			populateFrameWithField(frame, fieldName, fieldValue.(float64))
		case *float64:
			populateFrameWithField(frame, fieldName, fieldValue.(*float64))
		case string:
			populateFrameWithField(frame, fieldName, fieldValue.(string))
		case *string:
			populateFrameWithField(frame, fieldName, fieldValue.(*string))
		case bool:
			populateFrameWithField(frame, fieldName, fieldValue.(bool))
		case *bool:
			populateFrameWithField(frame, fieldName, fieldValue.(*bool))
		case time.Time:
			populateFrameWithField(frame, fieldName, fieldValue.(time.Time))
		case *time.Time:
			populateFrameWithField(frame, fieldName, fieldValue.(*time.Time))
		case json.RawMessage:
			populateFrameWithField(frame, fieldName, fieldValue.(json.RawMessage))
		case *json.RawMessage:
			populateFrameWithField(frame, fieldName, fieldValue.(*json.RawMessage))
		case data.EnumItemIndex:
			populateFrameWithField(frame, fieldName, fieldValue.(data.EnumItemIndex))
		case *data.EnumItemIndex:
			populateFrameWithField(frame, fieldName, fieldValue.(*data.EnumItemIndex))
		default:
			err := fmt.Errorf("field '%s' specified with unsupported type %T", fieldName, fieldValue)
			log.DefaultLogger.Error(err.Error())
			panic(err)
		}
	}

	return frame
}

func NewErrorFrame(errorMessage string) *data.Frame {
	frame := data.NewFrame("error")
	populateFrameWithField(frame, OpcodeFieldName, "error")
	populateFrameWithField(frame, "@error", errorMessage)

	return frame
}

func NewErrorClearFrame() *data.Frame {
	frame := data.NewFrame("error-clear")
	populateFrameWithField(frame, OpcodeFieldName, "error-clear")

	return frame
}

type grafanaSupportedFieldType interface {
	int8 | *int8 | int16 | *int16 | int32 | *int32 | int64 | *int64 | uint8 | *uint8 | uint16 | *uint16 | uint32 | *uint32 | uint64 | *uint64 | float32 | *float32 | float64 | *float64 | string | *string | bool | *bool | time.Time | *time.Time | json.RawMessage | *json.RawMessage | data.EnumItemIndex | *data.EnumItemIndex
}

func populateFrameWithField[T grafanaSupportedFieldType](frame *data.Frame, fieldName string, fieldValue T) *data.Frame {
	fieldIndex := len(frame.Fields)
	frame.Fields = append(frame.Fields,
		data.NewField(fieldName, nil, make([]T, 1)),
	)

	frame.Fields[fieldIndex].Set(0, fieldValue)

	return frame
}
