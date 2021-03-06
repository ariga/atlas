// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package destructive_test

import (
	"context"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlcheck/destructive"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/stretchr/testify/require"
)

func TestAnalyzer_DropTable(t *testing.T) {
	var (
		report *sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{},
			File: &sqlcheck.File{
				File: testFile{name: "1.sql"},
				Changes: []*sqlcheck.Change{
					{
						Stmt: "DROP TABLE `users`",
						Changes: schema.Changes{
							&schema.DropTable{
								T: schema.NewTable("users").
									SetSchema(schema.New("test")),
							},
						},
					},
					{
						Stmt: "DROP TABLE `posts`",
						Changes: schema.Changes{
							&schema.DropTable{
								T: schema.NewTable("posts").
									SetSchema(schema.New("test")),
							},
						},
					},
					{
						Stmt: "CREATE TABLE `posts`",
						Changes: schema.Changes{
							&schema.AddTable{
								T: schema.NewTable("posts").
									SetSchema(schema.New("test")),
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
	az := destructive.New(destructive.Options{})
	err := az.Analyze(context.Background(), pass)
	require.NoError(t, err)
	require.Equal(t, `Destructive changes detected in file 1.sql`, report.Text)
	require.Len(t, report.Diagnostics, 2)
	require.Equal(t, `Dropping table "users"`, report.Diagnostics[0].Text)
	require.Equal(t, `Dropping table "posts"`, report.Diagnostics[1].Text)
}

func TestAnalyzer_SkipTemporaryTable(t *testing.T) {
	var (
		report *sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{},
			File: &sqlcheck.File{
				File: testFile{name: "1.sql"},
				Changes: []*sqlcheck.Change{
					{
						Stmt: "DROP TABLE `users`",
						Changes: schema.Changes{
							&schema.AddTable{
								T: schema.NewTable("users").
									SetSchema(schema.New("test")),
							},
							&schema.DropTable{
								T: schema.NewTable("users").
									SetSchema(schema.New("test")),
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
	az := destructive.New(destructive.Options{})
	err := az.Analyze(context.Background(), pass)
	require.NoError(t, err)
	require.Nil(t, report, "no report")
}

func TestAnalyzer_DropSchema(t *testing.T) {
	var (
		report *sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{},
			File: &sqlcheck.File{
				File: testFile{name: "1.sql"},
				Changes: []*sqlcheck.Change{
					{
						Stmt: "DROP SCHEMA `test`",
						Changes: schema.Changes{
							&schema.DropSchema{
								S: schema.New("test").
									AddTables(
										schema.NewTable("users"),
										schema.NewTable("orders"),
									),
							},
						},
					},
					{
						Stmt: "DROP SCHEMA `market`",
						Changes: schema.Changes{
							&schema.DropSchema{
								S: schema.New("market"),
							},
						},
					},
					{
						Stmt: "CREATE DATABASE `market`",
						Changes: schema.Changes{
							&schema.AddSchema{
								S: schema.New("market"),
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
	az := destructive.New(destructive.Options{})
	err := az.Analyze(context.Background(), pass)
	require.NoError(t, err)
	require.Equal(t, `Destructive changes detected in file 1.sql`, report.Text)
	require.Len(t, report.Diagnostics, 2)
	require.Equal(t, `Dropping non-empty schema "test" with 2 tables`, report.Diagnostics[0].Text)
	require.Equal(t, `Dropping schema "market"`, report.Diagnostics[1].Text)
}

func TestAnalyzer_DropColumn(t *testing.T) {
	var (
		report sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{Name: "mysql"},
			File: &sqlcheck.File{
				File: testFile{name: "1.sql"},
				Changes: []*sqlcheck.Change{
					{
						Stmt: "ALTER TABLE `pets`",
						Changes: []schema.Change{
							&schema.ModifyTable{
								T: schema.NewTable("pets").
									SetSchema(schema.New("test")),
								Changes: schema.Changes{
									&schema.DropColumn{
										C: schema.NewColumn("c").
											SetGeneratedExpr(&schema.GeneratedExpr{Type: "STORED"}),
									},
								},
							},
						},
					},
					{
						Stmt: "ALTER TABLE `pets`",
						Changes: []schema.Change{
							&schema.ModifySchema{
								S: schema.New("test"),
								Changes: schema.Changes{
									&schema.ModifyAttr{
										From: &schema.Charset{V: "utf8"},
										To:   &schema.Charset{V: "latin1"},
									},
								},
							},
						},
					},
				},
			},
			Reporter: sqlcheck.ReportWriterFunc(func(r sqlcheck.Report) {
				report = r
			}),
		}
	)
	az := destructive.New(destructive.Options{})
	err := az.Analyze(context.Background(), pass)
	require.NoError(t, err)
	require.Len(t, report.Diagnostics, 1)
	require.Equal(t, `Dropping non-virtual column "c"`, report.Diagnostics[0].Text)
}

func TestAnalyzer_Options(t *testing.T) {
	var (
		off  bool
		pass = &sqlcheck.Pass{
			Dev: &sqlclient.Client{Name: "mysql"},
			File: &sqlcheck.File{
				File: testFile{name: "1.sql"},
				Changes: []*sqlcheck.Change{
					{
						Stmt: "DROP DATABASE `test`",
						Changes: schema.Changes{
							&schema.DropSchema{
								S: schema.New("test"),
							},
						},
					},
					{
						Stmt: "DROP TABLE `users`",
						Changes: schema.Changes{
							&schema.DropTable{
								T: schema.NewTable("users").
									SetSchema(schema.New("test")),
							},
						},
					},
					{
						Stmt: "ALTER TABLE `pets`",
						Changes: []schema.Change{
							&schema.ModifyTable{
								T: schema.NewTable("pets").
									SetSchema(schema.New("test")),
								Changes: schema.Changes{
									&schema.DropColumn{
										C: schema.NewColumn("c"),
									},
								},
							},
						},
					},
				},
			},
			Reporter: sqlcheck.ReportWriterFunc(func(r sqlcheck.Report) {
				t.Fatal("unexpected report")
			}),
		}
	)
	az := destructive.New(destructive.Options{
		DropTable:  &off,
		DropSchema: &off,
		DropColumn: &off,
	})
	err := az.Analyze(context.Background(), pass)
	require.NoError(t, err)
}

type testFile struct {
	name string
	migrate.File
}

func (t testFile) Name() string {
	return t.name
}
