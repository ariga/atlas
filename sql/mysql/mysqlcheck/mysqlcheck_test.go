// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysqlcheck_test

import (
	"context"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/mysql"
	_ "ariga.io/atlas/sql/mysql/mysqlcheck"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/stretchr/testify/require"
)

func TestDataDepend_MySQL_ImplicitUpdate(t *testing.T) {
	var (
		report *sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{
				Name:   "mysql",
				Driver: &mysql.Driver{},
			},
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
										schema.NewIntColumn("a", mysql.TypeInt),
										schema.NewIntColumn("b", mysql.TypeInt),
										schema.NewFloatColumn("c", mysql.TypeFloat),
										schema.NewStringColumn("d", mysql.TypeVarchar, schema.StringSize(10)),
										schema.NewEnumColumn("e", schema.EnumValues("foo", "bar")),
										schema.NewTimeColumn("f", mysql.TypeTimestamp),
									),
								Changes: []schema.Change{
									&schema.AddColumn{C: schema.NewIntColumn("b", mysql.TypeInt)},
									&schema.AddColumn{C: schema.NewFloatColumn("c", mysql.TypeFloat)},
									&schema.AddColumn{C: schema.NewStringColumn("d", mysql.TypeVarchar, schema.StringSize(10))},
									&schema.AddColumn{C: schema.NewEnumColumn("e", schema.EnumValues("foo", "bar"))},
									&schema.AddColumn{C: schema.NewTimeColumn("f", mysql.TypeTimestamp)},
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
	azs, err := sqlcheck.AnalyzerFor(mysql.DriverName, nil)
	require.NoError(t, err)
	require.NoError(t, sqlcheck.Analyzers(azs).Analyze(context.Background(), pass))
	require.Equal(t, report.Diagnostics[0].Text, `Adding a non-nullable "int" column "b" on table "users" without a default value implicitly sets existing rows with 0`)
	require.Equal(t, report.Diagnostics[1].Text, `Adding a non-nullable "float" column "c" on table "users" without a default value implicitly sets existing rows with 0`)
	require.Equal(t, report.Diagnostics[2].Text, `Adding a non-nullable "varchar" column "d" on table "users" without a default value implicitly sets existing rows with ""`)
	require.Equal(t, report.Diagnostics[3].Text, `Adding a non-nullable "enum" column "e" on table "users" without a default value implicitly sets existing rows with "foo"`)
	require.Equal(t, report.Diagnostics[4].Text, `Adding a non-nullable "timestamp" column "f" on table "users" without a default value implicitly sets existing rows with CURRENT_TIMESTAMP`)
}

func TestDataDepend_MySQL8_ImplicitUpdate(t *testing.T) {
	drv := &mysql.Driver{}
	drv.V = "8.0.19"
	var (
		report *sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{
				Name:   "mysql",
				Driver: drv,
			},
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
										schema.NewIntColumn("a", mysql.TypeInt),
										schema.NewTimeColumn("b", mysql.TypeTimestamp),
									),
								Changes: []schema.Change{
									&schema.AddColumn{C: schema.NewTimeColumn("b", mysql.TypeTimestamp)},
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
	azs, err := sqlcheck.AnalyzerFor(mysql.DriverName, nil)
	require.NoError(t, err)
	require.NoError(t, sqlcheck.Analyzers(azs).Analyze(context.Background(), pass))
	require.Equal(t,
		report.Diagnostics[0].Text,
		`Adding a non-nullable "timestamp" column "b" on table "users" without a default value implicitly sets existing rows with 0000-00-00 00:00:00`,
		"explicit_defaults_for_timestamp is enabled by default for versions >= 8.0.2",
	)
}

func TestDataDepend_MySQL_MightFail(t *testing.T) {
	var (
		report *sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{
				Name:   "mysql",
				Driver: &mysql.Driver{},
			},
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
										schema.NewIntColumn("a", mysql.TypeInt),
										schema.NewTimeColumn("b", mysql.TypeDate),
										schema.NewTimeColumn("c", mysql.TypeDateTime),
										schema.NewSpatialColumn("d", mysql.TypePoint),
									),
								Changes: []schema.Change{
									&schema.AddColumn{C: schema.NewTimeColumn("b", mysql.TypeDate)},
									&schema.AddColumn{C: schema.NewTimeColumn("c", mysql.TypeDateTime)},
									&schema.AddColumn{C: schema.NewSpatialColumn("d", mysql.TypePoint)},
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
	azs, err := sqlcheck.AnalyzerFor(mysql.DriverName, nil)
	require.NoError(t, err)
	require.NoError(t, sqlcheck.Analyzers(azs).Analyze(context.Background(), pass))
	require.Equal(t, report.Diagnostics[0].Text, `Adding a non-nullable "date" column "b" will fail in case table "users" is not empty`)
	require.Equal(t, report.Diagnostics[1].Text, `Adding a non-nullable "datetime" column "c" will fail in case table "users" is not empty`)
	require.Equal(t, report.Diagnostics[2].Text, `Adding a non-nullable "point" column "d" will fail in case table "users" is not empty`)
}

func TestDataDepend_Maria_ImplicitUpdate(t *testing.T) {
	drv := &mysql.Driver{}
	drv.V = "10.7.1-MariaDB"
	var (
		report *sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{
				Name:   "mysql",
				Driver: drv,
			},
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
										schema.NewIntColumn("a", mysql.TypeInt),
										schema.NewIntColumn("b", mysql.TypeText),
										schema.NewJSONColumn("c", mysql.TypeJSON),
										schema.NewTimeColumn("d", mysql.TypeDate),
										schema.NewTimeColumn("e", mysql.TypeDateTime),
										schema.NewSpatialColumn("f", mysql.TypePoint),
										schema.NewTimeColumn("first_ts", mysql.TypeTimestamp),
										schema.NewTimeColumn("second_ts", mysql.TypeTimestamp),
									),
								Changes: []schema.Change{
									&schema.AddColumn{C: schema.NewStringColumn("b", mysql.TypeText)},
									&schema.AddColumn{C: schema.NewJSONColumn("c", mysql.TypeJSON)},
									&schema.AddColumn{C: schema.NewTimeColumn("d", mysql.TypeDate)},
									&schema.AddColumn{C: schema.NewTimeColumn("e", mysql.TypeDateTime)},
									&schema.AddColumn{C: schema.NewSpatialColumn("f", mysql.TypePoint)},
									&schema.AddColumn{C: schema.NewTimeColumn("first_ts", mysql.TypeTimestamp)},
									&schema.AddColumn{C: schema.NewTimeColumn("second_ts", mysql.TypeTimestamp)},
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
	azs, err := sqlcheck.AnalyzerFor(mysql.DriverName, nil)
	require.NoError(t, err)
	require.NoError(t, sqlcheck.Analyzers(azs).Analyze(context.Background(), pass))
	require.Equal(t, report.Diagnostics[0].Text, `Adding a non-nullable "text" column "b" on table "users" without a default value implicitly sets existing rows with ""`)
	require.Equal(t, report.Diagnostics[1].Text, `Adding a non-nullable "json" column "c" on table "users" without a default value implicitly sets existing rows with ""`)
	require.Equal(t, report.Diagnostics[2].Text, `Adding a non-nullable "date" column "d" on table "users" without a default value implicitly sets existing rows with 00:00:00`)
	require.Equal(t, report.Diagnostics[3].Text, `Adding a non-nullable "datetime" column "e" on table "users" without a default value implicitly sets existing rows with 00:00:00`)
	require.Equal(t, report.Diagnostics[4].Text, `Adding a non-nullable "point" column "f" on table "users" without a default value implicitly sets existing rows with ""`)
	require.Equal(t, report.Diagnostics[5].Text, `Adding a non-nullable "timestamp" column "first_ts" on table "users" without a default value implicitly sets existing rows with CURRENT_TIMESTAMP`)
	require.Equal(t, report.Diagnostics[6].Text, `Adding a non-nullable "timestamp" column "second_ts" on table "users" without a default value implicitly sets existing rows with 0000-00-00 00:00:00`)

}

type testFile struct {
	name string
	migrate.File
}

func (t testFile) Name() string {
	return t.name
}
