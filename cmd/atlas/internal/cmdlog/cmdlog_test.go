// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"text/template"

	"ariga.io/atlas/cmd/atlas/internal/cmdlog"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	_ "ariga.io/atlas/sql/sqlite"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestSchemaInspect_MarshalJSON(t *testing.T) {
	report := &cmdlog.SchemaInspect{
		Realm: schema.NewRealm(
			schema.New("test").
				AddTables(
					schema.NewTable("users").
						AddColumns(
							&schema.Column{
								Name: "id",
								Type: &schema.ColumnType{Raw: "bigint"},
							},
							&schema.Column{
								Name: "name",
								Type: &schema.ColumnType{Raw: "varchar(255)"},
							},
						),
					schema.NewTable("posts").
						AddColumns(
							&schema.Column{
								Name: "id",
								Type: &schema.ColumnType{Raw: "bigint"},
							},
							&schema.Column{
								Name: "text",
								Type: &schema.ColumnType{Raw: "text"},
							},
						),
				),
			schema.New("temp"),
		),
	}
	b, err := report.MarshalJSON()
	require.NoError(t, err)
	ident, err := json.MarshalIndent(json.RawMessage(b), "", "  ")
	require.NoError(t, err)
	require.Equal(t, `{
  "schemas": [
    {
      "name": "test",
      "tables": [
        {
          "name": "users",
          "columns": [
            {
              "name": "id",
              "type": "bigint"
            },
            {
              "name": "name",
              "type": "varchar(255)"
            }
          ]
        },
        {
          "name": "posts",
          "columns": [
            {
              "name": "id",
              "type": "bigint"
            },
            {
              "name": "text",
              "type": "text"
            }
          ]
        }
      ]
    },
    {
      "name": "temp"
    }
  ]
}`, string(ident))

	report = &cmdlog.SchemaInspect{Error: io.EOF}
	b, err = report.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, `{"Error":"EOF"}`, string(b))
}

func TestSchemaInspect_MarshalSQL(t *testing.T) {
	client, err := sqlclient.Open(context.Background(), "sqlite://ci?mode=memory&_fk=1")
	require.NoError(t, err)
	defer client.Close()
	report := &cmdlog.SchemaInspect{
		Client: client,
		Realm: schema.NewRealm(
			schema.New("main").
				AddTables(
					schema.NewTable("users").
						AddColumns(
							schema.NewIntColumn("id", "int"),
						),
				),
		),
	}
	b, err := report.MarshalSQL()
	require.NoError(t, err)
	require.Equal(t, "-- Create \"users\" table\nCREATE TABLE `users` (`id` int NOT NULL);\n", b)
}

func TestSchemaInspect_EncodeSQL(t *testing.T) {
	ctx := context.Background()
	client, err := sqlclient.Open(ctx, "sqlite://ci?mode=memory&_fk=1")
	require.NoError(t, err)
	defer client.Close()
	err = client.ApplyChanges(ctx, schema.Changes{
		&schema.AddTable{
			T: schema.NewTable("users").
				AddColumns(
					schema.NewIntColumn("id", "int"),
					schema.NewStringColumn("name", "text"),
				),
		},
	})
	require.NoError(t, err)
	realm, err := client.InspectRealm(ctx, nil)
	require.NoError(t, err)

	var (
		b    bytes.Buffer
		tmpl = template.Must(template.New("format").Funcs(cmdlog.InspectTemplateFuncs).Parse(`{{ sql . }}`))
	)
	require.NoError(t, tmpl.Execute(&b, &cmdlog.SchemaInspect{Client: client, Realm: realm}))
	require.Equal(t, "-- Create \"users\" table\nCREATE TABLE `users` (`id` int NOT NULL, `name` text NOT NULL);\n", b.String())
	b.Reset()
	require.NoError(t, tmpl.Execute(&b, &cmdlog.SchemaInspect{Error: errors.New("failure")}))
	require.Equal(t, "failure", b.String())
}

func TestMigrateSet(t *testing.T) {
	var (
		b   bytes.Buffer
		log = &cmdlog.MigrateSet{}
	)
	require.NoError(t, cmdlog.MigrateSetTemplate.Execute(&b, log))
	require.Empty(t, b.String())

	log.Current = &migrate.Revision{Version: "1"}
	require.NoError(t, cmdlog.MigrateSetTemplate.Execute(&b, log))
	require.Empty(t, b.String())

	log.Current = &migrate.Revision{Version: "1"}
	log.Removed(&migrate.Revision{Version: "2"})
	log.Removed(&migrate.Revision{Version: "3", Description: "desc"})
	require.NoError(t, cmdlog.MigrateSetTemplate.Execute(&b, log))
	require.Equal(t, `Current version is 1 (2 removed):

  - 2
  - 3 (desc)

`, b.String())

	b.Reset()
	log.Set(&migrate.Revision{Version: "1.1", Description: "desc"})
	require.NoError(t, cmdlog.MigrateSetTemplate.Execute(&b, log))
	require.Equal(t, `Current version is 1 (1 set, 2 removed):

  + 1.1 (desc)
  - 2
  - 3 (desc)

`, b.String())

	b.Reset()
	log.Current, log.Revisions = nil, nil
	log.Removed(&migrate.Revision{Version: "2"})
	log.Removed(&migrate.Revision{Version: "3", Description: "desc"})
	require.NoError(t, cmdlog.MigrateSetTemplate.Execute(&b, log))
	require.Equal(t, `All revisions deleted (2 in total):

  - 2
  - 3 (desc)

`, b.String())
}
