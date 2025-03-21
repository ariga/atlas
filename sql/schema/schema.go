// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schema

import (
	"reflect"
)

type (
	// A Realm or a database describes a domain of schema resources that are logically connected
	// and can be accessed and queried in the same connection (e.g. a physical database instance).
	Realm struct {
		Schemas []*Schema
		Attrs   []Attr
		Objects []Object // Realm-level objects (e.g., users or extensions).
	}

	// A Schema describes a database schema (i.e. named database).
	Schema struct {
		Name    string
		Realm   *Realm
		Tables  []*Table
		Views   []*View
		Funcs   []*Func
		Procs   []*Proc
		Attrs   []Attr   // Attrs and options.
		Objects []Object // Schema-level objects (e.g., types or sequences).
	}

	// An Object represents a generic database object.
	// Note that this interface is implemented by some top-level types
	// to describe their relationship, and by driver specific types.
	Object interface {
		obj()
	}

	// A Table represents a table definition.
	Table struct {
		Name        string
		Schema      *Schema
		Columns     []*Column
		Indexes     []*Index
		PrimaryKey  *Index
		ForeignKeys []*ForeignKey
		Attrs       []Attr     // Attrs, constraints and options.
		Triggers    []*Trigger // Triggers on the table.
		Deps        []Object   // Objects this table depends on.
		Refs        []Object   // Objects that depends on this table.
	}

	// A View represents a view definition.
	View struct {
		Name     string
		Def      string
		Schema   *Schema
		Columns  []*Column
		Attrs    []Attr     // Attrs and options.
		Indexes  []*Index   // Indexes on materialized view.
		Triggers []*Trigger // Triggers on the view.
		Deps     []Object   // Objects this view depends on.
		Refs     []Object   // Objects that depends on this view.
	}

	// A Column represents a column definition.
	Column struct {
		Name    string
		Type    *ColumnType
		Default Expr
		Attrs   []Attr
		Indexes []*Index
		// Foreign keys that this column is
		// part of their child columns.
		ForeignKeys []*ForeignKey
	}
	// NamedDefault defines a named default expression.
	NamedDefault struct {
		Expr
		Name  string
		Attrs []Attr
	}
	// ColumnType represents a column type that is implemented by the dialect.
	ColumnType struct {
		Type Type
		Raw  string
		Null bool
	}

	// An Index represents an index definition.
	Index struct {
		Name   string
		Unique bool
		// Table or View that this index belongs to.
		Table *Table
		View  *View
		Attrs []Attr
		Parts []*IndexPart
	}

	// An IndexPart represents an index part that
	// can be either an expression or a column.
	IndexPart struct {
		// SeqNo represents the sequence number of the key part
		// in the index.
		SeqNo int
		// Desc indicates if the key part is stored in descending
		// order. All databases use ascending order as default.
		Desc  bool
		X     Expr
		C     *Column
		Attrs []Attr
	}

	// A ForeignKey represents an index definition.
	ForeignKey struct {
		Symbol     string // Constraint name, if exists.
		Table      *Table
		Columns    []*Column
		RefTable   *Table
		RefColumns []*Column
		OnUpdate   ReferenceOption
		OnDelete   ReferenceOption
		Attrs      []Attr
	}

	// A Trigger represents a trigger definition.
	Trigger struct {
		Name string
		// Table or View that this trigger belongs to.
		Table      *Table
		View       *View
		ActionTime TriggerTime    // BEFORE, AFTER, or INSTEAD OF.
		Events     []TriggerEvent // INSERT, UPDATE, DELETE, etc.
		For        TriggerFor     // FOR EACH ROW or FOR EACH STATEMENT.
		Body       string         // Trigger body only.
		Attrs      []Attr         // WHEN, REFERENCING, etc.
		Deps       []Object       // Objects this trigger depends on.
		Refs       []Object       // Objects that depend on this trigger.
	}

	// TriggerTime represents the trigger action time.
	TriggerTime string

	// TriggerFor represents the trigger FOR EACH spec.
	TriggerFor string

	// TriggerEvent represents the trigger event.
	TriggerEvent struct {
		Name    string    // Name of the event (e.g. INSERT, UPDATE, DELETE).
		Columns []*Column // Columns that might be associated with the event.
	}

	// Func represents a function definition.
	Func struct {
		Name   string
		Schema *Schema
		Args   []*FuncArg
		Ret    Type
		Body   string   // Function body only.
		Lang   string   // Language (e.g. SQL, PL/pgSQL, etc.).
		Attrs  []Attr   // Extra driver specific attributes.
		Deps   []Object // Objects this function depends on.
		Refs   []Object // Objects that depend on this function.
	}

	// Proc represents a procedure definition.
	Proc struct {
		Name   string
		Schema *Schema
		Args   []*FuncArg
		Body   string   // Function body only.
		Lang   string   // Language (e.g. SQL, PL/pgSQL, etc.).
		Attrs  []Attr   // Extra driver specific attributes.
		Deps   []Object // Objects this function depends on.
		Refs   []Object // Objects that depend on this Proc.
	}

	// A FuncArg represents a single function argument.
	FuncArg struct {
		Name    string      // Optional name.
		Type    Type        // Argument type.
		Default Expr        // Default value.
		Mode    FuncArgMode // Argument mode.
		Attrs   []Attr      // Extra driver specific attributes.
	}

	// FuncArgMode represents a function argument mode.
	FuncArgMode string
)

