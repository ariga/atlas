package schema

import (
	"fmt"
	"strconv"
)

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
		Columns  []*ColumnRef
		Attrs    []*SpecAttr
		Children []*ResourceSpec
	}

	// ForeignKeySpec holds a specification for a foreign key of a table.
	ForeignKeySpec struct {
		Symbol     string
		Columns    []*ColumnRef
		RefColumns []*ColumnRef
		OnUpdate   string
		OnDelete   string
		Attrs      []*SpecAttr
		Children   []*ResourceSpec
	}

	// IndexSpec holds a specification for an index of a table.
	IndexSpec struct {
		Name     string
		Columns  []*ColumnRef
		Unique   bool
		Attrs    []*SpecAttr
		Children []*ResourceSpec
	}

	// ColumnRef is a reference to a Column described in another spec.
	ColumnRef struct {
		Name  string
		Table string
	}

	// TableRef is a reference to a Table described in another spec.
	TableRef struct {
		Name   string
		Schema string
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
	// LiteralValue implements Value and represents a literal value (string, number, etc.)
	LiteralValue struct {
		V string
	}
	ListValue struct {
		V []string
	}
)

// Attr returns the value of the ColumnSpec attribute named `name` and reports whether such an attribute exists.
func (c *ColumnSpec) Attr(name string) (*SpecAttr, bool) {
	return getAttrVal(c.Attrs, name)
}

// Attr returns the value of the TableSpec attribute named `name` and reports whether such an attribute exists.
func (t *TableSpec) Attr(name string) (*SpecAttr, bool) {
	return getAttrVal(t.Attrs, name)
}

func getAttrVal(attrs []*SpecAttr, name string) (*SpecAttr, bool) {
	for _, attr := range attrs {
		if attr.K == name {
			return attr, true
		}
	}
	return nil, false
}

// Int returns an integer from the Value of the SpecAttr. If The value is not a LiteralValue or the value
// cannot be converted to an integer an error is returned.
func (a *SpecAttr) Int() (int, error) {
	lit, ok := a.V.(*LiteralValue)
	if !ok {
		return 0, fmt.Errorf("schema: cannot read attribute %q as literal", a.K)
	}
	s, err := strconv.Atoi(lit.V)
	if err != nil {
		return 0, fmt.Errorf("schema: cannot read attribute %q as integer", a.K)
	}
	return s, nil
}

// StringList returns a slice of strings from the Value of the SpecAttr. If The value is not a ListValue or the its
// values cannot be converted to strings an error is returned.
func (a *SpecAttr) StringList() ([]string, error) {
	lst, ok := a.V.(*ListValue)
	if !ok {
		return nil, fmt.Errorf("schema: attribute %q is not a list", a.K)
	}
	out := make([]string, 0, len(lst.V))
	for _, item := range lst.V {
		unquote, err := strconv.Unquote(item)
		if err != nil {
			return nil, fmt.Errorf("schema: failed parsing item %q to string: %w", item, err)
		}
		out = append(out, unquote)
	}
	return out, nil
}


func (*LiteralValue) val() {}
func (*ListValue) val() {}

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
