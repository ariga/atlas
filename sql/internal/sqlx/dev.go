// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlx

import (
	"context"
	"fmt"
	"strings"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
)

// DevDriver is a driver that provides additional functionality
// to interact with the development database.
type DevDriver struct {
	// A Driver connected to the dev database.
	Driver migrate.Driver
	// PathObject allows providing a custom function to patch
	// objects that hold a schema reference.
	PatchObject func(*schema.Schema, schema.Object)
}

// NormalizeRealm implements the schema.Normalizer interface.
//
// The implementation converts schema objects in "natural form" (e.g. HCL or DSL)
// to their "normal presentation" in the database, by creating them temporarily in
// a "dev database", and then inspects them from there.
func (d *DevDriver) NormalizeRealm(ctx context.Context, r *schema.Realm) (nr *schema.Realm, err error) {
	restore, err := d.Driver.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if rerr := restore(ctx); rerr != nil {
			if err != nil {
				rerr = fmt.Errorf("%w: %v", err, rerr)
			}
			err = rerr
		}
	}()
	var (
		changes []schema.Change
		opts    = &schema.InspectRealmOption{
			Schemas: make([]string, 0, len(r.Schemas)),
		}
	)
	for _, o := range r.Objects {
		changes = append(changes, &schema.AddObject{
			O: o,
			Extra: []schema.Clause{
				&schema.IfNotExists{},
			},
		})
	}
	name2pos := make(key2pos)
	for _, s := range r.Schemas {
		k, _ := name2pos.put(s.Attrs, keyS, s.Name)
		opts.Schemas = append(opts.Schemas, s.Name)
		changes = append(changes, &schema.AddSchema{
			S: s,
			Extra: []schema.Clause{
				&schema.IfNotExists{},
			},
		})
		for _, t := range s.Tables {
			changes = append(changes, addTableChange(t)...)
			// If the table was loaded with its position,
			// record the position of its children.
			if tk, ok := name2pos.put(t.Attrs, k, keyT, t.Name); ok {
				name2pos.putTable(t, tk)
			}
		}
		for _, v := range s.Views {
			changes = append(changes, addViewChange(v)...)
			// If the view was loaded with its position,
			// record the position of its children.
			if vk, ok := name2pos.put(v.Attrs, k, keyV, v.Name); ok {
				name2pos.putView(v, vk)
			}
		}
		for _, o := range s.Objects {
			changes = append(changes, &schema.AddObject{O: o})
		}
		for _, f := range s.Funcs {
			name2pos.put(f.Attrs, k, keyFn, f.Name)
			changes = append(changes, &schema.AddFunc{F: f})
		}
		for _, p := range s.Procs {
			name2pos.put(p.Attrs, k, keyPr, p.Name)
			changes = append(changes, &schema.AddProc{P: p})
		}
	}
	if err := d.Driver.ApplyChanges(ctx, changes); err != nil {
		return nil, err
	}
	if nr, err = d.Driver.InspectRealm(ctx, opts); err != nil {
		return nil, err
	}
	if len(name2pos) > 0 {
		name2pos.patchRealm(nr)
	}
	return nr, nil
}

// NormalizeSchema returns the normal representation of the given database. See NormalizeRealm for more info.
func (d *DevDriver) NormalizeSchema(ctx context.Context, s *schema.Schema) (*schema.Schema, error) {
	restore, err := d.Driver.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if rerr := restore(ctx); rerr != nil {
			if err != nil {
				rerr = fmt.Errorf("%w: %v", err, rerr)
			}
			err = rerr
		}
	}()
	dev, err := d.Driver.InspectSchema(ctx, "", &schema.InspectOptions{
		Mode: schema.InspectSchemas,
	})
	if err != nil {
		return nil, err
	}
	// Modify dev-schema attributes if needed.
	changes, err := d.Driver.SchemaDiff(
		schema.New(dev.Name).AddAttrs(dev.Attrs...),
		schema.New(dev.Name).AddAttrs(s.Attrs...),
	)
	if err != nil {
		return nil, err
	}
	prevName := s.Name
	s.Name = dev.Name
	name2pos := make(key2pos)
	k, _ := name2pos.put(s.Attrs, keyS, s.Name)
	for _, t := range s.Tables {
		// If objects are not strongly connected.
		if t.Schema != s {
			t.Schema = s
		}
		for _, c := range t.Columns {
			if e, ok := c.Type.Type.(*schema.EnumType); ok && e.Schema != s {
				e.Schema = s
			}
		}
		changes = append(changes, addTableChange(t)...)
		if tk, ok := name2pos.put(t.Attrs, k, keyT, t.Name); ok {
			name2pos.putTable(t, tk)
		}
	}
	for _, v := range s.Views {
		// If objects are not strongly connected.
		if v.Schema != s {
			v.Schema = s
		}
		changes = append(changes, addViewChange(v)...)
		if vk, ok := name2pos.put(v.Attrs, k, keyV, v.Name); ok {
			name2pos.putView(v, vk)
		}
	}
	for _, o := range s.Objects {
		if d.PatchObject != nil {
			d.PatchObject(s, o)
		}
		changes = append(changes, &schema.AddObject{O: o})
	}
	for _, f := range s.Funcs {
		changes = append(changes, &schema.AddFunc{F: f})
	}
	for _, p := range s.Procs {
		changes = append(changes, &schema.AddProc{P: p})
	}
	if err := d.Driver.ApplyChanges(ctx, changes, func(opts *migrate.PlanOptions) {
		noQualifier := ""
		opts.SchemaQualifier = &noQualifier
		opts.Mode = migrate.PlanModeInPlace
	}); err != nil {
		return nil, err
	}
	ns, err := d.Driver.InspectSchema(ctx, "", nil)
	if err != nil {
		return nil, err
	}
	// Preserve the original schema name and attributes.
	ns.Name = prevName
	for _, a := range s.Attrs {
		schema.ReplaceOrAppend(&ns.Attrs, a)
	}
	if len(name2pos) > 0 {
		name2pos.patchSchema(ns)
	}
	return ns, err
}

