// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/internal/schemautil"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/schema/schemaspec"
)

// FormatType converts schema type to its column form in the database.
// An error is returned if the type cannot be recognized.
func (d *Driver) FormatType(t schema.Type) (string, error) {
	var f string
	switch t := t.(type) {
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
		case p == 0 || s > 0:
			return "", fmt.Errorf("postgres: decimal type must have precision > 0: %d", p)
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
	default:
		return "", fmt.Errorf("postgres: invalid schema type: %T", t)
	}
	return f, nil
}

// ConvertSchema converts a schemaspec.Schema and its associated tables into a schema.Schema.
func (d *Driver) ConvertSchema(spec *schemaspec.Schema, tables []*schemaspec.Table) (*schema.Schema, error) {
	return schemautil.ConvertSchema(spec, tables, d.ConvertTable)
}

// ConvertTable converts a schemaspec.Table to a schema.Table. Table conversion is done without converting
// ForeignKeySpecs into ForeignKeys, as the target tables do not necessarily exist in the schema
// at this point. Instead, the linking is done by the ConvertSchema function.
func (d *Driver) ConvertTable(spec *schemaspec.Table, parent *schema.Schema) (*schema.Table, error) {
	return schemautil.ConvertTable(spec, parent, d.ConvertColumn, d.ConvertPrimaryKey, d.ConvertIndex)
}

// ConvertPrimaryKey converts a schemaspec.PrimaryKey to a schema.Index.
func (d *Driver) ConvertPrimaryKey(spec *schemaspec.PrimaryKey, parent *schema.Table) (*schema.Index, error) {
	return schemautil.ConvertPrimaryKey(spec, parent)
}

// ConvertIndex converts an schemaspec.Index to a schema.Index.
func (d *Driver) ConvertIndex(spec *schemaspec.Index, parent *schema.Table) (*schema.Index, error) {
	return schemautil.ConvertIndex(spec, parent)
}

// ConvertColumn converts a schemaspec.Column into a schema.Column.
func (d *Driver) ConvertColumn(spec *schemaspec.Column, _ *schema.Table) (*schema.Column, error) {
	const driver = "postgres"
	if override := spec.Override(sqlx.VersionPermutations(driver, d.version)...); override != nil {
		if err := schemautil.Override(spec, override); err != nil {
			return nil, err
		}
	}
	return schemautil.ConvertColumn(spec, d.ConvertColumnType)
}

// ConvertColumnType converts a schemaspec.Column into a concrete MySQL schema.Type.
func (d *Driver) ConvertColumnType(spec *schemaspec.Column) (schema.Type, error) {
	switch schemaspec.Type(spec.Type) {
	case schemaspec.TypeInt, schemaspec.TypeInt8, schemaspec.TypeInt16,
		schemaspec.TypeInt64, schemaspec.TypeUint, schemaspec.TypeUint8,
		schemaspec.TypeUint16, schemaspec.TypeUint64:
		return convertInteger(spec)
	case schemaspec.TypeString:
		return convertString(spec)
	case schemaspec.TypeEnum:
		return convertEnum(spec)
	case schemaspec.TypeDecimal:
		return convertDecimal(spec)
	case schemaspec.TypeFloat:
		return convertFloat(spec)
	case schemaspec.TypeTime:
		return &schema.TimeType{T: tTimestamp}, nil
	case schemaspec.TypeBinary:
		return &schema.BinaryType{T: tBytea}, nil
	case schemaspec.TypeBoolean:
		return &schema.BoolType{T: tBoolean}, nil
	}
	return parseRawType(spec)
}

func convertInteger(spec *schemaspec.Column) (schema.Type, error) {
	if strings.HasPrefix(spec.Type, "u") {
		return nil, fmt.Errorf("postgres: unsigned integers currently not supported")
	}
	typ := &schema.IntegerType{}
	switch schemaspec.Type(spec.Type) {
	case schemaspec.TypeInt8:
		return nil, fmt.Errorf("postgres: 8-bit integers not supported")
	case schemaspec.TypeInt16:
		typ.T = tSmallInt
	case schemaspec.TypeInt:
		typ.T = tInteger
	case schemaspec.TypeInt64:
		typ.T = tBigInt
	default:
		return nil, fmt.Errorf("mysql: unknown integer column type %q", spec.Type)
	}
	return typ, nil
}

// maxCharSize defines the maximum size of limited character types in Postgres (10 MB).
// https://github.com/postgres/postgres/blob/REL_13_STABLE/src/include/access/htup_details.h#L585
const maxCharSize = 10 << 20

func convertString(spec *schemaspec.Column) (schema.Type, error) {
	st := &schema.StringType{
		Size: 255,
	}
	if attr, ok := spec.Attr("size"); ok {
		s, err := attr.Int()
		if err != nil {
			return nil, err
		}
		st.Size = s
	}
	switch {
	case st.Size < maxCharSize:
		st.T = tVarChar
	default:
		st.T = tText
	}
	return st, nil
}

func convertEnum(spec *schemaspec.Column) (schema.Type, error) {
	attr, ok := spec.Attr("values")
	if !ok {
		return nil, errors.New("postgres: expected enum fields to have values")
	}
	list, err := attr.Strings()
	if err != nil {
		return nil, err
	}
	return &schema.EnumType{Values: list}, nil
}

func convertDecimal(spec *schemaspec.Column) (schema.Type, error) {
	dt := &schema.DecimalType{
		T: tDecimal,
	}
	if precision, ok := spec.Attr("precision"); ok {
		p, err := precision.Int()
		if err != nil {
			return nil, err
		}
		dt.Precision = p
	}
	if scale, ok := spec.Attr("scale"); ok {
		s, err := scale.Int()
		if err != nil {
			return nil, err
		}
		dt.Scale = s
	}
	return dt, nil
}

func convertFloat(spec *schemaspec.Column) (schema.Type, error) {
	ft := &schema.FloatType{
		T: tReal,
	}
	if precision, ok := spec.Attr("precision"); ok {
		p, err := precision.Int()
		if err != nil {
			return nil, err
		}
		ft.Precision = p
	}
	if ft.Precision > 23 {
		ft.T = tDouble
	}
	return ft, nil
}

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

func parseRawType(spec *schemaspec.Column) (schema.Type, error) {
	cm, err := parseColumn(spec.Type)
	if err != nil {
		return nil, err
	}
	return columnType(cm), nil
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
	case tVarChar:
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
