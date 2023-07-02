// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package pgparse

import (
	"fmt"

	"ariga.io/atlas/cmd/atlas/internal/sqlparse/parseutil"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"golang.org/x/exp/slices"
)

// Parser implements the sqlparse.Parser
type Parser struct{}

// ColumnFilledBefore checks if the column was filled before the given position.
func (p *Parser) ColumnFilledBefore(f migrate.File, t *schema.Table, c *schema.Column, pos int) (bool, error) {
	return parseutil.MatchStmtBefore(f, pos, func(s *migrate.Stmt) (bool, error) {
		stmt, err := parser.ParseOne(s.Text)
		if err != nil {
			return false, err
		}
		u, ok := stmt.AST.(*tree.Update)
		if !ok || !tableUpdated(u, t) {
			return false, nil
		}
		// Accept UPDATE that fills all rows or those with NULL values as we cannot
		// determine if NULL values were filled in case there is a custom filtering.
		affectC := func() bool {
			if u.Where == nil {
				return true
			}
			x, ok := u.Where.Expr.(*tree.ComparisonExpr)
			if !ok || x.Operator != tree.IsNotDistinctFrom || x.SubOperator != tree.EQ {
				return false
			}
			return x.Left.String() == c.Name && x.Right == tree.DNull
		}()
		idx := slices.IndexFunc(u.Exprs, func(x *tree.UpdateExpr) bool {
			return slices.Contains(x.Names, tree.Name(c.Name)) && x.Expr != tree.DNull
		})
		// Ensure the column was filled.
		return affectC && idx != -1, nil
	})
}

// CreateViewAfter checks if a view was created after the position with the given name to a table.
func (p *Parser) CreateViewAfter(f migrate.File, old, new string, pos int) (bool, error) {
	return parseutil.MatchStmtAfter(f, pos, func(s *migrate.Stmt) (bool, error) {
		stmt, err := parser.ParseOne(s.Text)
		if err != nil {
			return false, err
		}
		v, ok := stmt.AST.(*tree.CreateView)
		if !ok || v.AsSource == nil || v.Name.String() != old {
			return false, nil
		}
		sc, ok := v.AsSource.Select.(*tree.SelectClause)
		if !ok || len(sc.From.Tables) != 1 {
			return false, nil
		}
		return tree.AsString(sc.From.Tables[0]) == new, nil
	})
}

// FixChange fixes the changes according to the given statement.
func (p *Parser) FixChange(_ migrate.Driver, s string, changes schema.Changes) (schema.Changes, error) {
	stmt, err := parser.ParseOne(s)
	if err != nil {
		return nil, err
	}
	switch stmt := stmt.AST.(type) {
	case *tree.AlterTable:
		if r, ok := renameColumn(stmt); ok {
			modify, err := expectModify(changes)
			if err != nil {
				return nil, err
			}
			parseutil.RenameColumn(modify, r)
		}
	case *tree.RenameIndex:
		modify, err := expectModify(changes)
		if err != nil {
			return nil, err
		}
		parseutil.RenameIndex(modify, &parseutil.Rename{
			From: stmt.Index.String(),
			To:   stmt.NewName.String(),
		})
	case *tree.RenameTable:
		changes = parseutil.RenameTable(changes, &parseutil.Rename{
			From: stmt.Name.String(),
			To:   stmt.NewName.String(),
		})
	case *tree.CreateIndex:
		modify, err := expectModify(changes)
		if err != nil {
			return nil, err
		}
		i := schema.Changes(modify.Changes).IndexAddIndex(stmt.Name.String())
		if i == -1 {
			return nil, fmt.Errorf("AddIndex %q command not found", stmt.Name)
		}
		add := modify.Changes[i].(*schema.AddIndex)
		if slices.IndexFunc(add.Extra, func(c schema.Clause) bool {
			_, ok := c.(*postgres.Concurrently)
			return ok
		}) == -1 && stmt.Concurrently {
			add.Extra = append(add.Extra, &postgres.Concurrently{})
		}
	}
	return changes, nil
}

// renameColumn returns the renamed column exists in the statement, is any.
func renameColumn(stmt *tree.AlterTable) (*parseutil.Rename, bool) {
	for _, c := range stmt.Cmds {
		if r, ok := c.(*tree.AlterTableRenameColumn); ok {
			return &parseutil.Rename{
				From: r.Column.String(),
				To:   r.NewName.String(),
			}, true
		}
	}
	return nil, false
}

func expectModify(changes schema.Changes) (*schema.ModifyTable, error) {
	if len(changes) != 1 {
		return nil, fmt.Errorf("unexpected number fo changes: %d", len(changes))
	}
	modify, ok := changes[0].(*schema.ModifyTable)
	if !ok {
		return nil, fmt.Errorf("expected modify-table change for alter-table statement, but got: %T", changes[0])
	}
	return modify, nil
}

// tableUpdated checks if the table was updated in the statement.
func tableUpdated(u *tree.Update, t *schema.Table) bool {
	at, ok := u.Table.(*tree.AliasedTableExpr)
	if !ok {
		return false
	}
	n, ok := at.Expr.(*tree.TableName)
	return ok && n.Table() == t.Name && (n.Schema() == "" || n.Schema() == t.Schema.Name)
}
