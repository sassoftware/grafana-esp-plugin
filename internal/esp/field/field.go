/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package field

import (
	"fmt"
	"strings"
)

type Field struct {
	Name  string
	Value any
}

func New(fieldName string, fieldValue any) Field {
	field := Field{
		Name:  fieldName,
		Value: fieldValue,
	}

	return field
}

func (field Field) String() string {
	return fmt.Sprintf("Field{name=%s, value=%s}", field.Name, field.Value)
}

type SchemaType int

const (
	Array SchemaType = iota
	Blob
	Double
	Int
	Timestamp
	String
	Date
)

var (
	fieldTypeMap = map[string]SchemaType{
		"array(dbl)": Array,
		"array(i32)": Array,
		"array(i64)": Array,
		"blob":       Blob,
		"date":       Date,
		"double":     Double,
		"int32":      Int,
		"int64":      Int,
		"money":      Double,
		"rstring":    String,
		"stamp":      Timestamp,
		"string":     String,
	}
)

func ParseFieldTypeFromString(str string) (SchemaType, error) {
	fieldType, ok := fieldTypeMap[strings.ToLower(str)]
	if !ok {
		return fieldType, fmt.Errorf("unknown schema field type: %s", str)
	}
	return fieldType, nil
}

func IsFieldNameInternal(name string) bool {
	return strings.HasPrefix(name, "@")
}
