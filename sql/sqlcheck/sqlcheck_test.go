// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlcheck_test

import (
	"context"
	"reflect"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/stretchr/testify/require"
)

func TestDestructive(t *testing.T) {
	sqlcheck.Destructive.Register("mysql", func(ctx context.Context, p *sqlcheck.Pass) (diags []sqlcheck.Diagnostic, _ error) {
		for _, cs := range p.File.Changes {
			for _, c := range cs.Changes {
				_, ok := c.(*schema.ModifyTable)
				if !ok {
					continue
				}
				// A fake driver-level diagnostic.
				diags = append(diags, sqlcheck.Diagnostic{Text: "modify table", Pos: cs.Pos})
			}
		}
		return
	})
	var (
		report sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{Name: "mysql"},
			File: &sqlcheck.File{
				File: testFile{name: "1.sql"},
				Changes: []*sqlcheck.Change{
					{
						Stmt: "DROP DATABASE `test`",
						Changes: schema.Changes{
							&schema.DropSchema{S: schema.New("test")},
						},
					},
					{
						Stmt: "DROP TABLE `users`",
						Changes: schema.Changes{
							&schema.DropTable{T: schema.NewTable("users")},
						},
					},
				},
			},
			Reporter: sqlcheck.ReportWriterFunc(func(r sqlcheck.Report) {
				report = r
			}),
		}
	)
	err := sqlcheck.Destructive.Analyze(context.Background(), pass)
	require.NoError(t, err)
	require.Equal(t, `Destructive changes detected in file "1.sql"`, report.Text)
	require.Len(t, report.Diagnostics, 2)

	pass.File.Changes = append(pass.File.Changes, &sqlcheck.Change{
		Stmt: "MODIFY TABLE `pets`",
		Changes: schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("pets"),
				Changes: schema.Changes{
					&schema.ModifyColumn{
						Change: schema.ChangeType,
						From:   schema.NewIntColumn("c", "int"),
						To:     schema.NewDecimalColumn("c", "decimal"),
					},
				},
			},
		},
	})
	err = sqlcheck.Destructive.Analyze(context.Background(), pass)
	require.NoError(t, err)
	require.Len(t, report.Diagnostics, 3)
}

func TestRenames(t *testing.T) {
	var (
		report sqlcheck.Report
		pass   = &sqlcheck.Pass{
			Dev: &sqlclient.Client{Name: "mysql", Driver: &mockDriver{}},
			File: &sqlcheck.File{
				File: testFile{name: "1.sql"},
				Changes: []*sqlcheck.Change{
					{
						Stmt: "MODIFY TABLE `users`",
						Changes: schema.Changes{
							&schema.ModifyTable{
								T: schema.NewTable("users"),
								Changes: schema.Changes{
									&schema.DropColumn{C: schema.NewBoolColumn("name", "boolean")},
									&schema.AddColumn{C: schema.NewBoolColumn("username", "boolean")},
								}},
						},
					},
				},
			},
			Reporter: sqlcheck.ReportWriterFunc(func(r sqlcheck.Report) {
				report = r
			}),
		}
	)
	err := sqlcheck.Renames.Analyze(context.Background(), pass)
	require.NoError(t, err)
	require.Equal(t, `Potential renames detected in file "1.sql"`, report.Text)
	require.Len(t, report.Diagnostics, 1)
}

type testFile struct {
	name string
	migrate.File
}

func (t testFile) Name() string {
	return t.name
}

type mockDriver struct {
	migrate.Driver
}

func (m mockDriver) TableDiff(from, to *schema.Table) (changes []schema.Change, _ error) {
	for _, c1 := range from.Columns {
		c2, ok := to.Column(c1.Name)
		if !ok {
			continue
		}
		if change := m.ColumnDiff(c1, c2); change != schema.NoChange {
			changes = append(changes, &schema.ModifyColumn{
				From:   c1,
				To:     c2,
				Change: change,
			})
		}
	}
	return
}

func (m *mockDriver) ColumnDiff(from, to *schema.Column) (change schema.ChangeKind) {
	// This is used to mock a change different to a Type change.
	if len(from.Attrs) != len(to.Attrs) {
		change |= schema.ChangeComment
	}
	if reflect.TypeOf(from.Type.Type) != reflect.TypeOf(to.Type.Type) {
		change |= schema.ChangeType
	}
	return
}
