// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdext_test

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/cmdext"
	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlclient"
	_ "ariga.io/atlas/sql/sqlite"

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

func TestEntLoader_LoadState(t *testing.T) {
	ctx := context.Background()
	drv, err := sqlclient.Open(ctx, "sqlite://test?mode=memory&_fk=1")
	require.NoError(t, err)
	u, err := url.Parse("ent://../migrate/ent/schema")
	require.NoError(t, err)
	l, ok := cmdext.States.Loader("ent")
	require.True(t, ok)
	state, err := l.LoadState(ctx, &cmdext.LoadStateOptions{
		Dev:  drv,
		URLs: []*url.URL{u},
	})
	require.NoError(t, err)
	realm, err := state.ReadState(ctx)
	require.NoError(t, err)
	require.Len(t, realm.Schemas, 1)
	require.Len(t, realm.Schemas[0].Tables, 1)
	revT := realm.Schemas[0].Tables[0]
	require.Equal(t, "atlas_schema_revisions", revT.Name)
}

func TestEntLoader_MigrateDiff(t *testing.T) {
	ctx := context.Background()
	drv, err := sqlclient.Open(ctx, "sqlite://test?mode=memory&_fk=1")
	require.NoError(t, err)
	dir, err := migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)
	d, ok := cmdext.States.Differ([]string{"ent://../migrate/ent/schema?globalid=1"})
	require.True(t, ok)
	err = d.MigrateDiff(ctx, &cmdext.MigrateDiffOptions{
		Name: "boring",
		Dev:  drv,
		Dir:  dir,
		To:   []string{"ent://../migrate/ent/schema?globalid=1"},
	})
	require.NoError(t, err)
	files, err := dir.Files()
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(files[0].Name(), "_boring.sql"))

	_, ok = cmdext.States.Differ([]string{"ent://../migrate/ent/schema"})
	require.False(t, ok, "skipping schemas without globalid")
}
