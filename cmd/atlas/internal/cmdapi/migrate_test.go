// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/cmdlog"
	migrate2 "ariga.io/atlas/cmd/atlas/internal/migrate"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"ariga.io/atlas/sql/sqlite"
	_ "ariga.io/atlas/sql/sqlite"
	_ "ariga.io/atlas/sql/sqlite/sqlitecheck"

	"github.com/fatih/color"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {
	_, err := runCmd(migrateCmd())
	require.NoError(t, err)
}

func TestMigrate_Import(t *testing.T) {
	for _, tool := range []string{"dbmate", "flyway", "golang-migrate", "goose", "liquibase"} {
		p := t.TempDir()
		t.Run(tool, func(t *testing.T) { // remove this once --dir-format is removed. Test is kept to ensure BC.
			path := filepath.FromSlash("testdata/import/" + tool)
			out, err := runCmd(
				migrateImportCmd(),
				"--from", "file://"+path,
				"--to", "file://"+p,
				"--dir-format", tool,
			)
			require.NoError(t, err)
			require.Zero(t, out)

			path += "_gold"
			ex, err := os.ReadDir(path)
			require.NoError(t, err)
			ac, err := os.ReadDir(p)
			require.NoError(t, err)
			require.Equal(t, len(ex)+1, len(ac)) // sum file

			for i := range ex {
				e, err := os.ReadFile(filepath.Join(path, ex[i].Name()))
				require.NoError(t, err)
				a, err := os.ReadFile(filepath.Join(p, ex[i].Name()))
				require.NoError(t, err)
				require.Equal(t, string(e), string(a))
			}
		})
		p = t.TempDir()
		t.Run(tool, func(t *testing.T) {
			path := filepath.FromSlash("testdata/import/" + tool)
			out, err := runCmd(
				migrateImportCmd(),
				"--from", fmt.Sprintf("file://%s?format=%s", path, tool),
				"--to", "file://"+p,
			)
			require.NoError(t, err)
			require.Zero(t, out)

			path += "_gold"
			ex, err := os.ReadDir(path)
			require.NoError(t, err)
			ac, err := os.ReadDir(p)
			require.NoError(t, err)
			require.Equal(t, len(ex)+1, len(ac)) // sum file

			for i := range ex {
				e, err := os.ReadFile(filepath.Join(path, ex[i].Name()))
				require.NoError(t, err)
				a, err := os.ReadFile(filepath.Join(p, ex[i].Name()))
				require.NoError(t, err)
				require.Equal(t, string(e), string(a))
			}
		})
	}
}

