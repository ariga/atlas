// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package ydb

import (
	"testing"

	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/require"
)

func TestConvert_FormatType(t *testing.T) {
	tests := []struct {
		name     string
		typ      schema.Type
		expected string
		wantErr  bool
	}{
		// Boolean type
		{name: "bool", typ: &schema.BoolType{T: TypeBool}, expected: TypeBool},

		// Integer types (signed)
		{name: "int8", typ: &schema.IntegerType{T: TypeInt8}, expected: TypeInt8},
		{name: "int16", typ: &schema.IntegerType{T: TypeInt16}, expected: TypeInt16},
		{name: "int32", typ: &schema.IntegerType{T: TypeInt32}, expected: TypeInt32},
		{name: "int64", typ: &schema.IntegerType{T: TypeInt64}, expected: TypeInt64},

		// Integer types (unsigned)
		{name: "uint8", typ: &schema.IntegerType{T: TypeUint8, Unsigned: true}, expected: TypeUint8},
		{name: "uint16", typ: &schema.IntegerType{T: TypeUint16, Unsigned: true}, expected: TypeUint16},
		{name: "uint32", typ: &schema.IntegerType{T: TypeUint32, Unsigned: true}, expected: TypeUint32},
		{name: "uint64", typ: &schema.IntegerType{T: TypeUint64, Unsigned: true}, expected: TypeUint64},

		// Integer types (signed to unsigned conversion)
		{name: "int8_to_uint8", typ: &schema.IntegerType{T: TypeInt8, Unsigned: true}, expected: TypeUint8},
		{name: "int16_to_uint16", typ: &schema.IntegerType{T: TypeInt16, Unsigned: true}, expected: TypeUint16},
		{name: "int32_to_uint32", typ: &schema.IntegerType{T: TypeInt32, Unsigned: true}, expected: TypeUint32},
		{name: "int64_to_uint64", typ: &schema.IntegerType{T: TypeInt64, Unsigned: true}, expected: TypeUint64},

		// Float types
		{name: "float", typ: &schema.FloatType{T: TypeFloat}, expected: TypeFloat},
		{name: "double", typ: &schema.FloatType{T: TypeDouble}, expected: TypeDouble},

		// Decimal types
		{name: "decimal_with_precision_scale", typ: &schema.DecimalType{T: TypeDecimal, Precision: 10, Scale: 2}, expected: "decimal(10,2)"},
		{name: "decimal_with_precision_zero_scale", typ: &schema.DecimalType{T: TypeDecimal, Precision: 10, Scale: 0}, expected: "decimal(10,0)"},
		{name: "decimal_max_precision", typ: &schema.DecimalType{T: TypeDecimal, Precision: 35, Scale: 10}, expected: "decimal(35,10)"},
		{name: "decimal_invalid_precision_zero", typ: &schema.DecimalType{T: TypeDecimal, Precision: 0, Scale: 0}, wantErr: true},
		{name: "decimal_invalid_precision_too_large", typ: &schema.DecimalType{T: TypeDecimal, Precision: 36, Scale: 0}, wantErr: true},

		// Serial types
		{name: "smallserial", typ: &SerialType{T: TypeSmallSerial}, expected: TypeSmallSerial},
		{name: "serial2", typ: &SerialType{T: TypeSerial2}, expected: TypeSerial2},
		{name: "serial", typ: &SerialType{T: TypeSerial}, expected: TypeSerial},
		{name: "serial4", typ: &SerialType{T: TypeSerial4}, expected: TypeSerial4},
		{name: "serial8", typ: &SerialType{T: TypeSerial8}, expected: TypeSerial8},
		{name: "bigserial", typ: &SerialType{T: TypeBigSerial}, expected: TypeBigSerial},

		// String/Binary types
		{name: "string", typ: &schema.BinaryType{T: TypeString}, expected: TypeString},
		{name: "utf8", typ: &schema.StringType{T: TypeUtf8}, expected: TypeUtf8},

		// JSON types
		{name: "json", typ: &schema.JSONType{T: TypeJSON}, expected: TypeJSON},
		{name: "jsondocument", typ: &schema.JSONType{T: TypeJSONDocument}, expected: TypeJSONDocument},

		// YSON type
		{name: "yson", typ: YsonType{T: TypeYson}, expected: TypeYson},

		// UUID type
		{name: "uuid", typ: &schema.UUIDType{T: TypeUUID}, expected: TypeUUID},

		// Date/Time types
		{name: "date", typ: &schema.TimeType{T: TypeDate}, expected: TypeDate},
		{name: "date32", typ: &schema.TimeType{T: TypeDate32}, expected: TypeDate32},
		{name: "datetime", typ: &schema.TimeType{T: TypeDateTime}, expected: TypeDateTime},
		{name: "datetime64", typ: &schema.TimeType{T: TypeDateTime64}, expected: TypeDateTime64},
		{name: "timestamp", typ: &schema.TimeType{T: TypeTimestamp}, expected: TypeTimestamp},
		{name: "timestamp64", typ: &schema.TimeType{T: TypeTimestamp64}, expected: TypeTimestamp64},
		{name: "interval", typ: &schema.TimeType{T: TypeInterval}, expected: TypeInterval},
		{name: "interval64", typ: &schema.TimeType{T: TypeInterval64}, expected: TypeInterval64},

		// Timezone-aware date/time types
		{name: "tzdate", typ: &schema.TimeType{T: TypeTzDate}, expected: TypeTzDate},
		{name: "tzdate32", typ: &schema.TimeType{T: TypeTzDate32}, expected: TypeTzDate32},
		{name: "tzdatetime", typ: &schema.TimeType{T: TypeTzDateTime}, expected: TypeTzDateTime},
		{name: "tzdatetime64", typ: &schema.TimeType{T: TypeTzDateTime64}, expected: TypeTzDateTime64},
		{name: "tztimestamp", typ: &schema.TimeType{T: TypeTzTimestamp}, expected: TypeTzTimestamp},
		{name: "tztimestamp64", typ: &schema.TimeType{T: TypeTzTimestamp64}, expected: TypeTzTimestamp64},

		// Optional type
		{name: "optional_int32", typ: OptionalType{T: "Optional<int32>", InnerType: &schema.IntegerType{T: TypeInt32}}, expected: "Optional<int32>"},

		// Error cases
		{name: "unsupported_type", typ: &schema.UnsupportedType{T: "unknown"}, wantErr: true},
		{name: "unsupported_integer_type", typ: &schema.IntegerType{T: "unsupported"}, wantErr: true},
		{name: "unsupported_float_type", typ: &schema.FloatType{T: "unsupported"}, wantErr: true},
		{name: "unsupported_json_type", typ: &schema.JSONType{T: "unsupported"}, wantErr: true},
		{name: "unsupported_time_type", typ: &schema.TimeType{T: "unsupported"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatType(tt.typ)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestConvert_ParseType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected schema.Type
		wantErr  bool
	}{
		// Boolean type
		{name: "bool", input: TypeBool, expected: &schema.BoolType{T: TypeBool}},

		// Integer types (signed)
		{name: "int8", input: TypeInt8, expected: &schema.IntegerType{T: TypeInt8, Unsigned: false}},
		{name: "int16", input: TypeInt16, expected: &schema.IntegerType{T: TypeInt16, Unsigned: false}},
		{name: "int32", input: TypeInt32, expected: &schema.IntegerType{T: TypeInt32, Unsigned: false}},
		{name: "int64", input: TypeInt64, expected: &schema.IntegerType{T: TypeInt64, Unsigned: false}},

		// Integer types (unsigned)
		{name: "uint8", input: TypeUint8, expected: &schema.IntegerType{T: TypeUint8, Unsigned: true}},
		{name: "uint16", input: TypeUint16, expected: &schema.IntegerType{T: TypeUint16, Unsigned: true}},
		{name: "uint32", input: TypeUint32, expected: &schema.IntegerType{T: TypeUint32, Unsigned: true}},
		{name: "uint64", input: TypeUint64, expected: &schema.IntegerType{T: TypeUint64, Unsigned: true}},

		// Float types
		{name: "float", input: TypeFloat, expected: &schema.FloatType{T: TypeFloat, Precision: 24}},
		{name: "double", input: TypeDouble, expected: &schema.FloatType{T: TypeDouble, Precision: 53}},

		// Decimal types
		{name: "decimal", input: "decimal(10,2)", expected: &schema.DecimalType{T: TypeDecimal, Precision: 10, Scale: 2}},
		{name: "decimal_max_precision", input: "decimal(35,10)", expected: &schema.DecimalType{T: TypeDecimal, Precision: 35, Scale: 10}},
		{name: "decimal_zero_scale", input: "decimal(10,0)", expected: &schema.DecimalType{T: TypeDecimal, Precision: 10, Scale: 0}},

		// Serial types
		{name: "smallserial", input: TypeSmallSerial, expected: &SerialType{T: TypeSmallSerial}},
		{name: "serial2", input: TypeSerial2, expected: &SerialType{T: TypeSerial2}},
		{name: "serial", input: TypeSerial, expected: &SerialType{T: TypeSerial}},
		{name: "serial4", input: TypeSerial4, expected: &SerialType{T: TypeSerial4}},
		{name: "serial8", input: TypeSerial8, expected: &SerialType{T: TypeSerial8}},
		{name: "bigserial", input: TypeBigSerial, expected: &SerialType{T: TypeBigSerial}},

		// String/Binary types
		{name: "string", input: TypeString, expected: &schema.BinaryType{T: TypeString}},
		{name: "utf8", input: TypeUtf8, expected: &schema.StringType{T: TypeUtf8}},

		// JSON types
		{name: "json", input: TypeJSON, expected: &schema.JSONType{T: TypeJSON}},
		{name: "jsonDocument", input: TypeJSONDocument, expected: &schema.JSONType{T: TypeJSONDocument}},

		// YSON type
		{name: "yson", input: TypeYson, expected: &YsonType{T: TypeYson}},

		// UUID type
		{name: "uuid", input: TypeUUID, expected: &schema.UUIDType{T: TypeUUID}},

		// Date/Time types
		{name: "date", input: TypeDate, expected: &schema.TimeType{T: TypeDate}},
		{name: "date32", input: TypeDate32, expected: &schema.TimeType{T: TypeDate32}},
		{name: "datetime", input: TypeDateTime, expected: &schema.TimeType{T: TypeDateTime}},
		{name: "datetime64", input: TypeDateTime64, expected: &schema.TimeType{T: TypeDateTime64}},
		{name: "timestamp", input: TypeTimestamp, expected: &schema.TimeType{T: TypeTimestamp}},
		{name: "timestamp64", input: TypeTimestamp64, expected: &schema.TimeType{T: TypeTimestamp64}},
		{name: "interval", input: TypeInterval, expected: &schema.TimeType{T: TypeInterval}},
		{name: "interval64", input: TypeInterval64, expected: &schema.TimeType{T: TypeInterval64}},

		// Timezone-aware date/time types
		{name: "tzdate", input: TypeTzDate, expected: &schema.TimeType{T: TypeTzDate}},
		{name: "tzdate32", input: TypeTzDate32, expected: &schema.TimeType{T: TypeTzDate32}},
		{name: "tzdatetime", input: TypeTzDateTime, expected: &schema.TimeType{T: TypeTzDateTime}},
		{name: "tzdatetime64", input: TypeTzDateTime64, expected: &schema.TimeType{T: TypeTzDateTime64}},
		{name: "tztimestamp", input: TypeTzTimestamp, expected: &schema.TimeType{T: TypeTzTimestamp}},
		{name: "tztimestamp64", input: TypeTzTimestamp64, expected: &schema.TimeType{T: TypeTzTimestamp64}},

		// Optional types
		{name: "optional_int32", input: "Optional<int32>", expected: &OptionalType{T: "optional<int32>", InnerType: &schema.IntegerType{T: TypeInt32, Unsigned: false}}},
		{name: "optional_utf8", input: "Optional<utf8>", expected: &OptionalType{T: "optional<utf8>", InnerType: &schema.StringType{T: TypeUtf8}}},
		{name: "optional_bool", input: "Optional<bool>", expected: &OptionalType{T: "optional<bool>", InnerType: &schema.BoolType{T: TypeBool}}},

		// Unsupported/unknown types
		{name: "unknown_type", input: "unknown", expected: &schema.UnsupportedType{T: "unknown"}},

		// Error cases
		{name: "empty_string", input: "", wantErr: true},
		{name: "invalid_decimal_missing_params", input: "decimal", wantErr: true},
		{name: "invalid_decimal_precision", input: "decimal(100,2)", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseType(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}
