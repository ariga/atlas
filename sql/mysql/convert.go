// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"fmt"
	"strings"

	"ariga.io/atlas/sql/schema"
)

// FormatType converts schema type to its column form in the database.
// An error is returned if the type cannot be recognized.
func FormatType(t schema.Type) (string, error) {
	var f string
	switch t := t.(type) {
	case *BitType:
		f = strings.ToLower(t.T)
	case *schema.BoolType:
		// Map all flavors to a single form.
		switch f = strings.ToLower(t.T); f {
		case "bool", "boolean", "tinyint", "tinyint(1)":
			f = "bool"
		}
	case *schema.BinaryType:
		f = strings.ToLower(t.T)
		if f == tVarBinary || f == tBinary {
			// Zero is also a valid length.
			f = fmt.Sprintf("%s(%d)", f, t.Size)
		}
	case *schema.DecimalType:
		f = strings.ToLower(t.T)
		if f == tDecimal || f == tNumeric {
			// In MySQL, NUMERIC is implemented as DECIMAL.
			f = fmt.Sprintf("decimal(%d,%d)", t.Precision, t.Scale)
		}
	case *schema.EnumType:
		f = fmt.Sprintf("enum(%s)", formatValues(t.Values))
	case *schema.FloatType:
		f = strings.ToLower(t.T)
		// FLOAT with precision > 24, become DOUBLE.
		// Also, REAL is a synonym for DOUBLE (if REAL_AS_FLOAT was not set).
		if f == tFloat && t.Precision > 24 || f == tReal {
			f = tDouble
		}
	case *schema.IntegerType:
		f = strings.ToLower(t.T)
		if t.Unsigned {
			f += " unsigned"
		}
	case *schema.JSONType:
		f = strings.ToLower(t.T)
	case *SetType:
		f = fmt.Sprintf("enum(%s)", formatValues(t.Values))
	case *schema.StringType:
		f = strings.ToLower(t.T)
		if f == tChar || f == tVarchar {
			// Zero is also a valid length.
			f = fmt.Sprintf("%s(%d)", f, t.Size)
		}
	case *schema.SpatialType:
		f = strings.ToLower(t.T)
	case *schema.TimeType:
		f = strings.ToLower(t.T)
	case *schema.UnsupportedType:
		return "", fmt.Errorf("mysql: unsupported type: %q", t.T)
	default:
		return "", fmt.Errorf("mysql: invalid schema type: %T", t)
	}
	return f, nil
}

// mustFormat calls to FormatType and panics in case of error.
func mustFormat(t schema.Type) string {
	s, err := FormatType(t)
	if err != nil {
		panic(err)
	}
	return s
}

// formatValues formats ENUM and SET values.
func formatValues(vs []string) string {
	values := make([]string, len(vs))
	for i := range vs {
		values[i] = vs[i]
		if !strings.HasPrefix(values[i], "'") || !strings.HasSuffix(values[i], "'") {
			values[i] = "'" + values[i] + "'"
		}
	}
	return strings.Join(values, ",")
}
