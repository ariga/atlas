// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package atlascmd

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
	"strings"
	"testing"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	_ "ariga.io/atlas/sql/sqlite"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {
	_, err := runCmd(Root, "migrate")
	require.NoError(t, err)
}

func TestMigrate_Diff(t *testing.T) {
	p := t.TempDir()

	// Expect no clean dev error.
	s, err := runCmd(
		Root, "migrate", "diff",
		"name",
		"--dir", "file://"+p,
		"--dev-url", openSQLite(t, "create table t (c int);"),
		"--to", hclURL(t),
	)
	require.True(t, strings.HasPrefix(s, "Error: sql/migrate: connected database is not clean"))
	require.EqualError(t, err, "sql/migrate: connected database is not clean")

	// Works.
	s, err = runCmd(
		Root, "migrate", "diff",
		"name",
		"--dir", "file://"+p,
		"--dev-url", openSQLite(t, ""),
		"--to", hclURL(t),
	)
	require.Zero(t, s)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(p, fmt.Sprintf("%s_name.sql", time.Now().Format("20060102150405"))))
	require.FileExists(t, filepath.Join(p, "atlas.sum"))

	// A lock will prevent diffing.
	sqlclient.Register("sqlitelock", sqlclient.OpenerFunc(func(ctx context.Context, u *url.URL) (*sqlclient.Client, error) {
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
		"--dev-url", fmt.Sprintf("sqlitelock://file:%s?cache=shared&_fk=1", filepath.Join(p, "test.db")),
		"--to", hclURL(t),
	)
	require.True(t, strings.HasPrefix(s, "Error: "+lockErr))
	require.EqualError(t, err, lockErr)
}

func TestMigrate_Validate(t *testing.T) {
	s, err := runCmd(Root, "migrate", "validate", "--dir", "file://testdata")
	require.Zero(t, s)
	require.NoError(t, err)
}

func TestMigrate_ValidateError(t *testing.T) {
	if os.Getenv("DO_VALIDATE") == "1" {
		runCmd(Root, "migrate", "validate", "--dir", "file://testdata")
		return
	}
	f := filepath.Join("testdata", "new.sql")
	require.NoError(t, os.WriteFile(f, []byte("contents"), 0600))
	defer os.Remove(f)
	cmd := exec.Command(os.Args[0], "-test.run=TestMigrate_ValidateError") //nolint:gosec
	cmd.Env = append(os.Environ(), "DO_VALIDATE=1")
	err := cmd.Run()
	if err, ok := err.(*exec.ExitError); ok && !err.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exist status 1", err)
}

func TestMigrate_Hash(t *testing.T) {
	s, err := runCmd(Root, "migrate", "hash", "--dir", "file://testdata")
	require.Zero(t, s)
	require.NoError(t, err)

	p := t.TempDir()
	err = copyFile(filepath.Join("testdata", "20220318104614_initial.sql"), filepath.Join(p, "20220318104614_initial.sql"))
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
}

func TestMigrate_HashError(t *testing.T) {
	if os.Getenv("DO_HASH") == "1" {
		runCmd(Root, "migrate", "hash", "--dir", "file://"+os.Getenv("MIGRATION_DIR"))
		return
	}
	p := t.TempDir()
	err := copyFile(filepath.Join("testdata", "20220318104614_initial.sql"), filepath.Join(p, "20220318104614_initial.sql"))
	require.NoError(t, err)
	cmd := exec.Command(os.Args[0], "-test.run=TestMigrate_HashError") //nolint:gosec
	cmd.Env = append(os.Environ(), "DO_HASH=1", "MIGRATION_DIR="+p)
	err = cmd.Run()
	if err, ok := err.(*exec.ExitError); ok && !err.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exist status 1", err)
}

const hcl = `
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
	require.NoError(t, os.WriteFile(filepath.Join(p, "atlas.hcl"), []byte(hcl), 0600))
	return "file://" + filepath.Join(p, "atlas.hcl")
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

const lockErr = "lockErr"

func (d *sqliteLockerDriver) Lock(context.Context, string, time.Duration) (schema.UnlockFunc, error) {
	return func() error { return nil }, errors.New(lockErr)
}
