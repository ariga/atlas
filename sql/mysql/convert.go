package mysql

import (
	"fmt"
	"math"
	"strings"

	"ariga.io/atlas/sql/schema"
)

func ConvertSchema(spec *schema.SchemaSpec) (*schema.Schema, error) {
	out := &schema.Schema{
		Name: spec.Name,
		Spec: spec,
	}
	for _, ts := range spec.Tables {
		table, err := c.Table(ts, out)
		if err != nil {
			return nil, err
		}
		out.Tables = append(out.Tables, table)
	}
	return out, nil
}

func ConvertTable(spec *schema.TableSpec, parent *schema.Schema) (*schema.Table, error) {
	out := &schema.Table{
		Name:   spec.Name,
		Schema: parent,
		Spec:   spec,
	}
	for _, csp := range spec.Columns {
		col, err := c.Column(csp, out)
		if err != nil {
			return nil, err
		}
		out.Columns = append(out.Columns, col)
	}
	return out, nil
}

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
	columnType, err := c.columnType(spec)
	if err != nil {
		return nil, err
	}
	out.Type.Type = columnType
	return out, err
}

func columnType(spec *schema.ColumnSpec) (schema.Type, error) {
	return nil, nil
}

func (c *Converter) ColumnType(spec *schema.ColumnSpec) (schema.Type, error) {
	switch spec.Type {
	case "int", "int8", "int16", "int64", "uint", "uint8", "uint16", "uint64":
		return c.convertInteger(spec)
	case "string":
		return c.convertString(spec)
	case "binary":
		return c.convertBinary(spec)
		//case "enum":
		//	return s.convertEnum( spec)
		//case "boolean":
		//	return s.convertBool( spec)
		//case "decimal":
		//	return s.convertDecimal( spec)
		//case "float":
		//	return s.convertFloat( spec)
		//case "time":
		//	return s.convertTime( spec)
		//case "json":
		//	return s.convertJSON( spec)
	}
	return parseRawType(spec.Type)
}

func (c *Converter) convertInteger(spec *schema.ColumnSpec) (schema.Type, error) {
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

func (c *Converter) convertBinary(spec *schema.ColumnSpec) (schema.Type, error) {
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

func (c *Converter) convertString(spec *schema.ColumnSpec) (schema.Type, error) {
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
