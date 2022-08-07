// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	migrate2 "ariga.io/atlas/cmd/atlas/internal/migrate"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"ariga.io/atlas/sql/sqlite"
	_ "ariga.io/atlas/sql/sqlite"
	_ "ariga.io/atlas/sql/sqlite/sqlitecheck"

	"github.com/fatih/color"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {
	_, err := runCmd(Root, "migrate")
	require.NoError(t, err)
}

func TestMigrate_Apply(t *testing.T) {
	p := t.TempDir()
	// Disable text coloring in testing
	// to assert on string matching.
	color.NoColor = true

	// Fails on empty directory.
	s, err := runCmd(
		Root, "migrate", "apply",
		"--dir", "file://"+p,
		"-u", openSQLite(t, ""),
	)
	require.NoError(t, err)
	require.Equal(t, "The migration directory is synced with the database, no migration files to execute\n", s)

	// Fails on directory without sum file.
	require.NoError(t, os.Rename(
		filepath.FromSlash("testdata/sqlite/atlas.sum"),
		filepath.FromSlash("testdata/sqlite/atlas.sum.bak"),
	))
	t.Cleanup(func() {
		os.Rename(filepath.FromSlash("testdata/sqlite/atlas.sum.bak"), filepath.FromSlash("testdata/sqlite/atlas.sum"))
	})

	_, err = runCmd(
		Root, "migrate", "apply",
		"--dir", "file://testdata/sqlite",
		"--url", openSQLite(t, ""),
	)
	require.ErrorIs(t, err, migrate.ErrChecksumNotFound)
	require.NoError(t, os.Rename(
		filepath.FromSlash("testdata/sqlite/atlas.sum.bak"),
		filepath.FromSlash("testdata/sqlite/atlas.sum"),
	))

	// A lock will prevent execution.
	sqlclient.Register(
		"sqlitelockapply",
		sqlclient.OpenerFunc(func(ctx context.Context, u *url.URL) (*sqlclient.Client, error) {
			client, err := sqlclient.Open(ctx, strings.Replace(u.String(), u.Scheme, "sqlite", 1))
			if err != nil {
				return nil, err
			}
			client.Driver = &sqliteLockerDriver{client.Driver}
			return client, nil
		}),
		sqlclient.RegisterDriverOpener(func(db schema.ExecQuerier) (migrate.Driver, error) {
			drv, err := sqlite.Open(db)
			if err != nil {
				return nil, err
			}
			return &sqliteLockerDriver{drv}, nil
		}),
	)
	f, err := os.Create(filepath.Join(p, "test.db"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	s, err = runCmd(
		Root, "migrate", "apply",
		"--dir", "file://testdata/sqlite",
		"--url", fmt.Sprintf("sqlitelockapply://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")),
	)
	require.ErrorIs(t, err, errLock)
	require.True(t, strings.HasPrefix(s, "Error: sql/migrate: acquiring database lock: "+errLock.Error()))

	// Will work and print stuff to the console.
	s, err = runCmd(
		Root, "migrate", "apply",
		"--dir", "file://testdata/sqlite",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")),
		"1",
	)
	require.NoError(t, err)
	require.Contains(t, s, "20220318104614")                           // log to version
	require.Contains(t, s, "CREATE TABLE tbl (`col` int NOT NULL);")   // logs statement
	require.NotContains(t, s, "ALTER TABLE `tbl` ADD `col_2` bigint;") // does not execute second file
	require.Contains(t, s, "1 migrations")                             // logs amount of migrations
	require.Contains(t, s, "1 sql statements")

	// Transactions will be wrapped per file. If the second file has an error, first still is applied.
	s, err = runCmd(
		Root, "migrate", "apply",
		"--dir", "file://testdata/sqlite2",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test2.db")),
	)
	require.Error(t, err)
	require.Contains(t, s, "20220318104614")                           // log to version
	require.Contains(t, s, "CREATE TABLE tbl (`col` int NOT NULL);")   // logs statement
	require.Contains(t, s, "ALTER TABLE `tbl` ADD `col_2` bigint;")    // does execute first stmt first second file
	require.Contains(t, s, "ALTER TABLE `tbl` ADD `col_3` bigint;")    // does execute second stmt first second file
	require.NotContains(t, s, "ALTER TABLE `tbl` ADD `col_4` bigint;") // but not third
	require.Contains(t, s, "1 migrations ok (1 with errors)")          // logs amount of migrations
	require.Contains(t, s, "2 sql statements ok (1 with errors)")      // logs amount of statement
	require.Contains(t, s, "Error: Execution had errors:")             // logs error summary
	require.Contains(t, s, "near \"asdasd\": syntax error")            // logs error summary

	c, err := sqlclient.Open(context.Background(), fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test2.db")))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, c.Close())
	})
	sch, err := c.InspectSchema(context.Background(), "", nil)
	tbl, ok := sch.Table("tbl")
	require.True(t, ok)
	_, ok = tbl.Column("col_2")
	require.False(t, ok)
	_, ok = tbl.Column("col_3")
	require.False(t, ok)
	rrw, err := migrate2.NewEntRevisions(c)
	require.NoError(t, err)
	require.NoError(t, rrw.Init(context.Background()))
	revs, err := rrw.ReadRevisions(context.Background())
	require.NoError(t, err)
	require.Len(t, revs, 2)
	require.Equal(t, 1, revs[1].Applied)
	require.Equal(t, 3, revs[1].Total)

	// Running again will pick up the failed statement and try it again.
	s, err = runCmd(
		Root, "migrate", "apply",
		"--dir", "file://testdata/sqlite2",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test2.db")),
	)
	require.Error(t, err)
	require.NotContains(t, s, "20220318104614")                         // first version is applied already
	require.Contains(t, s, "20220318104615")                            // retry second (partially applied)
	require.NotContains(t, s, "CREATE TABLE tbl (`col` int NOT NULL);") // will not attempt stmts from first file
	require.NotContains(t, s, "ALTER TABLE `tbl` ADD `col_2` bigint;")  // does not execute first stmt first second file
	require.Contains(t, s, "ALTER TABLE `tbl` ADD `col_3` bigint;")     // does execute second stmt first second file
	require.NotContains(t, s, "ALTER TABLE `tbl` ADD `col_4` bigint;")  // but not third
	require.Contains(t, s, "0 migrations ok (1 with errors)")           // logs amount of migrations
	require.Contains(t, s, "0 sql statements ok (1 with errors)")       // logs amount of statement
	require.Contains(t, s, "Error: Execution had errors:")              // logs error summary
	require.Contains(t, s, "near \"asdasd\": syntax error")             // logs error summary

	// Editing an applied line will raise error.
	require.NoError(t, exec.Command("cp", "-r", "testdata/sqlite2", "testdata/sqlite3").Run())
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll("testdata/sqlite3"))
	})
	sed(t, "s/col_2/col_5/g", "testdata/sqlite3/20220318104615_second.sql")
	_, err = runCmd(Root, "migrate", "hash", "--force", "--dir", "file://testdata/sqlite3")
	require.NoError(t, err)
	s, err = runCmd(
		Root, "migrate", "apply",
		"--dir", "file://testdata/sqlite3",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test2.db")),
	)
	require.ErrorAs(t, err, &migrate.HistoryChangedError{})

	// Fixing the migration file will finish without errors.
	sed(t, "s/col_5/col_2/g", "testdata/sqlite3/20220318104615_second.sql")
	sed(t, "s/asdasd //g", "testdata/sqlite3/20220318104615_second.sql")
	_, err = runCmd(Root, "migrate", "hash", "--force", "--dir", "file://testdata/sqlite3")
	require.NoError(t, err)
	s, err = runCmd(
		Root, "migrate", "apply",
		"--dir", "file://testdata/sqlite3",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test2.db")),
	)
	require.NoError(t, err)
	require.Contains(t, s, "20220318104615")                        // retry second (partially applied)
	require.Contains(t, s, "ALTER TABLE `tbl` ADD `col_3` bigint;") // does execute second stmt first second file
	require.Contains(t, s, "ALTER TABLE `tbl` ADD `col_4` bigint;") // does execute second stmt first second file
	require.Contains(t, s, "1 migrations")                          // logs amount of migrations
	require.Contains(t, s, "2")                                     // logs amount of statement
	require.NotContains(t, s, "Error: Execution had errors:")       // logs error summary
	require.NotContains(t, s, "near \"asdasd\": syntax error")      // logs error summary

	// Running again will report database being in clean state.
	s, err = runCmd(
		Root, "migrate", "apply",
		"--dir", "file://testdata/sqlite3",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test2.db")),
	)
	require.NoError(t, err)
	require.Equal(t, "The migration directory is synced with the database, no migration files to execute\n", s)

	// Dry run will print the statements in second migration file without executing them. No changes to the revisions
	// will be done.
	s, err = runCmd(
		Root, "migrate", "apply",
		"--dir", "file://testdata/sqlite",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")),
		"--dry-run",
		"1",
	)
	require.NoError(t, err)
	require.Contains(t, s, "20220318104615")                        // log to version
	require.Contains(t, s, "ALTER TABLE `tbl` ADD `col_2` bigint;") // logs statement
	c1, err := sqlclient.Open(context.Background(), fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, c1.Close())
	})
	sch, err = c1.InspectSchema(context.Background(), "", nil)
	tbl, ok = sch.Table("tbl")
	require.True(t, ok)
	_, ok = tbl.Column("col_2")
	require.False(t, ok)
	rrw, err = migrate2.NewEntRevisions(c1)
	require.NoError(t, err)
	require.NoError(t, rrw.Init(context.Background()))
	revs, err = rrw.ReadRevisions(context.Background())
	require.NoError(t, err)
	require.Len(t, revs, 1)
}

