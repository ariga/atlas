package schemaspec

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

	// Resource is a generic container for resources described in configurations.
	Resource struct {
		Name     string
		Type     string
		Attrs    []*Attr
		Children []*Resource
	}

	// Schema holds a specification for a Schema.
	Schema struct {
		Name   string
		Tables []*Table
	}

	// Table holds a specification for an SQL table.
	Table struct {
		Name        string
		SchemaName  string
		Columns     []*Column
		PrimaryKey  *PrimaryKey
		ForeignKeys []*ForeignKey
		Indexes     []*Index
		Attrs       []*Attr
		Children    []*Resource
	}

	// Column holds a specification for a column in an SQL table.
	Column struct {
		Name      string
		Type      string        `override:"type"`
		Default   *LiteralValue `override:"default"`
		Null      bool          `override:"null"`
		Attrs     []*Attr
		Children  []*Resource
		Overrides []*Override
	}

	// PrimaryKey holds a specification for the primary key of a table.
	PrimaryKey struct {
		Columns  []*ColumnRef
		Attrs    []*Attr
		Children []*Resource
	}

	// ForeignKey holds a specification for a foreign key of a table.
	ForeignKey struct {
		Symbol     string
		Columns    []*ColumnRef
		RefColumns []*ColumnRef
		OnUpdate   string
		OnDelete   string
		Attrs      []*Attr
		Children   []*Resource
	}

	// Index holds a specification for an index of a table.
	Index struct {
		Name     string
		Columns  []*ColumnRef
		Unique   bool
		Attrs    []*Attr
		Children []*Resource
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

	// Override contains information about how to override some attributes of an Element
	// for a specific dialect and version. For example, to select a specific column type or add
	// special attributes when using MySQL, but not when using SQLite or Postgres.
	Override struct {
		Dialect string
		Version string
		Resource
	}

	// Element is an object that can be encoded into bytes to be written to a configuration file representing
	// Schema resources.
	Element interface {
		elem()
	}

	// Attr is an attribute of a Spec.
	Attr struct {
		K string
		V Value
	}

	// Value represents the value of a Attr.
	Value interface {
		val()
	}

	// LiteralValue implements Value and represents a literal value (string, number, etc.)
	LiteralValue struct {
		V string
	}

	// ListValue implements Value and represents a list of literal value (string, number, etc.)
	ListValue struct {
		V []string
	}

	// Overrider is the interface that wraps the Override method. Element types that implement
	// this interface can expose an Override object for specific dialect versions.
	Overrider interface {
		Override(versions ...string) *Override
	}

	// Attributer facilitates the reading and writing of attributes.
	Attributer interface {

		// Attr returns the Value of an attribute of the resource and reports whether an
		// attribute by that name was found.
		Attr(name string) (*Attr, bool)

		// SetAttr sets the attribute on the Element. If an Attr with the same name exists
		// it is replaced, if it does not exist it is replaced.
		SetAttr(*Attr)
	}
)

// Table returns the first table that matches the given name and reports whether such a table was found.
func (s *Schema) Table(name string) (*Table, bool) {
	for _, t := range s.Tables {
		if t.Name == name {
			return t, true
		}
	}
	return nil, false
}

// Column returns the first column that matches the given name and reports whether such a column was found.
func (t *Table) Column(name string) (*Column, bool) {
	for _, c := range t.Columns {
		if c.Name == name {
			return c, true
		}
	}
	return nil, false
}

// Index returns the first index that matches the given name and reports whether such a column was found.
func (t *Table) Index(name string) (*Index, bool) {
	for _, i := range t.Indexes {
		if i.Name == name {
			return i, true
		}
	}
	return nil, false
}

// Attr returns the value of the Column attribute named `name` and reports whether such an attribute exists.
func (c *Column) Attr(name string) (*Attr, bool) {
	return getAttrVal(c.Attrs, name)
}

func (c *Column) SetAttr(attr *Attr) {
	c.Attrs = replaceOrAppendAttr(c.Attrs, attr)
}

// Attr returns the value of the Table attribute named `name` and reports whether such an attribute exists.
func (t *Table) Attr(name string) (*Attr, bool) {
	return getAttrVal(t.Attrs, name)
}

// Override searches the Column's Overrides for ones matching any of the versions
// passed to it. It then creates an Override by merging the overrides for all of
// the matching versions in the order they were passed.
func (c *Column) Override(versions ...string) *Override {
	var override *Override
	for _, version := range versions {
		for _, o := range c.Overrides {
			if o.version() == version {
				if override == nil {
					override = o
				}
				override.Merge(&o.Resource)
			}
		}
	}
	return override
}

func getAttrVal(attrs []*Attr, name string) (*Attr, bool) {
	for _, attr := range attrs {
		if attr.K == name {
			return attr, true
		}
	}
	return nil, false
}

func replaceOrAppendAttr(attrs []*Attr, attr *Attr) []*Attr {
	for i, v := range attrs {
		if v.K == attr.K {
			attrs[i] = attr
			return attrs
		}
	}
	return append(attrs, attr)
}

// Int returns an integer from the Value of the Attr. If The value is not a LiteralValue or the value
// cannot be converted to an integer an error is returned.
func (a *Attr) Int() (int, error) {
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

// String returns a string from the Value of the Attr. If The value is not a LiteralValue
// an error is returned.  String values are expected to be quoted. If the value is not
// properly quoted an error is returned.
func (a *Attr) String() (string, error) {
	lit, ok := a.V.(*LiteralValue)
	if !ok {
		return "", fmt.Errorf("schema: cannot read attribute %q as literal", a.K)
	}
	return strconv.Unquote(lit.V)
}

// Bool returns a boolean from the Value of the Attr. If The value is not a LiteralValue or the value
// cannot be converted to a boolean an error is returned.
func (a *Attr) Bool() (bool, error) {
	lit, ok := a.V.(*LiteralValue)
	if !ok {
		return false, fmt.Errorf("schema: cannot read attribute %q as literal", a.K)
	}
	return strconv.ParseBool(lit.V)
}

// Strings returns a slice of strings from the Value of the Attr. If The value is not a ListValue or the its
// values cannot be converted to strings an error is returned.
func (a *Attr) Strings() ([]string, error) {
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

// Merge merges the attributes of another Resource into the Resource.
func (r *Resource) Merge(other *Resource) {
	for _, attr := range other.Attrs {
		r.SetAttr(attr)
	}
}

func (o *Override) version() string {
	v := o.Dialect
	if o.Version != "" {
		v += " " + o.Version
	}
	return v
}

func (r *Resource) Attr(name string) (*Attr, bool) {
	return getAttrVal(r.Attrs, name)
}

func (r *Resource) SetAttr(attr *Attr) {
	r.Attrs = replaceOrAppendAttr(r.Attrs, attr)
}

func (*LiteralValue) val() {}
func (*ListValue) val()    {}

func (*Resource) elem()   {}
func (*Attr) elem()       {}
func (*Column) elem()     {}
func (*Table) elem()      {}
func (*Schema) elem()     {}
func (*PrimaryKey) elem() {}
func (*ForeignKey) elem() {}
func (*Index) elem()      {}

func (*Column) spec()     {}
func (*Table) spec()      {}
func (*Schema) spec()     {}
func (*Resource) spec()   {}
func (*PrimaryKey) spec() {}
func (*ForeignKey) spec() {}
func (*Index) spec()      {}