func TestMigrate_Apply(t *testing.T) {
	var (
		p   = t.TempDir()
		ctx = context.Background()
	)
	// Disable text coloring in testing
	// to assert on string matching.
	color.NoColor = true

	// Fails on empty directory.
	s, err := runCmd(
		migrateApplyCmd(),
		"--dir", "file://"+p,
		"-u", openSQLite(t, ""),
	)
	require.NoError(t, err)
	require.Equal(t, "No migration files to execute\n", s)

	// Fails on directory without sum file.
	require.NoError(t, os.Rename(
		filepath.FromSlash("testdata/sqlite/atlas.sum"),
		filepath.FromSlash("testdata/sqlite/atlas.sum.bak"),
	))
	t.Cleanup(func() {
		os.Rename(filepath.FromSlash("testdata/sqlite/atlas.sum.bak"), filepath.FromSlash("testdata/sqlite/atlas.sum"))
	})

	_, err = runCmd(
		migrateApplyCmd(),
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
		migrateApplyCmd(),
		"--dir", "file://testdata/sqlite",
		"--url", fmt.Sprintf("sqlitelockapply://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")),
	)
	require.ErrorIs(t, err, errLock)
	require.True(t, strings.HasPrefix(s, "Error: acquiring database lock: "+errLock.Error()))

	// Apply zero throws error.
	for _, n := range []string{"-1", "0"} {
		_, err = runCmd(
			migrateApplyCmd(),
			"--dir", "file://testdata/sqlite",
			"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")),
			"--", n,
		)
		require.EqualError(t, err, fmt.Sprintf("cannot apply '%s' migration files", n))
	}

	// Will work and print stuff to the console.
	s, err = runCmd(
		migrateApplyCmd(),
		"--dir", "file://testdata/sqlite",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")),
		"1",
	)
	require.NoError(t, err)
	require.Contains(t, s, "20220318104614")                           // log to version
	require.Contains(t, s, "CREATE TABLE tbl (`col` int NOT NULL);")   // logs statement
	require.NotContains(t, s, "ALTER TABLE `tbl` ADD `col_2` bigint;") // does not execute second file
	require.Contains(t, s, "1 migration")                              // logs amount of migrations
	require.Contains(t, s, "1 sql statement")

	// Transactions will be wrapped per file. If the second file has an error, first still is applied.
	s, err = runCmd(
		migrateApplyCmd(),
		"--dir", "file://testdata/sqlite2",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test2.db")),
	)
	require.Error(t, err)
	require.Contains(t, s, "20220318104614")                           // log to version
	require.Contains(t, s, "CREATE TABLE tbl (`col` int NOT NULL);")   // logs statement
	require.Contains(t, s, "ALTER TABLE `tbl` ADD `col_2` bigint;")    // does execute first stmt first second file
	require.Contains(t, s, "ALTER TABLE `tbl` ADD `col_3` bigint;")    // does execute second stmt first second file
	require.NotContains(t, s, "ALTER TABLE `tbl` ADD `col_4` bigint;") // but not third
	require.Contains(t, s, "-- 1 migration ok, 1 with errors")         // logs amount of migrations
	require.Contains(t, s, "-- 2 sql statements ok, 1 with errors")    // logs amount of statement
	require.Contains(t, s, "near \"asdasd\": syntax error")            // logs error summary

	c, err := sqlclient.Open(ctx, fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test2.db")))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, c.Close())
	})
	sch, err := c.InspectSchema(ctx, "", nil)
	tbl, ok := sch.Table("tbl")
	require.True(t, ok)
	_, ok = tbl.Column("col_2")
	require.False(t, ok)
	_, ok = tbl.Column("col_3")
	require.False(t, ok)
	rrw, err := migrate2.NewEntRevisions(ctx, c)
	require.NoError(t, err)
	revs, err := rrw.ReadRevisions(ctx)
	require.NoError(t, err)
	require.Len(t, revs, 1)

	// Running again will pick up the failed statement and try it again.
	s, err = runCmd(
		migrateApplyCmd(),
		"--dir", "file://testdata/sqlite2",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test2.db")),
	)
	require.Error(t, err)
	require.Contains(t, s, "20220318104614")                            // currently applied version
	require.Contains(t, s, "20220318104615")                            // retry second (partially applied)
	require.NotContains(t, s, "CREATE TABLE tbl (`col` int NOT NULL);") // will not attempt stmts from first file
	require.Contains(t, s, "ALTER TABLE `tbl` ADD `col_2` bigint;")     // picks up first statement
	require.Contains(t, s, "ALTER TABLE `tbl` ADD `col_3` bigint;")     // does execute second stmt first second file
	require.NotContains(t, s, "ALTER TABLE `tbl` ADD `col_4` bigint;")  // but not third
	require.Contains(t, s, "-- 1 migration with errors")                // logs amount of migrations
	require.Contains(t, s, "-- 1 sql statement ok, 1 with errors")      // logs amount of statement
	require.Contains(t, s, "near \"asdasd\": syntax error")             // logs error summary

	// Editing an applied line will raise error.
	s, err = runCmd(
		migrateApplyCmd(),
		"--dir", "file://testdata/sqlite2",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test2.db")),
		"--tx-mode", "none",
	)
	t.Cleanup(func() {
		_ = os.RemoveAll("testdata/sqlite3")
	})
	require.NoError(t, exec.Command("cp", "-r", "testdata/sqlite2", "testdata/sqlite3").Run())
	sed(t, "s/col_2/col_5/g", "testdata/sqlite3/20220318104615_second.sql")
	_, err = runCmd(migrateHashCmd(), "--dir", "file://testdata/sqlite3")
	require.NoError(t, err)
	s, err = runCmd(
		migrateApplyCmd(),
		"--dir", "file://testdata/sqlite3",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test2.db")),
	)
	require.ErrorAs(t, err, &migrate.HistoryChangedError{})

	// Fixing the migration file will finish without errors.
	sed(t, "s/col_5/col_2/g", "testdata/sqlite3/20220318104615_second.sql")
	sed(t, "s/asdasd //g", "testdata/sqlite3/20220318104615_second.sql")
	_, err = runCmd(migrateHashCmd(), "--dir", "file://testdata/sqlite3")
	require.NoError(t, err)
	s, err = runCmd(
		migrateApplyCmd(),
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
		migrateApplyCmd(),
		"--dir", "file://testdata/sqlite3",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test2.db")),
	)
	require.NoError(t, err)
	require.Equal(t, "No migration files to execute\n", s)

	// Dry run will print the statements in second migration file without executing them.
	// No changes to the revisions will be done.
	s, err = runCmd(
		migrateApplyCmd(),
		"--dir", "file://testdata/sqlite",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")),
		"--dry-run",
		"1",
	)
	require.NoError(t, err)
	require.Contains(t, s, "20220318104615")                        // log to version
	require.Contains(t, s, "ALTER TABLE `tbl` ADD `col_2` bigint;") // logs statement
	c1, err := sqlclient.Open(ctx, fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, c1.Close())
	})
	sch, err = c1.InspectSchema(ctx, "", nil)
	tbl, ok = sch.Table("tbl")
	require.True(t, ok)
	_, ok = tbl.Column("col_2")
	require.False(t, ok)
	rrw, err = migrate2.NewEntRevisions(ctx, c1)
	require.NoError(t, err)
	revs, err = rrw.ReadRevisions(ctx)
	require.NoError(t, err)
	require.Len(t, revs, 1)

	// Prerequisites for testing missing migration behavior.
	c1, err = sqlclient.Open(ctx, fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test3.db")))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, c1.Close())
	})
	require.NoError(t, os.Rename(
		"testdata/sqlite3/20220318104615_second.sql",
		"testdata/sqlite3/20220318104616_second.sql",
	))
	_, err = runCmd(migrateHashCmd(), "--dir", "file://testdata/sqlite3")
	require.NoError(t, err)
	rrw, err = migrate2.NewEntRevisions(ctx, c1)
	require.NoError(t, err)
	require.NoError(t, rrw.Migrate(ctx))

	// No changes if the last revision has a greater version than the last migration.
	require.NoError(t, rrw.WriteRevision(ctx, &migrate.Revision{Version: "zzz"}))
	s, err = runCmd(
		migrateApplyCmd(),
		"--dir", "file://testdata/sqlite3",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test3.db")),
	)
	require.NoError(t, err)
	require.Equal(t, "No migration files to execute\n", s)

	// If the revision is before the last but after the first migration, only the last one is pending.
	_, err = c1.ExecContext(ctx, "DROP table `atlas_schema_revisions`")
	require.NoError(t, err)
	s, err = runCmd(
		migrateApplyCmd(), "1",
		"--dir", "file://testdata/sqlite3",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test3.db")),
	)
	require.NoError(t, rrw.WriteRevision(ctx, &migrate.Revision{Version: "20220318104615"}))
	require.NoError(t, err)
	s, err = runCmd(
		migrateApplyCmd(),
		"--dir", "file://testdata/sqlite3",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test3.db")),
	)
	require.NoError(t, err)
	require.NotContains(t, s, "20220318104614")                     // log to version
	require.Contains(t, s, "20220318104616")                        // log to version
	require.Contains(t, s, "ALTER TABLE `tbl` ADD `col_2` bigint;") // logs statement

	// If the revision is before every migration file, every file is pending.
	_, err = c1.ExecContext(ctx, "DROP table `atlas_schema_revisions`; DROP table `tbl`;")
	require.NoError(t, err)
	require.NoError(t, rrw.Migrate(ctx))
	require.NoError(t, rrw.WriteRevision(ctx, &migrate.Revision{Version: "1"}))
	require.NoError(t, err)
	s, err = runCmd(
		migrateApplyCmd(),
		"--dir", "file://testdata/sqlite3",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test3.db")),
	)
	require.NoError(t, err)
	require.Contains(t, s, "20220318104614")                         // log to version
	require.Contains(t, s, "20220318104616")                         // log to version
	require.Contains(t, s, "CREATE TABLE tbl (`col` int NOT NULL);") // logs statement
	require.Contains(t, s, "ALTER TABLE `tbl` ADD `col_2` bigint;")  // logs statement

	// If the revision is partially applied, error out.
	require.NoError(t, rrw.WriteRevision(ctx, &migrate.Revision{Version: "z", Description: "z", Total: 1}))
	require.NoError(t, err)
	_, err = runCmd(
		migrateApplyCmd(),
		"--dir", "file://testdata/sqlite3",
		"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test3.db")),
	)
	require.EqualError(t, err, migrate.MissingMigrationError{Version: "z", Description: "z"}.Error())
}

