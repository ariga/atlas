// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"ariga.io/atlas/sql/schema"

	"github.com/hashicorp/hcl/v2"
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
		rang      *hcl.Range
	}

	// Attr is an attribute of a Resource.
	Attr struct {
		K    string
		V    cty.Value
		rang *hcl.Range
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
		// type to a schema type (from document to database).
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

// IsRefTo indicates if the Type is a reference to specific schema type definition.
func (t *Type) IsRefTo(n string) bool {
	if !t.IsRef {
		return false
	}
	path, err := (&Ref{V: t.T}).ByType(n)
	return err == nil && len(path) > 0
}

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

// Float64 returns a float64 from the Value of the Attr. If The value is not a LiteralValue or the value
// cannot be converted to a float64 an error is returned.
func (a *Attr) Float64() (f float64, err error) {
	if err = gocty.FromCtyValue(a.V, &f); err != nil {
		return 0, err
	}
	return f, nil
}

// BigFloat returns a big.Float from the Value of the Attr. If The value is not a LiteralValue or the value
// cannot be converted to a big.Float an error is returned.
func (a *Attr) BigFloat() (*big.Float, error) {
	var f big.Float
	if err := gocty.FromCtyValue(a.V, &f); err != nil {
		return nil, err
	}
	return &f, nil
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
// an error is returned. String values are expected to be quoted. If the value is not
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

// PathIndex represents an index in a reference path.
type PathIndex struct {
	T string   // type
	V []string // identifiers
}

// Check if the path index is valid.
func (p *PathIndex) Check() error {
	if p.T == "" || len(p.V) == 0 {
		return fmt.Errorf("schemahcl: missing type or identifier %v", p)
	}
	for _, v := range p.V {
		if v == "" {
			return fmt.Errorf("schemahcl: empty identifier %v", p)
		}
	}
	return nil
}

// ByType returns the path index for the given type.
func (r *Ref) ByType(name string) ([]string, error) {
	if r == nil {
		return nil, fmt.Errorf("schemahcl: type %q was not found in nil reference", name)
	}
	path, err := r.Path()
	if err != nil {
		return nil, err
	}
	var vs []string
	for _, p := range path {
		switch {
		case p.T != name:
		case vs != nil:
			return nil, fmt.Errorf("schemahcl: multiple %q found in reference", name)
		default:
			if err := p.Check(); err != nil {
				return nil, err
			}
			vs = p.V
		}
	}
	if vs == nil {
		return nil, fmt.Errorf("schemahcl: missing %q in reference", name)
	}
	return vs, nil
}

// Path returns a parsed path including block types and their identifiers.
func (r *Ref) Path() (path []PathIndex, err error) {
	for i := 0; i < len(r.V); i++ {
		var part PathIndex
		switch idx := strings.IndexAny(r.V[i:], ".["); {
		case r.V[i] != '$':
			return nil, fmt.Errorf("schemahcl: missing type in reference %q", r.V[i:])
		case idx == -1:
			return nil, fmt.Errorf("schemahcl: missing identifier in reference %q", r.V[i:])
		default:
			part.T = r.V[i+1 : i+idx]
			i += idx
		}
	Ident:
		for i < len(r.V) {
			switch {
			// End of identifier before a type.
			case strings.HasPrefix(r.V[i:], ".$"):
				break Ident
			// Scan identifier.
			case r.V[i] == '.':
				v := r.V[i+1:]
				if idx := strings.IndexAny(v, ".["); idx != -1 {
					v = v[:idx]
				}
				part.V = append(part.V, v)
				i += 1 + len(v)
			// Scan attribute (["..."]).
			case strings.HasPrefix(r.V[i:], "[\""):
				idx := scanString(r.V[i+2:])
				if idx == -1 {
					return nil, fmt.Errorf("schemahcl: unterminated string in reference %q", r.V[i:])
				}
				v := r.V[i+2 : i+2+idx]
				i += 2 + idx
				if !strings.HasPrefix(r.V[i:], "\"]") {
					return nil, fmt.Errorf("schemahcl: missing ']' in reference %q", r.V[i:])
				}
				part.V = append(part.V, v)
				i += 2
			default:
				return nil, fmt.Errorf("schemahcl: invalid character in reference %q", r.V[i:])
			}
		}
		if err := part.Check(); err != nil {
			return nil, err
		}
		path = append(path, part)
	}
	return
}

// BuildRef from a path.
func BuildRef(path []PathIndex) *Ref {
	var v string
	for _, p := range path {
		switch {
		case len(p.V) == 1:
			v = addr(v, p.T, p.V[0], "")
		case len(p.V) == 2:
			v = addr(v, p.T, p.V[1], p.V[0])
		default:
			v = addr(v, p.T, "", "")
		}
	}
	return &Ref{V: v}
}

func scanString(s string) int {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '"':
			return i
		}
	}
	return -1
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

// SetRange sets the range of this attribute.
func (a *Attr) SetRange(p *hcl.Range) {
	a.rang = p
}

// Range returns the attribute range on the
// file, or nil if it is not set.
func (a *Attr) Range() *hcl.Range {
	return a.rang
}

// SetRange sets the range of this resource.
func (r *Resource) SetRange(p *hcl.Range) {
	r.rang = p
}

// Range returns the resource range on the
// file, or nil if it is not set.
func (r *Resource) Range() *hcl.Range {
	return r.rang
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

// Resources returns all child Resources by its type.
func (r *Resource) Resources(t string) []*Resource {
	if r == nil {
		return nil
	}
	var rs []*Resource
	for i := range r.Children {
		if r.Children[i].Type == t {
			rs = append(rs, r.Children[i])
		}
	}
	return rs
}

// Attr returns the Attr by the provided name and reports whether it was found.
func (r *Resource) Attr(name string) (*Attr, bool) {
	if at, ok := attrVal(r.Attrs, name); ok {
		return at, true
	}
	for _, r := range r.Children {
		if at, ok := attrVal(r.Attrs, name); ok && r.Type == "" {
			return at, true // Match on embedded resource.
		}
	}
	return nil, false
}

// SetAttr sets the Attr on the Resource. If r is nil, a zero value Resource
// is initialized. If an Attr with the same key exists, it is replaced by attr.
func (r *Resource) SetAttr(attr *Attr) {
	r.Attrs = replaceOrAppendAttr(r.Attrs, attr)
}

// EmbedAttr is like SetAttr but appends the attribute to an embedded
// resource, cause it to be marshaled after current blocks and attributes.
func (r *Resource) EmbedAttr(attr *Attr) {
	r.Children = append(r.Children, &Resource{
		Attrs: []*Attr{attr},
	})
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

// Float64Attr is a helper method for constructing *schemahcl.Attr instances that contain float64 value.
func Float64Attr(k string, v float64) *Attr {
	return &Attr{
		K: k,
		V: cty.NumberFloatVal(v),
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

// RefValue is a helper method for constructing a cty.Value that contains a Ref value.
func RefValue(v string) cty.Value {
	return cty.CapsuleVal(ctyRefType, &Ref{V: v})
}

// TypeValue is a helper method for constructing a cty.Value that contains a Type value.
func TypeValue(t *Type) cty.Value {
	return cty.CapsuleVal(ctyTypeSpec, t)
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

// ctyEnumString is a capsule type for EnumString.
var ctyEnumString = cty.Capsule("enum_string", reflect.TypeOf(EnumString{}))

// EnumString is a helper type that represents
// either an enum or a string value.
type EnumString struct {
	E, S string // Enum or string value.
}

// StringEnumsAttr is a helper method for constructing *schemahcl.Attr instances
// that contain list of elements that their values can be either enum or string.
func StringEnumsAttr(k string, elems ...*EnumString) *Attr {
	vv := make([]cty.Value, len(elems))
	for i, e := range elems {
		vv[i] = cty.CapsuleVal(ctyEnumString, e)
	}
	return &Attr{
		K: k,
		V: cty.ListVal(vv),
	}
}

// RangeAsPos builds a schema position from the give HCL range.
func RangeAsPos(r *hcl.Range) *schema.Pos {
	return &schema.Pos{
		Filename: r.Filename,
		Start:    r.Start,
		End:      r.End,
	}
}

// AppendPos appends the range to the attributes.
func AppendPos(attrs *[]schema.Attr, r *hcl.Range) {
	if r != nil {
		schema.ReplaceOrAppend(attrs, RangeAsPos(r))
	}
}
