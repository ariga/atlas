// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package naming_test

import (
	"context"
	"testing"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlcheck/naming"

	"github.com/stretchr/testify/require"
)

func TestOptions(t *testing.T) {
	var cfg struct {
		schemahcl.DefaultExtension
	}
	// language=hcl
	err := schemahcl.New().EvalBytes([]byte(`
naming {
  match   = "^[a-z]+$"
  message = "must be lowercase"
  index {
    match   = "^[a-z]+_idx$"
    message = "must be lowercase and end with _idx"
  } 
}
`), &cfg, nil)
	require.NoError(t, err)
	az, err := naming.New(cfg.Remain())
	require.NoError(t, err)
	var report *sqlcheck.Report
	err = az.Analyze(context.Background(), &sqlcheck.Pass{
		File: &sqlcheck.File{
			Changes: []*sqlcheck.Change{
				{
					Changes: schema.Changes{
						&schema.AddTable{T: schema.NewTable("Users")},
					},
					Stmt: &migrate.Stmt{
						Pos:  0,
						Text: "CREATE TABLE `Users`",
					},
				},
				{
					Changes: schema.Changes{
						&schema.ModifyTable{
							T: schema.NewTable("pets"),
							Changes: schema.Changes{
								&schema.AddIndex{
									I: schema.NewIndex("pet_name"),
								},
							},
						},
					},
					Stmt: &migrate.Stmt{
						Pos:  20,
						Text: "ALTER TABLE `pets` ADD INDEX `pet_name`",
					},
				},
			},
		},
		Reporter: sqlcheck.ReportWriterFunc(func(r sqlcheck.Report) {
			report = &r
		}),
	})
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Len(t, report.Diagnostics, 2)
	require.Equal(t, `Table named "Users" violates the naming policy: must be lowercase`, report.Diagnostics[0].Text)
	require.Equal(t, `Index named "pet_name" violates the naming policy: must be lowercase and end with _idx`, report.Diagnostics[1].Text)
}
