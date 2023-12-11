// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package myparse

import (
	"fmt"
	"strconv"

	"ariga.io/atlas/cmd/atlas/internal/sqlparse/parseutil"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pingcap/tidb/parser/opcode"
	"github.com/pingcap/tidb/parser/test_driver"
	"golang.org/x/exp/slices"
)

// Parser implements the sqlparse.Parser
type Parser struct{}

// ColumnFilledBefore checks if the column was filled before the given position.
func (p *Parser) ColumnFilledBefore(stmts []*migrate.Stmt, t *schema.Table, c *schema.Column, pos int) (bool, error) {
	pr := parser.New()
	return parseutil.MatchStmtBefore(stmts, pos, func(s *migrate.Stmt) (bool, error) {
		stmt, err := pr.ParseOneStmt(s.Text, "", "")
		if err != nil {
			return false, err
		}
		u, ok := stmt.(*ast.UpdateStmt)
		// Ensure the table was updated.
		if !ok || !tableUpdated(u, t) {
			return false, nil
		}
		// Accept UPDATE that fills all rows or those with NULL values as we cannot
		// determine if NULL values were filled in case there is a custom filtering.
		affectC := func() bool {
			if u.Where == nil {
				return true
			}
			is, ok := u.Where.(*ast.IsNullExpr)
			if !ok || is.Not {
				return false
			}
			n, ok := is.Expr.(*ast.ColumnNameExpr)
			return ok && n.Name.Name.O == c.Name
		}()
		idx := slices.IndexFunc(u.List, func(a *ast.Assignment) bool {
			return a.Column.Name.String() == c.Name && a.Expr != nil && a.Expr.GetType().GetType() != mysql.TypeNull
		})
		// Ensure the column was filled.
		return affectC && idx != -1, nil
	})
}

// ColumnFilledAfter checks if the column that matches the given value was filled after the position.
func (p *Parser) ColumnFilledAfter(stmts []*migrate.Stmt, t *schema.Table, c *schema.Column, pos int, match any) (bool, error) {
	switch i := slices.IndexFunc(stmts, func(s *migrate.Stmt) bool {
		return s.Pos >= pos
	}); i {
	case -1:
		return false, nil
	default:
		stmts = stmts[i:]
	}
	pr := parser.New()
	for _, s := range stmts {
		stmt, err := pr.ParseOneStmt(s.Text, "", "")
		if err != nil {
			return false, err
		}
		u, ok := stmt.(*ast.UpdateStmt)
		// Ensure the table was updated.
		if !ok || !tableUpdated(u, t) {
			continue
		}
		// Accept UPDATE that fills all rows or those with NULL values as we cannot
		// determine if NULL values were filled in case there is a custom filtering.
		affectC := func() bool {
			if u.Where == nil {
				return true
			}
			switch match.(type) {
			case nil:
				is, ok := u.Where.(*ast.IsNullExpr)
				if !ok || is.Not {
					return false
				}
				n, ok := is.Expr.(*ast.ColumnNameExpr)
				return ok && n.Name.Name.O == c.Name
			default:
				bin, ok := u.Where.(*ast.BinaryOperationExpr)
				if !ok || bin.Op != opcode.EQ {
					return false
				}
				l, r := bin.L, bin.R
				if _, ok := bin.L.(*ast.ColumnNameExpr); !ok {
					l, r = r, l
				}
				n, ok1 := l.(*ast.ColumnNameExpr)
				v, ok2 := r.(*test_driver.ValueExpr)
				if !ok1 || !ok2 || n.Name.Name.O != c.Name {
					return false
				}
				x := fmt.Sprint(match)
				if u, err := strconv.Unquote(x); err == nil {
					x = u
				}
				// String representations should be ~equal.
				return fmt.Sprint(v.Datum.GetValue()) == x
			}
		}()
		idx := slices.IndexFunc(u.List, func(a *ast.Assignment) bool {
			return a.Column.Name.String() == c.Name && a.Expr != nil && a.Expr.GetType().GetType() != mysql.TypeNull
		})
		// Ensure the column was filled.
		if affectC && idx != -1 {
			return true, nil
		}
	}
	return false, nil
}

// ColumnHasReferences checks if the column has an inline REFERENCES clause in the given CREATE or ALTER statement.
func (p *Parser) ColumnHasReferences(stmt *migrate.Stmt, c1 *schema.Column) (bool, error) {
	if stmt == nil {
		return false, nil
	}
	s, err := parser.New().ParseOneStmt(stmt.Text, "", "")
	if err != nil {
		return false, err
	}
	check := func(c2 *ast.ColumnDef) bool {
		idxR := slices.IndexFunc(c2.Options, func(o *ast.ColumnOption) bool {
			return o.Tp == ast.ColumnOptionReference
		})
		return c1.Name == c2.Name.Name.String() && idxR != -1
	}
	switch s := s.(type) {
	case *ast.CreateTableStmt:
		return slices.IndexFunc(s.Cols, check) != -1, nil
	case *ast.AlterTableStmt:
		return slices.IndexFunc(s.Specs, func(s *ast.AlterTableSpec) bool {
			return s.Tp == ast.AlterTableAddColumns && slices.IndexFunc(s.NewColumns, check) != -1
		}) != -1, nil
	}
	return false, nil
}

