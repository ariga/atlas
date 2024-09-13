// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlitecheck_test

import (
	"context"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"
	"ariga.io/atlas/sql/sqlite"
	_ "ariga.io/atlas/sql/sqlite/sqlitecheck"

	"github.com/stretchr/testify/require"
)

func TestDetectModifyTable(t *testing.T) {
	var (
		report *sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{
				Driver: func() migrate.Driver {
					drv := &sqlite.Driver{}
					drv.Differ = sqlite.DefaultDiff
					return drv
				}(),
			},
			File: &sqlcheck.File{
				File: testFile{name: "1.sql"},
				Changes: []*sqlcheck.Change{
					// A real drop.
					{
						Stmt: &migrate.Stmt{
							Text: "DROP TABLE `users`",
						},
						Changes: schema.Changes{
							&schema.DropTable{
								T: schema.NewTable("users").
									SetSchema(schema.New("main")),
							},
						},
					},
					// Table modification using a temporary table.
					{
						Stmt: &migrate.Stmt{
							Text: "PRAGMA foreign_keys = off;",
						},
					},
					{
						Stmt: &migrate.Stmt{
							Text: "CREATE TABLE `new_posts` (`text` text NOT NULL);",
						},
						Changes: schema.Changes{
							&schema.AddTable{
								T: schema.NewTable("new_posts").
									SetSchema(schema.New("main")).
									AddColumns(schema.NewStringColumn("text", "text")),
							},
						},
					},
					{
						Stmt: &migrate.Stmt{
							Text: "INSERT INTO `new_posts` (`text`) SELECT `text` FROM `posts`;",
						},
					},
					{
						Stmt: &migrate.Stmt{
							Text: "DROP TABLE `posts`",
						},
						Changes: schema.Changes{
							&schema.DropTable{
								T: schema.NewTable("posts").
									SetSchema(schema.New("main")).
									AddColumns(
										schema.NewNullStringColumn("text", "text"),
										schema.NewTimeColumn("posted_at", "timestamp"),
									),
							},
						},
					},
					{
						Stmt: &migrate.Stmt{
							Text: "ALTER TABLE `new_posts` RENAME TO `posts`;",
						},
						Changes: schema.Changes{
							&schema.DropTable{
								T: schema.NewTable("new_posts").
									SetSchema(schema.New("main")).
									AddColumns(schema.NewStringColumn("text", "text")),
							},
							&schema.AddTable{
								T: schema.NewTable("posts").
									SetSchema(schema.New("main")).
									AddColumns(schema.NewStringColumn("text", "text")),
							},
						},
					},
					{
						Stmt: &migrate.Stmt{
							Text: "PRAGMA foreign_keys = on;",
						},
					},
					// Another real drop.
					{
						Stmt: &migrate.Stmt{
							Text: "DROP TABLE `pets`",
						},
						Changes: schema.Changes{
							&schema.DropTable{
								T: schema.NewTable("pets").
									SetSchema(schema.New("main")),
							},
						},
					},
				},
			},
			Reporter: sqlcheck.ReportWriterFunc(func(r sqlcheck.Report) {
				report = &r
			}),
		}
	)
	azs, err := sqlcheck.AnalyzerFor(sqlite.DriverName, nil)
	require.NoError(t, err)
	require.Len(t, azs, 6)
	require.NoError(t, azs[0].Analyze(context.Background(), pass))
	err = azs[1].Analyze(context.Background(), pass)
	require.EqualError(t, err, "destructive changes detected")

	require.Equal(t, report.Text, "destructive changes detected")
	require.Len(t, report.Diagnostics, 3)
	require.Equal(t, report.Diagnostics[0].Text, `Dropping table "users"`)
	require.Equal(t, report.Diagnostics[1].Text, `Dropping non-virtual column "posted_at"`)
	require.Equal(t, report.Diagnostics[2].Text, `Dropping table "pets"`)

	require.NoError(t, azs[2].Analyze(context.Background(), pass))
	require.Equal(t, report.Text, "data dependent changes detected")
	require.Len(t, report.Diagnostics, 1)
	require.Equal(t, report.Diagnostics[0].Text, `Modifying nullable column "text" to non-nullable without default value might fail in case it contains NULL values`)
}

type testFile struct {
	name string
	migrate.File
}

func (t testFile) Name() string {
	return t.name
}