func TestMigrate_ApplyBaseline(t *testing.T) {
	p := t.TempDir()
	// Run migration with baseline should store this revision in the database.
	s, err := runCmd(
		Root, "migrate", "apply",
		"--dir", "file://testdata/baseline1",
		"--baseline", "1",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")),
	)
	require.NoError(t, err)
	require.Contains(t, s, "The migration directory is synced with the database, no migration files to execute")

	// Next run without baseline should run the migration from the baseline.
	s, err = runCmd(
		Root, "migrate", "apply",
		"--dir", "file://testdata/baseline2",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")),
	)
	require.NoError(t, err)
	require.Contains(t, s, "Migrating to version 20220318104615 from 1 (2 migrations in total)")
}

func TestMigrate_Diff(t *testing.T) {
	p := t.TempDir()

	// Will create migration directory if not existing.
	MigrateFlags.Force = false
	_, err := runCmd(
		Root, "migrate", "diff",
		"name",
		"--dir", "file://"+filepath.Join(p, "migrations"),
		"--dev-url", openSQLite(t, ""),
		"--to", hclURL(t))
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, "migrations", fmt.Sprintf("%s_name.sql", time.Now().UTC().Format("20060102150405"))))

	// Expect no clean dev error.
	p = t.TempDir()
	s, err := runCmd(
		Root, "migrate", "diff",
		"name",
		"--dir", "file://"+p,
		"--dev-url", openSQLite(t, "create table t (c int);"),
		"--to", hclURL(t),
	)
	require.ErrorAs(t, err, &migrate.NotCleanError{})
	require.ErrorContains(t, err, "found table \"t\"")

	// Works (on empty directory).
	s, err = runCmd(
		Root, "migrate", "diff",
		"name",
		"--dir", "file://"+p,
		"--dev-url", openSQLite(t, ""),
		"--to", hclURL(t),
	)
	require.NoError(t, err)
	require.Zero(t, s)
	require.FileExists(t, filepath.Join(p, fmt.Sprintf("%s_name.sql", time.Now().UTC().Format("20060102150405"))))
	require.FileExists(t, filepath.Join(p, "atlas.sum"))

	// A lock will prevent diffing.
	sqlclient.Register("sqlitelockdiff", sqlclient.OpenerFunc(func(ctx context.Context, u *url.URL) (*sqlclient.Client, error) {
		client, err := sqlclient.Open(ctx, strings.Replace(u.String(), u.Scheme, "sqlite", 1))
		if err != nil {
			return nil, err
		}
		client.Driver = &sqliteLockerDriver{Driver: client.Driver}
		return client, nil
	}))
	f, err := os.Create(filepath.Join(p, "test.db"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	s, err = runCmd(
		Root, "migrate", "diff",
		"name",
		"--dir", "file://"+t.TempDir(),
		"--dev-url", fmt.Sprintf("sqlitelockdiff://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")),
		"--to", hclURL(t),
	)
	require.True(t, strings.HasPrefix(s, "Error: "+errLock.Error()))
	require.ErrorIs(t, err, errLock)
}

