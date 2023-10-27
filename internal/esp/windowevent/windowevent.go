/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package windowevent

import (
	"fmt"
	"grafana-esp-plugin/internal/esp/field"
	"reflect"
	"strconv"
	"time"
)

type WindowEvent struct {
	Time   time.Time
	Opcode string
	Fields []field.Field
}

func New(eventTime time.Time, opcode string, fields []field.Field) WindowEvent {
	windowEvent := WindowEvent{
		Time:   eventTime,
		Opcode: opcode,
		Fields: fields,
	}

	return windowEvent
}

func (windowEvent WindowEvent) String() string {
	return fmt.Sprintf("WindowEvent{time=%s, opcode=%s, fields=%s}", windowEvent.Time, windowEvent.Opcode, windowEvent.Fields)
}

func ParseWindowEventTime(timestamp any) (*time.Time, error) {
	switch reflect.TypeOf(timestamp).Kind() {
	case reflect.String:
		var err error
		timestamp, err = strconv.Atoi(timestamp.(string))
		if err != nil {
			return nil, err
		}
		fallthrough
	case reflect.Int:
		timestamp = int64(timestamp.(int))
	case reflect.Uint64:
		timestamp = int64(timestamp.(uint64))
	default:
		err := fmt.Errorf("invalid argument type %T", timestamp)
		return nil, err
	}

	eventTime := time.UnixMicro(timestamp.(int64))

	return &eventTime, nil
}
