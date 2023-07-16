// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"text/template"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/stretchr/testify/require"
)

var (
	dbs         []io.Closer
	flagVersion string
)

func TestMain(m *testing.M) {
	flag.StringVar(&flagVersion, "version", "", "[mysql56, postgres10, tidb5, ...] what version to test")
	flag.Parse()
	code := m.Run()
	for _, db := range dbs {
		db.Close()
	}
	os.Exit(code)
}

// T holds the elements common between dialect tests.
type T interface {
	testing.TB
	url(string) string
	driver() migrate.Driver
	revisionsStorage() migrate.RevisionReadWriter
	realm() *schema.Realm
	loadRealm() *schema.Realm
	users() *schema.Table
	loadUsers() *schema.Table
	posts() *schema.Table
	loadPosts() *schema.Table
	loadTable(string) *schema.Table
	dropTables(...string)
	dropSchemas(...string)
	migrate(...schema.Change)
	diff(*schema.Table, *schema.Table) []schema.Change
	applyHcl(spec string)
	applyRealmHcl(spec string)
}

func testAddDrop(t T) {
	usersT := t.users()
	postsT := t.posts()
	petsT := &schema.Table{
		Name:   "pets",
		Schema: usersT.Schema,
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
			{Name: "owner_id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}, Null: true}},
		},
	}
	petsT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: postsT.Columns[0]}}}
	petsT.ForeignKeys = []*schema.ForeignKey{
		{Symbol: "owner_id", Table: petsT, Columns: petsT.Columns[1:], RefTable: usersT, RefColumns: usersT.Columns[:1]},
	}
	if tt, ok := t.(interface {
		pets(_, _ *schema.Table) *schema.Table
	}); ok {
		petsT = tt.pets(usersT, postsT)
	}
	t.dropTables(postsT.Name, usersT.Name, petsT.Name)
	t.migrate(&schema.AddTable{T: petsT}, &schema.AddTable{T: usersT}, &schema.AddTable{T: postsT})
	ensureNoChange(t, usersT, petsT, postsT)
	t.migrate(&schema.DropTable{T: usersT}, &schema.DropTable{T: postsT}, &schema.DropTable{T: petsT})
	// Ensure the realm has no tables.
	for _, s := range t.loadRealm().Schemas {
		require.Empty(t, s.Tables)
	}
}

func testRelation(t T) {
	usersT, postsT := t.users(), t.posts()
	t.dropTables(postsT.Name, usersT.Name)
	t.migrate(
		&schema.AddTable{T: usersT},
		&schema.AddTable{T: postsT},
	)
	ensureNoChange(t, postsT, usersT)
}

func testImplicitIndexes(t T, db *sql.DB) {
	const (
		name = "implicit_indexes"
		ddl  = "create table implicit_indexes(c1 int unique, c2 int unique, unique(c1,c2), unique(c2,c1))"
	)
	t.dropTables(name)
	_, err := db.Exec(ddl)
	require.NoError(t, err)
	current := t.loadTable(name)
	c1, c2 := schema.NewNullIntColumn("c1", "int"), schema.NewNullIntColumn("c2", "int")
	desired := schema.NewTable(name).
		AddColumns(c1, c2).
		AddIndexes(
			schema.NewUniqueIndex("").AddColumns(c1),
			schema.NewUniqueIndex("").AddColumns(c2),
			schema.NewUniqueIndex("").AddColumns(c1, c2),
			schema.NewUniqueIndex("").AddColumns(c2, c1),
		)
	changes := t.diff(current, desired)
	require.Empty(t, changes)
	desired.AddIndexes(
		schema.NewIndex("c1_key").AddColumns(c1),
		schema.NewIndex("c2_key").AddColumns(c2),
	)
	changes = t.diff(current, desired)
	require.NotEmpty(t, changes)
	t.migrate(&schema.ModifyTable{T: desired, Changes: changes})
	ensureNoChange(t, desired)
}

func testHCLIntegration(t T, full string, empty string) {
	t.applyHcl(full)
	users := t.loadUsers()
	posts := t.loadPosts()
	t.dropTables(users.Name, posts.Name)
	column, ok := users.Column("id")
	require.True(t, ok, "expected id column")
	require.Equal(t, "users", users.Name)
	column, ok = posts.Column("author_id")
	require.Equal(t, "author_id", column.Name)
	t.applyHcl(empty)
	require.Empty(t, t.realm().Schemas[0].Tables)
}

