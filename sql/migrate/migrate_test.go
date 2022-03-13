// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate_test

import (
	"context"
	"database/sql"
	_ "embed"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"text/template"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestPlanner_WritePlan(t *testing.T) {
	p := t.TempDir()
	d, err := migrate.NewLocalDir(p)
	require.NoError(t, err)
	plan := &migrate.Plan{
		Name: "add_t1_and_t2",
		Changes: []*migrate.Change{
			{Cmd: "CREATE TABLE t1(c int);", Reverse: "DROP TABLE t1 IF EXISTS"},
			{Cmd: "CREATE TABLE t2(c int)", Reverse: "DROP TABLE t2"},
		},
	}

	// DefaultFormatter
	pl := migrate.NewPlanner(nil, d, migrate.DisableChecksum())
	require.NotNil(t, pl)
	require.NoError(t, pl.WritePlan(plan))
	v := strconv.FormatInt(time.Now().Unix(), 10)
	require.Equal(t, countFiles(t, d), 2)
	requireFileEqual(t, filepath.Join(p, v+"_add_t1_and_t2.up.sql"), "CREATE TABLE t1(c int);\nCREATE TABLE t2(c int);\n")
	requireFileEqual(t, filepath.Join(p, v+"_add_t1_and_t2.down.sql"), "DROP TABLE t2;\nDROP TABLE t1 IF EXISTS;\n")

	// Custom formatter (creates only "up" migration files).
	fmt, err := migrate.NewTemplateFormatter(
		template.Must(template.New("").Parse("{{.Name}}.sql")),
		template.Must(template.New("").Parse("{{range .Changes}}{{println .Cmd}}{{end}}")),
	)
	require.NoError(t, err)
	pl = migrate.NewPlanner(nil, d, migrate.WithFormatter(fmt), migrate.DisableChecksum())
	require.NotNil(t, pl)
	require.NoError(t, pl.WritePlan(plan))
	require.Equal(t, countFiles(t, d), 3)
	requireFileEqual(t, filepath.Join(p, "add_t1_and_t2.sql"), "CREATE TABLE t1(c int);\nCREATE TABLE t2(c int)\n")
}

func TestPlanner_Plan(t *testing.T) {
	var (
		drv = &mockDriver{}
		ctx = context.Background()
	)
	d, err := migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)

	// nothing to do
	pl := migrate.NewPlanner(drv, d)
	plan, err := pl.Plan(ctx, "empty", migrate.Realm(nil))
	require.ErrorIs(t, err, migrate.ErrNoPlan)
	require.Nil(t, plan)

	// there are changes
	drv.changes = []schema.Change{
		&schema.AddTable{T: schema.NewTable("t1").AddColumns(schema.NewIntColumn("c", "int"))},
		&schema.AddTable{T: schema.NewTable("t2").AddColumns(schema.NewIntColumn("c", "int"))},
	}
	drv.plan = &migrate.Plan{
		Changes: []*migrate.Change{
			{Cmd: "CREATE TABLE t1(c int);"},
			{Cmd: "CREATE TABLE t2(c int);"},
		},
	}
	plan, err = pl.Plan(ctx, "", migrate.Realm(nil))
	require.NoError(t, err)
	require.Equal(t, drv.plan, plan)
}

