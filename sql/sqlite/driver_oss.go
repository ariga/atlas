// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package sqlite

import (
	"context"
	"fmt"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"
)

// InspectRealm returns schema descriptions of all resources in the given realm.
func (i *inspect) InspectRealm(ctx context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	schemas, err := i.databases(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(schemas) > 1 {
		return nil, fmt.Errorf("sqlite: multiple database files are not supported by the driver. got: %d", len(schemas))
	}
	if opts == nil {
		opts = &schema.InspectRealmOption{}
	}
	var (
		r    = schema.NewRealm(schemas...)
		mode = sqlx.ModeInspectRealm(opts)
	)
	if mode.Is(schema.InspectTables) {
		for _, s := range schemas {
			tables, err := i.tables(ctx, nil)
			if err != nil {
				return nil, err
			}
			s.AddTables(tables...)
			for _, t := range tables {
				if err := i.inspectTable(ctx, t); err != nil {
					return nil, err
				}
			}
		}
		sqlx.LinkSchemaTables(r.Schemas)
	}
	if mode.Is(schema.InspectViews) {
		if err := i.inspectViews(ctx, r, nil); err != nil {
			return nil, err
		}
	}
	if mode.Is(schema.InspectTriggers) {
		if err := i.inspectTriggers(ctx, r, nil); err != nil {
			return nil, err
		}
	}
	return schema.ExcludeRealm(r, opts.Exclude)
}

// InspectSchema returns schema descriptions of the tables in the given schema.
// If the schema name is empty, the "main" database is used.
func (i *inspect) InspectSchema(ctx context.Context, name string, opts *schema.InspectOptions) (*schema.Schema, error) {
	if name == "" {
		name = mainFile
	}
	schemas, err := i.databases(ctx, &schema.InspectRealmOption{
		Schemas: []string{name},
	})
	if err != nil {
		return nil, err
	}
	if len(schemas) == 0 {
		return nil, &schema.NotExistError{
			Err: fmt.Errorf("sqlite: schema %q was not found", name),
		}
	}
	if opts == nil {
		opts = &schema.InspectOptions{}
	}
	var (
		r    = schema.NewRealm(schemas...)
		mode = sqlx.ModeInspectSchema(opts)
	)
	if mode.Is(schema.InspectTables) {
		tables, err := i.tables(ctx, opts)
		if err != nil {
			return nil, err
		}
		r.Schemas[0].AddTables(tables...)
		for _, t := range tables {
			if err := i.inspectTable(ctx, t); err != nil {
				return nil, err
			}
		}
		sqlx.LinkSchemaTables(schemas)
	}
	if mode.Is(schema.InspectViews) {
		if err := i.inspectViews(ctx, r, opts); err != nil {
			return nil, err
		}
	}
	if mode.Is(schema.InspectTriggers) {
		if err := i.inspectTriggers(ctx, r, nil); err != nil {
			return nil, err
		}
	}
	return schema.ExcludeSchema(r.Schemas[0], opts.Exclude)
}

var (
	specOptions []schemahcl.Option
	scanFuncs   = &specutil.ScanFuncs{
		Table: convertTable,
		View:  convertView,
	}
)

func triggersSpec([]*schema.Trigger, *specutil.Doc) ([]*sqlspec.Trigger, error) {
	return nil, nil // unimplemented.
}

func (*inspect) inspectViews(context.Context, *schema.Realm, *schema.InspectOptions) error {
	return nil // unimplemented.
}

func (*inspect) inspectTriggers(context.Context, *schema.Realm, *schema.InspectOptions) error {
	return nil // unimplemented.
}

func (*state) addView(*schema.AddView) error {
	return nil // unimplemented.
}

func (*state) dropView(*schema.DropView) error {
	return nil // unimplemented.
}

func (*state) modifyView(*schema.ModifyView) error {
	return nil // unimplemented.
}

func (*state) renameView(*schema.RenameView) error {
	return nil // unimplemented.
}

func (*state) addTrigger(*schema.AddTrigger) error {
	return nil // unimplemented.
}

func (*state) dropTrigger(*schema.DropTrigger) error {
	return nil // unimplemented.
}

func verifyChanges(context.Context, []schema.Change) error {
	return nil // unimplemented.
}

// SupportChange reports if the change is supported by the differ.
func (*diff) SupportChange(c schema.Change) bool {
	switch c.(type) {
	case *schema.RenameConstraint:
		return false
	}
	return true
}