func testCLIMigrateApplyBC(t T, dialect string) {
	ctx := context.Background()

	t.dropSchemas("bc_test", "bc_test_2", "atlas_schema_revisions")
	t.dropTables("bc_tbl", "atlas_schema_revisions")
	t.migrate(&schema.AddSchema{S: schema.New("bc_test")})

	// Connection to schema with flag will respect flag (also mimics "old" behavior).
	out, err := exec.Command(
		execPath(t),
		"migrate", "apply",
		"--allow-dirty", // since database does contain more than one schema
		"--dir", "file://testdata/migrations/"+dialect,
		"--url", t.url("bc_test"),
		"--revisions-schema", "atlas_schema_revisions",
	).CombinedOutput()
	require.NoError(t, err, string(out))
	s, err := t.driver().InspectSchema(ctx, "atlas_schema_revisions", &schema.InspectOptions{
		Mode: ^schema.InspectViews,
	})
	require.NoError(t, err)
	_, ok := s.Table("atlas_schema_revisions")
	require.True(t, ok)

	// Connection to realm will see the existing schema and will not attempt to migrate.
	out, err = exec.Command(
		execPath(t),
		"migrate", "apply",
		"--dir", "file://testdata/migrations/"+dialect,
		"--url", t.url(""),
	).CombinedOutput()
	require.NoError(t, err, string(out))
	require.Equal(t, "No migration files to execute\n", string(out))

	// Connection to schema without flag will error.
	out, err = exec.Command(
		execPath(t),
		"migrate", "apply",
		"--dir", "file://testdata/migrations/"+dialect,
		"--url", t.url("bc_test"),
	).CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(out), "We couldn't find a revision table in the connected schema but found one in")

	// Providing the flag and we are good.
	out, err = exec.Command(
		execPath(t),
		"migrate", "apply",
		"--dir", "file://testdata/migrations/"+dialect,
		"--url", t.url("bc_test"),
		"--revisions-schema", "atlas_schema_revisions",
	).CombinedOutput()
	require.NoError(t, err)
	require.Equal(t, "No migration files to execute\n", string(out))

	// Providing the flag to the schema instead will work as well.
	t.migrate(
		&schema.DropSchema{S: schema.New("bc_test")},
		&schema.AddSchema{S: schema.New("bc_test")},
	)
	out, err = exec.Command(
		execPath(t),
		"migrate", "apply",
		"--dir", "file://testdata/migrations/"+dialect,
		"--url", t.url("bc_test"),
		"--revisions-schema", "bc_test",
	).CombinedOutput()
	require.NoError(t, err, string(out))
	require.NotContains(t, string(out), "No migration files to execute\n")

	// Consecutive attempts do not need the flag anymore.
	out, err = exec.Command(
		execPath(t),
		"migrate", "apply",
		"--dir", "file://testdata/migrations/"+dialect,
		"--url", t.url("bc_test"),
	).CombinedOutput()
	require.NoError(t, err)
	require.Equal(t, "No migration files to execute\n", string(out))

	// Last, if bound to schema and no "old" behavior extra schema does
	// exist, the revision table will be saved in the connected one.
	t.migrate(
		&schema.DropSchema{S: schema.New("atlas_schema_revisions")},
		&schema.DropSchema{S: schema.New("bc_test")},
		&schema.AddSchema{S: schema.New("bc_test_2")},
	)
	out, err = exec.Command(
		execPath(t),
		"migrate", "apply",
		"--allow-dirty", // since database does contain more than one schema
		"--dir", "file://testdata/migrations/"+dialect,
		"--url", t.url("bc_test_2"),
	).CombinedOutput()
	require.NoError(t, err, string(out))
	s, err = t.driver().InspectSchema(ctx, "atlas_schema_revisions", &schema.InspectOptions{
		Mode: ^schema.InspectViews,
	})
	require.True(t, schema.IsNotExistError(err))
	s, err = t.driver().InspectSchema(ctx, "bc_test_2", &schema.InspectOptions{
		Mode: ^schema.InspectViews,
	})
	require.NoError(t, err)
	_, ok = s.Table("atlas_schema_revisions")
	require.True(t, ok)
}

func testCLISchemaInspect(t T, h string, url string, eval schemahcl.Evaluator, args ...string) {
	t.dropTables("users")
	var expected schema.Schema
	err := evalBytes([]byte(h), &expected, eval)
	require.NoError(t, err)
	t.applyHcl(h)
	runArgs := []string{
		"schema",
		"inspect",
		"-u",
		url,
	}
	runArgs = append(runArgs, args...)
	cmd := exec.Command(execPath(t), runArgs...)
	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	require.NoError(t, cmd.Run(), stderr.String())
	var actual schema.Schema
	err = evalBytes(stdout.Bytes(), &actual, eval)
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Equal(t, expected, actual)
}

