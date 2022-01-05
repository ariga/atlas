// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action_test

import (
	"io/ioutil"
	"os"
	"testing"

	"ariga.io/atlas/cmd/action/internal"

	"github.com/stretchr/testify/require"
)

func Test_SqliteFileDoestNotExist(t *testing.T) {
	var tests = []struct {
		dsn      string
		expected string
	}{
		{
			dsn:      "test.db",
			expected: `failed opening "test.db": stat test.db: no such file or directory`,
		},
		{
			dsn:      "some_random_string_like_this",
			expected: `failed opening "some_random_string_like_this": stat some_random_string_like_this: no such file or directory`,
		},
		{
			dsn:      "file:/home/fred/data.db",
			expected: `failed opening "/home/fred/data.db": stat /home/fred/data.db: no such file or directory`,
		},
		{
			dsn:      "file:///home/fred/data.db",
			expected: `failed opening "/home/fred/data.db": stat /home/fred/data.db: no such file or directory`,
		},
		{
			dsn:      "file://localhost/home/fred/data.db",
			expected: `failed opening "/localhost/home/fred/data.db": stat /localhost/home/fred/data.db: no such file or directory`,
		},
		{
			dsn:      "file://darkstar/home/fred/data.db",
			expected: `failed opening "/darkstar/home/fred/data.db": stat /darkstar/home/fred/data.db: no such file or directory`,
		},
		{
			dsn:      "file:data.db?mode=ro&cache=private",
			expected: `failed opening "data.db": stat data.db: no such file or directory`,
		},
		{
			dsn:      "file:/home/fred/data.db?vfs=unix-dotfile",
			expected: `failed opening "/home/fred/data.db": stat /home/fred/data.db: no such file or directory`,
		},
		{
			dsn:      "file:data.db?mode=readonly",
			expected: `failed opening "data.db": stat data.db: no such file or directory`,
		},
		{
			dsn:      "asdad?cache=shared&mode=memory",
			expected: `failed opening "asdad": stat asdad: no such file or directory`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.dsn, func(t *testing.T) {
			err := action.SQLiteExists(tt.dsn)
			require.EqualError(t, err, tt.expected)
		})
	}
}

func Test_SQLiteFileExist(t *testing.T) {
	r := require.New(t)
	file, err := ioutil.TempFile("", "tmp")
	r.NoError(err)
	t.Cleanup(func() {
		err := os.Remove(file.Name())
		r.NoError(err)
	})
	dsn := "file://" + file.Name()
	err = action.SQLiteExists(dsn)
	r.NoError(err)
}

func Test_SQLiteInMemory(t *testing.T) {
	r := require.New(t)
	dsn := "file:test.db?cache=shared&mode=memory"
	err := action.SQLiteExists(dsn)
	r.NoError(err)
}
