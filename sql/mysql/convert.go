// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"fmt"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
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
		if f == tVarBinary {
			// Zero is also a valid length.
			f = fmt.Sprintf("%s(%d)", f, t.Size)
		}
	case *schema.DecimalType:
		if f = strings.ToLower(t.T); f != tDecimal && f != tNumeric {
			return "", fmt.Errorf("mysql: unexpected decimal type: %q", t.T)
		}
		switch p, s := t.Precision, t.Scale; {
		case p < 0 || s < 0:
			return "", fmt.Errorf("mysql: decimal type must have precision > 0 and scale >= 0: %d, %d", p, s)
		case p < s:
			return "", fmt.Errorf("mysql: decimal type must have precision >= scale: %d < %d", p, s)
		case p == 0 && s == 0:
			// The default value for precision is 10 (i.e. decimal(0,0) = decimal(10)).
			p = 10
			fallthrough
		case s == 0:
			// In standard SQL, the syntax DECIMAL(M) is equivalent to DECIMAL(M,0),
			f = fmt.Sprintf("decimal(%d)", p)
		default:
			f = fmt.Sprintf("decimal(%d,%d)", p, s)
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
		if !sqlx.IsQuoted(values[i], '"', '\'') {
			values[i] = "'" + values[i] + "'"
		}
	}
	return strings.Join(values, ",")
}
