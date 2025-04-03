// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	cmdmigrate "ariga.io/atlas/cmd/atlas/internal/migrate"

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
	report := cmdlog.NewSchemaInspect(
		context.Background(),
		client,
		schema.NewRealm(
			schema.New("main").
				AddTables(
					schema.NewTable("users").
						AddColumns(
							schema.NewIntColumn("id", "int"),
						),
				),
		),
	)
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
	require.NoError(t, tmpl.Execute(&b, cmdlog.NewSchemaInspect(ctx, client, realm)))
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
	require.NoError(t, tmpl.Execute(&b, cmdlog.NewSchemaInspect(context.Background(),
		client,
		schema.NewRealm(schema.New("main").AddTables(users))),
	))
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

	require.NoError(t, tmpl.Execute(&b, cmdlog.NewSchemaInspect(
		context.Background(),
		client, schema.NewRealm(schema.New("main").AddTables(users, posts)))),
	)
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
	require.NoError(t, tmpl.Execute(&b, cmdlog.NewSchemaInspect(
		context.Background(),
		client,
		schema.NewRealm(
			schema.New("main").AddTables(users),
			schema.New("temp").AddTables(posts),
		),
	)))
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
	require.NoError(t, tmpl.Execute(&b, cmdlog.NewSchemaInspect(
		context.Background(),
		client,
		schema.NewRealm(schema.New("main").AddTables(users)),
	)))
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
			schema.NewFloatColumn("time duration", "double precision"),
		)
	require.NoError(t, tmpl.Execute(&b, cmdlog.NewSchemaInspect(
		context.Background(),
		client,
		schema.NewRealm(schema.New("main").AddTables(users)),
	)))
	require.Equal(t, `erDiagram
    users {
      int id PK
      text name
      int best_friend_id FK
      double_precision time_duration
    }
    users |o--o| users : best_friend_id
`, b.String())
}

func TestSchemaInspect_RedactedURL(t *testing.T) {
	cmd := cmdlog.SchemaInspect{
		URL: "mysql://root:password@localhost:3306/test",
	}
	u, err := cmd.RedactedURL()
	require.NoError(t, err)
	require.Equal(t, "mysql://root:xxxxx@localhost:3306/test", u)
}

