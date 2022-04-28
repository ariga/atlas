// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlclient_test

import (
	"context"
	"database/sql"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"ariga.io/atlas/sql/sqlclient"
)

func TestRegisterOpen(t *testing.T) {
	c := &sqlclient.Client{}
	sqlclient.Register(
		"mysql",
		sqlclient.OpenerFunc(func(ctx context.Context, url *url.URL) (*sqlclient.Client, error) {
			return c, nil
		}),
		sqlclient.RegisterFlavours("maria"),
		sqlclient.RegisterURLParser(func(u *url.URL) *sqlclient.URL {
			return &sqlclient.URL{URL: u, DSN: "dsn", Schema: "schema"}
		}),
	)
	require.PanicsWithValue(
		t,
		"sql/sqlclient: Register opener is nil",
		func() { sqlclient.Register("mysql", nil) },
	)
	require.PanicsWithValue(
		t,
		"sql/sqlclient: Register called twice for mysql",
		func() {
			sqlclient.Register("mysql", sqlclient.OpenerFunc(func(ctx context.Context, url *url.URL) (*sqlclient.Client, error) {
				return c, nil
			}))
		},
	)
	c1, err := sqlclient.Open(context.Background(), "mysql://:3306")
	require.NoError(t, err)
	require.True(t, c == c1)
	require.Equal(t, "dsn", c.URL.DSN)
	require.Equal(t, "schema", c.URL.Schema)

	c1, err = sqlclient.Open(context.Background(), "maria://:3306")
	require.NoError(t, err)
	require.True(t, c == c1)
	require.Equal(t, "dsn", c.URL.DSN)
	require.Equal(t, "schema", c.URL.Schema)

	c1, err = sqlclient.Open(context.Background(), "postgres://:3306")
	require.EqualError(t, err, `sql/sqlclient: no opener was register with name "postgres"`)
}

func TestClient_AddClosers(t *testing.T) {
	var (
		i int
		c = &sqlclient.Client{DB: sql.OpenDB(nil)}
		f = closerFunc(func() error { i++; return nil })
	)
	c.AddClosers(f, f, f)
	require.NoError(t, c.Close())
	require.Equal(t, 3, i)
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }
