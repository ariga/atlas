// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysqlcheck

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlcheck/condrop"
	"ariga.io/atlas/sql/sqlcheck/datadepend"
	"ariga.io/atlas/sql/sqlcheck/destructive"
	"ariga.io/atlas/sql/sqlcheck/incompatible"
	"ariga.io/atlas/sql/sqlcheck/naming"
)

var (
	// codeImplicitUpdate is a MySQL specific code for reporting implicit update.
	codeImplicitUpdate = sqlcheck.Code("MY101")
	// codeInlineRef is a MySQL specific code for reporting columns with inline references.
	codeInlineRef = sqlcheck.Code("MY102")
)

func addNotNull(p *datadepend.ColumnPass) (diags []sqlcheck.Diagnostic, err error) {
	// Two types of reporting, implicit rows update and
	// changes that may cause the migration to fail.
	mightFail := func(tt string) {
		diags = append(diags, sqlcheck.Diagnostic{
			Pos: p.Change.Stmt.Pos,
			Text: fmt.Sprintf(
				"Adding a non-nullable %q column %q will fail in case table %q is not empty",
				tt, p.Column.Name, p.Table.Name,
			),
		})
	}
	implicitUpdate := func(tt, v string) {
		if !columnFilledAfter(p, v) {
			diags = append(diags, sqlcheck.Diagnostic{
				Code: codeImplicitUpdate,
				Pos:  p.Change.Stmt.Pos,
				Text: fmt.Sprintf(
					"Adding a non-nullable %q column %q on table %q without a default value implicitly sets existing rows with %s",
					tt, p.Column.Name, p.Table.Name, v,
				),
			})
		}
	}
	drv, ok := p.Dev.Driver.(*mysql.Driver)
	if !ok {
		return nil, fmt.Errorf("unexpected migrate driver %T", p.Dev.Driver)
	}
	switch ct := p.Column.Type.Type.(type) {
	case *mysql.BitType, *schema.BoolType, *schema.IntegerType, *schema.DecimalType, *schema.FloatType, *schema.BinaryType:
		if !sqlx.Has(p.Column.Attrs, &mysql.AutoIncrement{}) {
			tt, err := mysql.FormatType(p.Column.Type.Type)
			if err != nil {
				return nil, err
			}
			implicitUpdate(tt, "0")
		}
	case *schema.StringType:
		switch ct.T {
		case mysql.TypeVarchar, mysql.TypeChar:
			implicitUpdate(ct.T, `""`)
		case mysql.TypeText, mysql.TypeTinyText, mysql.TypeMediumText, mysql.TypeLongText:
			// On MySQL, Existing rows are updated with ''. Skip it
			// as we cannot propose and detect multi-steps migration
			// (ALTER + UPDATE) at this stage.
			if drv.Maria() {
				implicitUpdate(ct.T, `""`)
			}
		}
	case *schema.EnumType:
		if len(ct.Values) == 0 {
			return nil, fmt.Errorf("unexpected empty values for enum column %q.%q", p.Table.Name, p.Column.Name)
		}
		implicitUpdate("enum", strconv.Quote(ct.Values[0]))
	case *mysql.SetType:
		implicitUpdate("set", `""`)
	case *schema.JSONType:
		// On MySQL, Existing rows are updated with 'null' JSON. Same as TEXT
		// columns, we cannot propose multi-steps migration (ALTER + UPDATE)
		// as it cannot be detected at this stage.
		if drv.Maria() {
			implicitUpdate(ct.T, `""`)
		}
	case *schema.TimeType:
		switch ct.T {
		case mysql.TypeDate, mysql.TypeDateTime:
			if drv.Maria() {
				implicitUpdate(ct.T, "00:00:00")
			} else {
				// The suggested solution is to add a DEFAULT clause
				// with valid value or set the column to nullable.
				mightFail(ct.T)
			}
		case mysql.TypeYear:
			implicitUpdate(ct.T, "0000")
		case mysql.TypeTime:
			implicitUpdate(ct.T, "00:00:00")
		case mysql.TypeTimestamp:
			v := "CURRENT_TIMESTAMP"
			switch {
			case drv.Maria():
				// Maria has a special behavior for the first TIMESTAMP column.
				// See: https://mariadb.com/kb/en/timestamp/#automatic-values
				for i := 0; i < len(p.Table.Columns) && p.Table.Columns[i].Name != p.Column.Name; i++ {
					tt, err := mysql.FormatType(p.Table.Columns[i].Type.Type)
					if err != nil {
						return nil, err
					}
					if strings.HasPrefix(tt, mysql.TypeTimestamp) {
						v = "0000-00-00 00:00:00"
						break
					}
				}
			// Following MySQL 8.0.2, the explicit_defaults_for_timestamp
			// system variable is now enabled by default.
			case drv.GTE("8.0.2"):
				v = "0000-00-00 00:00:00"
			}
			implicitUpdate(ct.T, v)
		}
	case *schema.SpatialType:
		if drv.Maria() {
			implicitUpdate(ct.T, `""`)
		} else {
			// The suggested solution is to add the column as
			// null, update values and then set it to not-null.
			mightFail(ct.T)
		}
	}
	return
}

