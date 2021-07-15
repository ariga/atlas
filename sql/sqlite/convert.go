package sqlite

import (
	"fmt"
	"strings"

	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/schema/schemaspec"
)

// ConvertColumnType converts a schemaspec.Column into a concrete sqlite schema.Type.
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
	if strings.HasPrefix(spec.Type, "u") {
		// todo(rotemtam): support his once we can express CHECK(col >= 0)
		return nil, fmt.Errorf("sqlite: unsigned integers currently not supported")
	}
	typ := &schema.IntegerType{
		T: "integer",
	}
	switch t := spec.Type; t {
	case "int8", "uint8":
		typ.Size = 1
	case "int16", "uint16":
		typ.Size = 2

	case "int32", "uint32", "int", "integer", "uint":
		typ.Size = 4
	case "int64", "uint64":
		typ.Size = 8
	default:
		return nil, fmt.Errorf("sqlite: unknown integer column type %q", spec.Type)
	}
	return typ, nil
}

func convertBinary(spec *schemaspec.Column) (schema.Type, error) {
	bt := &schema.BinaryType{
		T: "blob",
	}
	return bt, nil
}

func convertString(spec *schemaspec.Column) (schema.Type, error) {
	st := &schema.StringType{
		T: "text",
	}
	if attr, ok := spec.Attr("size"); ok {
		s, err := attr.Int()
		if err != nil {
			return nil, err
		}
		st.Size = s
	}
	return st, nil
}

func convertEnum(spec *schemaspec.Column) (schema.Type, error) {
	// sqlite does not have a enum column type
	return &schema.StringType{T: "text"}, nil
}

func convertBoolean(spec *schemaspec.Column) (schema.Type, error) {
	return &schema.BoolType{T: "boolean"}, nil
}

func convertTime(spec *schemaspec.Column) (schema.Type, error) {
	return &schema.TimeType{T: "datetime"}, nil
}

func convertDecimal(spec *schemaspec.Column) (schema.Type, error) {
	dt := &schema.DecimalType{
		T: "decimal",
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
		T: "real",
	}
	if precision, ok := spec.Attr("precision"); ok {
		p, err := precision.Int()
		if err != nil {
			return nil, err
		}
		ft.Precision = p
	}
	return ft, nil
}
