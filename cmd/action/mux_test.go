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
		dsn      string
		expected string
	}{
		{
			dsn:      "sqlite://test.db",
			expected: `failed opening "test.db": stat test.db: no such file or directory`,
		},
		{
			dsn:      "sqlite://some_random_string_like_this",
			expected: `failed opening "some_random_string_like_this": stat some_random_string_like_this: no such file or directory`,
		},
		{
			dsn:      "sqlite://file:/home/fred/data.db",
			expected: `failed opening "/home/fred/data.db": stat /home/fred/data.db: no such file or directory`,
		},
		{
			dsn:      "sqlite://file:///home/fred/data.db",
			expected: `failed opening "/home/fred/data.db": stat /home/fred/data.db: no such file or directory`,
		},
		{
			dsn:      "sqlite://file://localhost/home/fred/data.db",
			expected: `failed opening "/localhost/home/fred/data.db": stat /localhost/home/fred/data.db: no such file or directory`,
		},
		{
			dsn:      "sqlite://file://darkstar/home/fred/data.db",
			expected: `failed opening "/darkstar/home/fred/data.db": stat /darkstar/home/fred/data.db: no such file or directory`,
		},
		{
			dsn:      "sqlite://file:data.db?mode=ro&cache=private",
			expected: `failed opening "data.db": stat data.db: no such file or directory`,
		},
		{
			dsn:      "sqlite://file:/home/fred/data.db?vfs=unix-dotfile",
			expected: `failed opening "/home/fred/data.db": stat /home/fred/data.db: no such file or directory`,
		},
		{
			dsn:      "sqlite://file:data.db?mode=readonly",
			expected: `failed opening "data.db": stat data.db: no such file or directory`,
		},
		{
			dsn:      "sqlite://asdad?cache=shared&mode=memory",
			expected: `failed opening "asdad": stat asdad: no such file or directory`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.dsn, func(t *testing.T) {
			_, err := action.SchemaNameFromDSN(tt.dsn)
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
	_, err = action.SchemaNameFromDSN(dsn)
	r.NoError(err)
}

func Test_SQLiteInMemory(t *testing.T) {
	r := require.New(t)
	_, err := action.SchemaNameFromDSN("sqlite://file:test.db?cache=shared&mode=memory")
	r.NoError(err)
}

func Test_PostgresSchemaDSN(t *testing.T) {
	var tests = []struct {
		dsn      string
		expected string
		wantErr  bool
	}{
		{
			dsn:      "postgres://localhost:5432/dbname?search_path=foo",
			expected: "foo",
		},
		{
			dsn:      "postgres://localhost:5432/dbname",
			expected: "",
		},
		{
			dsn:      "postgres://(bad:host)?search_path=foo",
			expected: "",
			wantErr:  true,
		},
		{
			dsn:      "postgres://localhost:5432/dbname?search_path=",
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.dsn, func(t *testing.T) {
			schema, err := action.SchemaNameFromDSN(tt.dsn)
			require.Equal(t, tt.wantErr, err != nil)
			require.Equal(t, tt.expected, schema)
		})
	}
}
