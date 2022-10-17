// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlclient"
	"github.com/stretchr/testify/require"
)

const (
	unformatted = `block  "x"  {
 x = 1
    y     = 2
}
`
	formatted = `block "x" {
  x = 1
  y = 2
}
`
)

func TestCmdSchemaDiff(t *testing.T) {
	// Creates the missing table.
	s, err := runCmd(
		schemaDiffCmd(),
		"--from", openSQLite(t, ""),
		"--to", openSQLite(t, "create table t1 (id int);"),
	)
	require.NoError(t, err)
	require.EqualValues(t, "-- Create \"t1\" table\nCREATE TABLE `t1` (`id` int NULL)\n", s)

	// No changes.
	s, err = runCmd(
		schemaDiffCmd(),
		"--from", openSQLite(t, ""),
		"--to", openSQLite(t, ""),
	)
	require.NoError(t, err)
	require.EqualValues(t, "Schemas are synced, no changes to be made.\n", s)

	// Desired state from migration directory requires dev database.
	_, err = runCmd(
		schemaDiffCmd(),
		"--from", "file://testdata/sqlite",
		"--to", openSQLite(t, ""),
	)
	require.EqualError(t, err, "--dev-url cannot be empty")

	// Desired state from migration directory.
	s, err = runCmd(
		schemaDiffCmd(),
		"--from", openSQLite(t, ""),
		"--to", "file://testdata/sqlite",
		"--dev-url", openSQLite(t, ""),
	)
	require.NoError(t, err)
	require.EqualValues(t, "-- Create \"tbl\" table\nCREATE TABLE `tbl` (`col` int NOT NULL, `col_2` bigint NULL)\n", s)

	// Desired state from migration directory.
	s, err = runCmd(
		schemaDiffCmd(),
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
		schemaDiffCmd(),
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
		schemaDiffCmd(),
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

	// Current state from migration directory with version, desired state from HCL - two missing columns.
	s, err = runCmd(
		schemaDiffCmd(),
		"--from", "file://testdata/sqlite?version=20220318104614",
		"--to", "file://"+p,
		"--dev-url", openSQLite(t, ""),
	)
	require.NoError(t, err)
	require.EqualValues(
		t,
		"-- Add column \"col_2\" to table: \"tbl\"\n"+
			"ALTER TABLE `tbl` ADD COLUMN `col_2` bigint NULL\n"+
			"-- Add column \"col_3\" to table: \"tbl\"\n"+
			"ALTER TABLE `tbl` ADD COLUMN `col_3` text NOT NULL\n",
		s,
	)

	// Current state from migration directory, desired state from multi file HCL - missing column.
	p = t.TempDir()
	var (
		one = filepath.Join(p, "one.hcl")
		two = filepath.Join(p, "two.hcl")
	)
	require.NoError(t, os.WriteFile(one, []byte(`table "tbl" {
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
	require.NoError(t, os.WriteFile(two, []byte(`schema "main" {}`), 0644))
	s, err = runCmd(
		schemaDiffCmd(),
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
	s, err = runCmd(
		schemaDiffCmd(),
		"--from", "file://testdata/sqlite",
		"--to", "file://"+one,
		"--to", "file://"+two,
		"--dev-url", openSQLite(t, ""),
	)
	require.NoError(t, err)
	require.EqualValues(
		t,
		"-- Add column \"col_3\" to table: \"tbl\"\nALTER TABLE `tbl` ADD COLUMN `col_3` text NOT NULL\n",
		s,
	)
}

func TestFmt(t *testing.T) {
	for _, tt := range []struct {
		name          string
		inputDir      map[string]string
		expectedDir   map[string]string
		expectedFile  string
		expectedOut   string
		args          []string
		expectedPrint bool
	}{
		{
			name: "specific file",
			inputDir: map[string]string{
				"test.hcl": unformatted,
			},
			expectedDir: map[string]string{
				"test.hcl": formatted,
			},
			args:        []string{"test.hcl"},
			expectedOut: "test.hcl\n",
		},
		{
			name: "current dir",
			inputDir: map[string]string{
				"test.hcl": unformatted,
			},
			expectedDir: map[string]string{
				"test.hcl": formatted,
			},
			expectedOut: "test.hcl\n",
		},
		{
			name: "multi path implicit",
			inputDir: map[string]string{
				"test.hcl":  unformatted,
				"test2.hcl": unformatted,
			},
			expectedDir: map[string]string{
				"test.hcl":  formatted,
				"test2.hcl": formatted,
			},
			expectedOut: "test.hcl\ntest2.hcl\n",
		},
		{
			name: "multi path explicit",
			inputDir: map[string]string{
				"test.hcl":  unformatted,
				"test2.hcl": unformatted,
			},
			expectedDir: map[string]string{
				"test.hcl":  formatted,
				"test2.hcl": formatted,
			},
			args:        []string{"test.hcl", "test2.hcl"},
			expectedOut: "test.hcl\ntest2.hcl\n",
		},
		{
			name: "formatted",
			inputDir: map[string]string{
				"test.hcl": formatted,
			},
			expectedDir: map[string]string{
				"test.hcl": formatted,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupFmtTest(t, tt.inputDir)
			out, err := runCmd(schemaFmtCmd(), tt.args...)
			require.NoError(t, err)
			assertDir(t, dir, tt.expectedDir)
			require.EqualValues(t, tt.expectedOut, out)
		})
	}
}

func TestSchema_Clean(t *testing.T) {
	var (
		u      = fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", filepath.Join(t.TempDir(), "test.db"))
		c, err = sqlclient.Open(context.Background(), u)
	)
	require.NoError(t, err)

	// Apply migrations onto database.
	_, err = runCmd(migrateApplyCmd(), "--dir", "file://testdata/sqlite", "--url", u)
	require.NoError(t, err)

	// Run clean and expect to be clean.
	_, err = runCmd(migrateApplyCmd(), "--dir", "file://testdata/sqlite", "--url", u)
	require.NoError(t, err)
	s, err := runCmd(schemaCleanCmd(), "--url", u, "--auto-approve")
	require.NoError(t, err)
	require.NotZero(t, s)
	require.NoError(t, c.Driver.(migrate.CleanChecker).CheckClean(context.Background(), nil))
}

func assertDir(t *testing.T, dir string, expected map[string]string) {
	act := make(map[string]string)
	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(dir, f.Name()))
		require.NoError(t, err)
		act[f.Name()] = string(contents)
	}
	require.EqualValues(t, expected, act)
}

func setupFmtTest(t *testing.T, inputDir map[string]string) string {
	wd, err := os.Getwd()
	require.NoError(t, err)
	dir, err := os.MkdirTemp(os.TempDir(), "fmt-test-")
	require.NoError(t, err)
	err = os.Chdir(dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(dir)
		os.Chdir(wd) //nolint:errcheck
	})
	for name, contents := range inputDir {
		file := path.Join(dir, name)
		err = os.WriteFile(file, []byte(contents), 0600)
	}
	require.NoError(t, err)
	return dir
}
