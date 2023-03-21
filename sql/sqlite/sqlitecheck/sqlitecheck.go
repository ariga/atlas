// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlitecheck

import (
	"context"
	"fmt"
	"strings"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlcheck/condrop"
	"ariga.io/atlas/sql/sqlcheck/datadepend"
	"ariga.io/atlas/sql/sqlcheck/destructive"
	"ariga.io/atlas/sql/sqlcheck/incompatible"
	"ariga.io/atlas/sql/sqlcheck/naming"
	"ariga.io/atlas/sql/sqlite"
)

// codeModNotNullC is an SQLite specific code for reporting modifying nullable columns to non-nullable.
var codeModNotNullC = sqlcheck.Code("LT101")

func addNotNull(p *datadepend.ColumnPass) (diags []sqlcheck.Diagnostic, err error) {
	tt, err := sqlite.FormatType(p.Column.Type.Type)
	if err != nil {
		return nil, err
	}
	return []sqlcheck.Diagnostic{
		{
			Pos: p.Change.Stmt.Pos,
			Text: fmt.Sprintf(
				"Adding a non-nullable %q column %q will fail in case table %q is not empty",
				tt, p.Column.Name, p.Table.Name,
			),
		},
	}, nil
}

func modifyNotNull(p *datadepend.ColumnPass) (diags []sqlcheck.Diagnostic, err error) {
	if p.Column.Default != nil || datadepend.ColumnFilled(p.File, p.Table, p.Column, p.Change.Stmt.Pos) {
		return nil, nil
	}
	return []sqlcheck.Diagnostic{
		{
			Pos:  p.Change.Stmt.Pos,
			Code: codeModNotNullC,
			Text: fmt.Sprintf("Modifying nullable column %q to non-nullable without default value might fail in case it contains NULL values", p.Column.Name),
		},
	}, nil
}

func init() {
	sqlcheck.Register(sqlite.DriverName, func(r *schemahcl.Resource) ([]sqlcheck.Analyzer, error) {
		ds, err := destructive.New(r)
		if err != nil {
			return nil, err
		}
		cd, err := condrop.New(r)
		if err != nil {
			return nil, err
		}
		dd, err := datadepend.New(r, datadepend.Handler{
			AddNotNull:    addNotNull,
			ModifyNotNull: modifyNotNull,
		})
		if err != nil {
			return nil, err
		}
		bc, err := incompatible.New(r)
		if err != nil {
			return nil, err
		}
		nm, err := naming.New(r)
		if err != nil {
			return nil, err
		}
		return []sqlcheck.Analyzer{
			sqlcheck.AnalyzerFunc(func(ctx context.Context, p *sqlcheck.Pass) error {
				var changes []*sqlcheck.Change
				// Detect sequence of changes using temporary table and transform them to one ModifyTable change.
				// See: https://www.sqlite.org/lang_altertable.html#making_other_kinds_of_table_schema_changes.
				for i := 0; i < len(p.File.Changes); i++ {
					if i+3 >= len(p.File.Changes) {
						changes = append(changes, p.File.Changes[i])
						continue
					}
					prevT, currT, ok := modifyUsingTemp(p.File.Changes[i], p.File.Changes[i+2], p.File.Changes[i+3])
					if !ok {
						changes = append(changes, p.File.Changes[i])
						continue
					}
					diff, err := p.Dev.Driver.TableDiff(prevT, currT)
					if err != nil {
						return nil
					}
					changes = append(changes, &sqlcheck.Change{
						Stmt: &migrate.Stmt{
							// Use the position of the first statement.
							Pos: p.File.Changes[i].Stmt.Pos,
							// A combined statement.
							Text: strings.Join([]string{
								p.File.Changes[i].Stmt.Text,
								p.File.Changes[i+2].Stmt.Text,
								p.File.Changes[i+3].Stmt.Text,
							}, "\n"),
						},
						Changes: schema.Changes{
							&schema.ModifyTable{
								T:       currT,
								Changes: diff,
							},
						},
					})
					i += 3
				}
				p.File.Changes = changes
				return nil
			}),
			ds, dd, cd, bc, nm,
		}, nil
	})
}

// modifyUsingTemp indicates if the 3 changes represents a table modification using
// the pattern mentioned in the link below: "CREATE", "INSERT", "DROP" and "RENAME".
func modifyUsingTemp(c1, c2, c3 *sqlcheck.Change) (from, to *schema.Table, _ bool) {
	if len(c1.Changes) != 1 || !isAddT(c1.Changes[0], "new_") || len(c2.Changes) != 1 || len(c3.Changes) == 0 {
		return nil, nil, false
	}
	// New table layout.
	add := c1.Changes[0].(*schema.AddTable)
	prefixed, name := add.T.Name, strings.TrimPrefix(add.T.Name, "new_")
	add.T.Name = name
	// Right after "INSERT", the "DROP T" is expected.
	if !isDropT(c2.Changes[0], name) {
		return nil, nil, false
	}
	drop := c2.Changes[0].(*schema.DropTable)
	// "RENAME T" is expected after "DROP T".
	if len(c3.Changes) == 1 && isRenameT(c3.Changes[0], prefixed, name) {
		return drop.T, add.T, true
	}
	// In case no parser is attached, "RENAME T" will be presented as "DROP T" and "ADD T".
	if len(c3.Changes) == 2 && isDropT(c3.Changes[0], prefixed) && isAddT(c3.Changes[1], name) {
		return drop.T, add.T, true
	}
	return nil, nil, false
}

func isAddT(c schema.Change, prefix string) bool {
	a, ok := c.(*schema.AddTable)
	return ok && strings.HasPrefix(a.T.Name, prefix)
}

func isDropT(c schema.Change, name string) bool {
	d, ok := c.(*schema.DropTable)
	return ok && d.T.Name == name
}

func isRenameT(c schema.Change, from, to string) bool {
	r, ok := c.(*schema.RenameTable)
	return ok && r.From.Name == from && r.To.Name == to
}
