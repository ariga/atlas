// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"text/template"

	"ariga.io/atlas/cmd/atlas/internal/cmdlog"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	_ "ariga.io/atlas/sql/sqlite"

	"github.com/fatih/color"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestSchemaInspect_MarshalJSON(t *testing.T) {
	report := &cmdlog.SchemaInspect{
		Realm: schema.NewRealm(
			schema.New("test").
				SetComment("schema comment").
				AddTables(
					schema.NewTable("users").
						SetCharset("charset").
						AddColumns(
							&schema.Column{
								Name: "id",
								Type: &schema.ColumnType{Raw: "bigint"},
							},
							&schema.Column{
								Name: "name",
								Type: &schema.ColumnType{Raw: "varchar(255)"},
								Attrs: []schema.Attr{
									&schema.Collation{V: "collate"},
								},
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
              "type": "varchar(255)",
              "collate": "collate"
            }
          ],
          "charset": "charset"
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
      ],
      "comment": "schema comment"
    },
    {
      "name": "temp"
    }
  ]
}`, string(ident))
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
}

func TestSchemaInspect_Mermaid(t *testing.T) {
	ctx := context.Background()
	client, err := sqlclient.Open(ctx, "sqlite://ci?mode=memory&_fk=1")
	require.NoError(t, err)
	defer client.Close()
	var (
		b     bytes.Buffer
		users = schema.NewTable("users").
			AddColumns(
				schema.NewIntColumn("id", "int"),
				schema.NewStringColumn("name", "text"),
			)
		tmpl = template.Must(template.New("format").Funcs(cmdlog.InspectTemplateFuncs).Parse(`{{ mermaid . }}`))
	)
	require.NoError(t, tmpl.Execute(&b, &cmdlog.SchemaInspect{
		Client: client,
		Realm:  schema.NewRealm(schema.New("main").AddTables(users)),
	}))
	require.Equal(t, `erDiagram
    users {
      int id
      text name
    }
`, b.String())

	b.Reset()
	users.SetPrimaryKey(
		schema.NewPrimaryKey(users.Columns[0]),
	)
	posts := schema.NewTable("posts").
		AddColumns(
			schema.NewIntColumn("id", "int"),
			schema.NewStringColumn("text", "text"),
		)
	posts.SetPrimaryKey(
		schema.NewPrimaryKey(posts.Columns...),
	)
	posts.AddForeignKeys(
		schema.NewForeignKey("owner_id").
			AddColumns(posts.Columns[0]).
			SetRefTable(users).
			AddRefColumns(users.Columns[0]),
	)

	require.NoError(t, tmpl.Execute(&b, &cmdlog.SchemaInspect{
		Client: client,
		Realm:  schema.NewRealm(schema.New("main").AddTables(users, posts)),
	}))
	require.Equal(t, `erDiagram
    users {
      int id PK
      text name
    }
    posts {
      int id PK,FK
      text text PK
    }
    posts }o--o| users : owner_id
`, b.String())

	b.Reset()
	require.NoError(t, tmpl.Execute(&b, &cmdlog.SchemaInspect{
		Client: client,
		Realm: schema.NewRealm(
			schema.New("main").AddTables(users),
			schema.New("temp").AddTables(posts),
		),
	}))
	require.Equal(t, `erDiagram
    main_users["main.users"] {
      int id PK
      text name
    }
    temp_posts["temp.posts"] {
      int id PK,FK
      text text PK
    }
    temp_posts }o--o| main_users : owner_id
`, b.String())

	b.Reset()
	users.
		AddColumns(
			schema.NewIntColumn("best_friend_id", "int"),
		).
		AddIndexes(
			schema.NewUniqueIndex("best_friend_id").
				AddColumns(users.Columns[2]),
		).
		AddForeignKeys(
			schema.NewForeignKey("best_friend_id").
				AddColumns(users.Columns[2]).
				SetRefTable(users).
				AddRefColumns(users.Columns[0]),
		)
	require.NoError(t, tmpl.Execute(&b, &cmdlog.SchemaInspect{
		Client: client,
		Realm:  schema.NewRealm(schema.New("main").AddTables(users)),
	}))
	require.Equal(t, `erDiagram
    users {
      int id PK
      text name
      int best_friend_id FK
    }
    users |o--o| users : best_friend_id
`, b.String())

	b.Reset()
	users.
		AddColumns(
			schema.NewFloatColumn("duration", "double precision"),
		)
	require.NoError(t, tmpl.Execute(&b, &cmdlog.SchemaInspect{
		Client: client,
		Realm:  schema.NewRealm(schema.New("main").AddTables(users)),
	}))
	require.Equal(t, `erDiagram
    users {
      int id PK
      text name
      int best_friend_id FK
      double_precision duration
    }
    users |o--o| users : best_friend_id
`, b.String())
}

func TestSchemaDiff_MarshalSQL(t *testing.T) {
	client, err := sqlclient.Open(context.Background(), "sqlite://ci?mode=memory&_fk=1")
	require.NoError(t, err)
	defer client.Close()
	diff := &cmdlog.SchemaDiff{
		Client: client,
		Changes: schema.Changes{
			&schema.AddTable{
				T: schema.NewTable("users").
					AddColumns(
						schema.NewIntColumn("id", "int"),
					),
			},
		},
	}
	b, err := diff.MarshalSQL()
	require.NoError(t, err)
	require.Equal(t, "-- Create \"users\" table\nCREATE TABLE `users` (`id` int NOT NULL);\n", b)
}

func TestMigrateSet(t *testing.T) {
	var (
		b   bytes.Buffer
		log = &cmdlog.MigrateSet{}
	)
	color.NoColor = true
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
