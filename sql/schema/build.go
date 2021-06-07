package schema

// Builder builds Schema elements from Spec objects. Builder holds all schema building logic
// that is shared between SQL dialects.
type Builder struct {
	SpecConverter
}

// SpecConverter converts Spec elements into dialect specific Schema elements. SpecConverter should be
// implemented per each dialect driver.
type SpecConverter interface {
	ColumnType(*ColumnSpec) (Type, error)
}

func (b *Builder) Build(spec *SchemaSpec) (*Schema, error) {
	out := &Schema{
		Name: spec.Name,
		Spec: spec,
	}
	for _, ts := range spec.Tables {
		table, err := b.BuildTable(ts, out)
		if err != nil {
			return nil, err
		}
		out.Tables = append(out.Tables, table)
	}
	return out, nil
}

func (b *Builder) BuildTable(spec *TableSpec, parent *Schema) (*Table, error) {
	out := &Table{
		Name:   spec.Name,
		Schema: parent,
		Spec:   spec,
	}
	for _, csp := range spec.Columns {
		col, err := b.BuildColumn(csp, out)
		if err != nil {
			return nil, err
		}
		out.Columns = append(out.Columns, col)
	}
	return out, nil
}

func (b *Builder) BuildColumn(spec *ColumnSpec, parent *Table) (*Column, error) {
	out := &Column{
		Name: spec.Name,
		Spec: spec,
		Type: &ColumnType{
			Null: spec.Null,
		},
	}
	if spec.Default != nil {
		out.Default = &Literal{V: *spec.Default}
	}
	columnType, err := b.ColumnType(spec)
	if err != nil {
		return nil, err
	}
	out.Type.Type = columnType
	return out, err
}

type DefaultSpecConverter struct {
}

func (d *DefaultSpecConverter) ColumnType(spec *ColumnSpec) (Type, error) {
	// TODO(rotemtam): implement this, most can be taken from HCL Converter code which should be deprecated.
	return nil, nil
}
