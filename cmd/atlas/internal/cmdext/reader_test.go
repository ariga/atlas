// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdext

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	cmdmigrate "ariga.io/atlas/cmd/atlas/internal/migrate"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	_ "ariga.io/atlas/sql/sqlite"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestSchemaDirState(t *testing.T) {
	ctx := context.Background()
	dev, err := sqlclient.Open(ctx, "sqlite://dev?mode=memory")
	require.NoError(t, err)
	p1, p2 := filepath.Join(t.TempDir(), cmdmigrate.DefaultDirName), filepath.Join(t.TempDir(), "schema")
	require.NoError(t, os.Mkdir(p1, 0755))
	require.NoError(t, os.Mkdir(p2, 0755))
	u1, err := url.Parse("file://" + p1)
	require.NoError(t, err)
	u2, err := url.Parse("file://" + p2)
	require.NoError(t, err)

	// Empty migration directory.
	sr, err := StateReaderSQL(ctx, &StateReaderConfig{
		Dev:  dev,
		URLs: []*url.URL{u1},
	})
	require.NoError(t, err)
	r, err := sr.ReadState(ctx)
	require.NoError(t, err)
	require.Empty(t, r.Schemas[0].Tables, "empty main schema (default SQLite schema)")

	// Sum file is required for migrations dir (named "migrations").
	d1, err := migrate.NewLocalDir(p1)
	require.NoError(t, err)
	require.NoError(t, d1.WriteFile("1.sql", []byte("CREATE TABLE t1 (id INTEGER PRIMARY KEY);")))
	_, err = StateReaderSQL(ctx, &StateReaderConfig{
		Dev:  dev,
		URLs: []*url.URL{u1},
	})
	require.Error(t, err, "checksum file not found")

	// Schema directory.
	d2, err := migrate.NewLocalDir(p2)
	require.NoError(t, err)
	require.NoError(t, d2.WriteFile("1.sql", []byte("CREATE TABLE t1 (id INTEGER PRIMARY KEY);")))
	sr, err = StateReaderSQL(ctx, &StateReaderConfig{
		Dev:  dev,
		URLs: []*url.URL{u2},
	})
	require.NoError(t, err)
	r, err = sr.ReadState(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, r.Schemas[0].Tables, "non-empty schema")

	// Exclude patterns.
	sr, err = StateReaderSQL(ctx, &StateReaderConfig{
		Dev:     dev,
		URLs:    []*url.URL{u2},
		Exclude: []string{"t1"},
	})
	require.NoError(t, err)
	r, err = sr.ReadState(ctx)
	require.NoError(t, err)
	require.Empty(t, r.Schemas[0].Tables, "empty schema after excluding table")

	// If schema contains a checksum file, it must be valid.
	require.NoError(t, d2.WriteFile(migrate.HashFileName, []byte("invalid")))
	_, err = StateReaderSQL(ctx, &StateReaderConfig{
		Dev:  dev,
		URLs: []*url.URL{u2},
	})
	require.Error(t, err, "invalid checksum file")
}

func TestStateReaderHCL(t *testing.T) {
	ctx := context.Background()
	dev, err := sqlclient.Open(ctx, "sqlite://dev?mode=memory")
	require.NoError(t, err)

	p := filepath.Join(t.TempDir(), "schema")
	require.NoError(t, os.Mkdir(p, 0755))

	// Write an empty schema file into the directory.
	os.WriteFile(p+"/schema.hcl", []byte(`
schema "default" {}
table "t1" {
  schema = schema.default
  column "id" {
    type = int
  }
  column "name" {
  	type = text
  }
}`), 0644)

	// Read schema file.
	u, err := url.Parse("file://" + p + "/schema.hcl")
	sr, err := StateReaderHCL(ctx, &StateReaderConfig{
		Dev:  dev,
		URLs: []*url.URL{u},
	})
	require.NoError(t, err)
	r, err := sr.ReadState(ctx)
	require.NoError(t, err)
	require.Equal(t, schema.NewRealm().AddSchemas(
		schema.New("default").AddTables(
			schema.NewTable("t1").AddColumns(
				schema.NewColumn("id").SetType(&schema.IntegerType{
					T: "int",
				}),
				schema.NewColumn("name").SetType(&schema.StringType{
					T: "text",
				}),
			),
		),
	), r)

	// Read schema file with exclude patterns.
	sr, err = StateReaderHCL(ctx, &StateReaderConfig{
		Dev:     dev,
		URLs:    []*url.URL{u},
		Exclude: []string{"*.name"},
	})
	require.NoError(t, err)
	r, err = sr.ReadState(ctx)
	require.NoError(t, err)
	_, exists := r.Schemas[0].Tables[0].Column("name")
	require.False(t, exists, "column 'name' should be excluded")
}
