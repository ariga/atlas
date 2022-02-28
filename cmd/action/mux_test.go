// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action_test

import (
	"io/ioutil"
	"os"
	"testing"

	"ariga.io/atlas/cmd/action"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func Test_ProviderNotSupported(t *testing.T) {
	u := action.NewMux()
	_, err := u.OpenAtlas("fake://open")
	require.Error(t, err)
}

func Test_RegisterProvider(t *testing.T) {
	u := action.NewMux()
	p := func(s string) (*action.Driver, error) { return nil, nil }
	require.NotPanics(t, func() { u.RegisterProvider("key", p) })
}

func Test_RegisterTwiceSameKeyFails(t *testing.T) {
	u := action.NewMux()
	p := func(s string) (*action.Driver, error) { return nil, nil }
	require.NotPanics(t, func() { u.RegisterProvider("key", p) })
	require.Panics(t, func() { u.RegisterProvider("key", p) })
}

func Test_GetDriverFails(t *testing.T) {
	u := action.NewMux()
	_, err := u.OpenAtlas("key://open")
	require.Error(t, err)
}

func Test_GetDriverSuccess(t *testing.T) {
	u := action.NewMux()
	p := func(s string) (*action.Driver, error) { return nil, nil }
	u.RegisterProvider("key", p)
	_, err := u.OpenAtlas("key://open")
	require.NoError(t, err)
}

func Test_SQLiteFileDoestNotExist(t *testing.T) {
	var tests = []struct {
		url      string
		expected string
	}{
		{
			url:      "sqlite://test.db",
			expected: `failed opening "test.db": stat test.db: no such file or directory`,
		},
		{
			url:      "sqlite://some_random_string_like_this",
			expected: `failed opening "some_random_string_like_this": stat some_random_string_like_this: no such file or directory`,
		},
		{
			url:      "sqlite://file:/home/fred/data.db",
			expected: `failed opening "/home/fred/data.db": stat /home/fred/data.db: no such file or directory`,
		},
		{
			url:      "sqlite://file:///home/fred/data.db",
			expected: `failed opening "/home/fred/data.db": stat /home/fred/data.db: no such file or directory`,
		},
		{
			url:      "sqlite://file://localhost/home/fred/data.db",
			expected: `failed opening "/localhost/home/fred/data.db": stat /localhost/home/fred/data.db: no such file or directory`,
		},
		{
			url:      "sqlite://file://darkstar/home/fred/data.db",
			expected: `failed opening "/darkstar/home/fred/data.db": stat /darkstar/home/fred/data.db: no such file or directory`,
		},
		{
			url:      "sqlite://file:data.db?mode=ro&cache=private",
			expected: `failed opening "data.db": stat data.db: no such file or directory`,
		},
		{
			url:      "sqlite://file:/home/fred/data.db?vfs=unix-dotfile",
			expected: `failed opening "/home/fred/data.db": stat /home/fred/data.db: no such file or directory`,
		},
		{
			url:      "sqlite://file:data.db?mode=readonly",
			expected: `failed opening "data.db": stat data.db: no such file or directory`,
		},
		{
			url:      "sqlite://asdad?cache=shared&mode=memory",
			expected: `failed opening "asdad": stat asdad: no such file or directory`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			_, err := action.SchemaNameFromURL(tt.url)
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
	dsn := "sqlite://file://" + file.Name()
	_, err = action.SchemaNameFromURL(dsn)
	r.NoError(err)
}

func Test_SQLiteInMemory(t *testing.T) {
	r := require.New(t)
	_, err := action.SchemaNameFromURL("sqlite://file:test.db?cache=shared&mode=memory")
	r.NoError(err)
}

func Test_PostgresSchemaDSN(t *testing.T) {
	var tests = []struct {
		url      string
		expected string
		wantErr  bool
	}{
		{
			url:      "postgres://localhost:5432/dbname?search_path=foo",
			expected: "foo",
		},
		{
			url:      "postgres://localhost:5432/dbname",
			expected: "",
		},
		{
			url:      "postgres://(bad:host)?search_path=foo",
			expected: "",
			wantErr:  true,
		},
		{
			url:      "postgres://localhost:5432/dbname?search_path=",
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			schema, err := action.SchemaNameFromURL(tt.url)
			require.Equal(t, tt.wantErr, err != nil)
			require.Equal(t, tt.expected, schema)
		})
	}
}