func TestSchemaDiff_MarshalSQL(t *testing.T) {
	client, err := sqlclient.Open(context.Background(), "sqlite://ci?mode=memory&_fk=1")
	require.NoError(t, err)
	defer client.Close()
	diff := cmdlog.NewSchemaDiff(context.Background(), client, nil, nil, schema.Changes{
		&schema.AddTable{
			T: schema.NewTable("users").
				AddColumns(
					schema.NewIntColumn("id", "int"),
				),
		},
	})
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

func TestMigrateApply(t *testing.T) {
	var (
		b   bytes.Buffer
		d   migrate.MemDir
		log = &cmdlog.MigrateApply{Start: time.Now()}
	)
	log.End = log.Start.Add(time.Millisecond * 10)
	color.NoColor = true
	require.NoError(t, cmdlog.MigrateApplyTemplate.Execute(&b, log))
	require.Equal(t, "No migration files to execute\n", b.String())

	require.NoError(t, d.WriteFile("20240116000001.sql", nil))
	require.NoError(t, d.WriteFile("20240116000002.sql", nil))
	require.NoError(t, d.WriteFile("20240116000003.sql", nil))
	files, err := d.Files()
	require.NoError(t, err)

	// Single file.
	b.Reset()
	log.Pending = files[:1]
	log.Target = files[0].Version()
	log.Applied = []*cmdlog.AppliedFile{
		{
			File:  files[0],
			Start: log.Start,
			End:   log.Start.Add(time.Millisecond * 5),
			Applied: []string{
				"CREATE TABLE users (id int NOT NULL);",
				"CREATE TABLE posts (id int NOT NULL);",
			},
		},
	}
	require.NoError(t, cmdlog.MigrateApplyTemplate.Execute(&b, log))
	require.Equal(t, `Migrating to version 20240116000001 (1 migrations in total):

  -- migrating version 20240116000001
    -> CREATE TABLE users (id int NOT NULL);
    -> CREATE TABLE posts (id int NOT NULL);
  -- ok (5ms)

  -------------------------
  -- 10ms
  -- 1 migration
  -- 2 sql statements
`, b.String())

	// Multiple files, with errors.
	b.Reset()
	log.Pending = files[1:3]
	log.Current = files[0].Version()
	log.Target = files[2].Version()
	log.Applied = []*cmdlog.AppliedFile{
		{
			File:  files[1],
			Start: log.Start,
			End:   log.Start.Add(time.Millisecond),
			Applied: []string{
				"CREATE TABLE t1 (id int NOT NULL);",
				"CREATE TABLE t2 (id int NOT NULL);",
			},
		},
		{
			File:  files[2],
			Start: log.Start,
			End:   log.Start.Add(time.Millisecond * 2),
			Applied: []string{
				"CREATE TABLE t3 (id int NOT NULL);",
				"CREATE TABLE t4 (id int NOT NULL);",
			},
			Error: &cmdlog.StmtError{
				Stmt: "CREATE TABLE t4 (id int NOT NULL);",
				Text: "table t4 already exists",
			},
		},
	}
	require.NoError(t, cmdlog.MigrateApplyTemplate.Execute(&b, log))
	require.Equal(t, `Migrating to version 20240116000003 from 20240116000001 (2 migrations in total):

  -- migrating version 20240116000002
    -> CREATE TABLE t1 (id int NOT NULL);
    -> CREATE TABLE t2 (id int NOT NULL);
  -- ok (1ms)

  -- migrating version 20240116000003
    -> CREATE TABLE t3 (id int NOT NULL);
    -> CREATE TABLE t4 (id int NOT NULL);
    table t4 already exists

  -------------------------
  -- 10ms
  -- 1 migration ok, 1 with errors
  -- 3 sql statements ok, 1 with errors
`, b.String())

	// Multiple files with checks, and without errors.
	b.Reset()
	log.Applied = []*cmdlog.AppliedFile{
		{
			File:  files[1],
			Start: log.Start,
			End:   log.Start.Add(time.Millisecond),
			Applied: []string{
				"CREATE TABLE t1 (id int NOT NULL);",
				"CREATE TABLE t2 (id int NOT NULL);",
			},
		},
		{
			File:  files[2],
			Start: log.Start,
			End:   log.Start.Add(time.Millisecond * 2),
			Checks: []*cmdlog.FileChecks{
				{
					Stmts: []*cmdlog.Check{
						{Stmt: "SELECT 1;"},
						{Stmt: "SELECT true;"},
					},
					Start: log.Start,
					End:   log.Start.Add(time.Millisecond * 2),
				},
			},
			Applied: []string{
				"CREATE TABLE t3 (id int NOT NULL);",
			},
		},
	}
	require.NoError(t, cmdlog.MigrateApplyTemplate.Execute(&b, log))
	require.Equal(t, `Migrating to version 20240116000003 from 20240116000001 (2 migrations in total):

  -- migrating version 20240116000002
    -> CREATE TABLE t1 (id int NOT NULL);
    -> CREATE TABLE t2 (id int NOT NULL);
  -- ok (1ms)

  -- checks before migrating version 20240116000003
    -> SELECT 1;
    -> SELECT true;
  -- ok (2ms)

  -- migrating version 20240116000003
    -> CREATE TABLE t3 (id int NOT NULL);
  -- ok (2ms)

  -------------------------
  -- 10ms
  -- 2 migrations
  -- 2 checks
  -- 3 sql statements
`, b.String())

	// Multiple files with check errors.
	b.Reset()
	log.Applied = []*cmdlog.AppliedFile{
		{
			File:  files[1],
			Start: log.Start,
			End:   log.Start.Add(time.Millisecond),
			Applied: []string{
				"CREATE TABLE t1 (id int NOT NULL);",
				"CREATE TABLE t2 (id int NOT NULL);",
			},
		},
		{
			File:  files[2],
			Start: log.Start,
			End:   log.Start.Add(time.Millisecond * 2),
			Checks: []*cmdlog.FileChecks{
				{
					Stmts: []*cmdlog.Check{
						{Stmt: "SELECT 1;"},
						{Stmt: "SELECT false;", Error: new(string)},
					},
					Start: log.Start,
					End:   log.Start.Add(time.Millisecond * 2),
					Error: &cmdlog.StmtError{Text: "assertion failure"},
				},
			},
			Error: &cmdlog.StmtError{
				Text: "assertion failure",
			},
		},
	}
	require.NoError(t, cmdlog.MigrateApplyTemplate.Execute(&b, log))
	require.Equal(t, `Migrating to version 20240116000003 from 20240116000001 (2 migrations in total):

  -- migrating version 20240116000002
    -> CREATE TABLE t1 (id int NOT NULL);
    -> CREATE TABLE t2 (id int NOT NULL);
  -- ok (1ms)

  -- checks before migrating version 20240116000003
    -> SELECT 1;
    -> SELECT false;
    assertion failure

  -------------------------
  -- 10ms
  -- 1 migration ok, 1 with errors
  -- 1 check ok, 1 failure
  -- 2 sql statements
`, b.String())

	// Multiple files with multiple checks.
	b.Reset()
	log.Applied = []*cmdlog.AppliedFile{
		{
			File:  files[1],
			Start: log.Start,
			End:   log.Start.Add(time.Millisecond),
			Applied: []string{
				"CREATE TABLE t1 (id int NOT NULL);",
				"CREATE TABLE t2 (id int NOT NULL);",
			},
		},
		{
			File:  files[2],
			Start: log.Start,
			End:   log.Start.Add(time.Millisecond * 2),
			Checks: []*cmdlog.FileChecks{
				{
					Name: "checks/1",
					Stmts: []*cmdlog.Check{
						{Stmt: "SELECT 1;"},
						{Stmt: "SELECT true;"},
					},
					Start: log.Start,
					End:   log.Start.Add(time.Millisecond * 2),
				},
				{
					Name: "checks/2",
					Stmts: []*cmdlog.Check{
						{Stmt: "SELECT 1;"},
						{Stmt: "SELECT false;", Error: new(string)},
					},
					Start: log.Start,
					End:   log.Start.Add(time.Millisecond * 2),
					Error: &cmdlog.StmtError{Text: "assertion failure"},
				},
			},
			Error: &cmdlog.StmtError{
				Text: "assertion failure",
			},
		},
	}
	require.NoError(t, cmdlog.MigrateApplyTemplate.Execute(&b, log))
	require.Equal(t, `Migrating to version 20240116000003 from 20240116000001 (2 migrations in total):

  -- migrating version 20240116000002
    -> CREATE TABLE t1 (id int NOT NULL);
    -> CREATE TABLE t2 (id int NOT NULL);
  -- ok (1ms)

  -- checks before migrating version 20240116000003
    -> SELECT 1;
    -> SELECT true;
  -- ok (2ms)

  -- checks before migrating version 20240116000003
    -> SELECT 1;
    -> SELECT false;
    assertion failure

  -------------------------
  -- 10ms
  -- 1 migration ok, 1 with errors
  -- 3 checks ok, 1 failure
  -- 2 sql statements
`, b.String())

	// Error during plan stage.
	b.Reset()
	log.Applied = nil
	log.Error = `sql/migrate: scanning statements from "20240116000003.sql": 5:115: unclosed quote '\''`
	require.NoError(t, cmdlog.MigrateApplyTemplate.Execute(&b, log))
	require.Equal(t, `Migrating to version 20240116000003 from 20240116000001 (2 migrations in total):

    sql/migrate: scanning statements from "20240116000003.sql": 5:115: unclosed quote '\''

  -------------------------
  -- 10ms
`, b.String())
}

func TestReporter_Status(t *testing.T) {
	var (
		buf strings.Builder
		ctx = context.Background()
	)

	// Clean.
	dir, err := migrate.NewLocalDir(filepath.Join("../migrate/testdata", "broken"))
	require.NoError(t, err)
	c, err := sqlclient.Open(ctx, "sqlite://?mode=memory")
	require.NoError(t, err)
	defer c.Close()
	rr := &cmdlog.StatusReporter{Client: c, Dir: dir}
	report, err := rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: No migration applied yet
  -- Next Version:    1
  -- Executed Files:  0
  -- Pending Files:   3
`, buf.String())

	// Applied one.
	buf.Reset()
	rrw, err := cmdmigrate.NewEntRevisions(ctx, c)
	require.NoError(t, err)
	require.NoError(t, rrw.Migrate(ctx))
	ex, err := migrate.NewExecutor(c.Driver, dir, rrw)
	require.NoError(t, err)
	require.NoError(t, ex.ExecuteN(ctx, 1))
	rr = &cmdlog.StatusReporter{Client: c, Dir: dir}
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: 1
  -- Next Version:    2
  -- Executed Files:  1
  -- Pending Files:   2
`, buf.String())

	// Applied two.
	buf.Reset()
	require.NoError(t, err)
	require.NoError(t, ex.ExecuteN(ctx, 1))
	rr = &cmdlog.StatusReporter{Client: c, Dir: dir}
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: 2
  -- Next Version:    3
  -- Executed Files:  2
  -- Pending Files:   1
`, buf.String())

	// Partial three.
	buf.Reset()
	require.NoError(t, err)
	require.Error(t, ex.ExecuteN(ctx, 1))
	rr = &cmdlog.StatusReporter{Client: c, Dir: dir}
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: 3 (1 statements applied)
  -- Next Version:    3 (1 statements left)
  -- Executed Files:  3 (last one partially)
  -- Pending Files:   1

Last migration attempt had errors:
  -- SQL:   THIS LINE ADDS A SYNTAX ERROR;
  -- ERROR: near "THIS": syntax error
`, buf.String())

	// Fixed three - okay.
	buf.Reset()
	dir2, err := migrate.NewLocalDir(filepath.Join("../migrate/testdata", "fixed"))
	require.NoError(t, err)
	*dir = *dir2
	require.NoError(t, err)
	require.NoError(t, ex.ExecuteN(ctx, 1))
	rr = &cmdlog.StatusReporter{Client: c, Dir: dir}
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: OK
  -- Current Version: 3
  -- Next Version:    Already at latest version
  -- Executed Files:  3
  -- Pending Files:   0
