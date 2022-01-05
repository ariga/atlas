// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package uri_test

import (
	"io/ioutil"
	"os"
	"testing"

	"ariga.io/atlas/cmd/action/internal/uri"

	"github.com/stretchr/testify/require"
)

func Test_SqliteFileDoestNotExist(t *testing.T) {
	var tests = []struct {
		dsn      string
		expected string
	}{
		{
			dsn:      "test.db",
			expected: "file test.db does not exist",
		},
		{
			dsn:      "some_random_string_like_this",
			expected: "file some_random_string_like_this does not exist",
		},
		{
			dsn:      "file:/home/fred/data.db",
			expected: "file /home/fred/data.db does not exist",
		},
		{
			dsn:      "file:///home/fred/data.db",
			expected: "file /home/fred/data.db does not exist",
		},
		{
			dsn:      "file://localhost/home/fred/data.db",
			expected: "file /localhost/home/fred/data.db does not exist",
		},
		{
			dsn:      "file://darkstar/home/fred/data.db",
			expected: "file /darkstar/home/fred/data.db does not exist",
		},
		{
			dsn:      "file:data.db?mode=ro&cache=private",
			expected: "file data.db does not exist",
		},
		{
			dsn:      "file:/home/fred/data.db?vfs=unix-dotfile",
			expected: "file /home/fred/data.db does not exist",
		},
		{
			dsn:      "file:data.db?mode=readonly",
			expected: "file data.db does not exist",
		},
		{
			dsn:      "asdad?cache=shared&mode=memory",
			expected: "file asdad does not exist",
		},
	}
	for _, tt := range tests {
		t.Run(tt.dsn, func(t *testing.T) {
			err := uri.SqliteExists(tt.dsn)
			require.EqualError(t, err, tt.expected)
		})
	}
}

func Test_SqliteFileExist(t *testing.T) {
	r := require.New(t)
	file, err := ioutil.TempFile("", "tmp")
	r.NoError(err)
	t.Cleanup(func() {
		err := os.Remove(file.Name())
		r.NoError(err)
	})
	dsn := "file://" + file.Name()
	err = uri.SqliteExists(dsn)
	r.NoError(err)
}

func Test_SqliteInMemory(t *testing.T) {
	r := require.New(t)
	dsn := "file:test.db?cache=shared&mode=memory"
	err := uri.SqliteExists(dsn)
	r.NoError(err)
}
