package schemacodec

type Spec interface {
	spec()
}

type SchemaSpec struct {
	Name   string
	Tables []*TableSpec
}

// TableSpec holds a specification for an SQL table.
type TableSpec struct {
	Name    string
	Columns []*ColumnSpec
}

// ColumnSpec holds a specification for a column in an SQL table.
type ColumnSpec struct {
	Name     string
	TypeName string
	Default  *string
	Null     bool
	Attrs    []*Attr
	Blocks   []*Block
}

func (*ColumnSpec) spec() {}
func (*TableSpec) spec()  {}
func (*SchemaSpec) spec() {}