`, buf.String())
}

func TestReporter_OutOfOrder(t *testing.T) {
	var (
		buf strings.Builder
		ctx = context.Background()
	)
	dir, err := migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, dir.WriteFile("1.sql", []byte("create table t1(c int);")))
	require.NoError(t, dir.WriteFile("2.sql", []byte("create table t2(c int);")))
	sum, err := dir.Checksum()
	require.NoError(t, err)
	require.NoError(t, migrate.WriteSumFile(dir, sum))
	c, err := sqlclient.Open(ctx, "sqlite://?mode=memory")
	require.NoError(t, err)
	defer c.Close()
	rr := &cmdlog.StatusReporter{Client: c, Dir: dir}

	rrw, err := cmdmigrate.NewEntRevisions(ctx, c)
	require.NoError(t, err)
	require.NoError(t, rrw.Migrate(ctx))
	report, err := rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: No migration applied yet
  -- Next Version:    1
  -- Executed Files:  0
  -- Pending Files:   2
`, buf.String())

	ex, err := migrate.NewExecutor(c.Driver, dir, rrw)
	require.NoError(t, err)
	require.NoError(t, ex.ExecuteN(ctx, 2))

	// One file was added out of order.
	buf.Reset()
	require.NoError(t, dir.WriteFile("1.5.sql", []byte("create table t1_5(c int);")))
	sum, err = dir.Checksum()
	require.NoError(t, err)
	require.NoError(t, migrate.WriteSumFile(dir, sum))
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: 2
  -- Next Version:    UNKNOWN
  -- Executed Files:  2
  -- Pending Files:   1 (out of order)

  ERROR: migration file 1.5.sql was added out of order. See: https://atlasgo.io/versioned/apply#non-linear-error
