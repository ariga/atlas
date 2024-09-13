// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgrescheck_test

import (
	"context"
	"testing"

	"github.com/s-sokolko/atlas/sql/migrate"
	"github.com/s-sokolko/atlas/sql/postgres"
	_ "github.com/s-sokolko/atlas/sql/postgres/postgrescheck"
	"github.com/s-sokolko/atlas/sql/schema"
	"github.com/s-sokolko/atlas/sql/sqlcheck"

	"github.com/stretchr/testify/require"
)

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
