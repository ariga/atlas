// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package condrop_test

import (
	"context"
	"testing"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlcheck/condrop"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/stretchr/testify/require"
)

func TestAnalyzer_DropForeignKey(t *testing.T) {
	var (
		report sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{Name: "mysql"},
			File: &sqlcheck.File{
				File: testFile{name: "1.sql"},
				Changes: []*sqlcheck.Change{
					{
						Stmt: &migrate.Stmt{
							Text: "ALTER TABLE `pets`",
						},
						Changes: []schema.Change{
							&schema.ModifyTable{
								T: schema.NewTable("pets").
									SetSchema(schema.New("test")),
								Changes: schema.Changes{
									&schema.DropColumn{
										C: schema.NewColumn("c"),
									},
									&schema.DropForeignKey{
										F: schema.NewForeignKey("owner_id").
											AddColumns(schema.NewColumn("owner_id")),
									},
									&schema.DropForeignKey{
										F: schema.NewForeignKey("c").
											AddColumns(schema.NewColumn("c"), schema.NewColumn("d")),
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
	az, err := condrop.New(&schemahcl.Resource{})
	require.NoError(t, err)
	err = az.Analyze(context.Background(), pass)
	require.NoError(t, err)
	require.Len(t, report.Diagnostics, 1)
	require.Equal(t, "constraint deletion detected", report.Text)
	require.Equal(t, `Dropping foreign-key constraint "owner_id"`, report.Diagnostics[0].Text)
}

type testFile struct {
	name string
	migrate.File
}

func (t testFile) Name() string {
	return t.name
}
