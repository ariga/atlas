// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"ariga.io/atlas/sql/sqlite"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestCmdSchemaDiff(t *testing.T) {
	// Creates the missing table.
	s, err := runCmd(
		newDiffCmd(),
		"--from", openSQLite(t, ""),
		"--to", openSQLite(t, "create table t1 (id int);"),
	)
	require.NoError(t, err)
	require.EqualValues(t, "-- Create \"t1\" table\nCREATE TABLE `t1` (`id` int NULL)\n", s)

	// No changes.
	s, err = runCmd(
		newDiffCmd(),
		"--from", openSQLite(t, ""),
		"--to", openSQLite(t, ""),
	)
	require.NoError(t, err)
	require.EqualValues(t, "Schemas are synced, no changes to be made.\n", s)

	// Desired state from migration directory requires dev database.
	_, err = runCmd(
		newDiffCmd(),
		"--from", "file://testdata/sqlite",
		"--to", openSQLite(t, ""),
	)
	require.EqualError(t, err, "--dev-url cannot be empty")

	// Desired state from migration directory.
	s, err = runCmd(
		newDiffCmd(),
		"--from", openSQLite(t, ""),
		"--to", "file://testdata/sqlite",
		"--dev-url", openSQLite(t, ""),
	)
	require.NoError(t, err)
	require.EqualValues(t, "-- Create \"tbl\" table\nCREATE TABLE `tbl` (`col` int NOT NULL, `col_2` bigint NULL)\n", s)

	// Desired state from migration directory.
	s, err = runCmd(
		newDiffCmd(),
		"--from", openSQLite(t, ""),
		"--to", "file://testdata/sqlite",
		"--dev-url", openSQLite(t, ""),
	)
	require.NoError(t, err)
	require.EqualValues(t, "-- Create \"tbl\" table\nCREATE TABLE `tbl` (`col` int NOT NULL, `col_2` bigint NULL)\n", s)

	// Current state from migration directory, desired state from HCL - synced.
	p := filepath.Join(t.TempDir(), "schema.hcl")
	require.NoError(t, os.WriteFile(p, []byte(`schema "main" {}
table "tbl" {
  schema = schema.main
  column "col" {
    type = int
  }
  column "col_2" {
    type = bigint
    null = true
  }
}`), 0644))
	s, err = runCmd(
		newDiffCmd(),
		"--from", "file://testdata/sqlite",
		"--to", "file://"+p,
		"--dev-url", openSQLite(t, ""),
	)
	require.NoError(t, err)
	require.EqualValues(t, "Schemas are synced, no changes to be made.\n", s)

	// Current state from migration directory, desired state from HCL - missing column.
	p = filepath.Join(t.TempDir(), "schema.hcl")
	require.NoError(t, os.WriteFile(p, []byte(`schema "main" {}
table "tbl" {
  schema = schema.main
  column "col" {
    type = int
  }
  column "col_2" {
    type = bigint
    null = true
  }
  column "col_3" {
    type = text
  }
}`), 0644))
	s, err = runCmd(
		newDiffCmd(),
		"--from", "file://testdata/sqlite",
		"--to", "file://"+p,
		"--dev-url", openSQLite(t, ""),
	)
	require.NoError(t, err)
	require.EqualValues(
		t,
		"-- Add column \"col_3\" to table: \"tbl\"\nALTER TABLE `tbl` ADD COLUMN `col_3` text NOT NULL\n",
		s,
	)
}

// openSQLite creates a sqlite db, seeds it with the seed query and returns the url to it.
func openSQLite(t *testing.T, seed string) string {
	f, err := os.CreateTemp("", "sqlite.db")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.Remove(f.Name())
	})
	dsn := fmt.Sprintf("file:%s?cache=shared&_fk=1", f.Name())
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
	})
	drv, err := sqlite.Open(db)
	require.NoError(t, err)
	if len(seed) > 0 {
		_, err := drv.ExecContext(context.Background(), seed)
		require.NoError(t, err)
	}
	return fmt.Sprintf("sqlite://%s", dsn)
}

func runCmd(cmd *cobra.Command, args ...string) (string, error) {
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}