func TestHash(t *testing.T) {
	p := t.TempDir()
	d, err := migrate.NewLocalDir(p)
	require.NoError(t, err)
	plan := &migrate.Plan{Name: "plan", Changes: []*migrate.Change{{Cmd: "cmd", Reverse: "rev"}}}
	pl := migrate.NewPlanner(nil, d)
	require.NotNil(t, pl)
	require.NoError(t, pl.WritePlan(plan))
	v := strconv.FormatInt(time.Now().Unix(), 10)
	require.Equal(t, countFiles(t, d), 3)
	requireFileEqual(t, filepath.Join(p, v+"_plan.up.sql"), "cmd;\n")
	requireFileEqual(t, filepath.Join(p, v+"_plan.down.sql"), "rev;\n")
	require.FileExists(t, filepath.Join(p, "atlas.sum"))

	p = t.TempDir()
	d, err = migrate.NewLocalDir(p)
	require.NoError(t, err)
	pl = migrate.NewPlanner(nil, d, migrate.DisableChecksum())
	require.NotNil(t, pl)
	require.NoError(t, pl.WritePlan(plan))
	require.Equal(t, countFiles(t, d), 2)
	requireFileEqual(t, filepath.Join(p, v+"_plan.up.sql"), "cmd;\n")
	requireFileEqual(t, filepath.Join(p, v+"_plan.down.sql"), "rev;\n")
}

func TestValidate(t *testing.T) {
	d, err := migrate.NewLocalDir("testdata")
	require.NoError(t, err)
	require.Nil(t, migrate.Validate(d))

	p := t.TempDir()
	d, err = migrate.NewLocalDir(p)
	require.NoError(t, err)
	require.NoError(t, d.WriteFile("atlas.sum", hash))
	require.Equal(t, migrate.ErrChecksumMismatch, migrate.Validate(d))
}

//go:embed testdata/atlas.sum
var hash []byte

func TestHash_MarshalText(t *testing.T) {
	d, err := migrate.NewLocalDir("testdata")
	require.NoError(t, err)
	h, err := migrate.HashSum(d)
	require.NoError(t, err)
	ac, err := h.MarshalText()
	require.Equal(t, hash, ac)
}

func TestHash_UnmarshalText(t *testing.T) {
	d, err := migrate.NewLocalDir("testdata")
	require.NoError(t, err)
	h, err := migrate.HashSum(d)
	require.NoError(t, err)
	var ac migrate.HashFile
	require.NoError(t, ac.UnmarshalText(hash))
	require.Equal(t, h, ac)
}

func TestGlobStateReader(t *testing.T) {
	var (
		drv = &mockDriver{}
		ctx = context.Background()
	)
	localFS, err := migrate.NewLocalDir("testdata")
	require.NoError(t, err)

	_, err = migrate.GlobStateReader(localFS, drv, "*.up.sql").ReadState(ctx)
	require.NoError(t, err)
	require.Equal(t, drv.executed, []string{"CREATE TABLE t(c int);"})

	_, err = migrate.GlobStateReader(localFS, drv, "*.down.sql").ReadState(ctx)
	require.NoError(t, err)
	require.Equal(t, drv.executed, []string{"CREATE TABLE t(c int);", "DROP TABLE IF EXISTS t;"})
}

func TestLocalDir(t *testing.T) {
	d, err := migrate.NewLocalDir("migrate.go")
	require.ErrorContains(t, err, "sql/migrate: \"migrate.go\" is not a dir")
	require.Nil(t, d)

	d, err = migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)
	require.NotNil(t, d)
	require.NoError(t, d.WriteFile("name", []byte("content")))
	f, err := d.Open("name")
	require.NoError(t, err)
	i, err := f.Stat()
	require.NoError(t, err)
	require.Equal(t, i.Name(), "name")
	c, err := io.ReadAll(f)
	require.NoError(t, err)
	require.Equal(t, "content", string(c))
}

type mockHashFS struct{ hash []byte }

func (h *mockHashFS) WriteHash(b []byte) error {
	h.hash = b
	return nil
}

func (h *mockHashFS) ReadHash() ([]byte, error) {
	return h.hash, nil
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

func countFiles(t *testing.T, d migrate.Dir) int {
	files, err := fs.ReadDir(d, "")
	require.NoError(t, err)
	return len(files)
}

func requireFileEqual(t *testing.T, name, contents string) {
	c, err := os.ReadFile(name)
	require.NoError(t, err)
	require.Equal(t, contents, string(c))
}
