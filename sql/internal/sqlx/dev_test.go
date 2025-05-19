// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlx

import (
	"context"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/require"
)

func TestDriver_NormalizeRealm(t *testing.T) {
	var (
		drv = &mockDriver{
			realm: schema.NewRealm(schema.New("test").SetCharset("utf8mb4")),
		}
		dev = &DevDriver{
			Driver: drv,
		}
		r = schema.NewRealm(schema.New("test"))
	)
	normal, err := dev.NormalizeRealm(context.Background(), r)
	require.NoError(t, err)
	require.Equal(t, normal, drv.realm)

	require.Len(t, drv.schemas, 1)
	require.Len(t, drv.changes, 1, "expect 1 call for creating the schema")
	require.Equal(t, &schema.AddSchema{
		S: r.Schemas[0],
		// The "IF NOT EXISTS" clause is added to make the
		// operation noop for schema like "public" in Postgres.
		Extra: []schema.Clause{&schema.IfNotExists{}},
	}, drv.changes[0])

	// Retain positions.
	r.Schemas[0].
		AddAttrs(
			schema.NewFilePos("schema.hcl").
				SetStart(hcl.Pos{Line: 1, Column: 1, Byte: 1}),
		).
		AddTables(
			schema.NewTable("t1").
				AddAttrs(
					schema.NewFilePos("schema.hcl").
						SetStart(hcl.Pos{Line: 2, Column: 2, Byte: 2}),
				).
				AddColumns(
					schema.NewIntColumn("id", "int").
						AddAttrs(
							schema.NewFilePos("schema.hcl").
								SetStart(hcl.Pos{Line: 3, Column: 3, Byte: 3}),
						),
				),
		)
	drv.realm.Schemas[0].AddTables(
		schema.NewTable("t1").AddColumns(schema.NewIntColumn("id", "int")),
	)
	normal, err = dev.NormalizeRealm(context.Background(), r)
	require.NoError(t, err)
	p := normal.Schemas[0].Pos()
	require.Equal(t, schema.NewFilePos("schema.hcl").SetStart(hcl.Pos{Line: 1, Column: 1, Byte: 1}), p)
	p = normal.Schemas[0].Tables[0].Pos()
	require.Equal(t, schema.NewFilePos("schema.hcl").SetStart(hcl.Pos{Line: 2, Column: 2, Byte: 2}), p)
	p = normal.Schemas[0].Tables[0].Columns[0].Pos()
	require.Equal(t, schema.NewFilePos("schema.hcl").SetStart(hcl.Pos{Line: 3, Column: 3, Byte: 3}), p)
}

type mockDriver struct {
	migrate.Driver
	// Inspect.
	schemas []string
	realm   *schema.Realm
	// Apply.
	changes []schema.Change
}

func (m *mockDriver) InspectRealm(_ context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	m.schemas = append(m.schemas, opts.Schemas...)
	return m.realm, nil
}

func (m *mockDriver) ApplyChanges(_ context.Context, changes []schema.Change, _ ...migrate.PlanOption) error {
	m.changes = append(m.changes, changes...)
	return nil
}

func (m *mockDriver) CheckClean(context.Context, *migrate.TableIdent) error {
	return nil
}

func (m *mockDriver) Snapshot(context.Context) (migrate.RestoreFunc, error) {
	return func(context.Context) error { return nil }, nil
}
