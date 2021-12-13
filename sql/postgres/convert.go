// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/schema"
)

// FormatType converts schema type to its column form in the database.
// An error is returned if the type cannot be recognized.
func FormatType(t schema.Type) (string, error) {
	var f string
	switch t := t.(type) {
	case *ArrayType:
		f = strings.ToLower(t.T)
	case *BitType:
		f = strings.ToLower(t.T)
		// BIT without a length is equivalent to BIT(1).
		if f == tBit && t.Len == 0 {
			f = fmt.Sprintf("%s(1)", f)
		}
	case *schema.BoolType:
		f = strings.ToLower(t.T)
	case *schema.BinaryType:
		f = strings.ToLower(t.T)
	case *CurrencyType:
		f = strings.ToLower(t.T)
	case *schema.EnumType:
		if t.T == "" {
			return "", errors.New("postgres: missing enum type name")
		}
		f = t.T
	case *schema.IntegerType:
		switch f = strings.ToLower(t.T); f {
		case tSmallInt, tInteger, tBigInt:
		case tInt2:
			f = tSmallInt
		case tInt, tInt4:
			f = tInteger
		case tInt8:
			f = tBigInt
		}
	case *schema.StringType:
		switch f = strings.ToLower(t.T); f {
		case tText:
		// CHAR(n) is alias for CHARACTER(n). If not length was
		// specified, the definition is equivalent to CHARACTER(1).
		case tChar, tCharacter:
			n := t.Size
			if n == 0 {
				n = 1
			}
			f = fmt.Sprintf("%s(%d)", tCharacter, n)
		// VARCHAR(n) is alias for CHARACTER VARYING(n). If not length
		// was specified, the type accepts strings of any size.
		case tVarChar, tCharVar:
			f = tCharVar
			if t.Size != 0 {
				f = fmt.Sprintf("%s(%d)", tCharVar, t.Size)
			}
		default:
			return "", fmt.Errorf("postgres: unexpected string type: %q", t.T)
		}
	case *schema.TimeType:
		switch f = strings.ToLower(t.T); f {
		// TIMESTAMPTZ is accepted as an abbreviation for TIMESTAMP WITH TIME ZONE.
		case tTimestampTZ:
			f = tTimestampWTZ
		// TIME be equivalent to TIME WITHOUT TIME ZONE.
		case tTime:
			f = tTimeWOTZ
		// TIMESTAMP be equivalent to TIMESTAMP WITHOUT TIME ZONE.
		case tTimestamp:
			f = tTimestampWOTZ
		}
	case *schema.FloatType:
		switch f = strings.ToLower(t.T); f {
		case tFloat4:
			f = tReal
		case tFloat8:
			f = tDouble
		}
	case *schema.DecimalType:
		switch f = strings.ToLower(t.T); f {
		case tNumeric:
		// The DECIMAL type is an alias for NUMERIC.
		case tDecimal:
			f = tNumeric
		default:
			return "", fmt.Errorf("postgres: unexpected decimal type: %q", t.T)
		}
		switch p, s := t.Precision, t.Scale; {
		case p == 0 && s == 0:
		case s < 0:
			return "", fmt.Errorf("postgres: decimal type must have scale >= 0: %d", s)
		case p == 0 && s > 0:
			return "", fmt.Errorf("postgres: decimal type must have precision between 1 and 1000: %d", p)
		case s == 0:
			f = fmt.Sprintf("%s(%d)", f, p)
		default:
			f = fmt.Sprintf("%s(%d,%d)", f, p, s)
		}
	case *SerialType:
		switch f = strings.ToLower(t.T); f {
		case tSmallSerial, tSerial, tBigSerial:
		case tSerial2:
			f = tSmallSerial
		case tSerial4:
			f = tSerial
		case tSerial8:
			f = tBigSerial
		default:
			return "", fmt.Errorf("postgres: unexpected serial type: %q", t.T)
		}
	case *schema.JSONType:
		f = strings.ToLower(t.T)
	case *UUIDType:
		f = strings.ToLower(t.T)
	case *schema.SpatialType:
		f = strings.ToLower(t.T)
	case *NetworkType:
		f = strings.ToLower(t.T)
	case *UserDefinedType:
		f = strings.ToLower(t.T)
	case *schema.UnsupportedType:
		return "", fmt.Errorf("postgres: unsupported type: %q", t.T)
	default:
		return "", fmt.Errorf("postgres: invalid schema type: %T", t)
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

// maxCharSize defines the maximum size of limited character types in Postgres (10 MB).
// https://github.com/postgres/postgres/blob/REL_13_STABLE/src/include/access/htup_details.h#L585
const maxCharSize = 10 << 20

// columnDesc represents a column descriptor.
type columnDesc struct {
	typ       string
	size      int64
	udt       string
	precision int64
	scale     int64
	typtype   string
	typid     int64
	parts     []string
}

func parseColumn(s string) (*columnDesc, error) {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '(' || r == ')' || r == ' ' || r == ','
	})
	var (
		err error
		c   = &columnDesc{
			typ:   parts[0],
			parts: parts,
		}
	)
	switch c.parts[0] {
	case tVarChar, tCharVar, tChar, tCharacter:
		if len(c.parts) > 1 {
			c.size, err = strconv.ParseInt(c.parts[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("postgres: parse size %q: %w", parts[1], err)
			}
		}
	case tDecimal, tNumeric:
		if len(parts) > 1 {
			c.precision, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("postgres: parse precision %q: %w", parts[1], err)
			}
		}
		if len(parts) > 2 {
			c.scale, err = strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("postgres: parse scale %q: %w", parts[1], err)
			}
		}
	case tBit:
		if err := parseBitParts(parts, c); err != nil {
			return nil, err
		}
	case tDouble, tFloat8:
		c.precision = 53
	case tReal, tFloat4:
		c.precision = 24
	default:
		c.typ = s
	}
	return c, nil
}

func parseBitParts(parts []string, c *columnDesc) error {
	if len(parts) == 1 {
		c.size = 1
		return nil
	}
	parts = parts[1:]
	if parts[0] == "varying" {
		c.typ = tBitVar
		parts = parts[1:]
	}
	if len(parts) == 0 {
		return nil
	}
	size, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return fmt.Errorf("postgres: parse size %q: %w", parts[1], err)
	}
	c.size = size
	return nil
}
