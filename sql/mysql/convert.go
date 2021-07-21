package mysql

import (
	"fmt"
	"math"
	"strings"

	"ariga.io/atlas/sql/internal/schemautil"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/schema/schemaspec"
)

// ConvertSchema converts a schemaspec.Schema into a schema.Schema.
func ConvertSchema(spec *schemaspec.Schema) (*schema.Schema, error) {
	return schemautil.ConvertSchema(spec, ConvertTable)
}

// ConvertTable converts a schemaspec.Table to a schema.Table. Table conversion is done without converting
// ForeignKeySpecs into ForeignKeys, as the target tables do not necessarily exist in the schema
// at this point. Instead, the linking is done by the ConvertSchema function.
func ConvertTable(spec *schemaspec.Table, parent *schema.Schema) (*schema.Table, error) {
	return schemautil.ConvertTable(spec, parent, ConvertColumn, ConvertPrimaryKey, ConvertIndex)
}

// ConvertPrimaryKey converts a schemaspec.PrimaryKey to a schema.Index.
func ConvertPrimaryKey(spec *schemaspec.PrimaryKey, parent *schema.Table) (*schema.Index, error) {
	return schemautil.ConvertPrimaryKey(spec, parent)
}

// ConvertIndex converts an schemaspec.Index to a schema.Index.
func ConvertIndex(spec *schemaspec.Index, parent *schema.Table) (*schema.Index, error) {
	return schemautil.ConvertIndex(spec, parent)
}

// ConvertColumn converts a schemaspec.Column into a schema.Column.
func ConvertColumn(spec *schemaspec.Column, parent *schema.Table) (*schema.Column, error) {
	if err := schemautil.OverrideFor(Name, spec); err != nil {
		return nil, err
	}
	return schemautil.ConvertColumn(spec, ConvertColumnType)
}

// ConvertColumnType converts a schemaspec.Column into a concrete MySQL schema.Type.
func ConvertColumnType(spec *schemaspec.Column) (schema.Type, error) {
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
