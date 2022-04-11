// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate_test

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
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
			{Cmd: "CREATE TABLE t1(c int)", Reverse: "DROP TABLE t1 IF EXISTS"},
			{Cmd: "CREATE TABLE t2(c int)", Reverse: "DROP TABLE t2"},
		},
	}

	// DefaultFormatter
	pl := migrate.NewPlanner(nil, d, migrate.DisableChecksum())
	require.NotNil(t, pl)
	require.NoError(t, pl.WritePlan(plan))
	v := time.Now().Format("20060102150405")
	require.Equal(t, countFiles(t, d), 1)
	requireFileEqual(t, d, v+"_add_t1_and_t2.sql", "CREATE TABLE t1(c int);\nCREATE TABLE t2(c int);\n")

	// Custom formatter (creates "up" and "down" migration files).
	fmt, err := migrate.NewTemplateFormatter(
		template.Must(template.New("").Parse("{{ .Name }}.up.sql")),
		template.Must(template.New("").Parse("{{ range .Changes }}{{ println .Cmd }}{{ end }}")),
		template.Must(template.New("").Parse("{{ .Name }}.down.sql")),
		template.Must(template.New("").Parse("{{ range .Changes }}{{ println .Reverse }}{{ end }}")),
	)
	require.NoError(t, err)
	pl = migrate.NewPlanner(nil, d, migrate.WithFormatter(fmt), migrate.DisableChecksum())
	require.NotNil(t, pl)
	require.NoError(t, pl.WritePlan(plan))
	require.Equal(t, countFiles(t, d), 3)
	requireFileEqual(t, d, "add_t1_and_t2.up.sql", "CREATE TABLE t1(c int)\nCREATE TABLE t2(c int)\n")
	requireFileEqual(t, d, "add_t1_and_t2.down.sql", "DROP TABLE t1 IF EXISTS\nDROP TABLE t2\n")
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

func TestHashSum(t *testing.T) {
	p := t.TempDir()
	d, err := migrate.NewLocalDir(p)
	require.NoError(t, err)
	plan := &migrate.Plan{Name: "plan", Changes: []*migrate.Change{{Cmd: "cmd"}}}
	pl := migrate.NewPlanner(nil, d)
	require.NotNil(t, pl)
	require.NoError(t, pl.WritePlan(plan))
	v := time.Now().Format("20060102150405")
	require.Equal(t, 2, countFiles(t, d))
	requireFileEqual(t, d, v+"_plan.sql", "cmd;\n")
	require.FileExists(t, filepath.Join(p, "atlas.sum"))

	p = t.TempDir()
	d, err = migrate.NewLocalDir(p)
	require.NoError(t, err)
	pl = migrate.NewPlanner(nil, d, migrate.DisableChecksum())
	require.NotNil(t, pl)
	require.NoError(t, pl.WritePlan(plan))
	require.Equal(t, 1, countFiles(t, d))
	requireFileEqual(t, d, v+"_plan.sql", "cmd;\n")
}