`, buf.String())

	// Multiple files were added our of order.
	buf.Reset()
	require.NoError(t, dir.WriteFile("1.6.sql", []byte("create table t1_6(c int);")))
	sum, err = dir.Checksum()
	require.NoError(t, err)
	require.NoError(t, migrate.WriteSumFile(dir, sum))
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: 2
  -- Next Version:    UNKNOWN
  -- Executed Files:  2
  -- Pending Files:   2 (out of order)

  ERROR: migration files 1.5.sql, 1.6.sql were added out of order. See: https://atlasgo.io/versioned/apply#non-linear-error
`, buf.String())

	// A mix of pending and out of order files.
	buf.Reset()
	require.NoError(t, dir.WriteFile("3.sql", []byte("create table t3(c int);")))
	sum, err = dir.Checksum()
	require.NoError(t, err)
	require.NoError(t, migrate.WriteSumFile(dir, sum))
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: 2
  -- Next Version:    UNKNOWN
  -- Executed Files:  2
  -- Pending Files:   3 (2 out of order)

  ERROR: migration files 1.5.sql, 1.6.sql were added out of order. See: https://atlasgo.io/versioned/apply#non-linear-error
`, buf.String())
}

func TestWarnOnce(t *testing.T) {
	b := &strings.Builder{}
	require.NoError(t, cmdlog.WarnOnce(b, "one"))
	require.NoError(t, cmdlog.WarnOnce(b, "two"))
	require.Equal(t, "one", b.String())

	var wg sync.WaitGroup
	b = &strings.Builder{}
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			require.NoError(t, cmdlog.WarnOnce(b, "one"))
		}()
	}
	wg.Wait()
	require.Equal(t, "one", b.String())

	// Ensure the type is not assignable.
	require.Panics(t, func() {
		var m sync.Map
		m.LoadOrStore(unassignable{}, "text")
	})
	b.Reset()
	require.NoError(t, cmdlog.WarnOnce(unassignable{Writer: b}, "done"))
	require.Equal(t, "done", b.String())
}

type unassignable struct {
	_ func()
	io.Writer
}

func (a unassignable) Write(p []byte) (n int, err error) { return a.Writer.Write(p) }
