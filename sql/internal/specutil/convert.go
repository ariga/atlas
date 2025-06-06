// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package specutil

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"

	"github.com/zclconf/go-cty/cty"
)

// List of convert function types.
type (
	ConvertTableFunc       func(*sqlspec.Table, *schema.Schema) (*schema.Table, error)
	ConvertTableColumnFunc func(*sqlspec.Column, *schema.Table) (*schema.Column, error)
	ConvertViewFunc        func(*sqlspec.View, *schema.Schema) (*schema.View, error)
	ConvertViewColumnFunc  func(*sqlspec.Column, *schema.View) (*schema.Column, error)
	ConvertTypeFunc        func(*sqlspec.Column) (schema.Type, error)
	ConvertPrimaryKeyFunc  func(*sqlspec.PrimaryKey, *schema.Table) (*schema.Index, error)
	ConvertIndexFunc       func(*sqlspec.Index, *schema.Table) (*schema.Index, error)
	ConvertViewIndexFunc   func(*sqlspec.Index, *schema.View) (*schema.Index, error)
	ConvertCheckFunc       func(*sqlspec.Check) (*schema.Check, error)
	ConvertFuncFunc        func(*sqlspec.Func, *schema.Schema) (*schema.Func, error)
	ConvertProcFunc        func(*sqlspec.Func, *schema.Schema) (*schema.Proc, error)
	ColumnTypeSpecFunc     func(schema.Type) (*sqlspec.Column, error)
	TableSpecFunc          func(*schema.Table) (*sqlspec.Table, error)
	TableColumnSpecFunc    func(*schema.Column, *schema.Table) (*sqlspec.Column, error)
	ViewSpecFunc           func(*schema.View) (*sqlspec.View, error)
	ViewColumnSpecFunc     func(*schema.Column, *schema.View) (*sqlspec.Column, error)
	PrimaryKeySpecFunc     func(*schema.Index) (*sqlspec.PrimaryKey, error)
	IndexSpecFunc          func(*schema.Index) (*sqlspec.Index, error)
	ForeignKeySpecFunc     func(*schema.ForeignKey) (*sqlspec.ForeignKey, error)
	CheckSpecFunc          func(*schema.Check) *sqlspec.Check
)

type (
	// ScanDoc represents a scanned HCL document.
	ScanDoc struct {
		Schemas      []*sqlspec.Schema
		Tables       []*sqlspec.Table
		Views        []*sqlspec.View
		Materialized []*sqlspec.View
		Funcs        []*sqlspec.Func
		Procs        []*sqlspec.Func
		Triggers     []*sqlspec.Trigger
	}

	// ScanFuncs represents a set of scan functions
	// used to convert the HCL document to the Realm.
	ScanFuncs struct {
		Table ConvertTableFunc
		View  ConvertViewFunc
		Func  ConvertFuncFunc
		Proc  ConvertProcFunc
		// Triggers add themselves to the relevant tables/views.
		Triggers func(*schema.Realm, []*sqlspec.Trigger) error
		// Objects add themselves to the realm.
		Objects func(*schema.Realm) error
		// Optional function to extend the foreign keys.
		ForeignKey func(*sqlspec.ForeignKey, *schema.ForeignKey) error
	}

	// SchemaFuncs represents a set of spec functions
	// used to convert the Schema object to an HCL document.
	SchemaFuncs struct {
		Table TableSpecFunc
		View  ViewSpecFunc
		Func  func(*schema.Func) (*sqlspec.Func, error)
		Proc  func(*schema.Proc) (*sqlspec.Func, error)
	}
	// RefNamer is an interface for objects that can
	// return their reference.
	RefNamer interface {
		// Ref returns the reference to the object.
		Ref() *schemahcl.Ref
	}
	// SpecTypeNamer is an interface for objects that can
	// return their spec type and name.
	SpecTypeNamer interface {
		SpecTyper
		// SpecName returns the spec name of the object.
		SpecName() string
	}
	// SpecTyper wraps the SpecType method. It allows objects
	// to describe what their spec type is.
	SpecTyper interface {
		// SpecType returns the spec type of the object.
		SpecType() string
	}
)

const (
	typeView         = "view"
	typeTable        = "table"
	typeColumn       = "column"
	typeIndex        = "index"
	typeSchema       = "schema"
	typeMaterialized = "materialized"
	typeFunction     = "function"
	typeProcedure    = "procedure"
	typeTrigger      = "trigger"
)

// typeName returns the type name of the given object.
func typeName(o schema.Object) string {
	attrOr := func(n string, attrs []schema.Attr) string {
		for _, a := range attrs {
			if t, ok := a.(SpecTyper); ok && t.SpecType() != "" {
				n = t.SpecType()
			}
		}
		return n
	}
	switch o := o.(type) {
	case nil:
	case *schema.Table:
		if o == nil {
			return typeTable
		}
		return attrOr(typeTable, o.Attrs)
	case *schema.View:
		switch {
		case o == nil:
			return typeView
		case o.Materialized():
			return attrOr(typeMaterialized, o.Attrs)
		default:
			return attrOr(typeView, o.Attrs)
		}
	case *schema.Func:
		if o == nil {
			return typeFunction
		}
		return attrOr(typeFunction, o.Attrs)
	case *schema.Proc:
		if o == nil {
			return typeProcedure
		}
		return attrOr(typeProcedure, o.Attrs)
	default:
		if ts, ok := o.(SpecTyper); ok && ts != nil && ts.SpecType() != "" {
			return ts.SpecType()
		}
	}
	return "object"
}

