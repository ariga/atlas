// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdext_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/cmdext"
	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/sqlclient"
	_ "ariga.io/atlas/sql/sqlite"
	_ "ariga.io/atlas/sql/sqlite/sqlitecheck"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestRuntimeVarSrc(t *testing.T) {
	var (
		v struct {
			V string `spec:"v"`
		}
		state = schemahcl.New(cmdext.DataSources...)
	)
	err := state.EvalBytes([]byte(`
data "runtimevar" "pass" {
  url = "constant://?val=hello+world&decoder=binary"
}

v = data.runtimevar.pass
`), &v, nil)
	require.EqualError(t, err, `data.runtimevar.pass: unsupported decoder: "binary"`)

	err = state.EvalBytes([]byte(`
data "runtimevar" "pass" {
  url = "constant://?val=hello+world"
}

v = data.runtimevar.pass
`), &v, nil)
	require.NoError(t, err)
	require.Equal(t, v.V, "hello world")

	err = state.EvalBytes([]byte(`
data "runtimevar" "pass" {
  url = "constant://?val=hello+world&decoder=string"
}

v = data.runtimevar.pass
`), &v, nil)
	require.NoError(t, err, "nop decoder")
	require.Equal(t, v.V, "hello world")
}

func TestQuerySrc(t *testing.T) {
	ctx := context.Background()
	u := fmt.Sprintf("sqlite3://file:%s?cache=shared&_fk=1", filepath.Join(t.TempDir(), "test.db"))
	drv, err := sqlclient.Open(context.Background(), u)
	require.NoError(t, err)
	_, err = drv.ExecContext(ctx, "CREATE TABLE users (name text)")
	require.NoError(t, err)
	_, err = drv.ExecContext(ctx, "INSERT INTO users (name) VALUES ('a8m')")
	require.NoError(t, err)

	var (
		v struct {
			C  int      `spec:"c"`
			V  string   `spec:"v"`
			Vs []string `spec:"vs"`
		}
		state = schemahcl.New(cmdext.DataSources...)
	)
	err = state.EvalBytes([]byte(fmt.Sprintf(`
data "sql" "user" {
  url   = %q
  query = "SELECT name FROM users"
}

c = data.sql.user.count
v = data.sql.user.value
vs = data.sql.user.values
`, u)), &v, nil)
	require.NoError(t, err)
	require.Equal(t, 1, v.C)
	require.Equal(t, "a8m", v.V)
	require.Equal(t, []string{"a8m"}, v.Vs)
}