const (
	keyS  = "schema"
	keyV  = "view"
	keyT  = "table"
	keyC  = "column"
	keyI  = "index"
	keyP  = "pk"
	keyF  = "fk"
	keyK  = "check"
	keyTg = "trigger"
	keyFn = "function"
	keyPr = "procedure"
)

type key2pos map[string]*schema.Pos

func (k key2pos) putTable(t *schema.Table, tk string) {
	for _, c := range t.Columns {
		k.put(c.Attrs, tk, keyC, c.Name)
	}
	for _, i := range t.Indexes {
		k.put(i.Attrs, tk, keyI, i.Name)
	}
	for _, f := range t.ForeignKeys {
		k.put(f.Attrs, tk, keyF, f.Symbol)
	}
	for _, ck := range t.Checks() {
		k.put(ck.Attrs, tk, keyK, ck.Name)
	}
	if t.PrimaryKey != nil {
		k.put(t.PrimaryKey.Attrs, tk, keyP, t.PrimaryKey.Name)
	}
	for _, r := range t.Triggers {
		k.put(r.Attrs, tk, keyTg, r.Name)
	}
}

func (k key2pos) putView(v *schema.View, vk string) {
	for _, c := range v.Columns {
		k.put(c.Attrs, vk, keyC, c.Name)
	}
	for _, i := range v.Indexes {
		k.put(i.Attrs, vk, keyI, i.Name)
	}
	for _, r := range v.Triggers {
		k.put(r.Attrs, vk, keyTg, r.Name)
	}
}

func (k key2pos) put(attrs []schema.Attr, typename ...string) (string, bool) {
	n := poskey(typename...)
	if p := (schema.Pos{}); Has(attrs, &p) {
		k[n] = &p
		return n, true
	}
	return n, false
}

func (k key2pos) patchRealm(r *schema.Realm) {
	for _, s := range r.Schemas {
		k.patchSchema(s)
	}
}

func (k key2pos) patchSchema(s *schema.Schema) {
	ks, _ := k.patch(&s.Attrs, keyS, s.Name)
	for _, t := range s.Tables {
		tk, ok := k.patch(&t.Attrs, ks, keyT, t.Name)
		if !ok {
			continue
		}
		for _, tr := range t.Triggers {
			k.patch(&tr.Attrs, tk, keyTg, tr.Name)
		}
		for _, c := range t.Columns {
			k.patch(&c.Attrs, tk, keyC, c.Name)
		}
		for _, i := range t.Indexes {
			k.patch(&i.Attrs, tk, keyI, i.Name)
		}
		for _, f := range t.ForeignKeys {
			k.patch(&f.Attrs, tk, keyF, f.Symbol)
		}
		for _, ck := range t.Checks() {
			k.patch(&ck.Attrs, tk, keyK, ck.Name)
		}
	}
	for _, v := range s.Views {
		vk, ok := k.patch(&v.Attrs, ks, keyV, v.Name)
		if !ok {
			continue
		}
		for _, tr := range v.Triggers {
			k.patch(&tr.Attrs, vk, keyTg, tr.Name)
		}
		for _, c := range v.Columns {
			k.patch(&c.Attrs, vk, keyC, c.Name)
		}
		for _, i := range v.Indexes {
			k.patch(&i.Attrs, vk, keyI, i.Name)
		}
	}
	for _, f := range s.Funcs {
		k.patch(&f.Attrs, ks, keyFn, f.Name)
	}
	for _, p := range s.Procs {
		k.patch(&p.Attrs, ks, keyPr, p.Name)
	}
}

func (k key2pos) patch(attrs *[]schema.Attr, typename ...string) (string, bool) {
	n := poskey(typename...)
	if p, ok := k[n]; ok {
		*attrs = append(*attrs, p)
		return n, true
	}
	return n, false
}

func poskey(typename ...string) string {
	return strings.Join(typename, ".")
}