func testCLISchemaInspectEnv(t T, h string, env string, eval schemahcl.Evaluator) {
	t.dropTables("users")
	var expected schema.Schema
	err := evalBytes([]byte(h), &expected, eval)
	require.NoError(t, err)
	t.applyHcl(h)
	cmd := exec.Command(execPath(t),
		"schema",
		"inspect",
		"--env",
		env,
	)
	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	require.NoError(t, cmd.Run(), stderr.String())
	var actual schema.Schema
	err = evalBytes(stdout.Bytes(), &actual, eval)
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Equal(t, expected, actual)
}

// initOnce controls that the cli will only be built once.
var initOnce sync.Once

func execPath(t testing.TB) string {
	initOnce.Do(func() {
		args := []string{
			"build",
			"-mod=mod",
			"-o", filepath.Join(os.TempDir(), "atlas"),
		}
		args = append(args, buildFlags...)
		args = append(args, "ariga.io/atlas/cmd/atlas")
		out, err := exec.Command("go", args...).CombinedOutput()
		require.NoError(t, err, string(out))
	})
	return filepath.Join(os.TempDir(), "atlas")
}

func testCLIMultiSchemaApply(t T, h string, url string, schemas []string, eval schemahcl.Evaluator) {
	f := filepath.Join(t.TempDir(), "schema.hcl")
	err := os.WriteFile(f, []byte(h), 0644)
	require.NoError(t, err)
	require.NoError(t, err)
	var expected schema.Realm
	err = evalBytes([]byte(h), &expected, eval)
	require.NoError(t, err)
	cmd := exec.Command(execPath(t),
		"schema",
		"apply",
		"-f",
		f,
		"-u",
		url,
		"-s",
		strings.Join(schemas, ","),
	)
	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	defer stdin.Close()
	_, err = io.WriteString(stdin, "\n")
	require.NoError(t, cmd.Run(), stderr.String())
	require.Contains(t, stdout.String(), `-- Add new schema named "test2"`)
}

func testCLIMultiSchemaInspect(t T, h string, url string, schemas []string, eval schemahcl.Evaluator) {
	var expected schema.Realm
	err := evalBytes([]byte(h), &expected, eval)
	require.NoError(t, err)
	t.applyRealmHcl(h)
	cmd := exec.Command(execPath(t),
		"schema",
		"inspect",
		"-u",
		url,
		"-s",
		strings.Join(schemas, ","),
	)
	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	require.NoError(t, cmd.Run(), stderr.String())
	var actual schema.Realm
	err = evalBytes(stdout.Bytes(), &actual, eval)
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Equal(t, expected, actual)
}

func testCLISchemaApply(t T, h string, url string, args ...string) {
	t.dropTables("users")
	f := filepath.Join(t.TempDir(), "schema.hcl")
	err := os.WriteFile(f, []byte(h), 0644)
	require.NoError(t, err)
	runArgs := []string{
		"schema",
		"apply",
		"-u",
		url,
		"-f",
		f,
		"--dev-url",
		url,
	}
	runArgs = append(runArgs, args...)
	cmd := exec.Command(execPath(t), runArgs...)
	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	defer stdin.Close()
	_, err = io.WriteString(stdin, "\n")
	require.NoError(t, err)
	require.NoError(t, cmd.Run(), stderr.String(), stdout.String())
	require.Empty(t, stderr.String(), stderr.String())
	require.Contains(t, stdout.String(), "-- Planned")
	u := t.loadUsers()
	require.NotNil(t, u)
}

func testCLISchemaApplyDry(t T, h string, url string) {
	t.dropTables("users")
	f := filepath.Join(t.TempDir(), "schema.hcl")
	err := os.WriteFile(f, []byte(h), 0644)
	require.NoError(t, err)
	cmd := exec.Command(execPath(t),
		"schema",
		"apply",
		"-u",
		url,
		"-f",
		f,
		"--dry-run",
	)
	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	defer stdin.Close()
	_, err = io.WriteString(stdin, "\n")
	require.NoError(t, err)
	require.NoError(t, cmd.Run(), stderr.String(), stdout.String())
	require.Empty(t, stderr.String(), stderr.String())
	require.Contains(t, stdout.String(), "-- Planned")
	require.NotContains(t, stdout.String(), "Are you sure?", "dry run should not prompt")
	realm := t.loadRealm()
	_, ok := realm.Schemas[0].Table("users")
	require.False(t, ok, "expected users table not to be created")
}

