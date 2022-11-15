// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package spanner

import (
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// FormatType converts schema type to its column form in the database.
func FormatType(t schema.Type) (string, error) {
	var f string
	switch t := t.(type) {
	case *ArrayType:
		f = t.T
	case *schema.BoolType:
		f = t.T
	case *schema.EnumType:
		f = t.T
	case *schema.IntegerType:
		f = t.T
	case *schema.TimeType:
		f = t.T
	case *schema.FloatType:
		f = t.T
	case *schema.DecimalType:
		f = t.T
	case *schema.JSONType:
		f = t.T
	case *schema.SpatialType:
		f = t.T
	case *schema.BinaryType:
		var size string
		if t.Size == nil || sqlx.Has(t.Attrs, &MaxSize{}) {
			size = "MAX"
		}
		if size == "" && t.Size != nil {
			size = fmt.Sprint(*t.Size)
		}
		f = fmt.Sprintf("%v(%v)", t.T, size)
	case *schema.StringType:
		size := fmt.Sprint(t.Size)
		if sqlx.Has(t.Attrs, &MaxSize{}) {
			size = "MAX"
		}
		f = fmt.Sprintf("%v(%v)", t.T, size)
	case *schema.UnsupportedType:
		return "", fmt.Errorf("spanner: unsupported type: %T(%q)", t, t.T)
	default:
		return "", fmt.Errorf("spanner: invalid schema type: %T", t)
	}
	return f, nil
}

// ParseType returns the schema.Type value represented by the given raw type.
func ParseType(c string) (schema.Type, error) {
	// A datatype may be zero or more names.
	if c == "" {
		return &schema.UnsupportedType{}, nil
	}
	cd, err := parseColumn(c)
	if err != nil {
		return &schema.UnsupportedType{
			T: c,
		}, nil
	}

	switch cd.typ {
	case TypeInt64:
		return &schema.IntegerType{
			T: TypeInt64,
		}, nil
	case TypeString:
		return &schema.StringType{
			T:    TypeString,
			Size: cd.size,
		}, nil
	case TypeBytes:
		s := cd.size
		return &schema.BinaryType{
			T:    TypeBytes,
			Size: &s,
		}, nil
	case TypeTimestamp:
		return &schema.TimeType{
			T: TypeTimestamp,
		}, nil
	case TypeDate:
		return &schema.TimeType{
			T: TypeDate,
		}, nil
	case TypeBool:
		return &schema.BoolType{
			T: TypeBool,
		}, nil
	default:
		return &schema.UnsupportedType{
			T: c,
		}, nil
	}
}

// parseColumn attempts to populate a columnDesc.
func parseColumn(s string) (*columnDesc, error) {
	var err error
	cd := &columnDesc{}
	// split up type into, base type, size, and other modifiers.
	m := sizedTypeRe.FindStringSubmatch(strings.ToUpper(s))
	if len(m) == 0 {
		return nil, fmt.Errorf("parseColumn: invalid type: %q", s)
	}
	cd.typ = m[1]
	if len(m) > 2 && m[2] != "" {
		if m[2] == "MAX" {
			cd.maxSize = true
		} else {
			cd.size, err = strconv.Atoi(m[2])
			if err != nil {
				return nil, fmt.Errorf("parseColumn: unable to convert %q to int: %w", m[2], err)
			}
		}
	}
	return cd, nil
}
