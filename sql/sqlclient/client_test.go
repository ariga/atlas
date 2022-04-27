// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlclient_test

import (
	"context"
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
	c1, err = sqlclient.Open(context.Background(), "maria://:3306")
	require.NoError(t, err)
	require.True(t, c == c1)
	c1, err = sqlclient.Open(context.Background(), "postgres://:3306")
	require.EqualError(t, err, `sql/sqlclient: no opener was register with name "postgres"`)
}
