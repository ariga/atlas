// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package ydb

import (
	"ariga.io/atlas/sql/schema"
)

// YDB primitive type names as defined in YDB documentation.
// See: https://ydb.tech/docs/en/yql/reference/types/
const (
	TypeBool = "bool"

	TypeInt8   = "int8"
	TypeInt16  = "int16"
	TypeInt32  = "int32"
	TypeInt64  = "int64"
	TypeUint8  = "uint8"
	TypeUint16 = "uint16"
	TypeUint32 = "uint32"
	TypeUint64 = "uint64"

	TypeFloat   = "float"
	TypeDouble  = "double"
	TypeDecimal = "decimal"

	TypeSmallSerial = "smallserial"
	TypeSerial2     = "serial2"
	TypeSerial      = "serial"
	TypeSerial4     = "serial4"
	TypeSerial8     = "serial8"
	TypeBigSerial   = "bigserial"

	TypeString       = "string"
	TypeUtf8         = "utf8"
	TypeJSON         = "json"
	TypeJSONDocument = "jsondocument"
	TypeYson         = "yson"
	TypeUUID         = "uuid"

	TypeDate        = "date"
	TypeDate32      = "date32"
	TypeDateTime    = "datetime"
	TypeDateTime64  = "datetime64"
	TypeTimestamp   = "timestamp"
	TypeTimestamp64 = "timestamp64"
	TypeInterval    = "interval"
	TypeInterval64  = "interval64"

	TypeTzDate        = "tzdate"
	TypeTzDate32      = "tzdate32"
	TypeTzDateTime    = "tzdatetime"
	TypeTzDateTime64  = "tzdatetime64"
	TypeTzTimestamp   = "tztimestamp"
	TypeTzTimestamp64 = "tztimestamp64"
)

type (
	// OptionalType represents nullable type
	OptionalType struct {
		schema.Type
		T         string
		InnerType schema.Type
	}

	// SerialType is used to implement type with auto increment
	SerialType struct {
		schema.Type
		T string
	}

	// YsonType represents YSON - JSON-like data format
	YsonType struct {
		schema.Type
		T string
	}
)

// Creates [SerialType] from corresponding [schema.IntegerType]
func SerialFromInt(intType *schema.IntegerType) *SerialType {
	serialType := &SerialType{}
	serialType.SetType(intType)
	return serialType
}

// Converts [SerialType] to corresponding [schema.IntegerType]
func (s *SerialType) IntegerType() *schema.IntegerType {
	t := &schema.IntegerType{T: TypeInt64}
	switch s.T {
	case TypeSerial2, TypeSmallSerial:
		t.T = TypeInt16
	case TypeSerial4, TypeSerial:
		t.T = TypeInt32
	case TypeSerial8, TypeBigSerial:
		t.T = TypeInt64
	}
	return t
}

// Sets [schema.IntegerType] as base underlying type for [SerialType]
func (s *SerialType) SetType(t *schema.IntegerType) {
	switch t.T {
	case TypeInt8, TypeInt16:
		s.T = TypeSerial2
	case TypeInt32:
		s.T = TypeSerial4
	case TypeInt64:
		s.T = TypeSerial8
	}
}
