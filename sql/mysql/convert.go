// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"errors"
	"fmt"
	"math"
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
	case *schema.BoolType:
		f = strings.ToLower(t.T)
	case *schema.BinaryType:
		f = strings.ToLower(t.T)
		if f == tVarBinary {
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
	default:
		return "", fmt.Errorf("mysql: invalid schema type: %T", t)
	}
	return f, nil
}

// mustFormat calls to FormatType and panics in case of error.
func (d *Driver) mustFormat(t schema.Type) string {
	s, err := d.FormatType(t)
	if err != nil {
		panic(err)
	}
	return s
}

// ConvertSchema converts a schemaspec.Schema into a schema.Schema.
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
	const driver = "mysql"
	if override := spec.Override(sqlx.VersionPermutations(driver, d.version)...); override != nil {
		if err := schemautil.Override(spec, override); err != nil {
			return nil, err
		}
	}
	return schemautil.ConvertColumn(spec, ConvertColumnType)
}

// ConvertColumnType converts a schemaspec.Column into a concrete MySQL schema.Type.
func ConvertColumnType(spec *schemaspec.Column) (schema.Type, error) {
	switch schemaspec.Type(spec.Type) {
	case schemaspec.TypeInt, schemaspec.TypeInt8, schemaspec.TypeInt16,
		schemaspec.TypeInt64, schemaspec.TypeUint, schemaspec.TypeUint8,
		schemaspec.TypeUint16, schemaspec.TypeUint64:
		return convertInteger(spec)
	case schemaspec.TypeString:
		return convertString(spec)
	case schemaspec.TypeBinary:
		return convertBinary(spec)
	case schemaspec.TypeEnum:
		return convertEnum(spec)
	case schemaspec.TypeBoolean:
		return convertBoolean(spec)
	case schemaspec.TypeDecimal:
		return convertDecimal(spec)
	case schemaspec.TypeFloat:
		return convertFloat(spec)
	case schemaspec.TypeTime:
		return convertTime(spec)
	}
	return parseRawType(spec.Type)
}

func convertInteger(spec *schemaspec.Column) (schema.Type, error) {
	typ := &schema.IntegerType{
		Unsigned: strings.HasPrefix(spec.Type, "u"),
	}
	switch spec.Type {
	case "int8", "uint8":
		typ.T = tTinyInt
	case "int16", "uint16":
		typ.T = tSmallInt
	case "int32", "uint32", "int", "integer", "uint":
		typ.T = tInt
	case "int64", "uint64":
		typ.T = tBigInt
	default:
		return nil, fmt.Errorf("mysql: unknown integer column type %q", spec.Type)
	}
	return typ, nil
}

func convertBinary(spec *schemaspec.Column) (schema.Type, error) {
	bt := &schema.BinaryType{}
	if attr, ok := spec.Attr("size"); ok {
		s, err := attr.Int()
		if err != nil {
			return nil, err
		}
		bt.Size = s
	}
	switch {
	case bt.Size == 0:
		bt.T = "blob"
	case bt.Size <= math.MaxUint8:
		bt.T = "tinyblob"
	case bt.Size > math.MaxUint8 && bt.Size <= math.MaxUint16:
		bt.T = "blob"
	case bt.Size > math.MaxUint16 && bt.Size <= 1<<24-1:
		bt.T = "mediumblob"
	case bt.Size > 1<<24-1 && bt.Size <= math.MaxUint32:
		bt.T = "longblob"
	default:
		return nil, fmt.Errorf("mysql: blob fields can be up to 4GB long")
	}
	return bt, nil
}

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
	case st.Size <= math.MaxUint16:
		st.T = "varchar"
	case st.Size > math.MaxUint16 && st.Size <= (1<<24-1):
		st.T = "mediumtext"
	case st.Size > (1<<24-1) && st.Size <= math.MaxUint32:
		st.T = "longtext"
	default:
		return nil, fmt.Errorf("mysql: string fields can be up to 4GB long")
	}
	return st, nil
}

func convertEnum(spec *schemaspec.Column) (schema.Type, error) {
	attr, ok := spec.Attr("values")
	if !ok {
		return nil, fmt.Errorf("mysql: expected enum fields to have values")
	}
	list, err := attr.Strings()
	if err != nil {
		return nil, err
	}
	return &schema.EnumType{Values: list}, nil
}

func convertBoolean(spec *schemaspec.Column) (schema.Type, error) {
	return &schema.BoolType{T: "boolean"}, nil
}

