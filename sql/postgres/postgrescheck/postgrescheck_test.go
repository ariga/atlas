// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgrescheck_test

import (
	"context"
	"testing"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/postgres/postgrescheck"
	_ "ariga.io/atlas/sql/postgres/postgrescheck"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"

	"github.com/stretchr/testify/require"
)

func TestConcurrentIndex(t *testing.T) {
	var cfg struct {
		schemahcl.DefaultExtension
	}
	// language=hcl
	err := schemahcl.New().EvalBytes([]byte(`
concurrent_index {
  error = true
}
`), &cfg, nil)
	require.NoError(t, err)
	require.NoError(t, err)
	az, err := postgrescheck.NewConcurrentIndex(cfg.Remain())
	require.NoError(t, err)

	t.Run("MissingConcurrent", func(t *testing.T) {
		var report *sqlcheck.Report
		err := az.Analyze(context.Background(), &sqlcheck.Pass{
			File: &sqlcheck.File{
				File: migrate.NewLocalFile("1.sql", []byte("CREATE INDEX i1 ON t(c);")),
				Changes: []*sqlcheck.Change{
					{
						Changes: schema.Changes{
							&schema.ModifyTable{
								T: schema.NewTable("Users").SetSchema(schema.New("public")),
								Changes: schema.Changes{
									&schema.AddIndex{
										I: schema.NewIndex("i1"),
									},
								},
							},
						},
						Stmt: &migrate.Stmt{
							Pos:  0,
							Text: "CREATE INDEX i1 ON t(c)",
						},
					},
				},
			},
			Reporter: sqlcheck.ReportWriterFunc(func(r sqlcheck.Report) {
				report = &r
			}),
		})
		require.EqualError(t, err, "concurrent index violations detected")
		require.Len(t, report.Diagnostics, 1)
		require.Equal(t, report.Diagnostics[0].Text, `Creating index "i1" non-concurrently causes write locks on the "Users" table`)
	})

	t.Run("MissingTxMode", func(t *testing.T) {
		var report *sqlcheck.Report
		err := az.Analyze(context.Background(), &sqlcheck.Pass{
			File: &sqlcheck.File{
				File: migrate.NewLocalFile("1.sql", []byte("CREATE INDEX CONCURRENTLY i1 ON t(c);")),
				Changes: []*sqlcheck.Change{
					{
						Changes: schema.Changes{
							&schema.ModifyTable{
								T: schema.NewTable("Users").SetSchema(schema.New("public")),
								Changes: schema.Changes{
									&schema.AddIndex{
										I: schema.NewIndex("i1"),
										Extra: []schema.Clause{
											&postgres.Concurrently{},
										},
									},
								},
							},
						},
						Stmt: &migrate.Stmt{
							Pos:  0,
							Text: "CREATE INDEX CONCURRENTLY i1 ON t(c)",
						},
					},
				},
			},
			Reporter: sqlcheck.ReportWriterFunc(func(r sqlcheck.Report) {
				report = &r
			}),
		})
		require.NoError(t, err)
		require.Len(t, report.Diagnostics, 1)
		require.Equal(t, report.Diagnostics[0].Text, "Indexes cannot be created or deleted concurrently within a transaction. Add the `atlas:txmode none` directive to the header to prevent this file from running in a transaction")
	})

	t.Run("MixedReport", func(t *testing.T) {
		var report *sqlcheck.Report
		err := az.Analyze(context.Background(), &sqlcheck.Pass{
			File: &sqlcheck.File{
				File: migrate.NewLocalFile("1.sql", []byte("CREATE INDEX CONCURRENTLY i1 ON t(c);DROP INDEX i2;")),
				Changes: []*sqlcheck.Change{
					{
						Changes: schema.Changes{
							&schema.ModifyTable{
								T: schema.NewTable("Users").SetSchema(schema.New("public")),
								Changes: schema.Changes{
									&schema.AddIndex{
										I: schema.NewIndex("i1"),
										Extra: []schema.Clause{
											&postgres.Concurrently{},
										},
									},
								},
							},
						},
						Stmt: &migrate.Stmt{
							Pos:  0,
							Text: "CREATE INDEX CONCURRENTLY i1 ON t(c)",
						},
					},
					{
						Changes: schema.Changes{
							&schema.ModifyTable{
								T: schema.NewTable("Users").SetSchema(schema.New("public")),
								Changes: schema.Changes{
									&schema.DropIndex{
										I: schema.NewIndex("i2"),
									},
								},
							},
						},
						Stmt: &migrate.Stmt{
							Pos:  0,
							Text: "DROP INDEX i2",
						},
					},
				},
			},
			Reporter: sqlcheck.ReportWriterFunc(func(r sqlcheck.Report) {
				report = &r
			}),
		})
		require.EqualError(t, err, "concurrent index violations detected")
		require.Len(t, report.Diagnostics, 2)
		require.Equal(t, report.Diagnostics[0].Text, "Indexes cannot be created or deleted concurrently within a transaction. Add the `atlas:txmode none` directive to the header to prevent this file from running in a transaction")
		require.Equal(t, report.Diagnostics[1].Text, `Dropping index "i2" non-concurrently causes write locks on the "Users" table`)
	})
}

func TestDataDepend_MightFail(t *testing.T) {
	var (
		report *sqlcheck.Report
		pass   = &sqlcheck.Pass{
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
										schema.NewIntColumn("a", postgres.TypeInt),
										schema.NewIntColumn("b", postgres.TypeInt),
									),
								Changes: []schema.Change{
									&schema.AddColumn{C: schema.NewTimeColumn("b", postgres.TypeInt)},
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
	azs, err := sqlcheck.AnalyzerFor(postgres.DriverName, nil)
	require.NoError(t, err)
	require.NoError(t, sqlcheck.Analyzers(azs).Analyze(context.Background(), pass))
	require.Equal(t, report.Diagnostics[0].Text, `Adding a non-nullable "int" column "b" will fail in case table "users" is not empty`)
}

type testFile struct {
	name string
	migrate.File
}

func (t testFile) Name() string {
	return t.name
}
