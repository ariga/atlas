// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package ydb

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/schema"
)

// FormatType converts a schema.Type to its YDB string representation.
func FormatType(typ schema.Type) (string, error) {
	var (
		formatted string
		err       error
	)

	switch t := typ.(type) {
	case *OptionalType:
		formatted = t.T
	case *schema.BoolType:
		formatted = TypeBool
	case *schema.IntegerType:
		formatted, err = formatIntegerType(t)
	case *schema.FloatType:
		formatted, err = formatFloatType(t)
	case *schema.DecimalType:
		formatted, err = formatDecimalType(t)
	case *SerialType:
		formatted = t.T
	case *schema.BinaryType:
		formatted = TypeString
	case *schema.StringType:
		formatted = TypeUtf8
	case *schema.JSONType:
		formatted, err = formatJSONType(t)
	case *YsonType:
		formatted = t.T
	case *schema.UUIDType:
		formatted = TypeUUID
	case *schema.TimeType:
		formatted, err = formatTimeType(t)
	case *schema.EnumType:
		err = errors.New("ydb: Enum can't be used as column data types for tables")
	case *schema.UnsupportedType:
		err = fmt.Errorf("ydb: unsupported type: %q", t.T)
	default:
		err = fmt.Errorf("ydb: unknown schema type: %T", t)
	}

	if err != nil {
		return "", err
	}
	return formatted, nil
}

func formatIntegerType(intType *schema.IntegerType) (string, error) {
	switch typ := strings.ToLower(intType.T); typ {
	case TypeInt8:
		if intType.Unsigned {
			return TypeUint8, nil
		}
		return TypeInt8, nil
	case TypeInt16:
		if intType.Unsigned {
			return TypeUint16, nil
		}
		return TypeInt16, nil
	case TypeInt32:
		if intType.Unsigned {
			return TypeUint32, nil
		}
		return TypeInt32, nil
	case TypeInt64:
		if intType.Unsigned {
			return TypeUint64, nil
		}
		return TypeInt64, nil
	case TypeUint8, TypeUint16, TypeUint32, TypeUint64:
		return typ, nil
	default:
		return "", fmt.Errorf("ydb: unsupported object identifier type: %q", intType.T)
	}
}

func formatFloatType(floatType *schema.FloatType) (string, error) {
	switch typ := strings.ToLower(floatType.T); typ {
	case TypeFloat, TypeDouble:
		return typ, nil
	default:
		return "", fmt.Errorf("ydb: unsupported object identifier type: %q", floatType.T)
	}
}

func formatDecimalType(decType *schema.DecimalType) (string, error) {
	if decType.Precision < 1 || decType.Precision > 35 {
		return "", fmt.Errorf("ydb: DECIMAL precision must be in [1, 35] range, but was %q", decType.Precision)
	}
	if decType.Scale < 0 || decType.Scale > decType.Precision {
		return "", fmt.Errorf("ydb: DECIMAL scale must be in [1, precision] range, but was %q", decType.Precision)
	}

	return fmt.Sprintf("%s(%d,%d)", TypeDecimal, decType.Precision, decType.Scale), nil
}

func formatJSONType(jsonType *schema.JSONType) (string, error) {
	typ := strings.ToLower(jsonType.T)
	switch typ {
	case TypeJSONDocument, TypeJSON:
		return typ, nil
	default:
		return "", fmt.Errorf("ydb: unsupported object identifier type: %q", jsonType.T)
	}
}

func formatTimeType(timeType *schema.TimeType) (string, error) {
	switch typ := strings.ToLower(timeType.T); typ {
	case TypeDate,
		TypeDate32,
		TypeDateTime,
		TypeDateTime64,
		TypeTimestamp,
		TypeTimestamp64,
		TypeInterval,
		TypeInterval64,
		TypeTzDate,
		TypeTzDate32,
		TypeTzDateTime,
		TypeTzDateTime64,
		TypeTzTimestamp,
		TypeTzTimestamp64:
		return typ, nil
	default:
		return "", fmt.Errorf("ydb: unsupported object identifier type: %q", timeType.T)
	}
}

// ParseType returns the schema.Type value represented by the given raw type.
// The raw value is expected to follow the format of input for the CREATE TABLE statement.
func ParseType(typ string) (schema.Type, error) {
	colDesc, err := parseColumn(strings.ToLower(typ))
	if err != nil {
		return nil, err
	}

	return columnType(colDesc)
}