func TestMigrate_ApplyMultiEnv(t *testing.T) {
	t.Run("FromVars", func(t *testing.T) {
		p := t.TempDir()
		h := `
variable "urls" {
  type = list(string)
}

env "local" {
  for_each = toset(var.urls)
  url = each.value
  dev = "sqlite://ci?mode=memory&cache=shared&_fk=1"
  migration {
    dir = "file://testdata/sqlite"
  }
}
`
		path := filepath.Join(p, "atlas.hcl")
		err := os.WriteFile(path, []byte(h), 0600)
		require.NoError(t, err)
		cmd := migrateCmd()
		cmd.AddCommand(migrateApplyCmd())
		s, err := runCmd(
			cmd, "apply",
			"-c", "file://"+path,
			"--env", "local",
			"--var", fmt.Sprintf("urls=sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test1.db")),
			"--var", fmt.Sprintf("urls=sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test2.db")),
		)
		require.NoError(t, err)
		require.Equal(t, 2, strings.Count(s, "Migrating to version 20220318104615 (2 migrations in total)"), "execution per environment")
		_, err = os.Stat(filepath.Join(p, "test1.db"))
		require.NoError(t, err)
		_, err = os.Stat(filepath.Join(p, "test2.db"))
		require.NoError(t, err)
	})

	t.Run("FromDataSrc", func(t *testing.T) {
		var (
			h = `
variable "url" {
  type = string
}

locals {
  string = "%test"
  bool   = true
  int    = 1
}

data "sql" "tenants" {
  url   = var.url
  query = <<EOS
SELECT name FROM tenants
	WHERE mode LIKE ? AND active = ? AND created = ?
EOS
  # Pass all types of arguments.
  args  = [local.string, local.bool, local.int]
}

env "local" {
  for_each = toset(data.sql.tenants.values)
  url = "sqlite://file:${each.value}?cache=shared&_fk=1"
  dev = "sqlite://ci?mode=memory&cache=shared&_fk=1"
  migration {
    dir = "file://testdata/sqlite"
  }
  log {
    migrate {
      apply = format(
        "{{ json . | json_merge %q }}",
        jsonencode({
          Tenant: each.value
        })
      )
    }
  }
}
`
			p    = t.TempDir()
			path = filepath.Join(p, "atlas.hcl")
			dbs  = []string{filepath.Join(p, "test1.db"), filepath.Join(p, "test2.db"), filepath.Join(p, "test3.db")}
		)
		err := os.WriteFile(path, []byte(h), 0600)
		require.NoError(t, err)
		db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&_fk=1", filepath.Join(p, "tenants.db")))
		require.NoError(t, err)
		_, err = db.Exec("CREATE TABLE `tenants` (`name` TEXT, `mode` TEXT DEFAULT 'test', `active` BOOL DEFAULT TRUE, `created` INT DEFAULT 1);")
		require.NoError(t, err)
		_, err = db.Exec("INSERT INTO `tenants` (`name`) VALUES (?), (?), (?)", dbs[0], dbs[1], dbs[2])
		require.NoError(t, err)

		cmd := migrateCmd()
		cmd.AddCommand(migrateApplyCmd())
		s, err := runCmd(
			cmd, "apply",
			"-c", "file://"+path,
			"--env", "local",
			"--var", fmt.Sprintf("url=sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "tenants.db")),
		)
		require.NoError(t, err)
		require.Equal(t, 3, strings.Count(s, `"Tenant"`))
		require.Equal(t, 3, strings.Count(s, `"Applied":[{"Applied":["CREATE TABLE tbl`), "execution per environment")
		for i := range dbs {
			_, err = os.Stat(dbs[i])
			require.NoError(t, err)
		}

		cmd = migrateCmd()
		cmd.AddCommand(migrateApplyCmd())
		s, err = runCmd(
			cmd, "apply",
			"-c", "file://"+path,
			"--env", "local",
			"--var", fmt.Sprintf("url=sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "tenants.db")),
		)
		require.NoError(t, err)
		for _, s := range strings.Split(s, "\n") {
			var r struct {
				Tenant string
				cmdlog.MigrateApply
			}
			require.NoError(t, json.Unmarshal([]byte(s), &r))
			require.Empty(t, r.Pending)
			require.Empty(t, r.Applied)
			require.NotEmpty(t, r.Tenant)
			require.Equal(t, "sqlite", r.Driver)
		}
		_, err = db.Exec("INSERT INTO `tenants` (`name`) VALUES (NULL)")
		require.NoError(t, err)
		_, err = runCmd(
			cmd, "apply",
			"-c", "file://"+path,
			"--env", "local",
			"--var", fmt.Sprintf("url=sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "tenants.db")),
			// Inject fake variable to enforce re-evaluation of the data source (skip cache).
			"--var", fmt.Sprintf("cache=%s", uuid.NewString()),
		)
		// Rows should represent real and consistent values.
		require.EqualError(t, err, "data.sql.tenants: unsupported row type: <nil>")

		_, err = db.Exec("DELETE FROM `tenants`")
		require.NoError(t, err)
		s, err = runCmd(
			cmd, "apply",
			"-c", "file://"+path,
			"--env", "local",
			"--var", fmt.Sprintf("url=sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "tenants.db")),
			// Inject fake variable to enforce re-evaluation of the data source (skip cache).
			"--var", fmt.Sprintf("cache=%s", uuid.NewString()),
		)
		// Empty list is expanded to zero blocks.
		require.EqualError(t, err, `env "local" not defined in config file`)
	})

	t.Run("TemplateDir", func(t *testing.T) {
		var (
			h = `
variable "path" {
  type = string
}

data "template_dir" "migrations" {
  path = var.path
  vars = {
    Env = atlas.env
  }
}

env "dev" {
  url = "sqlite://${atlas.env}?mode=memory&_fk=1"
  migration {
    dir = data.template_dir.migrations.url
  }
}

env "prod" {
  url = "sqlite://${atlas.env}?mode=memory&_fk=1"
  migration {
    dir = data.template_dir.migrations.url
  }
}
`
			p    = t.TempDir()
			path = filepath.Join(p, "atlas.hcl")
		)
		err := os.WriteFile(path, []byte(h), 0600)
		require.NoError(t, err)
		for _, e := range []string{"dev", "prod"} {
			cmd := migrateCmd()
			cmd.AddCommand(migrateApplyCmd())
			s, err := runCmd(
				cmd, "apply",
				"-c", "file://"+path,
				"--env", e,
				"--var", "path=testdata/templatedir",
			)
			require.NoError(t, err)
			require.Contains(t, s, "Migrating to version 2 (2 migrations in total):")
			require.Contains(t, s, fmt.Sprintf("create table %s1 (c text);", e))
			require.Contains(t, s, fmt.Sprintf("create table %s2 (c text);", e))
			require.Contains(t, s, fmt.Sprintf("create table users_%s2 (c text);", e))
		}
	})
}

func TestMigrate_ApplyTxMode(t *testing.T) {
	for _, mode := range []string{"none", "file", "all"} {
		t.Run(mode, func(t *testing.T) {
			p := t.TempDir()
			// Apply the first 2 migrations.
			s, err := runCmd(
				migrateApplyCmd(),
				"--dir", "file://testdata/sqlitetx",
				"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")),
				"--tx-mode", mode,
				"2",
			)
			require.NoError(t, err)
			require.NotEmpty(t, s)
			db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")))
			require.NoError(t, err)
			var n int
			require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM `friendships`").Scan(&n))
			require.Equal(t, 2, n)

			// Apply the rest.
			s, err = runCmd(
				migrateApplyCmd(),
				"--dir", "file://testdata/sqlitetx",
				"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")),
				"--tx-mode", mode,
			)
			require.NoError(t, err)
			require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM `friendships`").Scan(&n))
			require.Equal(t, 2, n)

			// For transactions check that the foreign keys are checked before the transaction is committed.
			if mode != "none" {
				// Apply the first 2 migrations for the faulty one.
				s, err = runCmd(
					migrateApplyCmd(),
					"--dir", "file://testdata/sqlitetx2",
					"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test_2.db")),
					"--tx-mode", mode,
					"2",
				)
				require.NoError(t, err)
				require.NotEmpty(t, s)
				db, err = sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&_fk=1", filepath.Join(p, "test_2.db")))
				require.NoError(t, err)
				require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM `friendships`").Scan(&n))
				require.Equal(t, 2, n)

				// Add an existing constraint.
				c, err := sqlclient.Open(context.Background(), fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test_2.db")))
				require.NoError(t, err)
				_, err = c.ExecContext(context.Background(), "PRAGMA foreign_keys = off; INSERT INTO `friendships` (`user_id`, `friend_id`) VALUES (3,3);PRAGMA foreign_keys = on;")
				require.NoError(t, err)
				require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM `friendships`").Scan(&n))
				require.Equal(t, 3, n)

				// Apply the rest, expect it to fail due to constraint error, but only the new one is reported.
				s, err = runCmd(
					migrateApplyCmd(),
					"--dir", "file://testdata/sqlitetx2",
					"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test_2.db")),
					"--tx-mode", mode,
				)
				require.EqualError(t, err, "sql/sqlite: foreign key mismatch: [{tbl:friendships ref:users row:4 index:1}]")
				require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM `friendships`").Scan(&n))
				require.Equal(t, 3, n) // was rolled back
			}
		})
	}
}

