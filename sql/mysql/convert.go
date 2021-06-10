package mysql

import "ariga.io/atlas/sql/schema"

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
	ct, err := columnType(spec)
	if err != nil {
		return nil, err
	}
	out.Type.Type = ct
	return out, err
}

func columnType(spec *schema.ColumnSpec) (schema.Type, error) {
	return nil, nil
}