// Nullability in YDB/YQL:
//
// YQL implements nullable types by wrapping them in Optional<T> containers.
// However, DDL statements do not support declaring arbitrary container types.
// Therefore, we assume that nested constructs (e.g. Optional<Optional<T>>) cannot
// occur when parsing types from DDL schemas.
type columnDecscriptor struct {
	strT      string
	nullable  bool
	precision int64
	scale     int64
	parts     []string
}

func parseColumn(typ string) (*columnDecscriptor, error) {
	if typ == "" {
		return nil, errors.New("ydb: unexpected empty column type")
	}

	var (
		err     error
		colDesc *columnDecscriptor
	)

	colDesc, typ = parseOptionalType(typ)

	parts := strings.FieldsFunc(typ, func(r rune) bool {
		return r == '(' || r == ')' || r == ' ' || r == ','
	})
	colDesc.strT = strings.ToLower(parts[0])
	colDesc.parts = parts

	switch colDesc.strT {
	case TypeDecimal:
		err = parseDecimalType(parts, colDesc)
	case TypeFloat:
		colDesc.precision = 24
	case TypeDouble:
		colDesc.precision = 53
	}

	if err != nil {
		return nil, err
	}
	return colDesc, nil
}

func parseOptionalType(typ string) (*columnDecscriptor, string) {
	colDesc := &columnDecscriptor{}

	if strings.HasPrefix(typ, "optional<") {
		colDesc.nullable = true
		typ = strings.TrimPrefix(typ, "optional<")
		typ = strings.TrimSuffix(typ, ">")
	}

	return colDesc, typ
}

func parseDecimalType(parts []string, colDesc *columnDecscriptor) error {
	if len(parts) < 3 {
		return errors.New("ydb: decimal should specify precision and scale")
	}

	precision, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || precision < 1 || precision > 35 {
		return fmt.Errorf("ydb: DECIMAL precision must be in range [1, 35], but was %q", parts[1])
	}

	scale, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || scale < 0 || scale > precision {
		return fmt.Errorf("ydb: DECIMAL scale must be in range [1, precision], but was %q", parts[1])
	}

	colDesc.precision = precision
	colDesc.scale = scale

	return nil
}

func columnType(colDesc *columnDecscriptor) (schema.Type, error) {
	var typ schema.Type

	if colDesc.nullable {
		colDesc.nullable = false
		innerType, err := columnType(colDesc)
		if err != nil {
			return nil, err
		}

		innerTypeStr, err := FormatType(innerType)
		if err != nil {
			return nil, err
		}

		return &OptionalType{
			T:         fmt.Sprintf("optional<%s>", innerTypeStr),
			InnerType: innerType,
		}, nil
	}

	switch strT := colDesc.strT; strT {
	case TypeBool:
		typ = &schema.BoolType{T: strT}
	case TypeInt8, TypeInt16, TypeInt32, TypeInt64:
		typ = &schema.IntegerType{
			T:        strT,
			Unsigned: false,
		}
	case TypeUint8, TypeUint16, TypeUint32, TypeUint64:
		typ = &schema.IntegerType{
			T:        strT,
			Unsigned: true,
		}
	case TypeFloat, TypeDouble:
		typ = &schema.FloatType{
			T:         strT,
			Precision: int(colDesc.precision),
		}
	case TypeDecimal:
		typ = &schema.DecimalType{
			T:         strT,
			Precision: int(colDesc.precision),
			Scale:     int(colDesc.scale),
		}
	case TypeSmallSerial, TypeSerial2, TypeSerial, TypeSerial4, TypeSerial8, TypeBigSerial:
		typ = &SerialType{T: strT}
	case TypeString:
		typ = &schema.BinaryType{T: strT}
	case TypeUtf8:
		typ = &schema.StringType{T: strT}
	case TypeJSON, TypeJSONDocument:
		typ = &schema.JSONType{T: strT}
	case TypeYson:
		typ = &YsonType{T: strT}
	case TypeUUID:
		typ = &schema.UUIDType{T: strT}
	case TypeDate,
		TypeDate32,
		TypeDateTime,
		TypeDateTime64,
		TypeTimestamp,
		TypeTimestamp64,
		TypeInterval,
		TypeInterval64,
		TypeTzDate,
		TypeTzDate32,
		TypeTzDateTime,
		TypeTzDateTime64,
		TypeTzTimestamp,
		TypeTzTimestamp64:
		typ = &schema.TimeType{T: strT}
	default:
		typ = &schema.UnsupportedType{T: strT}
	}

	return typ, nil
}