// CreateViewAfter checks if a view was created after the position with the given name to a table.
func (p *Parser) CreateViewAfter(stmts []*migrate.Stmt, old, new string, pos int) (bool, error) {
	pr := parser.New()
	return parseutil.MatchStmtAfter(stmts, pos, func(s *migrate.Stmt) (bool, error) {
		stmt, err := pr.ParseOneStmt(s.Text, "", "")
		if err != nil {
			return false, err
		}
		v, ok := stmt.(*ast.CreateViewStmt)
		if !ok || v.ViewName.Name.O != old {
			return false, nil
		}
		sc, ok := v.Select.(*ast.SelectStmt)
		if !ok || sc.From == nil || sc.From.TableRefs == nil || sc.From.TableRefs.Left == nil || sc.From.TableRefs.Right != nil {
			return false, nil
		}
		t, ok := sc.From.TableRefs.Left.(*ast.TableSource)
		if !ok || t.Source == nil {
			return false, nil
		}
		name, ok := t.Source.(*ast.TableName)
		return ok && name.Name.O == new, nil
	})
}

// FixChange fixes the changes according to the given statement.
func (p *Parser) FixChange(d migrate.Driver, s string, changes schema.Changes) (schema.Changes, error) {
	stmt, err := parser.New().ParseOneStmt(s, "", "")
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return changes, nil
	}
	switch stmt := stmt.(type) {
	case *ast.AlterTableStmt:
		if changes, err = renameTable(d, stmt, changes); err != nil {
			return nil, err
		}
		modify, ok := changes[0].(*schema.ModifyTable)
		if !ok {
			return nil, fmt.Errorf("expected modify-table change for alter-table statement, but got: %T", changes[0])
		}
		for _, r := range renameColumns(stmt) {
			parseutil.RenameColumn(modify, r)
		}
		for _, r := range renameIndexes(stmt) {
			parseutil.RenameIndex(modify, r)
		}
	case *ast.RenameTableStmt:
		for _, t := range stmt.TableToTables {
			changes = parseutil.RenameTable(
				changes,
				&parseutil.Rename{
					From: t.OldTable.Name.O,
					To:   t.NewTable.Name.O,
				})
		}
	}
	return changes, nil
}

// renameColumns returns all renamed columns that exist in the statement.
func renameColumns(stmt *ast.AlterTableStmt) (rename []*parseutil.Rename) {
	for _, s := range stmt.Specs {
		if s.Tp == ast.AlterTableRenameColumn {
			rename = append(rename, &parseutil.Rename{
				From: s.OldColumnName.Name.O,
				To:   s.NewColumnName.Name.O,
			})
		}
	}
	return
}

// renameIndexes returns all renamed indexes that exist in the statement.
func renameIndexes(stmt *ast.AlterTableStmt) (rename []*parseutil.Rename) {
	for _, s := range stmt.Specs {
		if s.Tp == ast.AlterTableRenameIndex {
			rename = append(rename, &parseutil.Rename{
				From: s.FromKey.O,
				To:   s.ToKey.O,
			})
		}
	}
	return
}

// renameTable fixes the changes from ALTER command with RENAME into ModifyTable and RenameTable.
func renameTable(drv migrate.Driver, stmt *ast.AlterTableStmt, changes schema.Changes) (schema.Changes, error) {
	var r *ast.AlterTableSpec
	for _, s := range stmt.Specs {
		if s.Tp == ast.AlterTableRenameTable {
			r = s
			break
		}
	}
	if r == nil {
		return changes, nil
	}
	if len(changes) != 2 {
		return nil, fmt.Errorf("unexpected number fo changes for ALTER command with RENAME clause: %d", len(changes))
	}
	i, j := changes.IndexDropTable(stmt.Table.Name.O), changes.IndexAddTable(r.NewTable.Name.O)
	if i == -1 {
		return nil, fmt.Errorf("DropTable %q change was not found in changes", stmt.Table.Name)
	}
	if j == -1 {
		return nil, fmt.Errorf("AddTable %q change was not found in changes", r.NewTable.Name)
	}
	fromT, toT := changes[0].(*schema.DropTable).T, changes[1].(*schema.AddTable).T
	fromT.Name = toT.Name
	diff, err := drv.TableDiff(fromT, toT)
	if err != nil {
		return nil, err
	}
	changeT := *toT
	changeT.Name = stmt.Table.Name.O
	return schema.Changes{
		// Modify the table first.
		&schema.ModifyTable{T: &changeT, Changes: diff},
		// Then, apply the RENAME.
		&schema.RenameTable{From: &changeT, To: toT},
	}, nil
}

// tableUpdated checks if the table was updated in the statement.
func tableUpdated(u *ast.UpdateStmt, t *schema.Table) bool {
	if u.TableRefs == nil || u.TableRefs.TableRefs == nil || u.TableRefs.TableRefs.Left == nil {
		return false
	}
	ts, ok := u.TableRefs.TableRefs.Left.(*ast.TableSource)
	if !ok {
		return false
	}
	n, ok := ts.Source.(*ast.TableName)
	return ok && n.Name.O == t.Name && (n.Schema.O == "" || n.Schema.O == t.Schema.Name)
}
