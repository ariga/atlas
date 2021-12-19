// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate_test

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestDir_Plan(t *testing.T) {
	var (
		m   = &mockDriver{}
		ctx = context.Background()
	)
	dir, err := migrate.NewDir(
		migrate.DirPath("testdata"),
		migrate.DirGlob("*.up.sql"),
		migrate.DirConn(m),
	)
	require.NoError(t, err)
	plan, err := dir.Plan(ctx, "plan_name", migrate.Realm(nil))
	require.Equal(t, migrate.ErrNoPlan, err)
	require.Nil(t, plan)
	require.Equal(t, []string{"CREATE TABLE t(c int);"}, m.executed)

	m.executed = nil
	dir, err = migrate.NewDir(
		migrate.DirPath("testdata"),
		migrate.DirGlob("*.sql"),
		migrate.DirConn(m),
	)
	require.NoError(t, err)
	plan, err = dir.Plan(ctx, "plan_name", migrate.Realm(nil))
	require.Equal(t, migrate.ErrNoPlan, err)
	require.Nil(t, plan)
	require.Equal(t, []string{"DROP TABLE IF EXISTS t;", "CREATE TABLE t(c int);", "CREATE TABLE t(c int);"}, m.executed)

	m.changes = append(m.changes, &schema.AddTable{T: &schema.Table{Name: "t1"}}, &schema.AddTable{T: &schema.Table{Name: "t2"}})
	m.plan = &migrate.Plan{
		Name: "add_t1_t2",
		Changes: []*migrate.Change{
			{Cmd: "CREATE TABLE t1(c int);"},
			{Cmd: "CREATE TABLE t2(c int);"},
		},
	}
	plan, err = dir.Plan(ctx, "plan_name", migrate.Realm(nil))
	require.NoError(t, err)
	require.NotNil(t, plan)
}

func TestDir_WritePlan_Compact(t *testing.T) {
	var (
		f   = &mockFS{}
		m   = &mockDriver{}
		ctx = context.Background()
	)
	dir, err := migrate.NewDir(
		migrate.DirFS(f),
		migrate.DirConn(m),
	)
	require.NoError(t, err)
	err = dir.WritePlan(&migrate.Plan{
		Name: "add_t1_t2",
		Changes: []*migrate.Change{
			{Cmd: "CREATE TABLE t1 (c int)"},
			{Cmd: "CREATE TABLE t2 (c int);"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "CREATE TABLE t1 (c int);\nCREATE TABLE t2 (c int);\n", f.files[0].F)

	f.files = nil
	dir, err = migrate.NewDir(
		migrate.DirFS(f),
		migrate.DirConn(m),
		migrate.DirTemplates("{{.Name}}.sql", `{{range $c := .Changes}}{{printf "--%s\n%s;\n" $c.Comment $c.Cmd}}{{end}}`),
	)
	require.NoError(t, err)
	err = dir.WritePlan(&migrate.Plan{
		Name: "add_t1_t2",
		Changes: []*migrate.Change{
			{Cmd: "CREATE TABLE t1 (c int)", Comment: "Create a new table named t1."},
			{Cmd: "CREATE TABLE t2 (c int)", Comment: "Create a new table named t2."},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "add_t1_t2.sql", f.files[0].N)
	require.Equal(t, "--Create a new table named t1.\nCREATE TABLE t1 (c int);\n--Create a new table named t2.\nCREATE TABLE t2 (c int);\n", f.files[0].F)

	err = dir.WritePlan(&migrate.Plan{
		Name: "add_t3",
		Changes: []*migrate.Change{
			{Cmd: "CREATE TABLE t3 (c int)", Comment: "Create a new table named t3."},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "add_t3.sql", f.files[1].N)
	require.Equal(t, "--Create a new table named t3.\nCREATE TABLE t3 (c int);\n", f.files[1].F)

	m.changes = append(m.changes, &schema.AddTable{T: &schema.Table{Name: "t1"}}, &schema.AddTable{T: &schema.Table{Name: "t2"}})
	m.plan = &migrate.Plan{Name: "schema", Changes: []*migrate.Change{{Cmd: "CREATE TABLE t1 (c int)"}, {Cmd: "CREATE TABLE t2 (c int)"}, {Cmd: "CREATE TABLE t3 (c int)"}}}
	err = dir.Compact(ctx, "schema", -1)
	require.NoError(t, err)
	require.Len(t, f.files, 1)
	require.Equal(t, "schema.sql", f.files[0].N)
	require.Equal(t, "--\nCREATE TABLE t1 (c int);\n--\nCREATE TABLE t2 (c int);\n--\nCREATE TABLE t3 (c int);\n", f.files[0].F)

}

type mockFS struct {
	fs.GlobFS
	files []struct{ N, F string }
}

func (f *mockFS) Glob(pattern string) ([]string, error) {
	var matches []string
	for i := range f.files {
		match, err := filepath.Match(pattern, f.files[i].N)
		if err != nil {
			return nil, err
		}
		if match {
			matches = append(matches, f.files[i].N)
		}
	}
	return matches, nil
}

func (f *mockFS) ReadFile(name string) ([]byte, error) {
	for i := range f.files {
		if f.files[i].N == name {
			return []byte(f.files[i].F), nil
		}
	}
	return nil, os.ErrNotExist
}

func (f *mockFS) WriteFile(name string, data []byte, _ fs.FileMode) error {
	f.files = append(f.files, struct{ N, F string }{N: name, F: string(data)})
	return nil
}

func (f *mockFS) RemoveFile(name string) error {
	for i := range f.files {
		if f.files[i].N == name {
			f.files = append(f.files[:i], f.files[i+1:]...)
			return nil
		}
	}
	return os.ErrNotExist
}

type mockDriver struct {
	migrate.Driver
	plan     *migrate.Plan
	changes  []schema.Change
	executed []string
}

func (m *mockDriver) ExecContext(_ context.Context, query string, _ ...interface{}) (sql.Result, error) {
	m.executed = append(m.executed, query)
	return nil, nil
}

func (m mockDriver) InspectRealm(context.Context, *schema.InspectRealmOption) (*schema.Realm, error) {
	return &schema.Realm{}, nil
}
func (m mockDriver) RealmDiff(_, _ *schema.Realm) ([]schema.Change, error) {
	return m.changes, nil
}
func (m mockDriver) PlanChanges(context.Context, string, []schema.Change) (*migrate.Plan, error) {
	return m.plan, nil
}
