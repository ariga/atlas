// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package incompatible_test

import (
	"context"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlcheck/incompatible"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/stretchr/testify/require"
)

func TestAnalyzer_RenameColumn(t *testing.T) {
	var (
		report *sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{},
			File: &sqlcheck.File{
				File: testFile{name: "1.sql"},
				Changes: []*sqlcheck.Change{
					{
						Stmt: &migrate.Stmt{
							Text: "CREATE TABLE `users` (`id` int)",
						},
						Changes: schema.Changes{
							&schema.AddTable{
								T: schema.NewTable("users").
									SetSchema(schema.New("test")).
									AddColumns(schema.NewIntColumn("id", "int")),
							},
						},
					},
					// Skip column that was added in the same file.
					{
						Stmt: &migrate.Stmt{
							Text: "ALTER TABLE `users` RENAME COLUMN `id` TO `uid`",
						},
						Changes: schema.Changes{
							&schema.ModifyTable{
								T: schema.NewTable("users").
									SetSchema(schema.New("test")).
									AddColumns(schema.NewIntColumn("uid", "int")),
								Changes: schema.Changes{
									&schema.RenameColumn{
										From: schema.NewIntColumn("id", "int"),
										To:   schema.NewIntColumn("uid", "int"),
									},
								},
							},
						},
					},
					// Skip if a new generated column was added with the previous name.
					{
						Stmt: &migrate.Stmt{
							Text: "ALTER TABLE `pets` RENAME COLUMN `id` TO `uid`",
						},
						Changes: schema.Changes{
							&schema.ModifyTable{
								T: schema.NewTable("pets").
									SetSchema(schema.New("test")).
									AddColumns(
										schema.NewIntColumn("uid", "int"),
										schema.NewIntColumn("id", "int").
											SetGeneratedExpr(&schema.GeneratedExpr{
												Expr: "uid", // point to the renamed column.
											}),
									),
								Changes: schema.Changes{
									&schema.RenameColumn{
										From: schema.NewIntColumn("id", "int"),
										To:   schema.NewIntColumn("uid", "int"),
									},
									&schema.AddColumn{
										C: schema.NewIntColumn("id", "int").
											SetGeneratedExpr(&schema.GeneratedExpr{
												Expr: "uid", // point to the renamed column.
											}),
									},
								},
							},
						},
					},
					// Skip if a new column was added with the previous name in consecutive statement.
					{
						Stmt: &migrate.Stmt{
							Text: "ALTER TABLE `pets` RENAME COLUMN `id` TO `uid`;ALTER TABLE `pets` ADD COLUMN `id` int;",
						},
						Changes: schema.Changes{
							&schema.ModifyTable{
								T: schema.NewTable("pets").
									SetSchema(schema.New("test")).
									AddColumns(
										schema.NewIntColumn("uid", "int"),
										schema.NewIntColumn("id", "int").
											SetGeneratedExpr(&schema.GeneratedExpr{
												Expr: "uid", // point to the renamed column.
											}),
									),
								Changes: schema.Changes{
									&schema.RenameColumn{
										From: schema.NewIntColumn("id", "int"),
										To:   schema.NewIntColumn("uid", "int"),
									},
								},
							},
							&schema.ModifyTable{
								T: schema.NewTable("pets").
									SetSchema(schema.New("test")).
									AddColumns(
										schema.NewIntColumn("uid", "int"),
										schema.NewIntColumn("id", "int").
											SetGeneratedExpr(&schema.GeneratedExpr{
												Expr: "uid", // point to the renamed column.
											}),
									),
								Changes: schema.Changes{
									&schema.AddColumn{
										C: schema.NewIntColumn("id", "int").
											SetGeneratedExpr(&schema.GeneratedExpr{
												Expr: "uid", // point to the renamed column.
											}),
									},
								},
							},
						},
					},
					// Detect rename.
					{
						Stmt: &migrate.Stmt{
							Text: "ALTER TABLE `cards` RENAME COLUMN `id` TO `uid`",
						},
						Changes: schema.Changes{
							&schema.ModifyTable{
								T: schema.NewTable("cards").
									SetSchema(schema.New("test")).
									AddColumns(
										schema.NewIntColumn("uid", "int"),
									),
								Changes: schema.Changes{
									&schema.RenameColumn{
										From: schema.NewIntColumn("id", "int"),
										To:   schema.NewIntColumn("uid", "int"),
									},
								},
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
	az, err := incompatible.New(nil)
	require.NoError(t, err)
	err = az.Analyze(context.Background(), pass)
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Len(t, report.Diagnostics, 1)
	require.Equal(t, "BC102", report.Diagnostics[0].Code)
	require.Equal(t, `Renaming column "id" to "uid"`, report.Diagnostics[0].Text)
}

func TestAnalyzer_RenameTable(t *testing.T) {
	var (
		report *sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{},
			File: &sqlcheck.File{
				File: testFile{
					name: "1.sql",
					stmts: []*migrate.Stmt{
						{Text: "CREATE TABLE `users` (`id` int)", Pos: 1},
						{Text: "ALTER TABLE `users` RENAME TO `Users`", Pos: 2},
						{Text: "ALTER TABLE `pets` RENAME TO `Pets`", Pos: 3},
						{Text: "ALTER TABLE `cards` RENAME TO `Cards`", Pos: 4},
						{Text: "CREATE VIEW `cards` AS SELECT * FROM `Cards`", Pos: 5},
					},
				},
				Parser: testParser{matchOn: "cards"},
				Changes: []*sqlcheck.Change{
					{
						Stmt: &migrate.Stmt{
							Text: "CREATE TABLE `users` (`id` int)",
						},
						Changes: schema.Changes{
							&schema.AddTable{
								T: schema.NewTable("users").
									SetSchema(schema.New("test")).
									AddColumns(schema.NewIntColumn("id", "int")),
							},
						},
					},
					// Skip table that was added in the same file.
					{
						Stmt: &migrate.Stmt{
							Text: "ALTER TABLE `users` RENAME TO `Users`",
						},
						Changes: schema.Changes{
							&schema.RenameTable{
								From: schema.NewTable("users").
									SetSchema(schema.New("test")),
								To: schema.NewTable("Users").
									SetSchema(schema.New("test")),
							},
						},
					},
					// Detect rename.
					{
						Stmt: &migrate.Stmt{
							Text: "ALTER TABLE `pets` RENAME TO `Pets`",
						},
						Changes: schema.Changes{
							&schema.RenameTable{
								From: schema.NewTable("pets").
									SetSchema(schema.New("test")),
								To: schema.NewTable("Pets").
									SetSchema(schema.New("test")),
							},
						},
					},
					// Skip rename if a view was created.
					{
						Stmt: &migrate.Stmt{
							Text: "ALTER TABLE `cards` RENAME TO `Cards`",
							Pos:  4,
						},
						Changes: schema.Changes{
							&schema.RenameTable{
								From: schema.NewTable("cards").
									SetSchema(schema.New("test")),
								To: schema.NewTable("Cards").
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
	az, err := incompatible.New(nil)
	require.NoError(t, err)
	err = az.Analyze(context.Background(), pass)
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Len(t, report.Diagnostics, 1)
	require.Equal(t, "BC101", report.Diagnostics[0].Code)
	require.Equal(t, `Renaming table "pets" to "Pets"`, report.Diagnostics[0].Text)
}

type testFile struct {
	name  string
	stmts []*migrate.Stmt
	migrate.File
}

func (t testFile) Name() string {
	return t.name
}

func (t testFile) StmtDecls() ([]*migrate.Stmt, error) {
	return t.stmts, nil
}

type testParser struct{ matchOn string }

func (t testParser) CreateViewAfter(_ migrate.File, old, _ string, _ int) (bool, error) {
	return t.matchOn == old, nil
}