// List of supported function argument modes.
const (
	FuncArgModeIn       FuncArgMode = "IN"
	FuncArgModeOut      FuncArgMode = "OUT"
	FuncArgModeInOut    FuncArgMode = "INOUT"
	FuncArgModeVariadic FuncArgMode = "VARIADIC"
)

// List of supported trigger action times.
const (
	TriggerTimeBefore  TriggerTime = "BEFORE"
	TriggerTimeAfter   TriggerTime = "AFTER"
	TriggerTimeInstead TriggerTime = "INSTEAD OF"
)

// List of supported trigger FOR EACH spec.
const (
	TriggerForRow  TriggerFor = "ROW"
	TriggerForStmt TriggerFor = "STATEMENT"
)

// List of supported trigger events.
var (
	TriggerEventInsert   = TriggerEvent{Name: "INSERT"}
	TriggerEventUpdate   = TriggerEvent{Name: "UPDATE"}
	TriggerEventDelete   = TriggerEvent{Name: "DELETE"}
	TriggerEventTruncate = TriggerEvent{Name: "TRUNCATE"}
)

// TriggerEventUpdateOf returns an UPDATE OF trigger event.
func TriggerEventUpdateOf(columns ...*Column) TriggerEvent {
	return TriggerEvent{Name: "UPDATE OF", Columns: columns}
}

// Schema returns the first schema that matched the given name.
func (r *Realm) Schema(name string) (*Schema, bool) {
	for _, s := range r.Schemas {
		if s.Name == name {
			return s, true
		}
	}
	return nil, false
}

// Object returns the first object that matched the given predicate.
func (r *Realm) Object(f func(Object) bool) (Object, bool) {
	for _, o := range r.Objects {
		if f(o) {
			return o, true
		}
	}
	return nil, false
}

// Pos of the schema, if exists.
func (s *Schema) Pos() (*Pos, bool) {
	for _, a := range s.Attrs {
		if p, ok := a.(*Pos); ok {
			return p, true
		}
	}
	return nil, false
}

// Table returns the first table that matched the given name.
func (s *Schema) Table(name string) (*Table, bool) {
	for _, t := range s.Tables {
		if t.Name == name {
			return t, true
		}
	}
	return nil, false
}

// View returns the first view that matched the given name.
func (s *Schema) View(name string) (*View, bool) {
	for _, v := range s.Views {
		if v.Name == name && !v.Materialized() {
			return v, true
		}
	}
	return nil, false
}

// Materialized returns the first materialized view that matched the given name.
func (s *Schema) Materialized(name string) (*View, bool) {
	for _, v := range s.Views {
		if v.Name == name && v.Materialized() {
			return v, true
		}
	}
	return nil, false
}

// Func returns the first function that matched the given name.
func (s *Schema) Func(name string) (*Func, bool) {
	for _, f := range s.Funcs {
		if f.Name == name {
			return f, true
		}
	}
	return nil, false
}

// Proc returns the first procedure that matched the given name.
func (s *Schema) Proc(name string) (*Proc, bool) {
	for _, p := range s.Procs {
		if p.Name == name {
			return p, true
		}
	}
	return nil, false
}

// Object returns the first object that matched the given predicate.
func (s *Schema) Object(f func(Object) bool) (Object, bool) {
	for _, o := range s.Objects {
		if f(o) {
			return o, true
		}
	}
	return nil, false
}

