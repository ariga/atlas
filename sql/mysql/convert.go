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

	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/schema/schemaspec"
	"ariga.io/atlas/sql/sqlspec"
)

// FormatType converts schema type to its column form in the database.
// An error is returned if the type cannot be recognized.
func (d *Driver) FormatType(t schema.Type) (string, error) {
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
func (d *Driver) ConvertSchema(spec *sqlspec.Schema, tables []*sqlspec.Table) (*schema.Schema, error) {
	return sqlspec.ConvertSchema(spec, tables, d.ConvertTable)
}

// ConvertTable converts a schemaspec.Table to a schema.Table. Table conversion is done without converting
// ForeignKeySpecs into ForeignKeys, as the target tables do not necessarily exist in the schema
// at this point. Instead, the linking is done by the ConvertSchema function.
func (d *Driver) ConvertTable(spec *sqlspec.Table, parent *schema.Schema) (*schema.Table, error) {
	return sqlspec.ConvertTable(spec, parent, d.ConvertColumn, d.ConvertPrimaryKey, d.ConvertIndex)
}

// ConvertPrimaryKey converts a schemaspec.PrimaryKey to a schema.Index.
func (d *Driver) ConvertPrimaryKey(spec *sqlspec.PrimaryKey, parent *schema.Table) (*schema.Index, error) {
	return sqlspec.ConvertPrimaryKey(spec, parent)
}

// ConvertIndex converts an schemaspec.Index to a schema.Index.
func (d *Driver) ConvertIndex(spec *sqlspec.Index, parent *schema.Table) (*schema.Index, error) {
	return sqlspec.ConvertIndex(spec, parent)
}

// ConvertColumn converts a schemaspec.Column into a schema.Column.
func (d *Driver) ConvertColumn(spec *sqlspec.Column, _ *schema.Table) (*schema.Column, error) {
	//const driver = "mysql"
	//if override := spec.Override(sqlx.VersionPermutations(driver, d.version)...); override != nil {
	//	if err := schemautil.Override(spec, override); err != nil {
	//		return nil, err
	//	}
	//}
	return sqlspec.ConvertColumn(spec, ConvertColumnType)
}

// ConvertColumnType converts a sqlspec.Column into a concrete MySQL schema.Type.
func ConvertColumnType(spec *sqlspec.Column) (schema.Type, error) {
	switch schemaspec.Type(spec.TypeName) {
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
		return convertBoolean()
	case schemaspec.TypeDecimal:
		return convertDecimal(spec)
	case schemaspec.TypeFloat:
		return convertFloat(spec)
	case schemaspec.TypeTime:
		return convertTime()
	}
	return parseRawType(spec.TypeName)
}

func convertInteger(spec *sqlspec.Column) (schema.Type, error) {
	typ := &schema.IntegerType{
		Unsigned: strings.HasPrefix(spec.TypeName, "u"),
	}
	switch spec.TypeName {
	case "int8", "uint8":
		typ.T = tTinyInt
	case "int16", "uint16":
		typ.T = tSmallInt
	case "int32", "uint32", "int", "integer", "uint":
		typ.T = tInt
	case "int64", "uint64":
		typ.T = tBigInt
	default:
		return nil, fmt.Errorf("mysql: unknown integer column type %q", spec.TypeName)
	}
	return typ, nil
}

func convertBinary(spec *sqlspec.Column) (schema.Type, error) {
	bt := &schema.BinaryType{}
	if attr, ok := spec.DefaultExtension.Extra.Attr("size"); ok {
		s, err := attr.Int()
		if err != nil {
			return nil, err
		}
		bt.Size = s
	}
	switch {
	case bt.Size == 0:
		bt.T = tBlob
	case bt.Size <= math.MaxUint8:
		bt.T = tTinyBlob
	case bt.Size > math.MaxUint8 && bt.Size <= math.MaxUint16:
		bt.T = tBlob
	case bt.Size > math.MaxUint16 && bt.Size <= 1<<24-1:
		bt.T = tMediumBlob
	case bt.Size > 1<<24-1 && bt.Size <= math.MaxUint32:
		bt.T = tLongBlob
	default:
		return nil, fmt.Errorf("mysql: blob fields can be up to 4GB long")
	}
	return bt, nil
}

func convertString(spec *sqlspec.Column) (schema.Type, error) {
	st := &schema.StringType{
		Size: 255,
	}
	if attr, ok := spec.DefaultExtension.Extra.Attr("size"); ok {
		s, err := attr.Int()
		if err != nil {
			return nil, err
		}
		st.Size = s
	}
	switch {
	case st.Size <= math.MaxUint16:
		st.T = tVarchar
	case st.Size > math.MaxUint16 && st.Size <= (1<<24-1):
		st.T = tMediumText
	case st.Size > (1<<24-1) && st.Size <= math.MaxUint32:
		st.T = tLongText
	default:
		return nil, fmt.Errorf("mysql: string fields can be up to 4GB long")
	}
	return st, nil
}

func convertEnum(spec *sqlspec.Column) (schema.Type, error) {
	attr, ok := spec.DefaultExtension.Extra.Attr("values")
	if !ok {
		return nil, fmt.Errorf("mysql: expected enum fields to have values")
	}
	list, err := attr.Strings()
	if err != nil {
		return nil, err
	}
	return &schema.EnumType{Values: list}, nil
}

func convertBoolean() (schema.Type, error) {
	return &schema.BoolType{T: "boolean"}, nil
}

func convertTime() (schema.Type, error) {
	return &schema.TimeType{T: "timestamp"}, nil
}

func convertDecimal(spec *sqlspec.Column) (schema.Type, error) {
	dt := &schema.DecimalType{
		T: tDecimal,
	}
	if precision, ok := spec.DefaultExtension.Extra.Attr("precision"); ok {
		p, err := precision.Int()
		if err != nil {
			return nil, err
		}
		dt.Precision = p
	}
	if scale, ok := spec.DefaultExtension.Extra.Attr("scale"); ok {
		s, err := scale.Int()
		if err != nil {
			return nil, err
		}
		dt.Scale = s
	}
	return dt, nil
}

func convertFloat(spec *sqlspec.Column) (schema.Type, error) {
	ft := &schema.FloatType{
		T: tFloat,
	}
	if precision, ok := spec.DefaultExtension.Extra.Attr("precision"); ok {
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
		ft.T = tDouble
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

// SchemaSpec converts from a concrete MySQL schema to Atlas specification.
func (d *Driver) SchemaSpec(schem *schema.Schema) (*sqlspec.Schema, []*sqlspec.Table, error) {
	return sqlspec.FromSchema(schem, d.TableSpec)
}

// TableSpec converts from a concrete MySQL schemaspec.Table to a schema.Table.
func (d *Driver) TableSpec(tab *schema.Table) (*sqlspec.Table, error) {
	return sqlspec.FromTable(tab, d.ColumnSpec, sqlspec.FromPrimaryKey, sqlspec.FromIndex, sqlspec.FromForeignKey)
}

// ColumnSpec converts from a concrete MySQL schema.Column into a schemaspec.Column.
func (d *Driver) ColumnSpec(col *schema.Column) (*sqlspec.Column, error) {
	return columnTypeSpec(col.Type.Type)
}

// columnTypeSpec converts from a concrete MySQL schema.Type into schemaspec.Column Type.
func columnTypeSpec(t schema.Type) (*sqlspec.Column, error) {
	switch t := t.(type) {
	case *schema.EnumType:
		return enumSpec(t)
	case *schema.IntegerType:
		return integerSpec(t)
	case *schema.StringType:
		return stringSpec(t)
	case *schema.DecimalType:
		return sqlspec.ColSpec("", "decimal",
			sqlspec.LitAttr("precision", strconv.Itoa(t.Precision)), sqlspec.LitAttr("scale", strconv.Itoa(t.Scale))), nil
	case *schema.BinaryType:
		return binarySpec(t)
	case *schema.BoolType:
		return &sqlspec.Column{TypeName: "boolean"}, nil
	case *schema.FloatType:
		return sqlspec.ColSpec("", "float", sqlspec.LitAttr("precision", strconv.Itoa(t.Precision))), nil
	case *schema.TimeType:
		return &sqlspec.Column{TypeName: t.T}, nil
	case *schema.JSONType:
		return &sqlspec.Column{TypeName: t.T}, nil
	case *schema.SpatialType:
		return &sqlspec.Column{TypeName: t.T}, nil
	case *schema.UnsupportedType:
		return &sqlspec.Column{TypeName: t.T}, nil
	default:
		return nil, fmt.Errorf("mysql: failed to convert column type %T to spec", t)
	}
}

func binarySpec(t *schema.BinaryType) (*sqlspec.Column, error) {
	switch t.T {
	case tBlob:
		return &sqlspec.Column{TypeName: "binary"}, nil
	case tTinyBlob, tMediumBlob, tLongBlob:
		s := strconv.Itoa(t.Size)
		return sqlspec.ColSpec("", "binary", sqlspec.LitAttr("size", s)), nil
	}
	return nil, errors.New("mysql: schema binary failed to convert")
}

func stringSpec(t *schema.StringType) (*sqlspec.Column, error) {
	switch t.T {
	case tVarchar, tMediumText, tLongText:
		s := strconv.Itoa(t.Size)
		return sqlspec.ColSpec("", "string", sqlspec.LitAttr("size", s)), nil
	}
	return nil, errors.New("mysql: schema string failed to convert")
}

func integerSpec(t *schema.IntegerType) (*sqlspec.Column, error) {
	switch t.T {
	case tInt:
		if t.Unsigned {
			return sqlspec.ColSpec("", "uint"), nil
		}
		return &sqlspec.Column{TypeName: "int"}, nil
	case tTinyInt:
		return &sqlspec.Column{TypeName: "int8"}, nil
	case tBigInt:
		if t.Unsigned {
			return sqlspec.ColSpec("", "uint64"), nil
		}
		return &sqlspec.Column{TypeName: "int64"}, nil
	}
	return nil, errors.New("mysql: schema integer failed to convert")
}

func enumSpec(t *schema.EnumType) (*sqlspec.Column, error) {
	if len(t.Values) == 0 {
		return nil, errors.New("mysql: schema enum fields to have values")
	}
	return sqlspec.ColSpec("", "enum", sqlspec.ListAttr("values", t.Values...)), nil
}
