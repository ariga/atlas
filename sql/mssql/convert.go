// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mssql

import (
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
	case *BitType:
		f = strings.ToLower(t.T)
	case *schema.IntegerType:
		f = strings.ToLower(t.T)
	case *schema.DecimalType:
		switch f = strings.ToLower(t.T); f {
		case TypeDecimal, typeISODec:
			f = TypeDecimal
		case TypeNumeric:
		default:
			return "", fmt.Errorf("mssql: unexpected decimal type: %q", t.T)
		}
		switch p, s := t.Precision, t.Scale; {
		case p == 0 && s == 0: // Let the database decide the precision and scale.
		case s < 0:
			return "", fmt.Errorf("mssql: decimal type must have scale >= 0: %d", s)
		case p >= 0 && s > p:
			return "", fmt.Errorf("mssql: decimal type must have scale <= precision: %d > %d", s, p)
		case s == 0:
			f = fmt.Sprintf("%s(%d)", f, p)
		default:
			f = fmt.Sprintf("%s(%d,%d)", f, p, s)
		}
	case *MoneyType:
		f = strings.ToLower(t.T)
	case *schema.FloatType:
		switch f = strings.ToLower(t.T); f {
		case TypeReal:
		case TypeFloat, typeISODoublePrecision:
			f = TypeFloat
			switch {
			case t.Precision > 0 && t.Precision <= 24:
				f = TypeReal // float(24) is an alias for real.
			case t.Precision == 0 || (t.Precision > 24 && t.Precision <= 53):
			default:
				return "", fmt.Errorf("mssql: precision for type float must be between 1 and 53: %d", t.Precision)
			}
		default:
			return "", fmt.Errorf("mssql: unexpected float type: %q", t.T)
		}
	case *schema.StringType:
		switch f = strings.ToLower(t.T); f {
		case TypeChar, typeISOCharacter:
			f = TypeChar
		case TypeVarchar, typeISOCharVarying, typeISOCharacterVarying:
			f = TypeVarchar
		case TypeNChar, typeISONationalChar, typeISONationalCharacter:
			f = TypeNChar
		case TypeNVarchar, typeISONationalCharVarying, typeISONationalCharacterVarying:
			f = TypeNVarchar
		case TypeText: // Deprecated types
			return f, nil
		case TypeNText, typeISONationalText: // Deprecated types
			f = TypeNText
			return f, nil
		default:
			return "", fmt.Errorf("mssql: unexpected string type: %q", t.T)
		}
		switch n := t.Size; n {
		case -1:
			f = fmt.Sprintf("%s(MAX)", f)
		case 0:
			n = 1
			fallthrough
		default:
			f = fmt.Sprintf("%s(%d)", f, n)
		}
	case *schema.BinaryType:
		switch f = strings.ToLower(t.T); f {
		case TypeBinary:
		case TypeVarBinary, typeANSIBinaryVarying:
			f = TypeVarBinary
		default:
			return "", fmt.Errorf("mssql: unexpected binary type: %q", t.T)
		}
		switch n := t.Size; {
		case n == nil || *n == 0:
			f = fmt.Sprintf("%s(1)", f)
		case *n == -1:
			if f != TypeVarBinary {
				return "", fmt.Errorf("mssql: invalid size for %q: %d", f, *n)
			}
			f = fmt.Sprintf("%s(MAX)", f)
		default:
			f = fmt.Sprintf("%s(%d)", f, *n)
		}
	case *schema.TimeType:
		f = strings.ToLower(t.T)
		switch f {
		case TypeDateTime2, TypeDateTimeOffset, TypeTime:
			s := defaultTimeScale
			if t.Scale != nil {
				s = *t.Scale
			}
			f = fmt.Sprintf("%s(%d)", f, s)
		}
	case *schema.SpatialType:
		f = strings.ToLower(t.T)
	case *HierarchyIDType:
		f = strings.ToLower(t.T)
	case *UniqueIdentifierType:
		f = strings.ToLower(t.T)
	case *UserDefinedType:
		f = strings.ToLower(t.T)
	case *XMLType:
		// We don't support typed-XML yet.
		// https://learn.microsoft.com/en-us/sql/relational-databases/xml/compare-typed-xml-to-untyped-xml
		f = strings.ToLower(t.T)
	case *schema.UnsupportedType:
		return "", fmt.Errorf("mssql: unsupported type: %q", t.T)
	default:
		return "", fmt.Errorf("mssql: invalid schema type: %T", t)
	}
	return f, nil
}

// columnDesc represents a column descriptor.
type columnDesc struct {
	typ         string // data_type
	size        int64
	precision   int64
	scale       int64
	userDefined bool
}

