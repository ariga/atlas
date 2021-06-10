package mysql

import "ariga.io/atlas/sql/schema"

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
