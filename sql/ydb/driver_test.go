// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package ydb

import (
	"context"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestDriver_CheckClean(t *testing.T) {
	s := schema.New("test")
	drv := &Driver{Inspector: &mockInspector{schema: s}, conn: &conn{database: "test"}}

	// Empty schema.
	err := drv.CheckClean(context.Background(), nil)
	require.NoError(t, err)

	// Revisions table found.
	s.AddTables(schema.NewTable("revisions"))
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Name: "revisions", Schema: "test"})
	require.NoError(t, err)

	// Multiple tables.
	s.Tables = []*schema.Table{schema.NewTable("a"), schema.NewTable("revisions")}
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Name: "revisions", Schema: "test"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "found table")

	// Realm level checks
	r := schema.NewRealm()
	drv.database = ""
	drv.Inspector = &mockInspector{realm: r}

	// Empty realm.
	err = drv.CheckClean(context.Background(), nil)
	require.NoError(t, err)

	// Revisions table found.
	s.Tables = []*schema.Table{schema.NewTable("revisions").SetSchema(s)}
	r.AddSchemas(s)
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Name: "revisions", Schema: "test"})
	require.NoError(t, err)

	// Unknown table.
	s.Tables[0].Name = "unknown"
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Schema: "test", Name: "revisions"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "found table")
}

func TestDriver_RealmRestoreFunc(t *testing.T) {
	var (
		apply   = &mockPlanApplier{}
		inspect = &mockInspector{}
		drv     = &Driver{
			Inspector:   inspect,
			Differ:      DefaultDiff,
			conn:        &conn{database: "test"},
			PlanApplier: apply,
		}
	)
	f := drv.RealmRestoreFunc(schema.NewRealm().AddSchemas(schema.New("local")))

	// No changes.
	inspect.realm = schema.NewRealm().AddSchemas(schema.New("local"))
	err := f(context.Background())
	require.NoError(t, err)
	require.Empty(t, apply.applied)

	// Table changes (table added that needs to be dropped).
	apply.applied = nil
	inspect.realm = schema.NewRealm().AddSchemas(schema.New("local").AddTables(schema.NewTable("t1")))
	err = f(context.Background())
	require.NoError(t, err)
	require.Len(t, apply.applied, 1)
	drop, ok := apply.applied[0].(*schema.DropTable)
	require.True(t, ok)
	require.Equal(t, "t1", drop.T.Name)
}

func TestDriver_SchemaRestoreFunc(t *testing.T) {
	var (
		apply   = &mockPlanApplier{}
		inspect = &mockInspector{}
		drv     = &Driver{
			Inspector:   inspect,
			Differ:      DefaultDiff,
			conn:        &conn{database: "test"},
			PlanApplier: apply,
		}
	)
	desired := schema.New("local")
	f := drv.SchemaRestoreFunc(desired)

	// No changes.
	inspect.schema = schema.New("local")
	err := f(context.Background())
	require.NoError(t, err)
	require.Empty(t, apply.applied)

	// Table changes (table added that needs to be dropped).
	apply.applied = nil
	inspect.schema = schema.New("local").AddTables(schema.NewTable("t1"))
	err = f(context.Background())
	require.NoError(t, err)
	require.Len(t, apply.applied, 1)
	drop, ok := apply.applied[0].(*schema.DropTable)
	require.True(t, ok)
	require.Equal(t, "t1", drop.T.Name)
}

func TestDriver_StmtBuilder(t *testing.T) {
	drv := &Driver{}
	opts := migrate.PlanOptions{}

	b := drv.StmtBuilder(opts)
	require.NotNil(t, b)

	// Test that builder uses backtick quoting
	b.Ident("table_name")
	require.Contains(t, b.String(), "`table_name`")
}

func TestDriver_ScanStmts(t *testing.T) {
	drv := &Driver{}

	input := `CREATE TABLE users (id Int32, name Utf8, PRIMARY KEY (id));
DROP TABLE users;`

	stmts, err := drv.ScanStmts(input)
	require.NoError(t, err)
	require.Len(t, stmts, 2)
}

// mockInspector is a mock implementation of schema.Inspector.
type mockInspector struct {
	schema.Inspector
	realm  *schema.Realm
	schema *schema.Schema
}

func (m *mockInspector) InspectSchema(context.Context, string, *schema.InspectOptions) (*schema.Schema, error) {
	if m.schema == nil {
		return nil, &schema.NotExistError{}
	}
	return m.schema, nil
}

func (m *mockInspector) InspectRealm(context.Context, *schema.InspectRealmOption) (*schema.Realm, error) {
	return m.realm, nil
}

// mockPlanApplier is a mock implementation of migrate.PlanApplier.
type mockPlanApplier struct {
	planned []schema.Change
	applied []schema.Change
}

func (m *mockPlanApplier) PlanChanges(_ context.Context, _ string, planned []schema.Change, _ ...migrate.PlanOption) (*migrate.Plan, error) {
	m.planned = append(m.planned, planned...)
	return nil, nil
}

func (m *mockPlanApplier) ApplyChanges(_ context.Context, applied []schema.Change, _ ...migrate.PlanOption) error {
	m.applied = append(m.applied, applied...)
	return nil
}
