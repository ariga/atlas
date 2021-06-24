package postgres

import (
	"fmt"
	"strconv"
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
	return parseRawType(spec)
}

func convertInteger(spec *schemaspec.Column) (schema.Type, error) {
	if strings.HasPrefix(spec.Type, "u") {
		return nil, fmt.Errorf("postgres: unsigned integers currently not supported")
	}
	typ := &schema.IntegerType{}
	switch spec.Type {
	case "int8":
		return nil, fmt.Errorf("postgres: 8-bit integers not supported")
	case "int16":
		typ.Size = 2
		typ.T = "smallint"
	case "int32", "int", "integer":
		typ.Size = 4
		typ.T = "integer"
	case "int64":
		typ.Size = 8
		typ.T = "bigint"
	default:
		return nil, fmt.Errorf("mysql: unknown integer column type %q", spec.Type)
	}
	return typ, nil
}

func convertBinary(spec *schemaspec.Column) (schema.Type, error) {
	return &schema.BinaryType{T: "bytea"}, nil
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
		st.T = "varchar"
	default:
		st.T = "text"
	}
	return st, nil
}

func convertEnum(spec *schemaspec.Column) (schema.Type, error) {
	attr, ok := spec.Attr("values")
	if !ok {
		return nil, fmt.Errorf("postgres: expected enum fields to have values")
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
	if ft.Precision > 23 {
		ft.T = "double precision"
	}
	return ft, nil
}

type columnMeta struct {
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

func parseColumn(s string) (*columnMeta, error) {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '(' || r == ')' || r == ' ' || r == ','
	})
	c := &columnMeta{
		typ:   parts[0],
		parts: parts,
	}
	var err error
	switch c.parts[0] {
	case "varchar":
		if len(c.parts) > 1 {
			c.size, err = strconv.ParseInt(c.parts[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("postgres: parse size %q: %w", parts[1], err)
			}
		}
	case "decimal", "numeric":
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
	case "bit":
		if err := parseBitParts(parts, c); err != nil {
			return nil, err
		}
	case "double":
		if len(parts) > 1 && parts[1] == "precision" {
			c.typ = "double precision"
			c.precision = 53
		}
		return nil, fmt.Errorf("postgres: error parsing double precision column")
	case "float8":
		c.precision = 53
	case "real", "float4":
		c.precision = 24
	}
	return c, nil
}

func parseBitParts(parts []string, c *columnMeta) error {
	if len(parts) == 1 {
		c.size = 1
		return nil
	}
	parts = parts[1:]
	if parts[0] == "varying" {
		c.typ = "bit varying"
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