func testCLISchemaApplyAutoApprove(t T, h string, url string, args ...string) {
	t.dropTables("users")
	f := filepath.Join(t.TempDir(), "schema.hcl")
	err := os.WriteFile(f, []byte(h), 0644)
	require.NoError(t, err)
	runArgs := []string{
		"schema",
		"apply",
		"-u",
		url,
		"-f",
		f,
		"--auto-approve",
	}
	runArgs = append(runArgs, args...)
	cmd := exec.Command(execPath(t), runArgs...)
	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	require.NoError(t, err)
	require.NoError(t, cmd.Run(), stderr.String(), stdout.String())
	require.Empty(t, stderr.String(), stderr.String())
	require.Contains(t, stdout.String(), "-- Planned")
	u := t.loadUsers()
	require.NotNil(t, u)
}

func testCLISchemaApplyFromMigrationDir(t T) {
	const (
		dbname  = "apply_migration_dir"
		devname = dbname + "_dev"
	)
	t.dropSchemas(dbname, devname)
	t.migrate(&schema.AddSchema{S: schema.New(dbname)}, &schema.AddSchema{S: schema.New(devname)})
	defer t.migrate(&schema.DropSchema{S: schema.New(dbname)}, &schema.DropSchema{S: schema.New(devname)})

	users, err := t.driver().SchemaDiff(schema.New(""), schema.New("").AddTables(t.users()))
	require.NoError(t, err)
	usersT := t.users()
	usersT.Name = "users_2"
	users2, err := t.driver().SchemaDiff(schema.New(""), schema.New("").AddTables(usersT))
	require.NoError(t, err)

	addUsers, err := t.driver().PlanChanges(context.Background(), "", users)
	require.NoError(t, err)
	addUsers2, err := t.driver().PlanChanges(context.Background(), "", users2)
	require.NoError(t, err)

	var (
		fn = func(c []*migrate.Change) string {
			var buf strings.Builder
			for _, c := range c {
				buf.WriteString(c.Cmd)
				buf.WriteString(";\n")
			}
			return buf.String()
		}
		one = fn(addUsers.Changes)
		two = fn(addUsers2.Changes)
	)

	p := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(p, "1.sql"), []byte(one), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(p, "2.sql"), []byte(two), 0644))

	require.NoError(t, exec.Command(
		execPath(t),
		"migrate", "hash",
		"--dir", "file://"+p,
	).Run())

	// All versions - must contain all migration files.
	out, err := exec.Command(
		execPath(t),
		"schema", "apply",
		"-u", t.url(dbname),
		"--to", "file://"+p,
		"--dev-url", t.url(devname),
		"--dry-run",
	).CombinedOutput()
	require.NoError(t, err)
	require.Contains(t, string(out), one)
	require.Contains(t, string(out), two)

	// One version - must contain only file one.
	out, err = exec.Command(
		execPath(t),
		"schema", "apply",
		"-u", t.url(dbname),
		"--to", "file://"+p+"?version=1",
		"--dev-url", t.url(devname),
		"--dry-run",
	).CombinedOutput()
	require.NoError(t, err)
	require.Contains(t, string(out), one)
	require.NotContains(t, string(out), two)
}

func testCLISchemaDiff(t T, url string) {
	t.dropTables("users")
	cmd := exec.Command(execPath(t),
		"schema",
		"diff",
		"--from",
		url,
		"--to",
		url,
	)
	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	require.NoError(t, cmd.Run(), stderr.String(), stdout.String())
	require.Empty(t, stderr.String(), stderr.String())
	require.Contains(t, stdout.String(), "Schemas are synced, no changes to be made.")
}

func ensureNoChange(t T, tables ...*schema.Table) {
	realm := t.loadRealm()
	require.Equal(t, len(realm.Schemas[0].Tables), len(tables))
	for i := range tables {
		tt, ok := realm.Schemas[0].Table(tables[i].Name)
		require.True(t, ok)
		changes := t.diff(tt, tables[i])
		require.Emptyf(t, changes, "changes should be empty for table %s, but instead was %#v", tt.Name, changes)
	}
}

func testAdvisoryLock(t *testing.T, l schema.Locker) {
	t.Run("One", func(t *testing.T) {
		unlock, err := l.Lock(context.Background(), "migrate", 0)
		require.NoError(t, err)
		_, err = l.Lock(context.Background(), "migrate", 0)
		require.Equal(t, schema.ErrLocked, err)
		require.NoError(t, unlock())
	})
	t.Run("Multi", func(t *testing.T) {
		var unlocks []schema.UnlockFunc
		for _, name := range []string{"a", "b", "c"} {
			unlock, err := l.Lock(context.Background(), name, 0)
			require.NoError(t, err)
			unlocks = append(unlocks, unlock)
		}
		for _, unlock := range unlocks {
			require.NoError(t, unlock())
		}
	})
}