func TestMigrate_ApplyTxModeDirective(t *testing.T) {
	for _, mode := range []string{txModeNone, txModeFile} {
		u := openSQLite(t, "")
		_, err := runCmd(
			migrateApplyCmd(),
			"--dir", "file://testdata/sqlitetx3",
			"--url", u,
			"--tx-mode", mode,
		)
		require.EqualError(t, err, `sql/migrate: executing statement "INSERT INTO t1 VALUES (1), (1);" from version "20220925094021": UNIQUE constraint failed: t1.a`)
		db, err := sql.Open("sqlite3", strings.TrimPrefix(u, "sqlite://"))
		require.NoError(t, err)
		var n int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name IN ('atlas_schema_revisions', 'users', 't1')").Scan(&n))
		require.Equal(t, 3, n)
		require.NoError(t, db.Close())
	}

	_, err := runCmd(
		migrateApplyCmd(),
		"--dir", "file://testdata/sqlitetx3",
		"--url", "sqlite://txmode?mode=memory&_fk=1",
		"--tx-mode", txModeAll,
	)
	require.EqualError(t, err, `cannot set txmode directive to "none" in "20220925094021_second.sql" when txmode "all" is set globally`)

	s, err := runCmd(
		migrateApplyCmd(),
		"--dir", "file://testdata/sqlitetx4",
		"--url", "sqlite://txmode?mode=memory&_fk=1",
		"--tx-mode", txModeAll,
		"--log", "{{ .Error }}",
	)
	require.EqualError(t, err, `unknown txmode "unknown" found in file directive "20220925094021_second.sql"`)
	// Errors should be attached to the report.
	require.Equal(t, s, `unknown txmode "unknown" found in file directive "20220925094021_second.sql"`)
}

func TestMigrate_ApplyExecOrder(t *testing.T) {
	p := t.TempDir()
	db := fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db"))
	require.NoError(t, os.Mkdir(filepath.Join(p, "migrations"), 0700))
	dir, err := migrate.NewLocalDir(filepath.Join(p, "migrations"))
	require.NoError(t, err)
	write := func(n, b string) {
		require.NoError(t, dir.WriteFile(n, []byte(b)))
		hash, err := dir.Checksum()
		require.NoError(t, err)
		require.NoError(t, migrate.WriteSumFile(dir, hash))
	}
	write("1.sql", "create table t1(c int);")
	write("3.sql", "create table t3(c int);")

	// First run.
	s, err := runCmd(
		migrateApplyCmd(),
		"--dir", "file://"+dir.Path(),
		"--url", db,
		"--format", "{{ len .Applied }}",
	)
	require.NoError(t, err)
	require.Equal(t, "2", s)

	// File was added out of order.
	write("2.sql", "create table t2(c int);")
	_, err = runCmd(
		migrateApplyCmd(),
		"--dir", "file://"+dir.Path(),
		"--url", db,
	)
	require.EqualError(t, err, "migration file 2.sql was added out of order. See: https://atlasgo.io/versioned/apply#non-linear-error")

	// The "linear" option is the default execution order.
	_, err = runCmd(
		migrateApplyCmd(),
		"--dir", "file://"+dir.Path(),
		"--url", db,
		"--exec-order", "linear",
	)
	require.EqualError(t, err, "migration file 2.sql was added out of order. See: https://atlasgo.io/versioned/apply#non-linear-error")

	// Keep linear order and skip files that were added out of order.
	s, err = runCmd(
		migrateApplyCmd(),
		"--dir", "file://"+dir.Path(),
		"--url", db,
		"--exec-order", "linear-skip",
	)
	require.NoError(t, err)
	require.Equal(t, "No migration files to execute\n", s)

	// Allow non-linear order.
	s, err = runCmd(
		migrateApplyCmd(),
		"--dir", "file://"+dir.Path(),
		"--url", db,
		"--exec-order", "non-linear",
		"--format", "{{ range .Applied }}{{ .Version }}{{ end }}",
	)
	require.NoError(t, err)
	require.Equal(t, "2", s)

	write("2.5.sql", "create table t25(c int);")
	write("4.sql", "create table t4(c int);")
	s, err = runCmd(
		migrateApplyCmd(),
		"--dir", "file://"+dir.Path(),
		"--url", db,
		"--exec-order", "non-linear",
		"--format", "{{ range .Applied }}{{ println .Version }}{{ end }}",
	)
	require.NoError(t, err)
	require.Equal(t, "2.5\n4\n", s)

	// There are no pending migrations, in all execution orders.
	for _, o := range []string{"linear", "linear-skip", "non-linear"} {
		s, err = runCmd(
			migrateApplyCmd(),
			"--dir", "file://"+dir.Path(),
			"--url", db,
			"--exec-order", o,
		)
		require.NoError(t, err)
		require.Equal(t, "No migration files to execute\n", s)
	}
}

func TestMigrate_ApplyBaseline(t *testing.T) {
	t.Run("FromFlags", func(t *testing.T) {
		p := t.TempDir()
		// Run migration with baseline should store this revision in the database.
		s, err := runCmd(
			migrateApplyCmd(),
			"--dir", "file://testdata/baseline1",
			"--baseline", "1",
			"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test1.db")),
		)
		require.NoError(t, err)
		require.Contains(t, s, "No migration files to execute")
		// Next run without baseline should run the migration from the baseline.
		s, err = runCmd(
			migrateApplyCmd(),
			"--dir", "file://testdata/baseline1",
			"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test1.db")),
		)
		require.NoError(t, err)
		require.Contains(t, s, "No migration files to execute")

		// Multiple migration files with baseline.
		s, err = runCmd(
			migrateApplyCmd(),
			"--dir", "file://testdata/baseline2",
			"--baseline", "1",
			"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test2.db")),
		)
		require.NoError(t, err)
		require.Contains(t, s, "Migrating to version 20220318104615 from 1 (2 migrations in total)")

		// Run all migration files and skip baseline.
		s, err = runCmd(
			migrateApplyCmd(),
			"--dir", "file://testdata/baseline2",
			"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test3.db")),
		)
		require.NoError(t, err)
		require.Contains(t, s, "Migrating to version 20220318104615 (3 migrations in total)")
	})

	t.Run("FromConfig", func(t *testing.T) {
		const h = `
env "local" {
  migration {
    baseline = "1"
  }
}`
		p := t.TempDir()
		path := filepath.Join(p, "atlas.hcl")
		err := os.WriteFile(path, []byte(h), 0600)
		require.NoError(t, err)
		cmd := migrateCmd()
		cmd.AddCommand(migrateApplyCmd())
		s, err := runCmd(
			cmd, "apply",
			"-c", "file://"+path,
			"--env", "local",
			"--dir", "file://testdata/baseline1",
			"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test1.db")),
		)
		require.NoError(t, err)
		require.Contains(t, s, "No migration files to execute")

		cmd = migrateCmd()
		cmd.AddCommand(migrateApplyCmd())
		s, err = runCmd(
			cmd, "apply",
			"-c", "file://"+path,
			"--env", "local",
			"--dir", "file://testdata/baseline2",
			"--url", fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(p, "test2.db")),
		)
		require.NoError(t, err)
		require.Contains(t, s, "Migrating to version 20220318104615 from 1 (2 migrations in total)")
	})
}