func TestValidate(t *testing.T) {
	// Add the sum file form the testdata dir without any files in it - should fail.
	p := t.TempDir()
	d, err := migrate.NewLocalDir(p)
	require.NoError(t, err)
	require.NoError(t, d.WriteFile("atlas.sum", hash))
	require.Equal(t, migrate.ErrChecksumMismatch, migrate.Validate(d))

	td := "testdata"
	d, err = migrate.NewLocalDir(td)
	t.Cleanup(func() {
		os.WriteFile(filepath.Join(td, "1_initial.up.sql"), upFile, 0644) //nolint:gosec
	})

	// Testdata is valid.
	require.NoError(t, err)
	require.Nil(t, migrate.Validate(d))
	require.NoError(t, err)

	// Making a manual change to the sum file should raise validation error.
	f, err := os.OpenFile(filepath.Join(td, "atlas.sum"), os.O_RDWR, os.ModeAppend)
	require.NoError(t, err)
	_, err = f.WriteString("foo")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	t.Cleanup(func() {
		os.WriteFile(filepath.Join(td, "atlas.sum"), hash, 0644) //nolint:gosec
	})
	require.Equal(t, migrate.ErrChecksumMismatch, migrate.Validate(d))
	os.WriteFile(filepath.Join(td, "atlas.sum"), hash, 0644) //nolint:gosec
	f, err = os.OpenFile(filepath.Join(td, "atlas.sum"), os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	require.NoError(t, err)
	_, err = f.WriteString("foo")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.Equal(t, migrate.ErrChecksumFormat, migrate.Validate(d))
	os.WriteFile(filepath.Join(td, "atlas.sum"), hash, 0644) //nolint:gosec

	// Changing the filename should raise validation error.
	require.NoError(t, os.Rename(filepath.Join(td, "1_initial.up.sql"), filepath.Join(td, "1_first.up.sql")))
	t.Cleanup(func() {
		os.Remove(filepath.Join(td, "1_first.up.sql")) //nolint:gosec
	})
	require.Equal(t, migrate.ErrChecksumMismatch, migrate.Validate(d))

	// Removing it as well (move it out of the dir).
	require.NoError(t, os.Remove(filepath.Join(td, "1_first.up.sql")))
	require.Equal(t, migrate.ErrChecksumMismatch, migrate.Validate(d))
}

var (
	//go:embed testdata/atlas.sum
	hash []byte
	//go:embed testdata/1_initial.up.sql
	upFile []byte
)

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
	d, err := migrate.NewLocalDir("testdata")
	require.NoError(t, err)

	_, err = migrate.GlobStateReader(d, drv, "*.up.sql").ReadState(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"CREATE TABLE t(c int);"}, drv.executed)
	require.Equal(t, 1, drv.lockCounter)
	require.Equal(t, 1, drv.unlockCounter)

	_, err = migrate.GlobStateReader(d, drv, "*.down.sql").ReadState(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"CREATE TABLE t(c int);", "DROP TABLE IF EXISTS t;"}, drv.executed)
	require.Equal(t, 2, drv.lockCounter)
	require.Equal(t, 2, drv.unlockCounter)

	drv.locked = true
	_, err = migrate.GlobStateReader(d, drv, "").ReadState(ctx)
	require.EqualError(t, err, "sql/migrate: acquiring database lock: lockErr")
	require.Equal(t, 2, drv.lockCounter)
	require.Equal(t, 2, drv.unlockCounter)
	drv.locked = false

	drv.realm = schema.Realm{Schemas: []*schema.Schema{{Name: "schema"}}}
	_, err = migrate.GlobStateReader(d, drv, "*.up.sql").ReadState(ctx)
	require.EqualError(t, err, "sql/migrate: connected database is not clean")
	require.Equal(t, 2, drv.lockCounter)
	require.Equal(t, 2, drv.unlockCounter)

	edrv := &emptyMockDriver{drv}
	_, err = migrate.GlobStateReader(d, edrv, "*.up.sql").ReadState(ctx)
	require.NoError(t, err)
	require.Equal(t, []schema.Change{
		&schema.DropSchema{
			S:     &schema.Schema{Name: "schema"},
			Extra: []schema.Clause{&schema.IfExists{}},
		},
	}, edrv.applied)
	require.Equal(t, 3, drv.lockCounter)
	require.Equal(t, 3, drv.unlockCounter)
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

type (
	mockDriver struct {
		migrate.Driver
		plan          *migrate.Plan
		changes       []schema.Change
		applied       []schema.Change
		realm         schema.Realm
		executed      []string
		locked        bool
		lockCounter   int
		unlockCounter int
	}
	emptyMockDriver struct{ *mockDriver }
)

func (m *mockDriver) ExecContext(_ context.Context, query string, _ ...interface{}) (sql.Result, error) {
	m.executed = append(m.executed, query)
	return nil, nil
}

func (m *mockDriver) InspectRealm(context.Context, *schema.InspectRealmOption) (*schema.Realm, error) {
	return &m.realm, nil
}
func (m *mockDriver) RealmDiff(_, _ *schema.Realm) ([]schema.Change, error) {
	return m.changes, nil
}
func (m *mockDriver) PlanChanges(context.Context, string, []schema.Change) (*migrate.Plan, error) {
	return m.plan, nil
}
func (m *mockDriver) ApplyChanges(_ context.Context, changes []schema.Change) error {
	m.applied = changes
	return nil
}
func (m *mockDriver) Lock(context.Context, string, time.Duration) (schema.UnlockFunc, error) {
	if m.locked {
		return nil, errors.New("lockErr")
	}
	m.lockCounter++
	m.locked = true
	return func() error {
		m.unlockCounter++
		m.locked = false
		return nil
	}, nil
}
func (m *emptyMockDriver) IsClean(context.Context) (bool, error) {
	return true, nil
}

func countFiles(t *testing.T, d migrate.Dir) int {
	files, err := fs.ReadDir(d, "")
	require.NoError(t, err)
	return len(files)
}

func requireFileEqual(t *testing.T, d migrate.Dir, name, contents string) {
	c, err := fs.ReadFile(d, name)
	require.NoError(t, err)
	require.Equal(t, contents, string(c))
}
