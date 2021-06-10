package mysql

import (
	"fmt"
	"math"
	"strings"

	"ariga.io/atlas/sql/schema"
)

// ConvertSchema converts a SchemaSpec into a Schema.
func ConvertSchema(spec *schema.SchemaSpec) (*schema.Schema, error) {
	out := &schema.Schema{
		Name: spec.Name,
		Spec: spec,
	}
	for _, ts := range spec.Tables {
		table, err := ConvertTable(ts, out)
		if err != nil {
			return nil, err
		}
		out.Tables = append(out.Tables, table)
	}
	return out, nil
}

// ConvertTable converts a TableSpec to a Table.
func ConvertTable(spec *schema.TableSpec, parent *schema.Schema) (*schema.Table, error) {
	out := &schema.Table{
		Name:   spec.Name,
		Schema: parent,
		Spec:   spec,
	}
	for _, csp := range spec.Columns {
		col, err := ConvertColumn(csp, out)
		if err != nil {
			return nil, err
		}
		out.Columns = append(out.Columns, col)
	}
	return out, nil
}

// ConvertColumn converts a ColumnSpec into a Column.
func ConvertColumn(spec *schema.ColumnSpec, parent *schema.Table) (*schema.Column, error) {
	out := &schema.Column{
		Name: spec.Name,
		Spec: spec,
		Type: &schema.ColumnType{
			Null: spec.Null,
		},
	}
	if spec.Default != nil {
		out.Default = &schema.Literal{V: *spec.Default}
	}
	ct, err := ConvertColumnType(spec)
	if err != nil {
		return nil, err
	}
	out.Type.Type = ct
	return out, err
}

func ConvertColumnType(spec *schema.ColumnSpec) (schema.Type, error) {
	switch spec.Type {
	case "int", "int8", "int16", "int64", "uint", "uint8", "uint16", "uint64":
		return convertInteger(spec)
	case "string":
		return convertString(spec)
	case "binary":
		return convertBinary(spec)
	case "enum":
		return convertEnum(spec)
	case "boolean":
		return convertBoolean(spec)
	case "decimal":
		return convertDecimal(spec)
	case "float":
		return convertFloat(spec)
	case "time":
		return convertTime(spec)
	}
	return parseRawType(spec.Type)
}

func convertInteger(spec *schema.ColumnSpec) (schema.Type, error) {
	typ := &schema.IntegerType{
		Unsigned: strings.HasPrefix(spec.Type, "u"),
	}
	switch spec.Type {
	case "int8", "uint8":
		typ.Size = 1
		typ.T = tTinyInt
	case "int16", "uint16":
		typ.Size = 2
		typ.T = tSmallInt
	case "int32", "uint32", "int", "integer", "uint":
		typ.Size = 4
		typ.T = tInt
	case "int64", "uint64":
		typ.Size = 8
		typ.T = tBigInt
	default:
		return nil, fmt.Errorf("mysql: unknown integer column type %q", spec.Type)
	}
	return typ, nil
}

func convertBinary(spec *schema.ColumnSpec) (schema.Type, error) {
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

func convertString(spec *schema.ColumnSpec) (schema.Type, error) {
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

func convertEnum(spec *schema.ColumnSpec) (schema.Type, error) {
	attr, ok := spec.Attr("values")
	if !ok {
		return nil, fmt.Errorf("mysql: expected enum fields to have values")
	}
	list, err := attr.StringList()
	if err != nil {
		return nil, err
	}
	return &schema.EnumType{Values: list}, nil
}

func convertBoolean(spec *schema.ColumnSpec) (schema.Type, error) {
	return &schema.BoolType{T: "boolean"}, nil
}

func convertTime(spec *schema.ColumnSpec) (schema.Type, error) {
	return &schema.TimeType{T: "timestamp"}, nil
}

func convertDecimal(spec *schema.ColumnSpec) (schema.Type, error) {
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

func convertFloat(spec *schema.ColumnSpec) (schema.Type, error) {
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