func convertTime(spec *schemaspec.Column) (schema.Type, error) {
	return &schema.TimeType{T: "timestamp"}, nil
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
		T: tFloat,
	}
	if precision, ok := spec.Attr("precision"); ok {
		p, err := precision.Int()
		if err != nil {
			return nil, err
		}
		ft.Precision = p
	}
	// A precision from 0 to 23 results in a 4-byte single-precision FLOAT column.
	// A precision from 24 to 53 results in an 8-byte double-precision DOUBLE column:
	// https://dev.mysql.com/doc/refman/8.0/en/floating-point-types.html
	if ft.Precision > 23 {
		ft.T = "double"
	}
	return ft, nil
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

// ColumnTypeSpec converts from a concrete MySQL schema.Type into schemaspec.Column Type.
func ColumnTypeSpec(t schema.Type) (*schemaspec.Column, error) {
	switch t := t.(type) {
	case *schema.EnumType:
		return enumSpec(t)
	case *schema.IntegerType:
		return integerSpec(t)
	case *schema.StringType:
		return stringSpec(t)
	case *schema.DecimalType:
		return decimalSpec(t)
	case *schema.BinaryType:
		return binarySpec(t)
	case *schema.BoolType:
		return boolSpec()
	case *schema.FloatType:
		return floatSpec(t)
	case *schema.TimeType:
		return timeSpec(t)
	case *schema.JSONType:
		return jsonSpec(t)
	case *schema.SpatialType:
		return spatialSpec(t)
	case *schema.UnsupportedType:
		return unsupportedSpec(t)
	default:
		return nil, fmt.Errorf("mysql: failed to convert column type %T to spec", t)
	}
}

func unsupportedSpec(t *schema.UnsupportedType) (*schemaspec.Column, error) {
	return schemautil.ColSpec("", t.T), nil
}

func spatialSpec(t *schema.SpatialType) (*schemaspec.Column, error) {
	return schemautil.ColSpec("", t.T), nil
}

func jsonSpec(t *schema.JSONType) (*schemaspec.Column, error) {
	return schemautil.ColSpec("", t.T), nil
}

func timeSpec(t *schema.TimeType) (*schemaspec.Column, error) {
	return schemautil.ColSpec("", t.T), nil
}

func floatSpec(t *schema.FloatType) (*schemaspec.Column, error) {
	p := strconv.Itoa(t.Precision)
	return schemautil.ColSpec("", "float", schemautil.LitAttr("precision", p)), nil
}

func boolSpec() (*schemaspec.Column, error) {
	return schemautil.ColSpec("", "boolean"), nil
}

func binarySpec(t *schema.BinaryType) (*schemaspec.Column, error) {
	switch t.T {
	case tBlob:
		return schemautil.ColSpec("", "binary"), nil
	case tTinyBlob, tMediumBlob, tLongBlob:
		s := strconv.Itoa(t.Size)
		return schemautil.ColSpec("", "binary", schemautil.LitAttr("size", s)), nil
	}
	return nil, errors.New("mysql: schema binary failed to convert")
}

func decimalSpec(t *schema.DecimalType) (*schemaspec.Column, error) {
	p := strconv.Itoa(t.Precision)
	s := strconv.Itoa(t.Scale)
	return schemautil.ColSpec("", "decimal", schemautil.LitAttr("precision", p), schemautil.LitAttr("scale", s)), nil
}

func stringSpec(t *schema.StringType) (*schemaspec.Column, error) {
	switch t.T {
	case tVarchar, tMediumText, tLongText:
		s := strconv.Itoa(t.Size)
		return schemautil.ColSpec("", "string", schemautil.LitAttr("size", s)), nil
	}
	return nil, errors.New("mysql: schema string failed to convert")
}

func integerSpec(t *schema.IntegerType) (*schemaspec.Column, error) {
	switch t.T {
	case tInt:
		if t.Unsigned {
			return schemautil.ColSpec("", "uint"), nil
		}
		return schemautil.ColSpec("", "int"), nil
	case tTinyInt:
		return schemautil.ColSpec("", "int8"), nil
	case tBigInt:
		if t.Unsigned {
			return schemautil.ColSpec("", "uint64"), nil
		}
		return schemautil.ColSpec("", "int64"), nil
	}
	return nil, errors.New("mysql: schema integer failed to convert")
}

func enumSpec(t *schema.EnumType) (*schemaspec.Column, error) {
	if len(t.Values) == 0 {
		return nil, errors.New("mysql: schema enum fields to have values")
	}
	return schemautil.ColSpec("", "enum", schemautil.ListAttr("values", t.Values...)), nil
}