func TestMigrate_Diff(t *testing.T) {
	p := t.TempDir()
	to := hclURL(t)

	// Will create migration directory if not existing.
	_, err := runCmd(
		migrateDiffCmd(),
		"name",
		"--dir", "file://"+filepath.Join(p, "migrations"),
		"--dev-url", openSQLite(t, ""),
		"--to", to,
	)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, "migrations", fmt.Sprintf("%s_name.sql", time.Now().UTC().Format("20060102150405"))))

	// Expect no clean dev error.
	p = t.TempDir()
	s, err := runCmd(
		migrateDiffCmd(),
		"name",
		"--dir", "file://"+p,
		"--dev-url", openSQLite(t, "create table t (c int);"),
		"--to", to,
	)
	require.ErrorAs(t, err, new(*migrate.NotCleanError))
	require.ErrorContains(t, err, "found table \"t\"")

	// Works (on empty directory).
	s, err = runCmd(
		migrateDiffCmd(),
		"name",
		"--dir", "file://"+p,
		"--dev-url", openSQLite(t, ""),
		"--to", to,
	)
	require.NoError(t, err)
	require.Zero(t, s)
	require.FileExists(t, filepath.Join(p, fmt.Sprintf("%s_name.sql", time.Now().UTC().Format("20060102150405"))))
	require.FileExists(t, filepath.Join(p, "atlas.sum"))

	// A lock will prevent diffing.
	sqlclient.Register("sqlitelockdiff", sqlclient.OpenerFunc(func(ctx context.Context, u *url.URL) (*sqlclient.Client, error) {
		u.Scheme = "sqlite"
		client, err := sqlclient.OpenURL(ctx, u)
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
		migrateDiffCmd(),
		"name",
		"--dir", "file://"+t.TempDir(),
		"--dev-url", fmt.Sprintf("sqlitelockdiff://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")),
		"--to", to,
	)
	require.True(t, strings.HasPrefix(s, "Error: acquiring database lock: "+errLock.Error()))
	require.ErrorIs(t, err, errLock)

	t.Run("Edit", func(t *testing.T) {
		p := t.TempDir()
		t.Setenv("EDITOR", "echo '-- Comment' >>")
		args := []string{
			"--edit",
			"--dir", "file://" + p,
			"--dev-url", openSQLite(t, ""),
			"--to", to,
		}
		_, err := runCmd(migrateDiffCmd(), args...)
		files, err := os.ReadDir(p)
		require.NoError(t, err)
		require.Len(t, files, 2)
		b, err := os.ReadFile(filepath.Join(p, files[0].Name()))
		require.NoError(t, err)
		require.Contains(t, string(b), "CREATE")
		require.True(t, strings.HasSuffix(string(b), "-- Comment\n"))
		require.Equal(t, "atlas.sum", files[1].Name())

		// Second run will have no effect.
		_, err = runCmd(migrateDiffCmd(), args...)
		require.NoError(t, err)
		files, err = os.ReadDir(p)
		require.NoError(t, err)
		require.Len(t, files, 2)
	})

	t.Run("Format", func(t *testing.T) {
		for f, out := range map[string]string{
			"{{sql .}}":            "CREATE TABLE `t` (`c` int NULL);",
			`{{- sql . "  " -}}`:   "CREATE TABLE `t` (\n  `c` int NULL\n);",
			"{{ sql . \"\t\" }}":   "CREATE TABLE `t` (\n\t`c` int NULL\n);",
			"{{sql $ \"  \t  \"}}": "CREATE TABLE `t` (\n  \t  `c` int NULL\n);",
		} {
			p := t.TempDir()
			d, err := migrate.NewLocalDir(p)
			require.NoError(t, err)
			// Works with indentation.
			s, err = runCmd(
				migrateDiffCmd(),
				"name",
				"--dir", "file://"+p,
				"--dev-url", openSQLite(t, ""),
				"--to", openSQLite(t, "create table t (c int);"),
				"--format", f,
			)
			require.NoError(t, err)
			require.Zero(t, s)
			files, err := d.Files()
			require.NoError(t, err)
			require.Len(t, files, 1)
			require.Equal(t, "-- Create \"t\" table\n"+out+"\n", string(files[0].Bytes()))
		}

		// Invalid use of sql.
		s, err = runCmd(
			migrateDiffCmd(),
			"name",
			"--dir", "file://"+p,
			"--dev-url", openSQLite(t, ""),
			"--to", openSQLite(t, "create table t (c int);"),
			"--format", `{{ if . }}{{ sql . "  " }}{{ end }}`,
		)
		require.EqualError(t, err, `'sql' can only be used to indent statements. got: {{if .}}{{sql . "  "}}{{end}}`)

		// Valid template.
		p := t.TempDir()
		d, err := migrate.NewLocalDir(p)
		require.NoError(t, err)
		s, err = runCmd(
			migrateDiffCmd(),
			"name",
			"--dir", "file://"+p,
			"--dev-url", openSQLite(t, ""),
			"--to", openSQLite(t, "create table t (c int);"),
			"--format", `{{ range .Changes }}{{ .Cmd }}{{ end }}`,
		)
		require.NoError(t, err)
		files, err := d.Files()
		require.NoError(t, err)
		require.Len(t, files, 1)
		require.Equal(t, "CREATE TABLE `t` (`c` int NULL)", string(files[0].Bytes()))
	})

	t.Run("ProjectFile", func(t *testing.T) {
		p := t.TempDir()
		h := `
variable "schema" {
  type = string
}

variable "dir" {
  type = string
}

variable "destructive" {
  type = bool
  default = false
}

env "local" {
  src = "file://${var.schema}"
  dev = "sqlite://ci?mode=memory&_fk=1"
  migration {
    dir = "file://${var.dir}"
  }
  diff {
    skip {
      drop_column = !var.destructive
    }
  }
}
`
		pathC := filepath.Join(p, "atlas.hcl")
		require.NoError(t, os.WriteFile(pathC, []byte(h), 0600))
		pathS := filepath.Join(p, "schema.sql")
		require.NoError(t, os.WriteFile(pathS, []byte(`CREATE TABLE t(c1 int, c2 int);`), 0600))
		pathD := t.TempDir()
		cmd := migrateCmd()
		cmd.AddCommand(migrateDiffCmd())
		s, err := runCmd(
			cmd, "diff", "initial",
			"-c", "file://"+pathC,
			"--env", "local",
			"--var", "schema="+pathS,
			"--var", "dir="+pathD,
		)
		require.NoError(t, err)
		require.Empty(t, s)
		d, err := migrate.NewLocalDir(pathD)
		require.NoError(t, err)
		files, err := d.Files()
		require.NoError(t, err)
		require.Len(t, files, 1)
		require.Equal(t, "-- Create \"t\" table\nCREATE TABLE `t` (`c1` int NULL, `c2` int NULL);\n", string(files[0].Bytes()))

		// Drop column should be skipped.
		require.NoError(t, os.WriteFile(pathS, []byte(`CREATE TABLE t(c1 int);`), 0600))
		cmd = migrateCmd()
		cmd.AddCommand(migrateDiffCmd())
		s, err = runCmd(
			cmd, "diff", "no_change",
			"-c", "file://"+pathC,
			"--env", "local",
			"--var", "schema="+pathS,
			"--var", "dir="+pathD,
		)
		require.NoError(t, err)
		require.Equal(t, "The migration directory is synced with the desired state, no changes to be made\n", s)
		files, err = d.Files()
		require.NoError(t, err)
		require.Len(t, files, 1)

		// Column is dropped when destructive is true.
		cmd = migrateCmd()
		cmd.AddCommand(migrateDiffCmd())
		s, err = runCmd(
			cmd, "diff", "second",
			"-c", "file://"+pathC,
			"--env", "local",
			"--var", "schema="+pathS,
			"--var", "dir="+pathD,
			"--var", "destructive=true",
		)
		require.NoError(t, err)
		require.Empty(t, s)
		files, err = d.Files()
		require.NoError(t, err)
		require.Len(t, files, 2)
	})

	t.Run("TemplateDir", func(t *testing.T) {
		var (
			h = `
variable "default" {
  type    = string
  default = "Default"
}

variable "schema_path" {
  type = string
}

variable "migrations_path" {
  type = string
}

data "hcl_schema" "app" {
  path = var.schema_path
  vars = {
    default = var.default
  }
}

data "template_dir" "app" {
  path = var.migrations_path
  vars = {
    default = var.default
  }
}

env "local" {
  src = data.hcl_schema.app.url
  dev = "sqlite://file?mode=memory&_fk=1"
  migration {
    dir = data.template_dir.app.url
  }
}
`
			p, md      = t.TempDir(), t.TempDir()
			cfg, state = filepath.Join(p, "atlas.hcl"), filepath.Join(p, "schema.hcl")
		)
		err := os.WriteFile(cfg, []byte(h), 0600)
		require.NoError(t, err)
		err = os.WriteFile(state, []byte(`variable "default" { type = string  }
schema "main" {}
table "users" {
  schema = schema.main
  column "c" {
    type = text
    default = var.default
  }
}
`), 0600)
		require.NoError(t, err)
		dir, err := migrate.NewLocalDir(md)
		require.NoError(t, err)
		err = dir.WriteFile("1.sql", []byte(`create table users (c text default "{{ .default }}" NOT NULL);`))
		require.NoError(t, err)

		cmd := migrateCmd()
		cmd.AddCommand(migrateHashCmd())
		_, err = runCmd(
			cmd, "hash",
			"--dir", "file://"+md,
		)
		require.NoError(t, err)
		f, err := dir.Open(migrate.HashFileName)
		require.NoError(t, err)
		b, err := io.ReadAll(f)
		require.NoError(t, err)
		require.NotEmpty(t, b)

		cmd = migrateCmd()
		cmd.AddCommand(migrateDiffCmd())
		s, err := runCmd(
			cmd, "diff",
			"-c", "file://"+cfg,
			"--env", "local",
			"--var", "migrations_path="+md,
			"--var", "schema_path="+state,
		)
		// Desired state and migration directory are in sync.
		require.NoError(t, err)
		require.Equal(t, "The migration directory is synced with the desired state, no changes to be made\n", s)
		f, err = dir.Open(migrate.HashFileName)
		require.NoError(t, err)
		nb, err := io.ReadAll(f)
		require.NoError(t, err)
		require.Equal(t, b, nb, "hash file should not be updated")

		// Update the desired state and run diff again.
		err = os.WriteFile(state, []byte(`variable "default" { type = string  }
schema "main" {}
table "users" {
  schema = schema.main
  column "c" {
    type = text
    default = var.default
  }
  column "d" {
    type = text
  }
}
`), 0600)
		require.NoError(t, err)
		_, err = runCmd(
			cmd, "diff",
			"-c", "file://"+cfg,
			"--env", "local",
			"--var", "migrations_path="+md,
			"--var", "schema_path="+state,
		)
		require.NoError(t, err)

		// Check files.
		files, err := dir.Files()
		require.NoError(t, err)
		require.Len(t, files, 2)
		require.Equal(t, "1.sql", files[0].Name())
		require.Equal(t, `create table users (c text default "{{ .default }}" NOT NULL);`, string(files[0].Bytes()), "should not update template files")
		require.Equal(t, "-- Add column \"d\" to table: \"users\"\nALTER TABLE `users` ADD COLUMN `d` text NOT NULL;\n", string(files[1].Bytes()))

		// Ensure the sum file is consistent.
		f, err = dir.Open(migrate.HashFileName)
		require.NoError(t, err)
		before, err := io.ReadAll(f)
		require.NoError(t, err)
		cmd = migrateCmd()
		cmd.AddCommand(migrateHashCmd())
		_, err = runCmd(
			cmd, "hash",
			"--dir", "file://"+md,
		)
		require.NoError(t, err)
		f, err = dir.Open(migrate.HashFileName)
		require.NoError(t, err)
		after, err := io.ReadAll(f)
		require.NoError(t, err)
		require.Equal(t, before, after)
	})
}