func TestMigrate_New(t *testing.T) {
	var (
		p = t.TempDir()
		v = time.Now().UTC().Format("20060102150405")
	)

	s, err := runCmd(Root, "migrate", "new", "--dir", "file://"+p)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, v+".sql"))
	require.FileExists(t, filepath.Join(p, "atlas.sum"))
	require.Equal(t, 2, countFiles(t, p))

	s, err = runCmd(Root, "migrate", "new", "my-migration-file", "--dir", "file://"+p)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, v+"_my-migration-file.sql"))
	require.FileExists(t, filepath.Join(p, "atlas.sum"))
	require.Equal(t, 3, countFiles(t, p))

	_, err = runCmd(Root, "migrate", "hash", "--dir", "file://"+p, "--force", "--dir-format", formatGolangMigrate)
	require.NoError(t, err)
	MigrateFlags.Force = false
	s, err = runCmd(Root, "migrate", "new", "golang-migrate", "--dir", "file://"+p, "--dir-format", formatGolangMigrate)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, v+"_golang-migrate.up.sql"))
	require.FileExists(t, filepath.Join(p, v+"_golang-migrate.down.sql"))
	require.Equal(t, 5, countFiles(t, p))

	s, err = runCmd(Root, "migrate", "hash", "--dir", "file://"+p, "--force", "--dir-format", formatGoose)
	require.NoError(t, err)
	MigrateFlags.Force = false
	s, err = runCmd(Root, "migrate", "new", "goose", "--dir", "file://"+p, "--dir-format", formatGoose)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, v+"_goose.sql"))
	require.Equal(t, 6, countFiles(t, p))

	s, err = runCmd(Root, "migrate", "new", "flyway", "--dir", "file://"+p, "--dir-format", formatFlyway)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, fmt.Sprintf("V%s__%s.sql", v, formatFlyway)))
	require.FileExists(t, filepath.Join(p, fmt.Sprintf("U%s__%s.sql", v, formatFlyway)))
	require.Equal(t, 8, countFiles(t, p))

	s, err = runCmd(Root, "migrate", "new", "liquibase", "--dir", "file://"+p, "--dir-format", formatLiquibase)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, v+"_liquibase.sql"))
	require.Equal(t, 9, countFiles(t, p))

	s, err = runCmd(Root, "migrate", "new", "dbmate", "--dir", "file://"+p, "--dir-format", formatDbmate)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, v+"_dbmate.sql"))
	require.Equal(t, 10, countFiles(t, p))

	f := filepath.Join("testdata", "mysql", "new.sql")
	require.NoError(t, os.WriteFile(f, []byte("contents"), 0600))
	t.Cleanup(func() { os.Remove(f) })
	s, err = runCmd(Root, "migrate", "new", "--dir", "file://testdata/mysql")
	require.NotZero(t, s)
	require.Error(t, err)
}