func testExecutor(t T) {
	usersT, postsT := t.users(), t.posts()
	petsT := &schema.Table{
		Name:   "pets",
		Schema: usersT.Schema,
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
			{Name: "owner_id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}, Null: true}},
		},
	}
	petsT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: postsT.Columns[0]}}}
	petsT.ForeignKeys = []*schema.ForeignKey{
		{Symbol: "owner_id", Table: petsT, Columns: petsT.Columns[1:], RefTable: usersT, RefColumns: usersT.Columns[:1]},
	}
	if tt, ok := t.(interface {
		pets(_, _ *schema.Table) *schema.Table
	}); ok {
		petsT = tt.pets(usersT, postsT)
	}
	t.dropTables(petsT.Name, postsT.Name, usersT.Name)
	t.Cleanup(func() {
		t.revisionsStorage().(*rrw).clean()
	})

	dir, err := migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)
	f, err := migrate.NewTemplateFormatter(
		template.Must(template.New("").Parse("{{ .Name }}.sql")),
		template.Must(template.New("").Parse(
			`{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ printf "%s;\n" .Cmd }}{{ end }}`,
		)),
	)
	require.NoError(t, err)
	pl := migrate.NewPlanner(t.driver(), dir, migrate.PlanFormat(f))
	require.NoError(t, err)

	require.NoError(t, pl.WritePlan(plan(t, "1_users", &schema.AddTable{T: usersT})))
	require.NoError(t, pl.WritePlan(plan(t, "2_posts", &schema.AddTable{T: postsT})))
	require.NoError(t, pl.WritePlan(plan(t, "3_pets", &schema.AddTable{T: petsT})))

	ex, err := migrate.NewExecutor(t.driver(), dir, t.revisionsStorage())
	require.NoError(t, err)
	require.NoError(t, ex.ExecuteN(context.Background(), 2)) // usersT and postsT
	require.Len(t, *t.revisionsStorage().(*rrw), 2)
	ensureNoChange(t, postsT, usersT)
	require.NoError(t, ex.ExecuteN(context.Background(), 1)) // petsT
	require.Len(t, *t.revisionsStorage().(*rrw), 3)
	ensureNoChange(t, petsT, postsT, usersT)

	require.ErrorIs(t, ex.ExecuteN(context.Background(), 1), migrate.ErrNoPendingFiles)
}

func plan(t T, name string, changes ...schema.Change) *migrate.Plan {
	p, err := t.driver().PlanChanges(context.Background(), name, changes)
	require.NoError(t, err)
	return p
}

type rrw []*migrate.Revision

func (r *rrw) Ident() *migrate.TableIdent {
	return &migrate.TableIdent{}
}

func (r *rrw) WriteRevision(_ context.Context, rev *migrate.Revision) error {
	for i, rev2 := range *r {
		if rev2.Version == rev.Version {
			(*r)[i] = rev
			return nil
		}
	}
	*r = append(*r, rev)
	return nil
}

func (r *rrw) ReadRevision(_ context.Context, v string) (*migrate.Revision, error) {
	for _, rev := range *r {
		if rev.Version == v {
			return rev, nil
		}
	}
	return nil, migrate.ErrRevisionNotExist
}

func (r *rrw) DeleteRevision(_ context.Context, v string) error {
	i := -1
	for j, r := range *r {
		if r.Version == v {
			i = j
			break
		}
	}
	if i == -1 {
		return nil
	}
	copy((*r)[i:], (*r)[i+1:])
	*r = (*r)[:len(*r)-1]
	return nil
}

func (r *rrw) ReadRevisions(context.Context) ([]*migrate.Revision, error) {
	return *r, nil
}

func (r *rrw) clean() {
	*r = []*migrate.Revision{}
}

var (
	buildFlags []string
	_          migrate.RevisionReadWriter = (*rrw)(nil)
	buildOnce  sync.Once
)

func cliPath(t *testing.T) string {
	path := filepath.Join(os.TempDir(), "atlas")
	buildOnce.Do(func() {
		args := append([]string{"build"}, buildFlags...)
		args = append(args, "-o", path, "ariga.io/atlas/cmd/atlas")
		out, err := exec.Command("go", args...).CombinedOutput()
		require.NoError(t, err, string(out))
	})
	return path
}

func evalBytes(b []byte, v any, ev schemahcl.Evaluator) error {
	p := hclparse.NewParser()
	if _, diag := p.ParseHCL(b, ""); diag.HasErrors() {
		return diag
	}
	return ev.Eval(p, v, nil)
}
