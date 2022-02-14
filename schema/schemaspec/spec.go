package schemaspec

import (
	"fmt"
	"reflect"
	"strconv"
)

type (
	// Resource is a generic container for resources described in configurations.
	Resource struct {
		Name     string
		Type     string
		Attrs    []*Attr
		Children []*Resource
	}

	// Attr is an attribute of a Resource.
	Attr struct {
		K string
		V Value
	}

	// Value represents the value of an Attr.
	Value interface {
		val()
	}

	// LiteralValue implements Value and represents a literal value (string, number, etc.)
	LiteralValue struct {
		V string
	}

	// RawExpr implements Value and represents any raw expression.
	RawExpr struct {
		X string
	}

	// ListValue implements Value and represents a list of Values.
	ListValue struct {
		V []Value
	}

	// Ref implements Value and represents a reference to another Resource.
	// The path to a Resource under the root Resource is expressed as "$<type>.<name>..."
	// recursively. For example, a resource of type "table" that is named "users" and is a direct
	// child of the root Resource's address shall be "$table.users". A child resource of that table
	// of type "column" and named "id", shall be referenced as "$table.users.$column.id", and so on.
	Ref struct {
		V string
	}

	// Marshaler is the interface implemented by types that can marshal objects into a
	// valid Atlas DDL representation.
	Marshaler interface {
		MarshalSpec(interface{}) ([]byte, error)
	}

	// Unmarshaler is the interface implemented by types that can unmarshal an Atlas DDL
	// representation into an object.
	Unmarshaler interface {
		UnmarshalSpec([]byte, interface{}) error
	}

	// MarshalerFunc is the function type that is implemented by the MarshalSpec
	// method of the Marshaler interface.
	MarshalerFunc func(interface{}) ([]byte, error)

	// UnmarshalerFunc is the function type that is implemented by the UnmarshalSpec
	// method of the Unmarshaler interface.
	UnmarshalerFunc func([]byte, interface{}) error

	// TypeSpec represents a specification for defining a Type.
	TypeSpec struct {
		// Name is the identifier for the type in an Atlas DDL document.
		Name string
		// T is the database identifier for the type.
		T          string
		Attributes []*TypeAttr
		// RType is the reflect.Type of the schema.Type used to describe the TypeSpec.
		// This field is optional and used to determine the TypeSpec in cases where the
		// schema.Type does not have a `T` field.
		RType reflect.Type
		// Printer is an optional function used by the drivers to get a string representation of the
		// represented Type, that can be parsed by the driver into a schema.Type.
		Printer func(*Type) (string, error)
	}

	// TypeAttr describes an attribute of a TypeSpec, for example `varchar` fields
	// can have a `size` attribute.
	TypeAttr struct {
		// Name should be a snake_case of related the schema.Type struct field.
		Name     string
		Kind     reflect.Kind
		Required bool
	}

	// Type represents the type of the field in a schema.
	Type struct {
		T     string
		Attrs []*Attr
		IsRef bool
	}
)

// Int returns an int from the Value of the Attr. If The value is not a LiteralValue or the value
// cannot be converted to an integer an error is returned.
func (a *Attr) Int() (int, error) {
	i, err := a.Int64()
	if err != nil {
		return 0, err
	}
	return int(i), nil
}

// Int64 returns an int64 from the Value of the Attr. If The value is not a LiteralValue or the value
// cannot be converted to an integer an error is returned.
func (a *Attr) Int64() (int64, error) {
	lit, ok := a.V.(*LiteralValue)
	if !ok {
		return 0, fmt.Errorf("schema: cannot read attribute %q as literal", a.K)
	}
	i, err := strconv.ParseInt(lit.V, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("schema: cannot read attribute %q as integer: %w", a.K, err)
	}
	return i, nil
}

// String returns a string from the Value of the Attr. If The value is not a LiteralValue
// an error is returned.  String values are expected to be quoted. If the value is not
// properly quoted an error is returned.
func (a *Attr) String() (string, error) {
	return StrVal(a.V)
}

// Bool returns a boolean from the Value of the Attr. If The value is not a LiteralValue or the value
// cannot be converted to a boolean an error is returned.
func (a *Attr) Bool() (bool, error) {
	lit, ok := a.V.(*LiteralValue)
	if !ok {
		return false, fmt.Errorf("schema: cannot read attribute %q as literal", a.K)
	}
	b, err := strconv.ParseBool(lit.V)
	if err != nil {
		return false, fmt.Errorf("schema: cannot read attribute %q as bool: %w", a.K, err)
	}
	return b, nil
}

