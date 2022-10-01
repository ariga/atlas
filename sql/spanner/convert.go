// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package spanner

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/schema"
)

// FormatType converts schema type to its column form in the database.
func FormatType(t schema.Type) (string, error) {
	var f string
	switch t := t.(type) {
	case *schema.BoolType:
		f = t.T
	case *schema.BinaryType:
		f = t.T
	case *schema.EnumType:
		f = t.T
	case *schema.IntegerType:
		f = t.T
	case *schema.StringType:
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
	case *BytesType:
		siz := fmt.Sprint(t.Size)
		if t.Size == -1 {
			siz = "MAX"
		}
		f = fmt.Sprintf("%v(%v)", t.T, siz)
	case *StringType:
		siz := fmt.Sprint(t.Size)
		if t.Size == -1 {
			siz = "MAX"
		}
		f = fmt.Sprintf("%v(%v)", t.T, siz)
	case *schema.UnsupportedType:
		return "", fmt.Errorf("spanner: unsupported type: %T(%q)", t, t.T)
	default:
		return "", fmt.Errorf("spanner: invalid schema type: %T", t)
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
		return &StringType{
			T:         TypeString,
			Size:      cd.size,
			SizeIsMax: cd.sizeIsMax,
		}, nil
	case TypeBytes:
		return &BytesType{
			T:         TypeBytes,
			Size:      cd.size,
			SizeIsMax: cd.sizeIsMax,
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
		return &schema.TimeType{
			T: TypeBool,
		}, nil
	default:
		return &schema.UnsupportedType{
			T: c,
		}, nil
	}
}

func parseColumn(s string) (*columnDesc, error) {
	cd := &columnDesc{}
	// split up type into, base type, size, and other modifiers.
	re := regexp.MustCompile(`(\w+)(?:\((-?\d+|MAX)\))?`)
	m := re.FindStringSubmatch(strings.ToUpper(s))
	if len(m) == 0 {
		return nil, fmt.Errorf("parseColumn: invalid type: %q", s)
	}
	cd.typ = m[1]
	if len(m) > 2 {
		size, _ := strconv.Atoi(m[2])
		cd.size = size
		if m[2] == "max" {
			cd.sizeIsMax = true
		}
	}
	return cd, nil
}

// columnDesc represents a column descriptor.
type columnDesc struct {
	typ       string
	size      int
	sizeIsMax bool
}
