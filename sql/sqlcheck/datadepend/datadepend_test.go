// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package datadepend_test

import (
	"context"
	"testing"

	"ariga.io/atlas/schemahcl"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlcheck/datadepend"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/stretchr/testify/require"
)

func TestAnalyzer_AddUniqueIndex(t *testing.T) {
	var (
		report *sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{},
			File: &sqlcheck.File{
				File: testFile{name: "1.sql"},
				Changes: []*sqlcheck.Change{
					{
						Stmt: &migrate.Stmt{
							Text: "ALTER TABLE users",
						},
						Changes: schema.Changes{
							&schema.ModifyTable{
								T: schema.NewTable("users").
									SetSchema(schema.New("test")).
									AddColumns(
										schema.NewColumn("a"),
										schema.NewColumn("b"),
										schema.NewColumn("c"),
										schema.NewColumn("d"),
									),
								Changes: []schema.Change{
									// Ignore new created columns.
									&schema.AddColumn{
										C: schema.NewColumn("a"),
									},
									&schema.AddIndex{
										I: schema.NewUniqueIndex("idx_a").
											AddColumns(schema.NewColumn("a")),
									},
									// Report on existing column.
									&schema.AddIndex{
										I: schema.NewUniqueIndex("idx_b").
											AddColumns(schema.NewColumn("b")),
									},
									// Report on existing columns.
									&schema.AddIndex{
										I: schema.NewUniqueIndex("idx_c_d").
											AddColumns(
												schema.NewColumn("c"),
												schema.NewColumn("d"),
											),
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
	az, err := datadepend.New(nil, datadepend.Handler{})
	require.NoError(t, err)
	err = az.Analyze(context.Background(), pass)
	require.NoError(t, err)
	require.Equal(t, "data dependent changes detected", report.Text)
	require.Len(t, report.Diagnostics, 2)
	require.Equal(t, `Adding a unique index "idx_b" on table "users" might fail in case column "b" contains duplicate entries`, report.Diagnostics[0].Text)
	require.Equal(t, `Adding a unique index "idx_c_d" on table "users" might fail in case columns "c", "d" contain duplicate entries`, report.Diagnostics[1].Text)
}

func TestAnalyzer_ModifyUniqueIndex(t *testing.T) {
	var (
		report *sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{},
			File: &sqlcheck.File{
				File: testFile{name: "1.sql"},
				Changes: []*sqlcheck.Change{
					{
						Stmt: &migrate.Stmt{
							Text: "ALTER TABLE users",
						},
						Changes: schema.Changes{
							&schema.ModifyTable{
								T: schema.NewTable("users").
									SetSchema(schema.New("test")).
									AddColumns(
										schema.NewColumn("a"),
										schema.NewColumn("b"),
									),
								Changes: []schema.Change{
									// Ignore new created columns.
									&schema.AddColumn{
										C: schema.NewColumn("a"),
									},
									&schema.ModifyIndex{
										From: schema.NewIndex("idx_a").
											AddColumns(schema.NewColumn("a")),
										To: schema.NewUniqueIndex("idx_a").
											AddColumns(schema.NewColumn("a")),
									},
									// Report on existing columns.
									&schema.ModifyIndex{
										From: schema.NewIndex("idx_b").
											AddColumns(schema.NewColumn("b")),
										To: schema.NewUniqueIndex("idx_b").
											AddColumns(schema.NewColumn("b")),
										Change: schema.ChangeUnique,
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
	az, err := datadepend.New(nil, datadepend.Handler{})
	require.NoError(t, err)
	err = az.Analyze(context.Background(), pass)
	require.NoError(t, err)
	require.Equal(t, "data dependent changes detected", report.Text)
	require.Len(t, report.Diagnostics, 1)
	require.Equal(t, `Modifying an index "idx_b" on table "users" might fail in case of duplicate entries`, report.Diagnostics[0].Text)
}

func TestAnalyzer_ModifyNullability(t *testing.T) {
	var (
		report *sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{},
			File: &sqlcheck.File{
				File: testFile{name: "1.sql"},
				Changes: []*sqlcheck.Change{
					{
						Stmt: &migrate.Stmt{
							Text: "ALTER TABLE users",
						},
						Changes: schema.Changes{
							&schema.ModifyTable{
								T: schema.NewTable("users").
									SetSchema(schema.New("test")).
									AddColumns(
										schema.NewNullIntColumn("a", "int"),
									),
								Changes: []schema.Change{
									&schema.ModifyColumn{
										From:   schema.NewNullIntColumn("a", "int"),
										To:     schema.NewIntColumn("a", "int"),
										Change: schema.ChangeNull,
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
	az, err := datadepend.New(nil, datadepend.Handler{})
	require.NoError(t, err)
	err = az.Analyze(context.Background(), pass)
	require.NoError(t, err)
	require.Equal(t, "data dependent changes detected", report.Text)
	require.Len(t, report.Diagnostics, 1)
	require.Equal(t, `Modifying nullable column "a" to non-nullable might fail in case it contains NULL values`, report.Diagnostics[0].Text)
}

func TestAnalyzer_Options(t *testing.T) {
	var (
		report *sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{},
			File: &sqlcheck.File{
				File: testFile{name: "1.sql"},
				Changes: []*sqlcheck.Change{
					{
						Stmt: &migrate.Stmt{
							Text: "ALTER TABLE users",
						},
						Changes: schema.Changes{
							&schema.ModifyTable{
								T: schema.NewTable("users").
									SetSchema(schema.New("test")).
									AddColumns(
										schema.NewColumn("a"),
										schema.NewColumn("b"),
									),
								Changes: []schema.Change{
									&schema.AddIndex{
										I: schema.NewIndex("idx_a").
											AddColumns(schema.NewColumn("a")),
									},
									&schema.ModifyIndex{
										From: schema.NewIndex("idx_b").
											AddColumns(schema.NewColumn("b")),
										To: schema.NewUniqueIndex("idx_b").
											AddColumns(schema.NewColumn("b")),
										Change: schema.ChangeUnique,
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
	az, err := datadepend.New(&schemahcl.Resource{
		Children: []*schemahcl.Resource{
			{
				Type: "data_depend",
				Attrs: []*schemahcl.Attr{
					schemahcl.BoolAttr("error", true),
				},
			},
		},
	}, datadepend.Handler{})
	require.NoError(t, err)
	err = az.Analyze(context.Background(), pass)
	require.EqualError(t, err, "data dependent changes detected")
	require.NotNil(t, report)
}

type testFile struct {
	name string
	migrate.File
}

func (t testFile) Name() string {
	return t.name
}