func TestMigrate_Validate(t *testing.T) {
	// Without re-playing.
	MigrateFlags.DevURL = "" // global flags are set from other tests ...
	s, err := runCmd(Root, "migrate", "validate", "--dir", "file://testdata/mysql")
	require.Zero(t, s)
	require.NoError(t, err)

	f := filepath.Join("testdata", "mysql", "new.sql")
	require.NoError(t, os.WriteFile(f, []byte("contents"), 0600))
	t.Cleanup(func() { os.Remove(f) })
	s, err = runCmd(Root, "migrate", "validate", "--dir", "file://testdata/mysql")
	require.NotZero(t, s)
	require.Error(t, err)
	require.NoError(t, os.Remove(f))

	// Replay migration files if a dev-url is given.
	p := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(p, "1_initial.sql"), []byte("create table t1 (c1 int)"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(p, "2_second.sql"), []byte("create table t2 (c2 int)"), 0644))
	_, err = runCmd(Root, "migrate", "hash", "--force", "--dir", "file://"+p)
	require.NoError(t, err)
	s, err = runCmd(
		Root, "migrate", "validate",
		"--dir", "file://"+p,
		"--dev-url", openSQLite(t, ""),
	)
	require.Zero(t, s)
	require.NoError(t, err)

	// Should fail since the files are not compatible with SQLite.
	_, err = runCmd(Root, "migrate", "validate", "--dir", "file://testdata/mysql", "--dev-url", openSQLite(t, ""))
	require.Error(t, err)
}

