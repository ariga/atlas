// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlx

import (
	"context"
	"fmt"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
)

// DevDriver is a driver that provides additional functionality
// to interact with the development database.
type DevDriver struct {
	// A Driver connected to the dev database.
	Driver interface {
		migrate.Driver
		migrate.CleanChecker
		migrate.Snapshoter
	}
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
	for _, s := range r.Schemas {
		opts.Schemas = append(opts.Schemas, s.Name)
		changes = append(changes, &schema.AddSchema{
			S: s,
			Extra: []schema.Clause{
				&schema.IfNotExists{},
			},
		})
		for _, t := range s.Tables {
			changes = append(changes, &schema.AddTable{T: t})
			for _, r := range t.Triggers {
				changes = append(changes, &schema.AddTrigger{T: r})
			}
		}
		for _, v := range s.Views {
			changes = append(changes, &schema.AddView{V: v})
			for _, r := range v.Triggers {
				changes = append(changes, &schema.AddTrigger{T: r})
			}
		}
		for _, o := range s.Objects {
			changes = append(changes, &schema.AddObject{O: o})
		}
		for _, f := range s.Funcs {
			changes = append(changes, &schema.AddFunc{F: f})
		}
		for _, p := range s.Procs {
			changes = append(changes, &schema.AddProc{P: p})
		}
	}
	if err := d.Driver.ApplyChanges(ctx, changes); err != nil {
		return nil, err
	}
	nr, err = d.Driver.InspectRealm(ctx, opts)
	return
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
	}
	for _, v := range s.Views {
		// If objects are not strongly connected.
		if v.Schema != s {
			v.Schema = s
		}
		changes = append(changes, addViewChange(v)...)
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
	return ns, err
}
