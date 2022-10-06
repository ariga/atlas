// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
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
			out := runFmt(t, tt.args)
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
	MigrateFlags.Apply.BaselineVersion = ""
	_, err = runCmd(Root, "migrate", "apply", "--dir", "file://testdata/sqlite", "--url", u)
	require.NoError(t, err)

	// Run clean and expect to be clean.
	_, err = runCmd(Root, "migrate", "apply", "--dir", "file://testdata/sqlite", "--url", u)
	require.NoError(t, err)
	s, err := runCmd(Root, "schema", "clean", "--url", u, "--auto-approve")
	require.NoError(t, err)
	require.NotZero(t, s)
	require.NoError(t, c.Driver.(migrate.CleanChecker).CheckClean(context.Background(), nil))
}

func runFmt(t *testing.T, args []string) string {
	var out bytes.Buffer
	SchemaFmt.ResetCommands() // Detach from sub-commands and parents, needed to skip input validation done by them.
	SchemaFmt.SetOut(&out)
	SchemaFmt.SetArgs(args)
	err := SchemaFmt.Execute()
	require.NoError(t, err)
	return out.String()
}

func assertDir(t *testing.T, dir string, expected map[string]string) {
	act := make(map[string]string)
	files, err := ioutil.ReadDir(dir)
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