// columnFilledAfter checks if the column with the given value was filled after the current change.
func columnFilledAfter(pass *datadepend.ColumnPass, matchValue string) bool {
	p, ok := pass.File.Parser.(interface {
		ColumnFilledAfter([]*migrate.Stmt, *schema.Table, *schema.Column, int, any) (bool, error)
	})
	if !ok {
		return false
	}
	stmts, err := migrate.FileStmtDecls(pass.Dev, pass.File)
	if err != nil {
		return false
	}
	filled, _ := p.ColumnFilledAfter(stmts, pass.Table, pass.Column, pass.Change.Stmt.Pos, matchValue)
	return filled
}

// inlineRefs is an analyzer function that detects column definitions with the REFERENCES
// clause and suggest replacing them with an explicit foreign-key definition.
func inlineRefs(_ context.Context, p *sqlcheck.Pass) error {
	var diags []sqlcheck.Diagnostic
	parse, ok := p.File.Parser.(interface {
		ColumnHasReferences(*migrate.Stmt, *schema.Column) (bool, error)
	})
	if !ok {
		return nil
	}
	for _, sc := range p.File.Changes {
		for _, c := range sc.Changes {
			switch c := c.(type) {
			case *schema.AddTable:
				for i := range c.T.Columns {
					if hasR, _ := parse.ColumnHasReferences(sc.Stmt, c.T.Columns[i]); hasR {
						diags = append(diags, sqlcheck.Diagnostic{
							Pos:  sc.Stmt.Pos,
							Code: codeInlineRef,
							Text: fmt.Sprintf("Defining column %q on table %q with inline REFERENCES is ignored by MySQL", c.T.Columns[i].Name, c.T.Name),
						})
					}
				}
			case *schema.ModifyTable:
				for _, mc := range c.Changes {
					add, ok := mc.(*schema.AddColumn)
					if !ok {
						continue
					}
					if hasR, _ := parse.ColumnHasReferences(sc.Stmt, add.C); hasR {
						diags = append(diags, sqlcheck.Diagnostic{
							Pos:  sc.Stmt.Pos,
							Code: codeInlineRef,
							Text: fmt.Sprintf("Defining column %q on table %q with inline REFERENCES is ignored by MySQL", add.C.Name, c.T.Name),
						})
					}
				}
			}
		}
	}
	if len(diags) > 0 {
		p.Reporter.WriteReport(sqlcheck.Report{Text: "inline REFERENCES detected", Diagnostics: diags})
	}
	return nil
}

func analyzers(r *schemahcl.Resource) ([]sqlcheck.Analyzer, error) {
	ds, err := destructive.New(r)
	if err != nil {
		return nil, err
	}
	cd, err := condrop.New(r)
	if err != nil {
		return nil, err
	}
	dd, err := datadepend.New(r, datadepend.Handler{
		AddNotNull: addNotNull,
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
	return []sqlcheck.Analyzer{ds, dd, cd, bc, nm, sqlcheck.AnalyzerFunc(inlineRefs)}, nil
}