func TestMigrate_Hash(t *testing.T) {
	s, err := runCmd(Root, "migrate", "hash", "--dir", "file://testdata/mysql")
	require.Zero(t, s)
	require.NoError(t, err)

	p := t.TempDir()
	err = copyFile(filepath.Join("testdata", "mysql", "20220318104614_initial.sql"), filepath.Join(p, "20220318104614_initial.sql"))
	require.NoError(t, err)

	s, err = runCmd(Root, "migrate", "hash", "--dir", "file://"+p, "--force")
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, "atlas.sum"))
	d, err := ioutil.ReadFile(filepath.Join(p, "atlas.sum"))
	require.NoError(t, err)
	dir, err := migrate.NewLocalDir(p)
	require.NoError(t, err)
	sum, err := migrate.HashSum(dir)
	require.NoError(t, err)
	b, err := sum.MarshalText()
	require.NoError(t, err)
	require.Equal(t, d, b)

	p = t.TempDir()
	require.NoError(t, copyFile(
		filepath.Join("testdata", "mysql", "20220318104614_initial.sql"),
		filepath.Join(p, "20220318104614_initial.sql"),
	))
	s, err = runCmd(Root, "migrate", "hash", "--dir", "file://"+os.Getenv("MIGRATION_DIR"))
	require.NotZero(t, s)
	require.Error(t, err)
}

