// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package specutil

import (
	"fmt"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// VarAttr is a helper method for constructing *schemahcl.Attr instances that contain a variable reference.
func VarAttr(k, v string) *schemahcl.Attr {
	return schemahcl.RefAttr(k, &schemahcl.Ref{V: v})
}

type (
	// SchemaSpec is returned by driver convert functions to
	// marshal a *schema.Schema into top-level spec objects.
	SchemaSpec struct {
		Schema       *sqlspec.Schema
		Tables       []*sqlspec.Table
		Views        []*sqlspec.View
		Funcs        []*sqlspec.Func
		Procs        []*sqlspec.Func
		Materialized []*sqlspec.View
		// Collected triggers to convert into spec.
		Triggers []*schema.Trigger
	}
	// RealmFuncs represents the functions that used
	// to convert the schema.Realm into HCL spec document.
	RealmFuncs struct {
		Schema   func(*schema.Schema) (*SchemaSpec, error)
		Triggers func([]*schema.Trigger, *Doc) ([]*sqlspec.Trigger, error)
	}
	// Doc represents the common HCL spec document.
	Doc struct {
		Tables       []*sqlspec.Table   `spec:"table"`
		Views        []*sqlspec.View    `spec:"view"`
		Materialized []*sqlspec.View    `spec:"materialized"`
		Funcs        []*sqlspec.Func    `spec:"function"`
		Procs        []*sqlspec.Func    `spec:"procedure"`
		Triggers     []*sqlspec.Trigger `spec:"trigger"`
		Schemas      []*sqlspec.Schema  `spec:"schema"`
	}
)

// Marshal marshals v into an Atlas DDL document using a schemahcl.Marshaler. Marshal uses the given
// schemaSpec function to convert a *schema.Schema into *sqlspec.Schema, []*sqlspec.Table and []*sqlspec.View.
func Marshal(v any, marshaler schemahcl.Marshaler, funcs RealmFuncs) ([]byte, error) {
	var (
		d  = &Doc{}
		ts []*schema.Trigger
	)
	switch s := v.(type) {
	case *schema.Schema:
		spec, err := funcs.Schema(s)
		if err != nil {
			return nil, fmt.Errorf("specutil: failed converting schema to spec: %w", err)
		}
		d.Tables = spec.Tables
		d.Views = spec.Views
		d.Materialized = spec.Materialized
		d.Schemas = []*sqlspec.Schema{spec.Schema}
		d.Funcs = spec.Funcs
		d.Procs = spec.Procs
		ts = spec.Triggers
	case *schema.Realm:
		for _, s := range s.Schemas {
			spec, err := funcs.Schema(s)
			if err != nil {
				return nil, fmt.Errorf("specutil: failed converting schema to spec: %w", err)
			}
			d.Tables = append(d.Tables, spec.Tables...)
			d.Views = append(d.Views, spec.Views...)
			d.Materialized = append(d.Materialized, spec.Materialized...)
			d.Schemas = append(d.Schemas, spec.Schema)
			d.Funcs = append(d.Funcs, spec.Funcs...)
			d.Procs = append(d.Procs, spec.Procs...)
			ts = append(ts, spec.Triggers...)
		}
		if err := QualifyObjects(d.Tables); err != nil {
			return nil, err
		}
		if err := QualifyObjects(d.Views); err != nil {
			return nil, err
		}
		if err := QualifyObjects(d.Materialized); err != nil {
			return nil, err
		}
		if err := QualifyObjects(d.Funcs); err != nil {
			return nil, err
		}
		if err := QualifyObjects(d.Procs); err != nil {
			return nil, err
		}
		if err := QualifyReferences(d.Tables, s); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("specutil: failed marshaling spec. %T is not supported", v)
	}
	if funcs.Triggers != nil {
		specs, err := funcs.Triggers(ts, d)
		if err != nil {
			return nil, err
		}
		d.Triggers = specs
	}
	return marshaler.MarshalSpec(d)
}

// SchemaObject describes a top-level schema object
// that might be qualified, e.g. a table or a view.
type SchemaObject interface {
	Label() string
	QualifierLabel() string
	SetQualifier(string)
	SchemaRef() *schemahcl.Ref
}

// QualifyObjects sets the Qualifier field equal to the schema
// name in any objects with duplicate names in the provided specs.
func QualifyObjects[T SchemaObject](specs []T) error {
	var (
		schemas = make(map[string]bool, len(specs))
		byLabel = make(map[string]map[string][]T, len(specs))
	)
	// Loop first and qualify schema objects with the same label.
	// For example, two tables named "users" reside in different
	// schemas are converted to: ("s1", "users") and ("s2", "users").
	for _, v := range specs {
		l := v.Label()
		q, err := SchemaName(v.SchemaRef())
		if err != nil {
			return err
		}
		if _, ok := byLabel[l]; !ok {
			byLabel[l] = make(map[string][]T)
		}
		byLabel[l][q] = append(byLabel[l][q], v)
	}
	for _, v := range byLabel {
		// Multiple objects with the same label on the same
		// schema are not qualified (repeatable blocks).
		if len(v) == 1 {
			continue
		}
		for q, sv := range v {
			for _, s := range sv {
				s.SetQualifier(q)
				schemas[q] = true
			}
		}
	}
	// After objects were qualified, they might be conflicted with different
	// resources that labeled with the schema name. e.g., ("s1", "users") and
	// ("s1"). To resolve this conflict, we qualify these objects as well.
	for _, v := range specs {
		if v.QualifierLabel() == "" && schemas[v.Label()] {
			schemaName, err := SchemaName(v.SchemaRef())
			if err != nil {
				return err
			}
			v.SetQualifier(schemaName)
		}
	}
	return nil
}

// ColumnRefFinder returns a function that finds a column reference by its table and its schema.
func ColumnRefFinder(specs []*sqlspec.Table) func(s, t, c string) (*schemahcl.Ref, error) {
	var refs map[struct{ s, t string }]*sqlspec.Table
	return func(s, t, c string) (*schemahcl.Ref, error) {
		// Lazily initialize refs.
		if refs == nil {
			refs = make(map[struct{ s, t string }]*sqlspec.Table, len(specs))
			for _, st := range specs {
				ns, err := SchemaName(st.Schema)
				if err != nil {
					return nil, err
				}
				refs[struct{ s, t string }{s: ns, t: st.Name}] = st
			}
		}
		r, ok := refs[struct{ s, t string }{s: s, t: t}]
		if !ok {
			return nil, fmt.Errorf("table %q.%q was not found", s, t)
		}
		if r.Qualifier != "" {
			return QualifiedExternalColRef(c, r.Name, r.Qualifier), nil
		}
		return ExternalColumnRef(c, r.Name), nil
	}
}

// QualifyReferences qualifies any reference with qualifier.
func QualifyReferences(tableSpecs []*sqlspec.Table, realm *schema.Realm) error {
	type cref struct{ s, t string }
	byRef := make(map[cref]*sqlspec.Table)
	for _, t := range tableSpecs {
		r := cref{s: t.Qualifier, t: t.Name}
		if byRef[r] != nil {
			return fmt.Errorf("duplicate references were found for: %v", r)
		}
		byRef[r] = t
	}
	for _, t := range tableSpecs {
		sname, err := SchemaName(t.Schema)
		if err != nil {
			return err
		}
		s1, ok := realm.Schema(sname)
		if !ok {
			return fmt.Errorf("schema %q was not found in realm", sname)
		}
		t1, ok := s1.Table(t.Name)
		if !ok {
			return fmt.Errorf("table %q.%q was not found in realm", sname, t.Name)
		}
		for _, fk := range t.ForeignKeys {
			fk1, ok := t1.ForeignKey(fk.Symbol)
			if !ok {
				return fmt.Errorf("table %q.%q.%q was not found in realm", sname, t.Name, fk.Symbol)
			}
			for i, c := range fk.RefColumns {
				if r, ok := byRef[cref{s: fk1.RefTable.Schema.Name, t: fk1.RefTable.Name}]; ok && r.Qualifier != "" {
					fk.RefColumns[i] = QualifiedExternalColRef(fk1.RefColumns[i].Name, r.Name, r.Qualifier)
				} else if r, ok := byRef[cref{t: fk1.RefTable.Name}]; ok && r.Qualifier == "" {
					fk.RefColumns[i] = ExternalColumnRef(fk1.RefColumns[i].Name, r.Name)
				} else {
					return fmt.Errorf("missing reference for column %q in %q.%q.%q", c.V, sname, t.Name, fk.Symbol)
				}
			}
		}
	}
	return nil
}

// HCLBytesFunc returns a helper that evaluates an HCL document from a byte slice instead
// of from an hclparse.Parser instance.
func HCLBytesFunc(ev schemahcl.Evaluator) func(b []byte, v any, inp map[string]cty.Value) error {
	return func(b []byte, v any, inp map[string]cty.Value) error {
		parser := hclparse.NewParser()
		if _, diag := parser.ParseHCL(b, ""); diag.HasErrors() {
			return diag
		}
		return ev.Eval(parser, v, inp)
	}
}