// Column returns the first column that matched the given name.
func (t *Table) Column(name string) (*Column, bool) {
	for _, c := range t.Columns {
		if c.Name == name {
			return c, true
		}
	}
	return nil, false
}

// Pos of the table, if exists.
func (t *Table) Pos() (*Pos, bool) {
	for _, a := range t.Attrs {
		if p, ok := a.(*Pos); ok {
			return p, true
		}
	}
	return nil, false
}

// Index returns the first index that matched the given name.
func (t *Table) Index(name string) (*Index, bool) {
	for _, i := range t.Indexes {
		if i.Name == name {
			return i, true
		}
	}
	return nil, false
}

// ForeignKey returns the first foreign-key that matched the given symbol (constraint name).
func (t *Table) ForeignKey(symbol string) (*ForeignKey, bool) {
	for _, f := range t.ForeignKeys {
		if f.Symbol == symbol {
			return f, true
		}
	}
	return nil, false
}

// Trigger returns the first trigger that matches the given name.
func (t *Table) Trigger(name string) (*Trigger, bool) {
	for _, r := range t.Triggers {
		if r.Name == name {
			return r, true
		}
	}
	return nil, false
}

// Checks of the table.
func (t *Table) Checks() (ck []*Check) {
	for _, a := range t.Attrs {
		if c, ok := a.(*Check); ok {
			ck = append(ck, c)
		}
	}
	return ck
}

// Pos of the view, if exists.
func (v *View) Pos() (*Pos, bool) {
	for _, a := range v.Attrs {
		if p, ok := a.(*Pos); ok {
			return p, true
		}
	}
	return nil, false
}

// Materialized reports if the view is materialized.
func (v *View) Materialized() bool {
	for _, a := range v.Attrs {
		if _, ok := a.(*Materialized); ok {
			return true
		}
	}
	return false
}

// SetMaterialized reports if the view is materialized.
func (v *View) SetMaterialized(b bool) *View {
	if b {
		ReplaceOrAppend(&v.Attrs, &Materialized{})
	} else {
		v.Attrs = RemoveAttr[*Materialized](v.Attrs)
	}
	return v
}

// Column returns the first column that matched the given name.
func (v *View) Column(name string) (*Column, bool) {
	for _, c := range v.Columns {
		if c.Name == name {
			return c, true
		}
	}
	return nil, false
}

// Index returns the first index that matched the given name.
func (v *View) Index(name string) (*Index, bool) {
	for _, i := range v.Indexes {
		if i.Name == name {
			return i, true
		}
	}
	return nil, false
}

// Trigger returns the first trigger that matches the given name.
func (v *View) Trigger(name string) (*Trigger, bool) {
	for _, r := range v.Triggers {
		if r.Name == name {
			return r, true
		}
	}
	return nil, false
}

// AsTable returns a table that represents the view.
func (v *View) AsTable() *Table {
	return NewTable(v.Name).
		SetSchema(v.Schema).
		AddColumns(v.Columns...)
}

// Pos of the column, if exists.
func (c *Column) Pos() (*Pos, bool) {
	for _, a := range c.Attrs {
		if p, ok := a.(*Pos); ok {
			return p, true
		}
	}
	return nil, false
}

// Pos of the index, if exists.
func (i *Index) Pos() (*Pos, bool) {
	for _, a := range i.Attrs {
		if p, ok := a.(*Pos); ok {
			return p, true
		}
	}
	return nil, false
}

// Pos of the index part, if exists.
func (p *IndexPart) Pos() (*Pos, bool) {
	for _, a := range p.Attrs {
		if p, ok := a.(*Pos); ok {
			return p, true
		}
	}
	return nil, false
}

// Pos of the check, if exists.
func (c *Check) Pos() (*Pos, bool) {
	for _, a := range c.Attrs {
		if p, ok := a.(*Pos); ok {
			return p, true
		}
	}
	return nil, false
}

// Pos of the foreign-key, if exists.
func (f *ForeignKey) Pos() (*Pos, bool) {
	for _, a := range f.Attrs {
		if p, ok := a.(*Pos); ok {
			return p, true
		}
	}
	return nil, false
}

// Column returns the first column that matches the given name.
func (f *ForeignKey) Column(name string) (*Column, bool) {
	for _, c := range f.Columns {
		if c.Name == name {
			return c, true
		}
	}
	return nil, false
}

