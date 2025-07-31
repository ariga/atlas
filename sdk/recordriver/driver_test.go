// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package recordriver

import (
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDriver(t *testing.T) {
	db, err := sql.Open("recordriver", "t1")
	require.NoError(t, err)
	defer db.Close()
	SetResponse("t1", "select sqlite_version()", &Response{
		Cols: []string{"sqlite_version()"},
		Data: [][]driver.Value{{"3.30.1"}},
	})
	for i := 0; i < 3; i++ {
		query, err := db.Query("select sqlite_version()")
		require.NoError(t, err)
		defer query.Close()
		var rows []string
		for query.Next() {
			var version string
			err = query.Scan(&version)
			require.NoError(t, err)
			rows = append(rows, version)
			require.Equal(t, "3.30.1", version)
		}
		require.Len(t, rows, 1)
		hi, ok := Session("t1")
		require.True(t, ok)
		require.Len(t, hi.Queries, i+1)

	}
}

func TestInputs(t *testing.T) {
	db, err := sql.Open("recordriver", "t1")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Query("select * from t where id = ?", 1)
	require.NoError(t, err)
}