// Ref returns the string representation of the Attr. If the value is not a Ref or the value
// an error is returned.
func (a *Attr) Ref() (string, error) {
	ref, ok := a.V.(*Ref)
	if !ok {
		return "", fmt.Errorf("schema: cannot read attribute %q as ref", a.K)
	}
	return ref.V, nil
}

// Strings returns a slice of strings from the Value of the Attr. If The value is not a ListValue or its
// values cannot be converted to strings an error is returned.
func (a *Attr) Strings() ([]string, error) {
	lst, ok := a.V.(*ListValue)
	if !ok {
		return nil, fmt.Errorf("schema: attribute %q is not a list", a.K)
	}
	out := make([]string, 0, len(lst.V))
	for _, item := range lst.V {
		sv, err := StrVal(item)
		if err != nil {
			return nil, fmt.Errorf("schemaspec: failed parsing item %q to string: %w", item, err)
		}
		out = append(out, sv)
	}
	return out, nil
}

// Bools returns a slice of bools from the Value of the Attr. If The value is not a ListValue or its
// values cannot be converted to bools an error is returned.
func (a *Attr) Bools() ([]bool, error) {
	lst, ok := a.V.(*ListValue)
	if !ok {
		return nil, fmt.Errorf("schemaspec: attribute %q is not a list", a.K)
	}
	out := make([]bool, 0, len(lst.V))
	for _, item := range lst.V {
		b, err := BoolVal(item)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

// Resource returns the first child Resource by its type and reports whether it was found.
func (r *Resource) Resource(t string) (*Resource, bool) {
	for i := range r.Children {
		if r.Children[i].Type == t {
			return r.Children[i], true
		}
	}
	return nil, false
}

// Attr returns the Attr by the provided name and reports whether it was found.
func (r *Resource) Attr(name string) (*Attr, bool) {
	return attrVal(r.Attrs, name)
}

// SetAttr sets the Attr on the Resource. If r is nil, a zero value Resource
// is initialized. If an Attr with the same key exists, it is replaced by attr.
func (r *Resource) SetAttr(attr *Attr) {
	if r == nil {
		*r = Resource{}
	}
	r.Attrs = replaceOrAppendAttr(r.Attrs, attr)
}

// MarshalSpec implements Marshaler.
func (f MarshalerFunc) MarshalSpec(v interface{}) ([]byte, error) {
	return f(v)
}

// UnmarshalSpec implements Unmarshaler.
func (f UnmarshalerFunc) UnmarshalSpec(data []byte, v interface{}) error {
	return f(data, v)
}

func attrVal(attrs []*Attr, name string) (*Attr, bool) {
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

// StrVal returns the raw string representation of v. If v is not a *LiteralValue
// it returns an error. If the raw string representation of v cannot be read as
// a string by unquoting it, an error is returned as well.
func StrVal(v Value) (string, error) {
	lit, ok := v.(*LiteralValue)
	if !ok {
		return "", fmt.Errorf("schemaspec: expected %T to be LiteralValue", v)
	}
	return strconv.Unquote(lit.V)
}

// BoolVal returns the bool representation of v. If v is not a *LiteralValue
// it returns an error. If the raw string representation of v cannot be read as
// a bool, an error is returned as well.
func BoolVal(v Value) (bool, error) {
	lit, ok := v.(*LiteralValue)
	if !ok {
		return false, fmt.Errorf("schemaspec: expected %T to be LiteralValue", v)
	}
	b, err := strconv.ParseBool(lit.V)
	if err != nil {
		return false, fmt.Errorf("schemaspec: failed parsing %q as bool: %w", lit.V, err)
	}
	return b, nil
}

// Attr returns a TypeAttr by name and reports if one was found.
func (s *TypeSpec) Attr(name string) (*TypeAttr, bool) {
	for _, ta := range s.Attributes {
		if ta.Name == name {
			return ta, true
		}
	}
	return nil, false
}

func (*LiteralValue) val() {}
func (*RawExpr) val()      {}
func (*ListValue) val()    {}
func (*Ref) val()          {}
func (*Type) val()         {}

var (
	_ Unmarshaler = UnmarshalerFunc(nil)
	_ Marshaler   = MarshalerFunc(nil)
)