// RefColumn returns the first referenced column that matches the given name.
func (f *ForeignKey) RefColumn(name string) (*Column, bool) {
	for _, c := range f.RefColumns {
		if c.Name == name {
			return c, true
		}
	}
	return nil, false
}

// ReferenceOption for constraint actions.
type ReferenceOption string

// Reference options (actions) specified by ON UPDATE and ON DELETE
// subclauses of the FOREIGN KEY clause.
const (
	NoAction   ReferenceOption = "NO ACTION"
	Restrict   ReferenceOption = "RESTRICT"
	Cascade    ReferenceOption = "CASCADE"
	SetNull    ReferenceOption = "SET NULL"
	SetDefault ReferenceOption = "SET DEFAULT"
)

type (
	// A Type represents a database type. The types below implements this
	// interface and can be used for describing schemas.
	//
	// The Type interface can also be implemented outside this package as follows:
	//
	//	type SpatialType struct {
	//		schema.Type
	//		T string
	//	}
	//
	//	var t schema.Type = &SpatialType{T: "point"}
	//
	Type interface {
		typ()
	}

	// EnumType represents an enum type.
	EnumType struct {
		T      string   // Optional type.
		Values []string // Enum values.
		Schema *Schema  // Optional schema.
	}

	// BinaryType represents a type that stores a binary data.
	BinaryType struct {
		T    string
		Size *int
	}

	// StringType represents a string type.
	StringType struct {
		T     string
		Size  int
		Attrs []Attr
	}

	// BoolType represents a boolean type.
	BoolType struct {
		T string
	}

	// IntegerType represents an int type.
	IntegerType struct {
		T        string
		Unsigned bool
		Attrs    []Attr
	}

	// DecimalType represents a fixed-point type that stores exact numeric values.
	DecimalType struct {
		T         string
		Precision int
		Scale     int
		Unsigned  bool
	}

	// FloatType represents a floating-point type that stores approximate numeric values.
	FloatType struct {
		T         string
		Unsigned  bool
		Precision int
	}

	// TimeType represents a date/time type.
	TimeType struct {
		T         string
		Precision *int
		Scale     *int
		Attrs     []Attr
	}

	// JSONType represents a JSON type.
	JSONType struct {
		T string
	}

	// SpatialType represents a spatial/geometric type.
	SpatialType struct {
		T string
	}

	// A UUIDType defines a UUID type.
	UUIDType struct {
		T string
	}

	// UnsupportedType represents a type that is not supported by the drivers.
	UnsupportedType struct {
		T string
	}

	// TypeParser is an interface that is required be implemented by
	// different drivers for parsing column types from their database
	// forms to the schema representation.
	TypeParser interface {
		// ParseType converts the raw database type to its schema.Type representation.
		ParseType(string) (Type, error)
	}

	// TypeFormatter is an interface that is required to be implemented by
	// different drivers to format column types into their corresponding
	// database forms.
	TypeFormatter interface {
		// FormatType converts a schema type to its column form in the database.
		FormatType(Type) (string, error)
	}

	// TypeParseFormatter that groups the TypeParser and TypeFormatter interfaces.
	TypeParseFormatter interface {
		TypeParser
		TypeFormatter
	}
)

type (
	// Expr defines an SQL expression in schema DDL.
	//
	// The Expr interface can also be implemented outside this package as follows:
	//
	// 	type NamedDefault struct {
	// 		schema.Expr
	// 		Name string
	// 	}
	// 	// Underlying returns the underlying expression.
	// 	func (e *NamedDefault) Underlying() schema.Expr { return e.Expr }
	//
	//  var e schema.Expr = &NamedDefault{Expr: &schema.Literal{V: "bar"}, Name: "foo"}
	Expr interface {
		expr()
	}

	// Literal represents a basic literal expression like 1, or '1'.
	// String literals are usually quoted with single or double quotes.
	Literal struct {
		V string
	}

	// RawExpr represents a raw expression like "uuid()" or "current_timestamp()".
	// Unlike literals, raw expression are usually inlined as is on migration.
	RawExpr struct {
		X string
	}
)