// ParseType returns the schema.Type value represented by the given raw type.
// The raw value is expected to follow the format in PostgreSQL information schema
// or as an input for the CREATE TABLE statement.
func ParseType(typ string) (schema.Type, error) {
	d, err := parseColumn(typ)
	if err != nil {
		return nil, err
	}
	return columnType(d)
}

func parseColumn(s string) (*columnDesc, error) {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '(' || r == ')' || r == ' ' || r == ','
	})
	var (
		err error
		typ = strings.ToLower(parts[0])
		c   = &columnDesc{
			typ: typ,
		}
	)
	switch typ {
	case TypeDecimal, TypeNumeric:
		if len(parts) > 1 {
			c.precision, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("mssql: parse precision %q: %w", parts[1], err)
			}
		}
		if len(parts) > 2 {
			c.scale, err = strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("mssql: parse scale %q: %w", parts[1], err)
			}
		}
	case TypeFloat, TypeReal:
		if len(parts) > 1 {
			c.precision, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("mssql: parse precision %q: %w", parts[1], err)
			}
		}
	case TypeBinary, TypeChar, TypeNChar:
		if len(parts) > 1 {
			if c.size, err = strconv.ParseInt(parts[1], 10, 64); err != nil {
				return nil, fmt.Errorf("mssql: parse size %q: %w", parts[1], err)
			}
		}
	case TypeVarBinary, TypeVarchar, TypeNVarchar:
		if len(parts) > 1 {
			// MAX is a special value for the maximum length.
			if strings.ToLower(parts[1]) == "max" {
				c.size = -1
			} else if c.size, err = strconv.ParseInt(parts[1], 10, 64); err != nil {
				return nil, fmt.Errorf("mssql: parse size %q: %w", parts[1], err)
			}
		}
	case TypeDateTime2, TypeDateTimeOffset, TypeTime:
		if len(parts) > 1 {
			if c.scale, err = strconv.ParseInt(parts[1], 10, 64); err != nil {
				return nil, fmt.Errorf("mssql: parse scale %q: %w", parts[1], err)
			}
		}
	// Types without length, precision, or scale.
	case TypeDate, TypeDateTime, TypeSmallDateTime:
	case TypeBit:
	case TypeInt, TypeBigInt, TypeSmallInt, TypeTinyInt:
	case TypeMoney, TypeSmallMoney:
	case TypeText, TypeNText, TypeImage: // Deprecated types
	case TypeGeography, TypeGeometry, TypeHierarchyID, TypeRowVersion, TypeSQLVariant, TypeUniqueIdentifier, TypeXML:
	default:
		c.typ = s
	}
	return c, nil
}

func columnType(c *columnDesc) (schema.Type, error) {
	size, precision, scale := int(c.size), int(c.precision), int(c.scale)
	var typ schema.Type
	switch t := c.typ; strings.ToLower(t) {
	// numeric types
	case TypeBit:
		typ = &BitType{T: t}
	case TypeInt, TypeBigInt, TypeSmallInt, TypeTinyInt:
		typ = &schema.IntegerType{T: t}
	case TypeDecimal, TypeNumeric:
		typ = &schema.DecimalType{T: t, Precision: precision, Scale: scale}
	case TypeMoney, TypeSmallMoney:
		typ = &MoneyType{T: t}
	// Approximate numerics
	case TypeFloat, TypeReal:
		typ = &schema.FloatType{T: t, Precision: precision}
	// Character strings and Unicode character strings
	case TypeText, TypeNText: // Deprecated types
		fallthrough
	case TypeChar, TypeVarchar, TypeNChar, TypeNVarchar:
		typ = &schema.StringType{T: t, Size: size}
	case TypeImage: // Deprecated type
		fallthrough
	case TypeBinary, TypeVarBinary:
		bt := &schema.BinaryType{T: t}
		if size != 0 {
			bt.Size = &size
		}
		typ = bt
	// Date and time
	case TypeDate, TypeDateTime, TypeSmallDateTime:
		typ = &schema.TimeType{T: t}
	case TypeDateTime2, TypeDateTimeOffset, TypeTime:
		typ = &schema.TimeType{T: t, Precision: &precision, Scale: &scale}
	// Other types
	case TypeGeography, TypeGeometry:
		typ = &schema.SpatialType{T: t}
	case TypeHierarchyID:
		typ = &HierarchyIDType{T: t}
	case TypeUniqueIdentifier:
		typ = &UniqueIdentifierType{T: t}
	case TypeXML:
		typ = &XMLType{T: t}
	default:
		if c.userDefined {
			typ = &UserDefinedType{T: t}
		} else {
			typ = &schema.UnsupportedType{T: t}
		}
	}
	return typ, nil
}

const defaultTimeScale = 7