func TestMigrate_StatusJSON(t *testing.T) {
	p := t.TempDir()
	s, err := runCmd(
		migrateStatusCmd(),
		"--dir", "file://"+p,
		"-u", openSQLite(t, ""),
		"--format", "{{ json .Env.Driver }}",
	)
	require.NoError(t, err)
	require.Equal(t, `"sqlite"`, s)
}

func TestMigrate_Set(t *testing.T) {
	u := fmt.Sprintf("sqlite://file:%s?_fk=1", filepath.Join(t.TempDir(), "test.db"))
	_, err := runCmd(
		migrateApplyCmd(),
		"--dir", "file://testdata/sqlite",
		"--url", u,
	)
	require.NoError(t, err)

	s, err := runCmd(
		migrateSetCmd(),
		"--dir", "file://testdata/sqlite",
		"-u", u,
		"20220318104614",
	)
	require.NoError(t, err)
	require.Equal(t, `Current version is 20220318104614 (1 removed):

  - 20220318104615 (second)

`, s)
	s, err = runCmd(
		migrateSetCmd(),
		"--dir", "file://testdata/sqlite",
		"-u", u,
		"20220318104615",
	)
	require.NoError(t, err)
	require.Equal(t, `Current version is 20220318104615 (1 set):

  + 20220318104615 (second)

`, s)

	s, err = runCmd(
		migrateSetCmd(),
		"--dir", "file://testdata/baseline1",
		"-u", u,
	)
	require.NoError(t, err)
	require.Equal(t, `Current version is 1 (1 set, 2 removed):

  + 1 (baseline)
  - 20220318104614 (initial)
  - 20220318104615 (second)

`, s)

	s, err = runCmd(
		migrateSetCmd(),
		"--dir", filepath.Join("file://", t.TempDir()), // empty dir.
		"-u", u,
	)
	require.NoError(t, err)
	require.Equal(t, `All revisions deleted (1 in total):

  - 1 (baseline)

`, s)

	// Empty database.
	u = fmt.Sprintf("sqlite://file:%s?_fk=1", filepath.Join(t.TempDir(), "test.db"))
	_, err = runCmd(
		migrateSetCmd(),
		"--dir", "file://testdata/sqlite",
		"-u", u,
	)
	require.EqualError(t, err, "accepts 1 arg(s), received 0")

	s, err = runCmd(
		migrateSetCmd(),
		"--dir", "file://testdata/sqlite",
		"-u", u,
		"20220318104614",
	)
	require.NoError(t, err)
	require.Equal(t, `Current version is 20220318104614 (1 set):

  + 20220318104614 (initial)

`, s)
}