type (
	// Attr represents the interface that all attributes implement.
	Attr interface {
		attr()
	}

	// Comment describes a schema element comment.
	Comment struct {
		Text string
	}

	// Charset describes a column or a table character-set setting.
	Charset struct {
		V string
	}

	// Collation describes a column or a table collation setting.
	Collation struct {
		V string
	}

	// Check describes a CHECK constraint.
	Check struct {
		Name  string // Optional constraint name.
		Expr  string // Actual CHECK.
		Attrs []Attr // Additional attributes (e.g. ENFORCED).
	}

	// GeneratedExpr describes the expression used for generating
	// the value of a generated/virtual column.
	GeneratedExpr struct {
		Expr string
		Type string // Optional type. e.g. STORED or VIRTUAL.
	}

	// ViewCheckOption describes the standard 'WITH CHECK OPTION clause' of a view.
	ViewCheckOption struct {
		V string // LOCAL, CASCADED, NONE, or driver specific.
	}

	// Materialized is a schema attribute that attached to views to indicates
	// they are MATERIALIZED VIEWs.
	Materialized struct {
		Attr
	}

	// Pos is an attribute that holds the position of a schema element.
	Pos struct {
		// Filename is the name (or full path) of the file which loaded the schema element.
		Filename string

		// Start and End represent the bounds of this range.
		Start, End struct {
			Line, Column, Byte int // hcl.Pos fields.
		}
	}
)

// A list of known view check options.
const (
	ViewCheckOptionNone     = "NONE"
	ViewCheckOptionLocal    = "LOCAL"
	ViewCheckOptionCascaded = "CASCADED"
)

// objects.
func (*Table) obj()    {}
func (*View) obj()     {}
func (*Func) obj()     {}
func (*Proc) obj()     {}
func (*Trigger) obj()  {}
func (*EnumType) obj() {}

// constraints are objects.
func (*Index) obj()        {}
func (*Check) obj()        {}
func (*ForeignKey) obj()   {}
func (*NamedDefault) obj() {}

// expressions.
func (*Literal) expr() {}
func (*RawExpr) expr() {}

// types.
func (*BoolType) typ()        {}
func (*EnumType) typ()        {}
func (*TimeType) typ()        {}
func (*JSONType) typ()        {}
func (*FloatType) typ()       {}
func (*StringType) typ()      {}
func (*BinaryType) typ()      {}
func (*SpatialType) typ()     {}
func (*UUIDType) typ()        {}
func (*IntegerType) typ()     {}
func (*DecimalType) typ()     {}
func (*UnsupportedType) typ() {}

// attributes.
func (*Pos) attr()             {}
func (*Check) attr()           {}
func (*Comment) attr()         {}
func (*Charset) attr()         {}
func (*Collation) attr()       {}
func (*GeneratedExpr) attr()   {}
func (*ViewCheckOption) attr() {}

// SpecType returns the type of the spec.
func (e *EnumType) SpecType() string { return "enum" }

// SpecName returns the name of the spec.
func (e *EnumType) SpecName() string { return e.T }

// Underlying returns underlying the expression.
func (n *NamedDefault) Underlying() Expr {
	return n.Expr
}

// UnderlyingExpr returns the underlying expression of x.
func UnderlyingExpr(x Expr) Expr {
	if w, ok := x.(interface{ Underlying() Expr }); ok {
		return UnderlyingExpr(w.Underlying())
	}
	return x
}

// UnderlyingType returns the underlying type of t.
func UnderlyingType(t Type) Type {
	if w, ok := t.(interface{ Underlying() Type }); ok {
		return UnderlyingType(w.Underlying())
	}
	return t
}

// IsType return true if somewhere in the type-chain of t1 is the same as t2.
func IsType(t1, t2 Type) bool {
	if t1 == nil || t2 == nil {
		return t1 == t2
	}
	return sameType(t1, t2, reflect.TypeOf(t2).Comparable())
}

func sameType(t1, t2 Type, targetComparable bool) bool {
	for {
		if targetComparable && t1 == t2 {
			return true
		}
		// Check if t1 implements the Is method.
		// Then call it to check if it is the same as t2.
		// This is useful for comparing types that are
		// not directly the same pointer.
		if x, ok := t1.(interface{ Is(Type) bool }); ok && x.Is(t2) {
			return true
		}
		// Check if t1 has an underlying type.
		// Then use it to compare with t2.
		if x, ok := t1.(interface{ Underlying() Type }); ok {
			if t1 = x.Underlying(); t1 != nil {
				continue
			}
		}
		return false
	}
}