// Scan populates the Realm from the schemas and table specs.
func Scan(r *schema.Realm, doc *ScanDoc, funcs *ScanFuncs) error {
	byName := make(map[string]*schema.Schema)
	for _, s := range doc.Schemas {
		s1 := schema.New(s.Name)
		if err := convertCommentFromSpec(s, &s1.Attrs); err != nil {
			return err
		}
		schemahcl.AppendPos(&s1.Attrs, s.Range)
		r.AddSchemas(s1)
		byName[s.Name] = s1
	}
	var (
		fks     = make(map[*schema.Table][]*sqlspec.ForeignKey)
		deps    = make(map[schema.Object][]*schemahcl.Ref, len(doc.Views))
		aliases = make(map[string]string)
	)
	for _, st := range doc.Tables {
		name, err := SchemaName(st.Schema)
		if err != nil {
			return fmt.Errorf("cannot extract schema name for table %q: %w", st.Name, err)
		}
		s, ok := byName[name]
		if !ok {
			return fmt.Errorf("schema %q not found for table %q", name, st.Name)
		}
		t, err := funcs.Table(st, s)
		if err != nil {
			return fmt.Errorf("cannot convert table %q: %w", st.Name, err)
		}
		if tn := typeName(t); tn != typeTable {
			aliases[tn] = typeTable
		}
		fks[t] = st.ForeignKeys
		s.AddTables(t)
		if d, ok := st.Attr("depends_on"); ok {
			refs, err := d.Refs()
			if err != nil {
				return fmt.Errorf("expect list of references for attribute table.%s.depends_on: %w", st.Name, err)
			}
			deps[t] = refs
		}
	}
	// Link the foreign keys.
	for t, fks := range fks {
		if err := linkForeignKeys(funcs, t, fks); err != nil {
			return err
		}
	}
	for _, sv := range doc.Views {
		name, err := SchemaName(sv.Schema)
		if err != nil {
			return fmt.Errorf("cannot extract schema name for view %q: %w", sv.Name, err)
		}
		s, ok := byName[name]
		if !ok {
			return fmt.Errorf("schema %q not found for view %q", name, sv.Name)
		}
		v, err := funcs.View(sv, s)
		if err != nil {
			return fmt.Errorf("cannot convert view %q: %w", sv.Name, err)
		}
		if tn := typeName(v); tn != typeView {
			aliases[tn] = typeView
		}
		s.AddViews(v)
		if d, ok := sv.Attr("depends_on"); ok {
			refs, err := d.Refs()
			if err != nil {
				return fmt.Errorf("expect list of references for attribute view.%s.depends_on: %w", sv.Name, err)
			}
			deps[v] = refs
		}
	}
	for _, m := range doc.Materialized {
		name, err := SchemaName(m.Schema)
		if err != nil {
			return fmt.Errorf("cannot extract schema name for materialized %q: %w", m.Name, err)
		}
		s, ok := byName[name]
		if !ok {
			return fmt.Errorf("schema %q not found for materialized %q", name, m.Name)
		}
		v, err := funcs.View(m, s)
		if err != nil {
			return fmt.Errorf("cannot convert materialized %q: %w", m.Name, err)
		}
		if tn := typeName(v); tn != typeMaterialized {
			aliases[tn] = typeMaterialized
		}
		s.AddViews(v.SetMaterialized(true))
		if d, ok := m.Attr("depends_on"); ok {
			refs, err := d.Refs()
			if err != nil {
				return fmt.Errorf("expect list of references for attribute materialized.%s.depends_on: %w", m.Name, err)
			}
			deps[v] = refs
		}
	}
	if funcs.Func != nil {
		for _, sf := range doc.Funcs {
			name, err := SchemaName(sf.Schema)
			if err != nil {
				return fmt.Errorf("cannot extract schema name for function %q: %w", sf.Name, err)
			}
			s, ok := byName[name]
			if !ok {
				return fmt.Errorf("schema %q not found for function %q", name, sf.Name)
			}
			f, err := funcs.Func(sf, s)
			if err != nil {
				return fmt.Errorf("cannot convert function %q: %w", sf.Name, err)
			}
			if tn := typeName(f); tn != typeFunction {
				aliases[tn] = typeFunction
			}
			s.AddFuncs(f)
			if d, ok := sf.Attr("depends_on"); ok {
				refs, err := d.Refs()
				if err != nil {
					return fmt.Errorf("expect list of references for attribute function.%s.depends_on: %w", f.Name, err)
				}
				deps[f] = refs
			}
		}
	}
	if funcs.Proc != nil {
		for _, sf := range doc.Procs {
			name, err := SchemaName(sf.Schema)
			if err != nil {
				return fmt.Errorf("cannot extract schema name for procedure %q: %w", sf.Name, err)
			}
			s, ok := byName[name]
			if !ok {
				return fmt.Errorf("schema %q not found for procedure %q", name, sf.Name)
			}
			f, err := funcs.Proc(sf, s)
			if err != nil {
				return fmt.Errorf("cannot convert procedure %q: %w", sf.Name, err)
			}
			if tn := typeName(f); tn != typeProcedure {
				aliases[tn] = typeProcedure
			}
			s.AddProcs(f)
			if d, ok := sf.Attr("depends_on"); ok {
				refs, err := d.Refs()
				if err != nil {
					return fmt.Errorf("expect list of references for attribute procedure.%s.depends_on: %w", f.Name, err)
				}
				deps[f] = refs
			}
		}
	}
	if funcs.Triggers != nil {
		if err := funcs.Triggers(r, doc.Triggers); err != nil {
			return err
		}
	}
	if funcs.Objects != nil {
		if err := funcs.Objects(r); err != nil {
			return err
		}
	}
	for o, refs := range deps {
		var err error
		switch o := o.(type) {
		case *schema.Table:
			err = fromDependsOn(fmt.Sprintf("%s.%s", typeName(o), o.Name), o, o.Schema, refs, aliases)
		case *schema.View:
			err = fromDependsOn(fmt.Sprintf("%s.%s", typeName(o), o.Name), o, o.Schema, refs, aliases)
		case *schema.Func:
			err = fromDependsOn(fmt.Sprintf("%s.%s", typeName(o), o.Name), o, o.Schema, refs, aliases)
		case *schema.Proc:
			err = fromDependsOn(fmt.Sprintf("%s.%s", typeName(o), o.Name), o, o.Schema, refs, aliases)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Table converts a sqlspec.Table to a schema.Table. Table conversion is done without converting
// ForeignKeySpecs into ForeignKeys, as the target tables do not necessarily exist in the schema
// at this point. Instead, the linking is done by the Schema function.
func Table(spec *sqlspec.Table, parent *schema.Schema, convertColumn ConvertTableColumnFunc,
	convertPK ConvertPrimaryKeyFunc, convertIndex ConvertIndexFunc, convertCheck ConvertCheckFunc) (*schema.Table, error) {
	t := &schema.Table{
		Name:   spec.Name,
		Schema: parent,
	}
	schemahcl.AppendPos(&t.Attrs, spec.Range)
	for _, cs := range spec.Columns {
		c, err := convertColumn(cs, t)
		if err != nil {
			return nil, err
		}
		schemahcl.AppendPos(&c.Attrs, cs.Range)
		t.AddColumns(c)
	}
	if spec.PrimaryKey != nil {
		pk, err := convertPK(spec.PrimaryKey, t)
		if err != nil {
			return nil, err
		}
		schemahcl.AppendPos(&pk.Attrs, spec.PrimaryKey.Range)
		t.SetPrimaryKey(pk)
	}
	for _, idx := range spec.Indexes {
		i, err := convertIndex(idx, t)
		if err != nil {
			return nil, err
		}
		schemahcl.AppendPos(&i.Attrs, idx.Range)
		t.AddIndexes(i)
	}
	for _, c := range spec.Checks {
		ck, err := convertCheck(c)
		if err != nil {
			return nil, err
		}
		schemahcl.AppendPos(&ck.Attrs, c.Range)
		t.AddChecks(ck)
	}
	if err := convertCommentFromSpec(spec, &t.Attrs); err != nil {
		return nil, err
	}
	return t, nil
}

// View converts a sqlspec.View to a schema.View.
func View(spec *sqlspec.View, parent *schema.Schema, convertC ConvertViewColumnFunc, convertI ConvertViewIndexFunc) (*schema.View, error) {
	as, ok := spec.Extra.Attr("as")
	if !ok {
		return nil, fmt.Errorf("missing 'as' definition for view %q", spec.Name)
	}
	def, err := as.String()
	if err != nil {
		return nil, fmt.Errorf("expect string definition for attribute view.%s.as: %w", spec.Name, err)
	}
	v := schema.NewView(spec.Name, def).SetSchema(parent)
	schemahcl.AppendPos(&v.Attrs, spec.Range)
	for _, cs := range spec.Columns {
		c, err := convertC(cs, v)
		if err != nil {
			return nil, err
		}
		schemahcl.AppendPos(&c.Attrs, cs.Range)
		v.AddColumns(c)
	}
	for _, idx := range spec.Indexes {
		i, err := convertI(idx, v)
		if err != nil {
			return nil, err
		}
		schemahcl.AppendPos(&i.Attrs, idx.Range)
		v.AddIndexes(i)
	}
	if err := convertCommentFromSpec(spec, &v.Attrs); err != nil {
		return nil, err
	}
	if c, ok := spec.Extra.Attr("check_option"); ok {
		o, err := c.String()
		if err != nil {
			return nil, fmt.Errorf("expect string definition for attribute view.%s.check_option: %w", spec.Name, err)
		}
		v.SetCheckOption(o)
	}
	return v, nil
}

// Column converts a sqlspec.Column into a schema.Column.
func Column(spec *sqlspec.Column, conv ConvertTypeFunc) (*schema.Column, error) {
	out := &schema.Column{
		Name: spec.Name,
		Type: &schema.ColumnType{
			Null: spec.Null,
		},
	}
	d, err := columnDefault(spec.Remain())
	if err != nil {
		return nil, err
	}
	out.Default = d
	ct, err := conv(spec)
	if err != nil {
		return nil, err
	}
	out.Type.Type = ct
	if err := convertCommentFromSpec(spec, &out.Attrs); err != nil {
		return nil, err
	}
	return out, err
}

func columnDefault(r *schemahcl.Resource) (schema.Expr, error) {
	defaultA, okA := r.Attr("default")
	defaultR, okR := r.Resource("default")
	switch {
	case okA && okR:
		return nil, errors.New("both default and default resource are set")
	case okA:
		v, err := Default(defaultA.V)
		if err != nil {
			return nil, err
		}
		return v, nil
	case okR:
		var spec struct {
			Name string    `spec:",name"`
			As   cty.Value `spec:"as"`
		}
		if err := defaultR.As(&spec); err != nil {
			return nil, err
		}
		v, err := Default(spec.As)
		if err != nil {
			return nil, err
		}
		return &schema.NamedDefault{Name: spec.Name, Expr: v}, nil
	default:
		return nil, nil
	}
}

// Default converts a cty.Value (as defined in the spec) into a schema.Expr.
func Default(d cty.Value) (schema.Expr, error) {
	if d.IsNull() {
		return nil, nil // no default.
	}
	var x schema.Expr
	switch {
	case d.Type() == cty.String:
		x = &schema.Literal{V: d.AsString()}
	case d.Type() == cty.Number:
		f := d.AsBigFloat()
		// If the number is an integer, convert it to an integer.
		if f.IsInt() {
			x = &schema.Literal{V: f.Text('f', -1)}
		} else {
			x = &schema.Literal{V: f.String()}
		}
	case d.Type() == cty.Bool:
		x = &schema.Literal{V: strconv.FormatBool(d.True())}
	case d.Type().IsCapsuleType():
		raw, ok := d.EncapsulatedValue().(*schemahcl.RawExpr)
		if !ok {
			return nil, fmt.Errorf("invalid default value %q", d.Type().FriendlyName())
		}
		x = &schema.RawExpr{X: raw.X}
	default:
		return nil, fmt.Errorf("unsupported type for default value: %T", d)
	}
	return x, nil
}

// Index converts a sqlspec.Index to a schema.Index. The optional arguments allow
// passing functions for mutating the created index-part (e.g. add attributes).
func Index(spec *sqlspec.Index, parent *schema.Table, partFns ...func(*sqlspec.IndexPart, *schema.IndexPart) error) (*schema.Index, error) {
	parts := make([]*schema.IndexPart, 0, len(spec.Columns)+len(spec.Parts))
	switch n, m := len(spec.Columns), len(spec.Parts); {
	case n == 0 && m == 0:
		return nil, fmt.Errorf("missing definition for index %q", spec.Name)
	case n > 0 && m > 0:
		return nil, fmt.Errorf(`multiple definitions for index %q, use "columns" or "on"`, spec.Name)
	case n > 0:
		for i, c := range spec.Columns {
			c, err := ColumnByRef(parent, c)
			if err != nil {
				return nil, err
			}
			parts = append(parts, &schema.IndexPart{
				SeqNo: i,
				C:     c,
			})
		}
	case m > 0:
		for i, p := range spec.Parts {
			part := &schema.IndexPart{SeqNo: i, Desc: p.Desc}
			switch {
			case p.Column == nil && p.Expr == "":
				return nil, fmt.Errorf(`"column" or "expr" are required for index %q at position %d`, spec.Name, i)
			case p.Column != nil && p.Expr != "":
				return nil, fmt.Errorf(`cannot use both "column" and "expr" in index %q at position %d`, spec.Name, i)
			case p.Expr != "":
				part.X = &schema.RawExpr{X: p.Expr}
			case p.Column != nil:
				c, err := ColumnByRef(parent, p.Column)
				if err != nil {
					return nil, err
				}
				part.C = c
			}
			for _, f := range partFns {
				if err := f(p, part); err != nil {
					return nil, err
				}
			}
			parts = append(parts, part)
		}
	}
	idx := &schema.Index{
		Name:   spec.Name,
		Unique: spec.Unique,
		Table:  parent,
		Parts:  parts,
	}
	if err := convertCommentFromSpec(spec, &idx.Attrs); err != nil {
		return nil, err
	}
	for _, p := range idx.Parts {
		if p.C != nil {
			p.C.AddIndexes(idx)
		}
	}
	return idx, nil
}

// Check converts a sqlspec.Check to a schema.Check.
func Check(spec *sqlspec.Check) (*schema.Check, error) {
	return &schema.Check{
		Name: spec.Name,
		Expr: spec.Expr,
	}, nil
}

// PrimaryKey converts a sqlspec.PrimaryKey to a schema.Index.
func PrimaryKey(spec *sqlspec.PrimaryKey, parent *schema.Table) (*schema.Index, error) {
	parts := make([]*schema.IndexPart, 0, len(spec.Columns))
	for seqno, c := range spec.Columns {
		c, err := ColumnByRef(parent, c)
		if err != nil {
			return nil, nil
		}
		parts = append(parts, &schema.IndexPart{
			SeqNo: seqno,
			C:     c,
		})
	}
	pk := &schema.Index{
		Name:  spec.Name,
		Table: parent,
		Parts: parts,
	}
	if err := convertCommentFromSpec(spec, &pk.Attrs); err != nil {
		return nil, err
	}
	for _, p := range pk.Parts {
		if p.C != nil {
			p.C.AddIndexes(pk)
		}
	}
	return pk, nil
}

// linkForeignKeys creates the foreign keys defined in the Table's spec by creating references
// to column in the provided Schema. It is assumed that all tables referenced FK definitions in the spec
// are reachable from the provided schema or its connected realm.
func linkForeignKeys(funcs *ScanFuncs, tbl *schema.Table, fks []*sqlspec.ForeignKey) error {
	for _, spec := range fks {
		fk := &schema.ForeignKey{Symbol: spec.Symbol, Table: tbl}
		schemahcl.AppendPos(&fk.Attrs, spec.Range)
		if spec.OnUpdate != nil {
			fk.OnUpdate = schema.ReferenceOption(FromVar(spec.OnUpdate.V))
		}
		if spec.OnDelete != nil {
			fk.OnDelete = schema.ReferenceOption(FromVar(spec.OnDelete.V))
		}
		if n, m := len(spec.Columns), len(spec.RefColumns); n != m {
			return fmt.Errorf("sqlspec: number of referencing and referenced columns do not match for foreign-key %q", fk.Symbol)
		}
		for _, ref := range spec.Columns {
			c, err := ColumnByRef(tbl, ref)
			if err != nil {
				return err
			}
			fk.Columns = append(fk.Columns, c)
		}
		for i, ref := range spec.RefColumns {
			t, c, err := externalRef(ref, tbl.Schema)
			if isLocalRef(ref) {
				t = fk.Table
				c, err = ColumnByRef(fk.Table, ref)
			}
			if err != nil {
				return err
			}
			if i > 0 && fk.RefTable != t {
				return fmt.Errorf("sqlspec: more than 1 table was referenced for foreign-key %q", fk.Symbol)
			}
			fk.RefTable = t
			fk.RefColumns = append(fk.RefColumns, c)
		}
		tbl.ForeignKeys = append(tbl.ForeignKeys, fk)
		if funcs.ForeignKey != nil {
			if err := funcs.ForeignKey(spec, fk); err != nil {
				return err
			}
		}
	}
	return nil
}

// FromSchema converts a schema.Schema into sqlspec.Schema and []sqlspec.Table.
func FromSchema(s *schema.Schema, funcs *SchemaFuncs) (*SchemaSpec, error) {
	spec := &SchemaSpec{
		Schema: &sqlspec.Schema{
			Name: s.Name,
		},
		Tables:       make([]*sqlspec.Table, 0, len(s.Tables)),
		Views:        make([]*sqlspec.View, 0, len(s.Views)),
		Materialized: make([]*sqlspec.View, 0, len(s.Views)),
	}
	for _, t := range s.Tables {
		table, err := funcs.Table(t)
		if err != nil {
			return nil, err
		}
		if s.Name != "" {
			table.Schema = SchemaRef(s.Name)
		}
		spec.Tables = append(spec.Tables, table)
		spec.Triggers = append(spec.Triggers, t.Triggers...)
	}
	for _, v := range s.Views {
		view, err := funcs.View(v)
		if err != nil {
			return nil, err
		}
		if s.Name != "" {
			view.Schema = SchemaRef(s.Name)
		}
		if v.Materialized() {
			spec.Materialized = append(spec.Materialized, view)
		} else {
			spec.Views = append(spec.Views, view)
		}
		spec.Triggers = append(spec.Triggers, v.Triggers...)
	}
	if funcs.Func != nil {
		for _, f := range s.Funcs {
			fn, err := funcs.Func(f)
			if err != nil {
				return nil, err
			}
			if s.Name != "" {
				fn.Schema = SchemaRef(s.Name)
			}
			spec.Funcs = append(spec.Funcs, fn)
		}
	}
	if funcs.Proc != nil {
		for _, p := range s.Procs {
			pr, err := funcs.Proc(p)
			if err != nil {
				return nil, err
			}
			if s.Name != "" {
				pr.Schema = SchemaRef(s.Name)
			}
			spec.Procs = append(spec.Procs, pr)
		}
	}
	convertCommentFromSchema(s.Attrs, &spec.Schema.Extra.Attrs)
	return spec, nil
}

// FromTable converts a schema.Table to a sqlspec.Table.
func FromTable(t *schema.Table, colFn TableColumnSpecFunc, pkFn PrimaryKeySpecFunc, idxFn IndexSpecFunc,
	fkFn ForeignKeySpecFunc, ckFn CheckSpecFunc) (*sqlspec.Table, error) {
	spec := &sqlspec.Table{
		Name: t.Name,
	}
	for _, c := range t.Columns {
		col, err := colFn(c, t)
		if err != nil {
			return nil, err
		}
		spec.Columns = append(spec.Columns, col)
	}
	if t.PrimaryKey != nil {
		pk, err := pkFn(t.PrimaryKey)
		if err != nil {
			return nil, err
		}
		spec.PrimaryKey = pk
	}
	for _, idx := range t.Indexes {
		i, err := idxFn(idx)
		if err != nil {
			return nil, err
		}
		spec.Indexes = append(spec.Indexes, i)
	}
	for _, fk := range t.ForeignKeys {
		f, err := fkFn(fk)
		if err != nil {
			return nil, err
		}
		spec.ForeignKeys = append(spec.ForeignKeys, f)
	}
	for _, attr := range t.Attrs {
		if c, ok := attr.(*schema.Check); ok {
			spec.Checks = append(spec.Checks, ckFn(c))
		}
	}
	if deps, ok := dependsOn(t.Schema.Realm, t.Deps); ok {
		// Embedding a resource push its attributes to the end.
		spec.Extra.Children = append(spec.Extra.Children, &schemahcl.Resource{Attrs: []*schemahcl.Attr{deps}})
	}
	convertCommentFromSchema(t.Attrs, &spec.Extra.Attrs)
	return spec, nil
}

// FromView converts a schema.View to a sqlspec.View.
func FromView(v *schema.View, colFn ViewColumnSpecFunc, idxFn IndexSpecFunc) (*sqlspec.View, error) {
	spec := &sqlspec.View{
		Name: v.Name,
	}
	for _, c := range v.Columns {
		cs, err := colFn(c, v)
		if err != nil {
			return nil, err
		}
		spec.Columns = append(spec.Columns, cs)
	}
	for _, idx := range v.Indexes {
		i, err := idxFn(idx)
		if err != nil {
			return nil, err
		}
		spec.Indexes = append(spec.Indexes, i)
	}
	// In case the view definition is multi-line,
	// format it as indented heredoc with two spaces.
	as := sqlspec.MightHeredoc(v.Def)
	embed := &schemahcl.Resource{
		Attrs: []*schemahcl.Attr{
			schemahcl.StringAttr("as", as),
		},
	}
	if c := (schema.ViewCheckOption{}); sqlx.Has(v.Attrs, &c) {
		switch strings.ToUpper(c.V) {
		case schema.ViewCheckOptionNone, "":
		case schema.ViewCheckOptionLocal, schema.ViewCheckOptionCascaded:
			embed.Attrs = append(embed.Attrs, VarAttr("check_option", c.V))
		default:
			embed.Attrs = append(embed.Attrs, schemahcl.StringAttr("check_option", c.V))
		}
	}
	if deps, ok := dependsOn(v.Schema.Realm, v.Deps); ok {
		embed.Attrs = append(embed.Attrs, deps)
	}
	convertCommentFromSchema(v.Attrs, &embed.Attrs)
	spec.Extra.Children = append(spec.Extra.Children, embed)
	return spec, nil
}

// dependsOn returns the depends_on attribute for the given objects.
func dependsOn(realm *schema.Realm, objects []schema.Object) (*schemahcl.Attr, bool) {
	var (
		n2s  = make(map[string][]*schema.Schema)
		name = func(t schema.Object, n string) string { return typeName(t) + "/" + n }
	)
	// Qualify references if there are objects with the same name.
	if realm != nil {
		for _, s := range realm.Schemas {
			for _, t := range s.Tables {
				n2s[name(t, t.Name)] = append(n2s[name(t, t.Name)], s)
			}
			for _, v := range s.Views {
				n2s[name(v, v.Name)] = append(n2s[name(v, v.Name)], s)
			}
			for _, f := range s.Funcs {
				if n := name(f, f.Name); !slices.Contains(n2s[n], s) {
					n2s[n] = append(n2s[n], s) // Count overload once.
				}
			}
			for _, p := range s.Procs {
				if n := name(p, p.Name); !slices.Contains(n2s[n], s) {
					n2s[n] = append(n2s[n], s) // Count overload once.
				}
			}
		}
	}
	var (
		refs = make(map[string]bool, len(objects))
		deps = make([]*schemahcl.Ref, 0, len(objects))
	)
	for _, o := range objects {
		path := make([]string, 0, 2)
		var n, s string
		switch d := o.(type) {
		case *schema.Table:
			n, s = d.Name, d.Schema.Name
		case *schema.View:
			n, s = d.Name, d.Schema.Name
		case *schema.Func:
			n, s = d.Name, d.Schema.Name
		case *schema.Proc:
			n, s = d.Name, d.Schema.Name
		case RefNamer:
			// If the object is a reference, add it to the depends_on list.
			deps = append(deps, d.Ref())
			continue
		}
		if len(n2s[name(o, n)]) > 1 {
			path = append(path, s)
		}
		if r := schemahcl.BuildRef([]schemahcl.PathIndex{
			{T: typeName(o), V: append(path, n)},
		}); !refs[r.V] {
			refs[r.V] = true
			deps = append(deps, r)
		}
	}
	if len(deps) > 0 {
		slices.SortFunc(deps, func(l, r *schemahcl.Ref) int {
			return strings.Compare(l.V, r.V)
		})
		return schemahcl.RefsAttr("depends_on", deps...), true
	}
	return nil, false
}

func fromDependsOn[T interface{ AddDeps(...schema.Object) T }](loc string, t T, ns *schema.Schema, refs []*schemahcl.Ref, aliases map[string]string) error {
	for i, r := range refs {
		p, err := r.Path()
		if err != nil {
			return fmt.Errorf("extract %s.depends_on references: %w", loc, err)
		}
		if len(p) == 0 {
			return fmt.Errorf("empty reference exists in %s.depends_on[%d]", loc, i)
		}
		q, n, err := RefName(r, p[0].T)
		if err != nil {
			return fmt.Errorf("extract %s name from %s.depends_on[%d]: %w", p[0].T, loc, i, err)
		}
		var o schema.Object
		switch tn := p[0].T; {
		case tn == typeTable, aliases[tn] == typeTable:
			o, err = findT(ns, q, n, func(s *schema.Schema, name string) (*schema.Table, bool) {
				if v, ok := s.Table(name); ok && typeName(v) == tn {
					return v, true
				}
				return nil, false
			})
		case tn == typeView, aliases[tn] == typeView:
			o, err = findT(ns, q, n, func(s *schema.Schema, name string) (*schema.View, bool) {
				if v, ok := s.View(name); ok && typeName(v) == tn {
					return v, true
				}
				return nil, false
			})
		case tn == typeMaterialized, aliases[tn] == typeMaterialized:
			o, err = findT(ns, q, n, func(s *schema.Schema, name string) (*schema.View, bool) {
				if v, ok := s.Materialized(name); ok && typeName(v) == tn {
					return v, true
				}
				return nil, false
			})
		case tn == typeFunction, aliases[tn] == typeFunction:
			o, err = findT(ns, q, n, func(s *schema.Schema, name string) (*schema.Func, bool) {
				if f, ok := s.Func(name); ok && typeName(f) == tn {
					return f, true
				}
				return nil, false
			})
		case tn == typeProcedure, aliases[tn] == typeProcedure:
			o, err = findT(ns, q, n, func(s *schema.Schema, name string) (*schema.Proc, bool) {
				if f, ok := s.Proc(name); ok && typeName(f) == tn {
					return f, true
				}
				return nil, false
			})
		default:
			if o, err = findT(ns, q, n, func(s *schema.Schema, name string) (schema.Object, bool) {
				return s.Object(func(o schema.Object) bool {
					if o, ok := o.(SpecTypeNamer); ok {
						return p[0].T == o.SpecType() && name == o.SpecName()
					}
					return false
				})
			}); err != nil {
				continue // Custom objects might be loaded in a different pass.
			}
		}
		if err != nil {
			return fmt.Errorf("find %s reference for %s.depends_on[%d]: %w", p[0].T, loc, i, err)
		}
		t.AddDeps(o)
	}
	return nil
}

// FromPrimaryKey converts schema.Index to a sqlspec.PrimaryKey.
func FromPrimaryKey(s *schema.Index) (*sqlspec.PrimaryKey, error) {
	c := make([]*schemahcl.Ref, 0, len(s.Parts))
	for _, v := range s.Parts {
		c = append(c, ColumnRef(v.C.Name))
	}
	return &sqlspec.PrimaryKey{
		Columns: c,
	}, nil
}

// FromColumn converts a *schema.Column into a *sqlspec.Column using the ColumnTypeSpecFunc.
func FromColumn(c *schema.Column, columnTypeSpec ColumnTypeSpecFunc) (*sqlspec.Column, error) {
	ct, err := columnTypeSpec(c.Type.Type)
	if err != nil {
		return nil, err
	}
	spec := &sqlspec.Column{
		Name: c.Name,
		Type: ct.Type,
		Null: c.Type.Null,
		DefaultExtension: schemahcl.DefaultExtension{
			Extra: schemahcl.Resource{Attrs: ct.DefaultExtension.Extra.Attrs},
		},
	}
	switch v := c.Default.(type) {
	case nil:
	case *schema.NamedDefault:
		lv, err := ColumnDefault(c)
		if err != nil {
			return nil, err
		}
		spec.Extra.Children = append(spec.Extra.Children, &schemahcl.Resource{
			Type:  "default",
			Name:  v.Name,
			Attrs: []*schemahcl.Attr{{K: "as", V: lv}},
		})
	default:
		lv, err := ColumnDefault(c)
		if err != nil {
			return nil, err
		}
		spec.Extra.Attrs = slices.Insert(spec.Extra.Attrs, 0, &schemahcl.Attr{K: "default", V: lv})
	}
	convertCommentFromSchema(c.Attrs, &spec.Extra.Attrs)
	return spec, nil
}

// FromGenExpr returns the spec for a generated expression.
func FromGenExpr(x schema.GeneratedExpr, t func(string) string) *schemahcl.Resource {
	return &schemahcl.Resource{
		Type: "as",
		Attrs: []*schemahcl.Attr{
			schemahcl.StringAttr("expr", x.Expr),
			VarAttr("type", t(x.Type)),
		},
	}
}

// ConvertGenExpr converts the "as" attribute or the block under the given resource.
func ConvertGenExpr(r *schemahcl.Resource, c *schema.Column, t func(string) string) error {
	asA, okA := r.Attr("as")
	asR, okR := r.Resource("as")
	switch {
	case okA && okR:
		return fmt.Errorf("multiple as definitions for column %q", c.Name)
	case okA:
		expr, err := asA.String()
		if err != nil {
			return err
		}
		c.Attrs = append(c.Attrs, &schema.GeneratedExpr{
			Type: t(""), // default type.
			Expr: expr,
		})
	case okR:
		var spec struct {
			Expr string `spec:"expr"`
			Type string `spec:"type"`
		}
		if err := asR.As(&spec); err != nil {
			return err
		}
		c.Attrs = append(c.Attrs, &schema.GeneratedExpr{
			Expr: spec.Expr,
			Type: t(spec.Type),
		})
	}
	return nil
}

// ColumnDefault converts the column default into cty.Value.
func ColumnDefault(c *schema.Column) (cty.Value, error) {
	var textlike bool
	if c.Type != nil {
		switch c.Type.Type.(type) {
		case *schema.StringType, *schema.EnumType:
			textlike = true
		}
	}
	switch x := schema.UnderlyingExpr(c.Default).(type) {
	case nil:
		return cty.NilVal, nil
	case *schema.RawExpr:
		return schemahcl.RawExprValue(&schemahcl.RawExpr{X: x.X}), nil
	case *schema.Literal:
		switch {
		case oneOfPrefix(x.V, "0x", "0X", "0b", "0B", "b'", "B'", "x'", "X'"):
			return schemahcl.RawExprValue(&schemahcl.RawExpr{X: x.V}), nil
		case sqlx.IsQuoted(x.V, '\'', '"'):
			// Normalize single quotes to double quotes.
			s, err := sqlx.Unquote(x.V)
			if err != nil {
				return cty.NilVal, err
			}
			return cty.StringVal(s), nil
		case strings.ToLower(x.V) == "true", strings.ToLower(x.V) == "false":
			return cty.BoolVal(strings.ToLower(x.V) == "true"), nil
		case sqlx.IsLiteralNumber(x.V) && !textlike:
			if strings.Contains(x.V, ".") {
				f, err := strconv.ParseFloat(x.V, 64)
				if err != nil {
					return cty.NilVal, err
				}
				return cty.NumberFloatVal(f), nil
			}
			switch i, err := strconv.ParseInt(x.V, 10, 64); {
			case errors.Is(err, strconv.ErrRange):
				u, err := strconv.ParseUint(x.V, 10, 64)
				if err != nil {
					return cty.NilVal, err
				}
				return cty.NumberUIntVal(u), nil
			case err != nil:
				return cty.NilVal, err
			default:
				return cty.NumberIntVal(i), nil
			}
		default:
			// Literal values (non-expressions) are returned as strings.
			return cty.StringVal(x.V), nil
		}
	default:
		return cty.NilVal, fmt.Errorf("converting expr %T to literal value for column %s", x, c.Name)
	}
}

// FromIndex converts schema.Index to sqlspec.Index.
func FromIndex(idx *schema.Index, partFns ...func(*schema.Index, *schema.IndexPart, *sqlspec.IndexPart) error) (*sqlspec.Index, error) {
	spec := &sqlspec.Index{Name: idx.Name, Unique: idx.Unique}
	convertCommentFromSchema(idx.Attrs, &spec.Extra.Attrs)
	spec.Parts = make([]*sqlspec.IndexPart, len(idx.Parts))
	for i, p := range idx.Parts {
		part := &sqlspec.IndexPart{Desc: p.Desc}
		switch {
		case p.C == nil && p.X == nil:
			return nil, fmt.Errorf("missing column or expression for key part of index %q", idx.Name)
		case p.C != nil && p.X != nil:
			return nil, fmt.Errorf("multiple key part definitions for index %q", idx.Name)
		case p.C != nil:
			part.Column = ColumnRef(p.C.Name)
		case p.X != nil:
			x, ok := p.X.(*schema.RawExpr)
			if !ok {
				return nil, fmt.Errorf("unexpected expression %T for index %q", p.X, idx.Name)
			}
			part.Expr = x.X
		}
		for _, f := range partFns {
			if err := f(idx, p, part); err != nil {
				return nil, err
			}
		}
		spec.Parts[i] = part
	}
	if parts, ok := columnsOnly(spec.Parts); ok {
		spec.Parts = nil
		spec.Columns = parts
		return spec, nil
	}
	return spec, nil
}

func columnsOnly(parts []*sqlspec.IndexPart) ([]*schemahcl.Ref, bool) {
	columns := make([]*schemahcl.Ref, len(parts))
	for i, p := range parts {
		if p.Desc || p.Column == nil || len(p.Extra.Attrs) != 0 {
			return nil, false
		}
		columns[i] = p.Column
	}
	return columns, true
}

// FromForeignKey converts schema.ForeignKey to sqlspec.ForeignKey.
func FromForeignKey(s *schema.ForeignKey) (*sqlspec.ForeignKey, error) {
	c := make([]*schemahcl.Ref, 0, len(s.Columns))
	for _, v := range s.Columns {
		c = append(c, ColumnRef(v.Name))
	}
	r := make([]*schemahcl.Ref, 0, len(s.RefColumns))
	for _, v := range s.RefColumns {
		ref := ColumnRef(v.Name)
		if s.Table != s.RefTable {
			ref = ExternalColumnRef(v.Name, s.RefTable.Name)
		}
		r = append(r, ref)
	}
	fk := &sqlspec.ForeignKey{
		Symbol:     s.Symbol,
		Columns:    c,
		RefColumns: r,
	}
	if s.OnUpdate != "" {
		fk.OnUpdate = &schemahcl.Ref{V: Var(string(s.OnUpdate))}
	}
	if s.OnDelete != "" {
		fk.OnDelete = &schemahcl.Ref{V: Var(string(s.OnDelete))}
	}
	return fk, nil
}

// FromCheck converts schema.Check to sqlspec.Check.
func FromCheck(s *schema.Check) *sqlspec.Check {
	return &sqlspec.Check{
		Name: s.Name,
		Expr: s.Expr,
	}
}

// SchemaName returns the name from a ref to a schema.
func SchemaName(ref *schemahcl.Ref) (string, error) {
	vs, err := ref.ByType(typeSchema)
	if err != nil {
		return "", err
	}
	if len(vs) != 1 {
		return "", fmt.Errorf("expected 1 schema ref, got %d", len(vs))
	}
	return vs[0], nil
}

// ColumnByRef returns a column from the table by its reference.
func ColumnByRef(tv interface {
	Column(string) (*schema.Column, bool)
}, ref *schemahcl.Ref) (*schema.Column, error) {
	vs, err := ref.ByType(typeColumn)
	if err != nil {
		return nil, err
	}
	if len(vs) != 1 {
		return nil, fmt.Errorf("expected 1 column ref, got %d", len(vs))
	}
	if c, ok := tv.Column(vs[0]); ok {
		return c, nil
	}
	switch tv := tv.(type) {
	case *schema.Table:
		return nil, fmt.Errorf("column %q was not found in table %s", vs[0], tv.Name)
	case *schema.View:
		return nil, fmt.Errorf("column %q was not found in view %s", vs[0], tv.Name)
	default:
		return nil, fmt.Errorf("column %q was not found in %T", vs[0], tv)
	}
}

// IndexByRef returns a index from the table/view by its reference.
func IndexByRef(tv interface {
	Index(string) (*schema.Index, bool)
}, ref *schemahcl.Ref) (*schema.Index, error) {
	vs, err := ref.ByType(typeIndex)
	if err != nil {
		return nil, err
	}
	if len(vs) != 1 {
		return nil, fmt.Errorf("expected 1 index ref, got %d", len(vs))
	}
	if c, ok := tv.Index(vs[0]); ok {
		return c, nil
	}
	switch tv := tv.(type) {
	case *schema.Table:
		return nil, fmt.Errorf("index %q was not found in table %s", vs[0], tv.Name)
	case *schema.View:
		return nil, fmt.Errorf("index %q was not found in view %s", vs[0], tv.Name)
	default:
		return nil, fmt.Errorf("index %q was not found in %T", vs[0], tv)
	}
}

func externalRef(ref *schemahcl.Ref, sch *schema.Schema) (*schema.Table, *schema.Column, error) {
	qualifier, name, err := TableName(ref)
	if err != nil {
		return nil, nil, err
	}
	t, err := findT(sch, qualifier, name, func(s *schema.Schema, name string) (*schema.Table, bool) {
		return s.Table(name)
	})
	if err != nil {
		return nil, nil, err
	}
	c, err := ColumnByRef(t, ref)
	if err != nil {
		return nil, nil, err
	}
	return t, c, nil
}

// findT finds the table/view referenced by ref in the provided schema. If the table/view
// is not in the provided schema.Schema other schemas in the connected schema.Realm are
// searched as well.
func findT[T schema.Object](sch *schema.Schema, qualifier, name string, findT func(*schema.Schema, string) (T, bool)) (t T, err error) {
	var (
		matches []T              // Found references.
		schemas []*schema.Schema // Schemas to search.
	)
	switch {
	case sch.Realm == nil || qualifier == sch.Name:
		schemas = []*schema.Schema{sch}
	case qualifier == "":
		schemas = sch.Realm.Schemas
	default:
		s, ok := sch.Realm.Schema(qualifier)
		if ok {
			schemas = []*schema.Schema{s}
		}
	}
	for _, s := range schemas {
		t, ok := findT(s, name)
		if ok {
			matches = append(matches, t)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		err = fmt.Errorf("referenced %s %q not found", typeName(t), name)
	default:
		err = fmt.Errorf("multiple reference %ss found for %q", typeName(t), name)
	}
	return
}

// TableName returns the qualifier and name from a reference to a table.
func TableName(ref *schemahcl.Ref) (string, string, error) {
	return RefName(ref, typeTable)
}

// RefName returns the qualifier and name from a reference.
func RefName(ref *schemahcl.Ref, typeName string) (qualifier, name string, err error) {
	vs, err := ref.ByType(typeName)
	if err != nil {
		return "", "", err
	}
	switch len(vs) {
	case 1:
		name = vs[0]
	case 2:
		qualifier, name = vs[0], vs[1]
	default:
		return "", "", fmt.Errorf("sqlspec: unexpected number of references in %q", vs)
	}
	return
}

func isLocalRef(r *schemahcl.Ref) bool {
	return strings.HasPrefix(r.V, "$column")
}

// ColumnRef returns the reference of a column by its name.
func ColumnRef(cName string) *schemahcl.Ref {
	return schemahcl.BuildRef([]schemahcl.PathIndex{
		{T: typeColumn, V: []string{cName}},
	})
}

// IndexRef returns the reference of a index by its name.
func IndexRef(name string) *schemahcl.Ref {
	return schemahcl.BuildRef([]schemahcl.PathIndex{
		{T: typeIndex, V: []string{name}},
	})
}

// ExternalColumnRef returns the reference of a column by its name and table name.
func ExternalColumnRef(cName, tName string) *schemahcl.Ref {
	return schemahcl.BuildRef([]schemahcl.PathIndex{
		{T: typeTable, V: []string{tName}},
		{T: typeColumn, V: []string{cName}},
	})
}

// QualifiedExternalColRef returns the reference of a column by its name and qualified table name.
func QualifiedExternalColRef(cName, tName, sName string) *schemahcl.Ref {
	return schemahcl.BuildRef([]schemahcl.PathIndex{
		{T: typeTable, V: []string{sName, tName}},
		{T: typeColumn, V: []string{cName}},
	})
}

// SchemaRef returns the schemahcl.Ref to the schema with the given name.
func SchemaRef(name string) *schemahcl.Ref {
	return schemahcl.BuildRef([]schemahcl.PathIndex{
		{T: typeSchema, V: []string{name}},
	})
}

// Attrer is the interface that wraps the Attr method.
type Attrer interface {
	Attr(string) (*schemahcl.Attr, bool)
}

// convertCommentFromSpec converts a spec comment attribute to a schema element attribute.
func convertCommentFromSpec(spec Attrer, attrs *[]schema.Attr) error {
	if c, ok := spec.Attr("comment"); ok {
		s, err := c.String()
		if err != nil {
			return err
		}
		*attrs = append(*attrs, &schema.Comment{Text: s})
	}
	return nil
}

// convertCommentFromSchema converts a schema element comment attribute to a spec comment attribute.
func convertCommentFromSchema(src []schema.Attr, target *[]*schemahcl.Attr) {
	var c schema.Comment
	if sqlx.Has(src, &c) {
		*target = append(*target, schemahcl.StringAttr("comment", c.Text))
	}
}

// ReferenceVars holds the HCL variables
// for foreign keys' referential-actions.
var ReferenceVars = []string{
	Var(string(schema.NoAction)),
	Var(string(schema.Restrict)),
	Var(string(schema.Cascade)),
	Var(string(schema.SetNull)),
	Var(string(schema.SetDefault)),
}

// Var formats a string as variable to make it HCL compatible.
// The result is simple, replace each space with underscore.
func Var(s string) string { return strings.ReplaceAll(s, " ", "_") }

// FromVar is the inverse function of Var.
func FromVar(s string) string { return strings.ReplaceAll(s, "_", " ") }

func oneOfPrefix(s string, ps ...string) bool {
	for _, p := range ps {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
