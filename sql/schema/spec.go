package schema

type (
	// Spec holds a specification for a schema resource (such as a Table, Column or Index).
	Spec interface {
		spec()
	}

	// Encoder is the interface that wraps the Encode method.
	//
	// Encoder takes a Spec and returns a byte slice representing that Spec in some configuration
	// format (for instance, HCL).
	Encoder interface {
		Encode(Spec) ([]byte, error)
	}

	// Decoder is the interface that wraps the Decode method.
	//
	// Decoder takes a byte slice representing a Spec and decodes it into a Spec.
	Decoder interface {
		Decode([]byte, Spec) error
	}

	// Codec wraps Encoder and Decoder.
	Codec interface {
		Encoder
		Decoder
	}

	// ResourceSpec is a generic container for resources described in configurations.
	ResourceSpec struct {
		Name     string
		Type     string
		Attrs    []*SpecAttr
		Children []*ResourceSpec
	}

	// SchemaSpec holds a specification for a Schema.
	SchemaSpec struct {
		Name   string
		Tables []*TableSpec
	}

	// TableSpec holds a specification for an SQL table.
	TableSpec struct {
		Name        string
		SchemaName  string
		Columns     []*ColumnSpec
		PrimaryKey  *PrimaryKeySpec
		ForeignKeys []*ForeignKeySpec
		Indexes     []*IndexSpec
		Attrs       []*SpecAttr
		Children    []*ResourceSpec
	}

	// ColumnSpec holds a specification for a column in an SQL table.
	ColumnSpec struct {
		Name     string
		Type     string
		Default  *string
		Null     bool
		Attrs    []*SpecAttr
		Children []*ResourceSpec
	}

	// PrimaryKeySpec holds a specification for the primary key of a table.
	PrimaryKeySpec struct {
		Columns  []string
		Attrs    []*SpecAttr
		Children []*ResourceSpec
	}

	// ForeignKeySpec holds a specification for a foreign key of a table.
	ForeignKeySpec struct {
		Symbol     string
		Columns    []string
		RefTable   string
		RefColumns []string
		OnUpdate   string
		OnDelete   string
		Attrs      []*SpecAttr
		Children   []*ResourceSpec
	}

	// IndexSpec holds a specification for an index of a table.
	IndexSpec struct {
		Name     string
		Columns  []string
		Unique   bool
		Attrs    []*SpecAttr
		Children []*ResourceSpec
	}

	// Element is an object that can be encoded into bytes to be written to a configuration file representing
	// Schema resources.
	Element interface {
		elem()
	}

	// SpecAttr is an attribute of a Spec.
	SpecAttr struct {
		K string
		V Value
	}
	// Value represents the value of a SpecAttr.
	Value interface {
		val()
	}
	// SpecLiteral implements Value and represents a literal value (string, number, etc.)
	SpecLiteral struct {
		V string
	}
)

func (*SpecLiteral) val() {}

func (*ResourceSpec) elem()   {}
func (*SpecAttr) elem()       {}
func (*ColumnSpec) elem()     {}
func (*TableSpec) elem()      {}
func (*SchemaSpec) elem()     {}
func (*PrimaryKeySpec) elem() {}
func (*ForeignKeySpec) elem() {}
func (*IndexSpec) elem()      {}

func (*ColumnSpec) spec()     {}
func (*TableSpec) spec()      {}
func (*SchemaSpec) spec()     {}
func (*ResourceSpec) spec()   {}
func (*PrimaryKeySpec) spec() {}
func (*ForeignKeySpec) spec() {}
func (*IndexSpec) spec()      {}