func TestMigrate_New(t *testing.T) {
	var (
		p = t.TempDir()
		v = time.Now().UTC().Format("20060102150405")
	)

	s, err := runCmd(migrateNewCmd(), "--dir", "file://"+p)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, v+".sql"))
	require.FileExists(t, filepath.Join(p, "atlas.sum"))
	require.Equal(t, 2, countFiles(t, p))

	s, err = runCmd(migrateNewCmd(), "my-migration-file", "--dir", "file://"+p)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, v+"_my-migration-file.sql"))
	require.FileExists(t, filepath.Join(p, "atlas.sum"))
	require.Equal(t, 3, countFiles(t, p))

	p = t.TempDir()
	s, err = runCmd(migrateNewCmd(), "golang-migrate", "--dir", "file://"+p, "--dir-format", migrate2.FormatGolangMigrate)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, v+"_golang-migrate.up.sql"))
	require.FileExists(t, filepath.Join(p, v+"_golang-migrate.down.sql"))
	require.Equal(t, 3, countFiles(t, p))

	p = t.TempDir()
	s, err = runCmd(migrateNewCmd(), "goose", "--dir", "file://"+p+"?format="+migrate2.FormatGoose)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, v+"_goose.sql"))
	require.Equal(t, 2, countFiles(t, p))

	p = t.TempDir()
	s, err = runCmd(migrateNewCmd(), "flyway", "--dir", "file://"+p+"?format="+migrate2.FormatFlyway)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, fmt.Sprintf("V%s__%s.sql", v, migrate2.FormatFlyway)))
	require.FileExists(t, filepath.Join(p, fmt.Sprintf("U%s__%s.sql", v, migrate2.FormatFlyway)))
	require.Equal(t, 3, countFiles(t, p))

	p = t.TempDir()
	s, err = runCmd(migrateNewCmd(), "liquibase", "--dir", "file://"+p+"?format="+migrate2.FormatLiquibase)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, v+"_liquibase.sql"))
	require.Equal(t, 2, countFiles(t, p))

	p = t.TempDir()
	s, err = runCmd(migrateNewCmd(), "dbmate", "--dir", "file://"+p+"?format="+migrate2.FormatDBMate)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, v+"_dbmate.sql"))
	require.Equal(t, 2, countFiles(t, p))

	f := filepath.Join("testdata", "mysql", "new.sql")
	require.NoError(t, os.WriteFile(f, []byte("contents"), 0600))
	t.Cleanup(func() { os.Remove(f) })
	s, err = runCmd(migrateNewCmd(), "--dir", "file://testdata/mysql")
	require.NotZero(t, s)
	require.Error(t, err)

	t.Run("Edit", func(t *testing.T) {
		p := t.TempDir()
		require.NoError(t, os.Setenv("EDITOR", "echo 'contents' >"))
		t.Cleanup(func() { require.NoError(t, os.Unsetenv("EDITOR")) })
		s, err = runCmd(migrateNewCmd(), "--dir", "file://"+p, "--edit")
		files, err := os.ReadDir(p)
		require.NoError(t, err)
		require.Len(t, files, 2)
		b, err := os.ReadFile(filepath.Join(p, files[0].Name()))
		require.NoError(t, err)
		require.Equal(t, "contents\n", string(b))
		require.Equal(t, "atlas.sum", files[1].Name())
	})
}

func TestMigrate_Validate(t *testing.T) {
	// Without re-playing.
	s, err := runCmd(migrateValidateCmd(), "--dir", "file://testdata/mysql")
	require.Zero(t, s)
	require.NoError(t, err)

	f := filepath.Join("testdata", "mysql", "new.sql")
	require.NoError(t, os.WriteFile(f, []byte("contents"), 0600))
	t.Cleanup(func() { os.Remove(f) })
	s, err = runCmd(migrateValidateCmd(), "--dir", "file://testdata/mysql")
	require.NotZero(t, s)
	require.Error(t, err)
	require.NoError(t, os.Remove(f))

	// Replay migration files if a dev-url is given.
	p := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(p, "1_initial.sql"), []byte("create table t1 (c1 int)"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(p, "2_second.sql"), []byte("create table t2 (c2 int)"), 0644))
	_, err = runCmd(migrateHashCmd(), "--dir", "file://"+p)
	require.NoError(t, err)
	s, err = runCmd(
		migrateValidateCmd(),
		"--dir", "file://"+p,
		"--dev-url", openSQLite(t, ""),
	)
	require.Zero(t, s)
	require.NoError(t, err)

	// Should fail since the files are not compatible with SQLite.
	_, err = runCmd(migrateValidateCmd(), "--dir", "file://testdata/mysql", "--dev-url", openSQLite(t, ""))
	require.Error(t, err)

	// Will report detailed information when there is a checksum mismatch.
	p = t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(p, "1_initial.sql"), []byte("create table t1 (c1 int)"), 0644))
	s, err = runCmd(migrateValidateCmd(), "--dir", "file://"+p)
	require.ErrorIs(t, err, migrate.ErrChecksumNotFound)
	require.Contains(t, s, "You have a checksum error")
	_, err = runCmd(migrateHashCmd(), "--dir", "file://"+p)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(p, "2_second.sql"), []byte("create table t2 (c2 int)"), 0644))
	s, err = runCmd(migrateValidateCmd(), "--dir", "file://"+p)
	csErr := &migrate.ChecksumError{}
	require.ErrorAs(t, err, &csErr)
	require.Equal(t, 3, csErr.Line)
	require.Equal(t, "2_second.sql", csErr.File)
	require.Equal(t, migrate.ReasonAdded, csErr.Reason)
	require.Contains(t, s, "You have a checksum error")
	require.Contains(t, s, "L3: 2_second.sql was added")
}

func TestMigrate_Hash(t *testing.T) {
	s, err := runCmd(migrateHashCmd(), "--dir", "file://testdata/mysql")
	require.Zero(t, s)
	require.NoError(t, err)

	// Prints a warning if --force flag is still used.
	s, err = runCmd(migrateHashCmd(), "--dir", "file://testdata/mysql", "--force")
	require.NoError(t, err)
	require.Equal(t, "Flag --force has been deprecated, you can safely omit it.\n", s)

	p := t.TempDir()
	err = copyFile(filepath.Join("testdata", "mysql", "20220318104614_initial.sql"), filepath.Join(p, "20220318104614_initial.sql"))
	require.NoError(t, err)

	s, err = runCmd(migrateHashCmd(), "--dir", "file://"+p)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, "atlas.sum"))
	d, err := os.ReadFile(filepath.Join(p, "atlas.sum"))
	require.NoError(t, err)
	dir, err := migrate.NewLocalDir(p)
	require.NoError(t, err)
	sum, err := dir.Checksum()
	require.NoError(t, err)
	b, err := sum.MarshalText()
	require.NoError(t, err)
	require.Equal(t, d, b)

	p = t.TempDir()
	require.NoError(t, copyFile(
		filepath.Join("testdata", "mysql", "20220318104614_initial.sql"),
		filepath.Join(p, "20220318104614_initial.sql"),
	))
	s, err = runCmd(migrateHashCmd(), "--dir", "file://"+os.Getenv("MIGRATION_DIR"))
	require.NotZero(t, s)
	require.Error(t, err)
}