func TestMigrate_Lint(t *testing.T) {
	p := t.TempDir()
	s, err := runCmd(
		Root, "migrate", "lint",
		"--dir", "file://"+p,
		"--dev-url", openSQLite(t, ""),
		"--latest", "1",
	)
	require.NoError(t, err)
	require.Empty(t, s)

	err = os.WriteFile(filepath.Join(p, "1.sql"), []byte("CREATE TABLE t(c int);"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(p, "2.sql"), []byte("DROP TABLE t;"), 0600)
	require.NoError(t, err)
	s, err = runCmd(
		Root, "migrate", "lint",
		"--dir", "file://"+p,
		"--dev-url", openSQLite(t, ""),
		"--latest", "1",
	)
	require.NoError(t, err)
	require.Equal(t, "Destructive changes detected in file 2.sql:\n\n\tL1: Dropping table \"t\"\n\n", s)
	s, err = runCmd(
		Root, "migrate", "lint",
		"--dir", "file://"+p,
		"--dev-url", openSQLite(t, ""),
		"--latest", "1",
		"--log", "{{ range .Files }}{{ .Name }}{{ end }}",
	)
	require.NoError(t, err)
	require.Equal(t, "2.sql", s)

	// Change files to golang-migrate format.
	require.NoError(t, os.Rename(filepath.Join(p, "1.sql"), filepath.Join(p, "1.up.sql")))
	require.NoError(t, os.Rename(filepath.Join(p, "2.sql"), filepath.Join(p, "1.down.sql")))
	s, err = runCmd(
		Root, "migrate", "lint",
		"--dir", "file://"+p,
		"--dir-format", "golang-migrate",
		"--dev-url", openSQLite(t, ""),
		"--latest", "2",
		"--log", "{{ range .Files }}{{ .Name }}:{{ len .Reports }}{{ end }}",
	)
	require.NoError(t, err)
	require.Equal(t, "1.up.sql:0", s)
}

const testSchema = `
schema "main" {
}

table "table" {
  schema = schema.main
  column "col" {
    type    = int
    comment = "column comment"
  }
  column "age" {
    type = int
  }
  column "price1" {
    type = int
  }
  column "price2" {
    type           = int
  }
  column "account_name" {
    type = varchar(32)
    null = true
  }
  column "created_at" {
    type    = datetime
    default = sql("current_timestamp")
  }
  primary_key {
    columns = [table.table.column.col]
  }
  index "index" {
    unique  = true
    columns = [
      table.table.column.col,
      table.table.column.age,
    ]
    comment = "index comment"
  }
  foreign_key "accounts" {
    columns = [
      table.table.column.account_name,
    ]
    ref_columns = [
      table.accounts.column.name,
    ]
    on_delete = SET_NULL
    on_update = "NO_ACTION"
  }
  check "positive price" {
    expr = "price1 > 0"
  }
  check {
    expr     = "price1 <> price2"
    enforced = true
  }
  check {
    expr     = "price2 <> price1"
    enforced = false
  }
  comment        = "table comment"
}

table "accounts" {
  schema = schema.main
  column "name" {
    type = varchar(32)
  }
  column "unsigned_float" {
    type     = float(10)
    unsigned = true
  }
  column "unsigned_decimal" {
    type     = decimal(10, 2)
    unsigned = true
  }
  primary_key {
    columns = [table.accounts.column.name]
  }
}`

func hclURL(t *testing.T) string {
	p := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(p, "schema.hcl"), []byte(testSchema), 0600))
	return "file://" + filepath.Join(p, "schema.hcl")
}

func copyFile(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()
	df, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer df.Close()
	_, err = io.Copy(df, sf)
	return err
}

type sqliteLockerDriver struct{ migrate.Driver }

var errLock = errors.New("lockErr")

func (d *sqliteLockerDriver) Lock(context.Context, string, time.Duration) (schema.UnlockFunc, error) {
	return func() error { return nil }, errLock
}

func (d *sqliteLockerDriver) Snapshot(ctx context.Context) (migrate.RestoreFunc, error) {
	return d.Driver.(migrate.Snapshoter).Snapshot(ctx)
}

func (d *sqliteLockerDriver) CheckClean(ctx context.Context, revT *migrate.TableIdent) error {
	return d.Driver.(migrate.CleanChecker).CheckClean(ctx, revT)
}

func countFiles(t *testing.T, p string) int {
	files, err := os.ReadDir(p)
	require.NoError(t, err)
	return len(files)
}

func sed(t *testing.T, r, p string) {
	args := []string{"-i"}
	if runtime.GOOS == "darwin" {
		args = append(args, ".bk")
	}
	buf, err := exec.Command("sed", append(args, r, p)...).CombinedOutput()
	require.NoError(t, err, string(buf))
}
