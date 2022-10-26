// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl

import (
	"fmt"
	"math/big"
	"reflect"

	"ariga.io/atlas/sql/schema"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

type (
	// Resource is a generic container for resources described in configurations.
	Resource struct {
		Name      string
		Qualifier string
		Type      string
		Attrs     []*Attr
		Children  []*Resource
	}

	// Attr is an attribute of a Resource.
	Attr struct {
		K string
		V cty.Value
	}

	// Ref implements Value and represents a reference to another Resource.
	// The path to a Resource under the root Resource is expressed as "$<type>.<name>..."
	// recursively. For example, a resource of type "table" that is named "users" and is a direct
	// child of the root Resource's address shall be "$table.users". A child resource of that table
	// of type "column" and named "id", shall be referenced as "$table.users.$column.id", and so on.
	Ref struct {
		V string
	}

	// RawExpr implements Value and represents any raw expression.
	RawExpr struct {
		X string
	}

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

		// Format is an optional formatting function.
		// If exists, it will be used instead the registry one.
		Format func(*Type) (string, error)

		// FromSpec is an optional function that can be attached
		// to the type spec and allows converting the schema spec
		// type to a schema type (from document to databse).
		FromSpec func(*Type) (schema.Type, error)

		// ToSpec is an optional function that can be attached
		// to the type spec and allows converting the schema type
		// to a schema spec type (from database to document).
		ToSpec func(schema.Type) (*Type, error)
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

// IsRef indicates if the attribute is a reference type.
func (a *Attr) IsRef() bool {
	if !a.V.Type().IsCapsuleType() {
		return false
	}
	_, ok := a.V.EncapsulatedValue().(*Ref)
	return ok
}

// IsRawExpr indicates if the attribute is a RawExpr type.
func (a *Attr) IsRawExpr() bool {
	if !a.V.Type().IsCapsuleType() {
		return false
	}
	_, ok := a.V.EncapsulatedValue().(*RawExpr)
	return ok
}

// IsType indicates if the attribute is a type spec.
func (a *Attr) IsType() bool {
	if !a.V.Type().IsCapsuleType() {
		return false
	}
	_, ok := a.V.EncapsulatedValue().(*Type)
	return ok
}

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
func (a *Attr) Int64() (i int64, err error) {
	if err = gocty.FromCtyValue(a.V, &i); err != nil {
		return 0, err
	}
	return i, nil
}

// String returns a string from the Value of the Attr. If The value is not a LiteralValue
// an error is returned.  String values are expected to be quoted. If the value is not
// properly quoted an error is returned.
func (a *Attr) String() (s string, err error) {
	if err = gocty.FromCtyValue(a.V, &s); err != nil {
		return "", err
	}
	return s, nil
}

// Bool returns a boolean from the Value of the Attr. If The value is not a LiteralValue or the value
// cannot be converted to a boolean an error is returned.
func (a *Attr) Bool() (b bool, err error) {
	if err = gocty.FromCtyValue(a.V, &b); err != nil {
		return false, err
	}
	return b, nil
}

// Ref extracts the reference from the Value of the Attr.
func (a *Attr) Ref() (string, error) {
	ref, ok := a.V.EncapsulatedValue().(*Ref)
	if !ok {
		return "", fmt.Errorf("schema: cannot read attribute %q as ref", a.K)
	}
	return ref.V, nil
}

// Type extracts the Type from the Attr.
func (a *Attr) Type() (*Type, error) {
	t, ok := a.V.EncapsulatedValue().(*Type)
	if !ok {
		return nil, fmt.Errorf("schema: cannot read attribute %q as type", a.K)
	}
	return t, nil
}

// RawExpr extracts the RawExpr from the Attr.
func (a *Attr) RawExpr() (*RawExpr, error) {
	if !a.IsRawExpr() {
		return nil, fmt.Errorf("schema: cannot read attribute %q as raw expression", a.K)
	}
	return a.V.EncapsulatedValue().(*RawExpr), nil
}

// Refs returns a slice of references.
func (a *Attr) Refs() ([]*Ref, error) {
	refs := make([]*Ref, 0, len(a.V.AsValueSlice()))
	for _, v := range a.V.AsValueSlice() {
		ref, ok := v.EncapsulatedValue().(*Ref)
		if !ok {
			return nil, fmt.Errorf("schema: cannot read attribute %q as ref", a.K)
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

// Strings returns a slice of strings from the Value of the Attr. If The value is not a ListValue or its
// values cannot be converted to strings an error is returned.
func (a *Attr) Strings() (vs []string, err error) {
	if a.V.Type().IsTupleType() {
		for _, v := range a.V.AsValueSlice() {
			var s string
			if err = gocty.FromCtyValue(v, &s); err != nil {
				return nil, err
			}
			vs = append(vs, s)
		}
		return vs, nil
	}
	if err = gocty.FromCtyValue(a.V, &vs); err != nil {
		return nil, err
	}
	return vs, nil
}

// Bools returns a slice of bools from the Value of the Attr. If The value is not a ListValue or its
// values cannot be converted to bools an error is returned.
func (a *Attr) Bools() (vs []bool, err error) {
	if a.V.Type().IsTupleType() {
		for _, v := range a.V.AsValueSlice() {
			var b bool
			if err = gocty.FromCtyValue(v, &b); err != nil {
				return nil, err
			}
			vs = append(vs, b)
		}
		return vs, nil
	}
	if err = gocty.FromCtyValue(a.V, &vs); err != nil {
		return nil, err
	}
	return vs, nil
}

// Resource returns the first child Resource by its type and reports whether it was found.
func (r *Resource) Resource(t string) (*Resource, bool) {
	if r == nil {
		return nil, false
	}
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
func (f MarshalerFunc) MarshalSpec(v any) ([]byte, error) {
	return f(v)
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

// Attr returns a TypeAttr by name and reports if one was found.
func (s *TypeSpec) Attr(name string) (*TypeAttr, bool) {
	for _, ta := range s.Attributes {
		if ta.Name == name {
			return ta, true
		}
	}
	return nil, false
}

var _ Marshaler = MarshalerFunc(nil)

// StringAttr is a helper method for constructing *schemahcl.Attr instances that contain string value.
func StringAttr(k string, v string) *Attr {
	return &Attr{
		K: k,
		V: cty.StringVal(v),
	}
}

// IntAttr is a helper method for constructing *schemahcl.Attr instances that contain int64 value.
func IntAttr(k string, v int) *Attr {
	return Int64Attr(k, int64(v))
}

// Int64Attr is a helper method for constructing *schemahcl.Attr instances that contain int64 value.
func Int64Attr(k string, v int64) *Attr {
	return &Attr{
		K: k,
		V: cty.NumberVal(new(big.Float).SetInt64(v).SetPrec(512)),
	}
}

// BoolAttr is a helper method for constructing *schemahcl.Attr instances that contain a boolean value.
func BoolAttr(k string, v bool) *Attr {
	return &Attr{
		K: k,
		V: cty.BoolVal(v),
	}
}

// RefAttr is a helper method for constructing *schemahcl.Attr instances that contain a Ref value.
func RefAttr(k string, v *Ref) *Attr {
	return &Attr{
		K: k,
		V: cty.CapsuleVal(ctyRefType, v),
	}
}

// StringsAttr is a helper method for constructing *schemahcl.Attr instances that contain list strings.
func StringsAttr(k string, vs ...string) *Attr {
	vv := make([]cty.Value, len(vs))
	for i, v := range vs {
		vv[i] = cty.StringVal(v)
	}
	return &Attr{
		K: k,
		V: cty.ListVal(vv),
	}
}

// RefsAttr is a helper method for constructing *schemahcl.Attr instances that contain list references.
func RefsAttr(k string, refs ...*Ref) *Attr {
	vv := make([]cty.Value, len(refs))
	for i, v := range refs {
		vv[i] = cty.CapsuleVal(ctyRefType, v)
	}
	return &Attr{
		K: k,
		V: cty.ListVal(vv),
	}
}

// RawAttr is a helper method for constructing *schemahcl.Attr instances that contain RawExpr value.
func RawAttr(k string, x string) *Attr {
	return &Attr{
		K: k,
		V: RawExprValue(&RawExpr{X: x}),
	}
}

// RawExprValue is a helper method for constructing a cty.Value that capsules a raw expression.
func RawExprValue(x *RawExpr) cty.Value {
	return cty.CapsuleVal(ctyRawExpr, x)
}