func TestMigrate_Lint(t *testing.T) {
	p := t.TempDir()
	s, err := runCmd(
		migrateLintCmd(),
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
		migrateLintCmd(),
		"--dir", "file://"+p,
		"--dev-url", openSQLite(t, ""),
		"--latest", "1",
	)
	require.Error(t, err)
	require.Regexp(t, `Analyzing changes from version 1 to 2 \(1 migration in total\):

  -- analyzing version 2
    -- destructive changes detected:
      -- L1: Dropping table "t" https://atlasgo.io/lint/analyzers#DS102
    -- suggested fix:
      -> Add a pre-migration check to ensure table "t" is empty before dropping it
  -- ok \(.+\)

  -------------------------
  -- .+
  -- 1 version with errors
  -- 1 schema change
  -- 1 diagnostic
`, s)
	s, err = runCmd(
		migrateLintCmd(),
		"--dir", "file://"+p,
		"--dev-url", openSQLite(t, ""),
		"--latest", "1",
		"--log", "{{ range .Files }}{{ .Name }}{{ end }}", // Backward compatibility with old flag name.
	)
	require.Error(t, err)
	require.Equal(t, "2.sql", s)

	t.Run("FromConfig", func(t *testing.T) {
		cfg := filepath.Join(p, "atlas.hcl")
		err := os.WriteFile(cfg, []byte(`
variable "error" {
  type    = bool
  default = false
}

lint {
  latest = 1
  destructive {
    error = var.error
  }
}
`), 0600)
		require.NoError(t, err)
		cmd := migrateCmd()
		cmd.AddCommand(migrateLintCmd())
		s, err := runCmd(
			cmd, "lint",
			"--dir", "file://"+p,
			"--dev-url", openSQLite(t, ""),
			"-c", "file://"+cfg,
		)
		require.NoError(t, err)
		require.Regexp(t, `Analyzing changes from version 1 to 2 \(1 migration in total\):

  -- analyzing version 2
    -- destructive changes detected:
      -- L1: Dropping table "t" https://atlasgo.io/lint/analyzers#DS102
    -- suggested fix:
      -> Add a pre-migration check to ensure table "t" is empty before dropping it
  -- ok \(.+\)

  -------------------------
  -- .+
  -- 1 version with warnings
  -- 1 schema change
  -- 1 diagnostic
`, s)

		cmd = migrateCmd()
		cmd.AddCommand(migrateLintCmd())
		s, err = runCmd(
			cmd, "lint",
			"--dir", "file://"+p,
			"--dev-url", openSQLite(t, ""),
			"-c", "file://"+cfg,
			"--var", "error=true",
		)
		require.Error(t, err)
		require.Regexp(t, `Analyzing changes from version 1 to 2 \(1 migration in total\):

  -- analyzing version 2
    -- destructive changes detected:
      -- L1: Dropping table "t" https://atlasgo.io/lint/analyzers#DS102
    -- suggested fix:
      -> Add a pre-migration check to ensure table "t" is empty before dropping it
  -- ok (.+)

  -------------------------
  -- .+
  -- 1 version with errors
  -- 1 schema change
  -- 1 diagnostic
`, s)
	})

	// Change files to golang-migrate format.
	require.NoError(t, os.Rename(filepath.Join(p, "1.sql"), filepath.Join(p, "1.up.sql")))
	require.NoError(t, os.Rename(filepath.Join(p, "2.sql"), filepath.Join(p, "1.down.sql")))
	s, err = runCmd(
		migrateLintCmd(),
		"--dir", "file://"+p+"?format="+migrate2.FormatGolangMigrate,
		"--dev-url", openSQLite(t, ""),
		"--latest", "2",
		"--format", "{{ range .Files }}{{ .Name }}:{{ len .Reports }}{{ end }}",
	)
	require.NoError(t, err)
	require.Equal(t, "1.up.sql:0", s)
	s, err = runCmd(
		migrateLintCmd(),
		"--dir", "file://"+p+"?format="+migrate2.FormatGolangMigrate,
		"--dev-url", openSQLite(t, ""),
		"--latest", "2",
		"--format", "{{ range .Files }}{{ .Name }}:{{ len .Reports }}{{ end }}",
		"--dir-format", migrate2.FormatGolangMigrate,
	)
	require.NoError(t, err)
	require.Equal(t, "1.up.sql:0", s)

	// Invalid files.
	err = os.WriteFile(filepath.Join(p, "2.up.sql"), []byte("BORING"), 0600)
	require.NoError(t, err)
	s, err = runCmd(
		migrateLintCmd(),
		"--dir", "file://"+p+"?format="+migrate2.FormatGolangMigrate,
		"--dev-url", openSQLite(t, ""),
		"--latest", "1",
	)
	require.Error(t, err)
	require.Regexp(t, `Analyzing changes from version 1.up to 2.up \(1 migration in total\):

  Error: executing statement: BORING: near "BORING": syntax error

  -------------------------
  -- .+
  -- 1 version with errors
`, s)
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
  }
  check {
    expr     = "price2 <> price1"
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

func lines(f migrate.File) []string {
	return strings.Split(strings.TrimSpace(string(f.Bytes())), "\n")
}

func TestMigrate_ApplyDirFormat(t *testing.T) {
	folder := func(t *testing.T) string {
		p := t.TempDir()
		upFile := filepath.Join(p, "1_initial.up.sql")
		downFile := filepath.Join(p, "1_initial.down.sql")
		require.NoError(t, os.WriteFile(upFile, []byte("CREATE TABLE users (id INT PRIMARY KEY);"), 0644))
		require.NoError(t, os.WriteFile(downFile, []byte("DROP TABLE users;"), 0644))
		_, err := runCmd(migrateHashCmd(), "--dir", "file://"+p+"?format=golang-migrate")
		require.NoError(t, err)
		return p
	}
	t.Run("golang-migrate dir-format flag", func(t *testing.T) {
		p := folder(t)
		s, err := runCmd(
			migrateApplyCmd(),
			"--dir", "file://"+p,
			"--dir-format", "golang-migrate",
			"--url", openSQLite(t, ""),
			"--dry-run",
		)
		require.NoError(t, err)
		require.Contains(t, s, "CREATE TABLE users (id INT PRIMARY KEY);")
		require.NotContains(t, s, "DROP TABLE users;")
		require.Contains(t, s, "1 migrations in total")
	})
	t.Run("golang-migrate atlas.hcl migration format", func(t *testing.T) {
		p := folder(t)
		// Create atlas.hcl config file with golang-migrate format
		configPath := filepath.Join(p, "atlas.hcl")
		config := fmt.Sprintf(`
env "test" {
  url = "%s"
  migration {
    dir = "file://%s"
    format = "golang-migrate"
  }
}`, openSQLite(t, ""), p)
		require.NoError(t, os.WriteFile(configPath, []byte(config), 0644))
		cmd := migrateCmd()
		cmd.AddCommand(migrateApplyCmd())
		s, err := runCmd(
			cmd, "apply",
			"-c", "file://"+configPath,
			"--env", "test",
			"--dry-run",
		)
		require.NoError(t, err)

		// Should only contain the up migration, not the down
		require.Contains(t, s, "CREATE TABLE users (id INT PRIMARY KEY);")
		require.NotContains(t, s, "DROP TABLE users;")
		require.Contains(t, s, "1 migrations in total")
	})
	t.Run("golang-migrate atlas.hcl migration format from query parameter", func(t *testing.T) {
		p := folder(t)
		// Create atlas.hcl config file with golang-migrate format
		configPath := filepath.Join(p, "atlas.hcl")
		config := fmt.Sprintf(`
env "test" {
  url = "%s"
  migration {
    dir = "file://%s?format=golang-migrate"
  }
}`, openSQLite(t, ""), p)
		require.NoError(t, os.WriteFile(configPath, []byte(config), 0644))
		cmd := migrateCmd()
		cmd.AddCommand(migrateApplyCmd())
		s, err := runCmd(
			cmd, "apply",
			"-c", "file://"+configPath,
			"--env", "test",
			"--dry-run",
		)
		require.NoError(t, err)

		// Should only contain the up migration, not the down
		require.Contains(t, s, "CREATE TABLE users (id INT PRIMARY KEY);")
		require.NotContains(t, s, "DROP TABLE users;")
		require.Contains(t, s, "1 migrations in total")
	})
}
