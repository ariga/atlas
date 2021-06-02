package entschema

import (
	"fmt"

	"ariga.io/atlas/sql/schema"
	sqlschema "entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
)

func Convert(graph *gen.Graph) (*schema.Schema, error) {
	s := &schema.Schema{Name: "ent"}
	for _, etbl := range graph.Tables() {
		tbl := &schema.Table{
			Schema: s,
			Name:   etbl.Name,
		}
		cols := make(map[string]*schema.Column)
		for _, ec := range etbl.Columns {
			col, err := convertColumn(ec)
			if err != nil {
				return nil, err
			}
			cols[ec.Name] = col
			tbl.Columns = append(tbl.Columns, col)
		}

		s.Tables = append(s.Tables, tbl)
	}
	return s, nil
}

func convertColumn(col *sqlschema.Column) (*schema.Column, error) {
	ct := &schema.ColumnType{
		Null: col.Nullable,
	}
	switch col.Type {
	case field.TypeString:
		ct.Type = &schema.StringType{
			T: "string",
		}
	case field.TypeInt, field.TypeInt8, field.TypeInt16, field.TypeInt32, field.TypeInt64:
		ct.Type = &schema.IntegerType{
			T: "integer",
		}
	case field.TypeUint, field.TypeUint8, field.TypeUint16, field.TypeUint32, field.TypeUint64:
		ct.Type = &schema.IntegerType{
			T:        "integer",
			Unsigned: true,
		}
	case field.TypeBool:
		ct.Type = &schema.BoolType{T: "boolean"}
	case field.TypeTime:
		ct.Type = &schema.TimeType{T: "time"}
	case field.TypeEnum:
		ct.Type = &schema.EnumType{Values: col.Enums}
	case field.TypeFloat32, field.TypeFloat64:
		ct.Type = &schema.FloatType{
			T:         "float",
		}
	case field.TypeUUID:
		ct.Type = &schema.BinaryType{
			T:    "binary",
			Size: 16,
		}
	case field.TypeBytes:
		ct.Type = &schema.BinaryType{
			T: "binary",
		}
	case field.TypeJSON:
		ct.Type = &schema.JSONType{
			T: "json",
		}
	default:
		return nil, fmt.Errorf("entschema: unsupported type %q", col.Type)
	}
	out := &schema.Column{
		Name: col.Name,
		Type: ct,
		//Default:     nil,
		//Attrs:       nil,
		//Indexes:     nil,
		//ForeignKeys: nil,
	}
	return out, nil
}
