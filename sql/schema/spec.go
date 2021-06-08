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

	Codec interface {
		Encoder
		Decoder
	}
)

// ResourceSpec is a generic container for resources described in configurations.
type ResourceSpec struct {
	Name     string
	Type     string
	Attrs    []*SpecAttr
	Children []*ResourceSpec
}

// SchemaSpec holds a specification for a Schema.
type SchemaSpec struct {
	Name   string
	Tables []*TableSpec
}

// TableSpec holds a specification for an SQL table.
type TableSpec struct {
	Name     string
	Columns  []*ColumnSpec
	Attrs    []*SpecAttr
	Children []*ResourceSpec
}

// ColumnSpec holds a specification for a column in an SQL table.
type ColumnSpec struct {
	Name     string
	TypeName string
	Default  *string
	Null     bool
	Attrs    []*SpecAttr
	Children []*ResourceSpec
}

// Element is an object that can be encoded into bytes to be written to a configuration file representing
// Schema resources.
type Element interface {
	elem()
}

// SpecAttr is an attribute of a Spec.
type SpecAttr struct {
	K string
	V SpecLiteral
}

// SpecLiteral is a literal value to be used in the V field of a SpecAttr.
type SpecLiteral interface {
	lit()
}

type String string
type Number float64
type Bool bool

func (String) lit() {}
func (Number) lit() {}
func (Bool) lit()   {}

func (String) elem()        {}
func (Number) elem()        {}
func (Bool) elem()          {}
func (*ResourceSpec) elem() {}
func (*SpecAttr) elem()     {}
func (*ColumnSpec) elem()   {}
func (*TableSpec) elem()    {}
func (*SchemaSpec) elem()   {}

func (*ColumnSpec) spec()   {}
func (*TableSpec) spec()    {}
func (*SchemaSpec) spec()   {}
func (*ResourceSpec) spec() {}
