// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !windows

package pgparse

import (
	"fmt"
	"slices"

	"ariga.io/atlas/cmd/atlas/internal/sqlparse/parseutil"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"

	pgquery "github.com/pganalyze/pg_query_go/v5"
)

// Parser implements the sqlparse.Parser
type Parser struct{}

// ColumnFilledBefore checks if the column was filled before the given position.
func (p *Parser) ColumnFilledBefore(stmts []*migrate.Stmt, t *schema.Table, c *schema.Column, pos int) (bool, error) {
	return parseutil.MatchStmtBefore(stmts, pos, func(s *migrate.Stmt) (bool, error) {
		tr, err := pgquery.Parse(s.Text)
		if err != nil {
			return false, err
		}
		idx := slices.IndexFunc(tr.Stmts, func(s *pgquery.RawStmt) bool {
			return s.Stmt.GetUpdateStmt() != nil
		})
		if idx == -1 {
			return false, nil
		}
		u := tr.Stmts[idx].Stmt.GetUpdateStmt()
		if u == nil || !matchTable(u.Relation, t) {
			return false, nil
		}
		// Accept UPDATE that fills all rows or those with NULL values as we cannot
		// determine if NULL values were filled in case there is a custom filtering.
		affectC := func() bool {
			if u.WhereClause == nil {
				return true
			}
			x := u.WhereClause.GetNullTest()
			if x == nil || x.GetNulltesttype() != pgquery.NullTestType_IS_NULL {
				return false
			}
			fields := x.GetArg().GetColumnRef().GetFields()
			return len(fields) == 1 && fields[0].GetString_().GetSval() == c.Name
		}()
		idx = slices.IndexFunc(u.TargetList, func(n *pgquery.Node) bool {
			r := n.GetResTarget()
			return r.GetName() == c.Name && !r.GetVal().GetAConst().GetIsnull()
		})
		// Ensure the column was filled.
		return affectC && idx != -1, nil
	})
}

// CreateViewAfter checks if a view was created after the position with the given name to a table.
func (p *Parser) CreateViewAfter(stmts []*migrate.Stmt, old, new string, pos int) (bool, error) {
	return parseutil.MatchStmtAfter(stmts, pos, func(s *migrate.Stmt) (bool, error) {
		tr, err := pgquery.Parse(s.Text)
		if err != nil {
			return false, err
		}
		idx := slices.IndexFunc(tr.Stmts, func(s *pgquery.RawStmt) bool {
			return s.Stmt.GetViewStmt() != nil
		})
		if idx == -1 {
			return false, nil
		}
		v := tr.Stmts[idx].Stmt.GetViewStmt()
		if v.GetView().GetRelname() != old {
			return false, nil
		}
		from := v.Query.GetSelectStmt().GetFromClause()
		if len(from) != 1 {
			return false, nil
		}
		return from[0].GetRangeVar().GetRelname() == new, nil
	})
}

// FixChange fixes the changes according to the given statement.
func (p *Parser) FixChange(_ migrate.Driver, s string, changes schema.Changes) (schema.Changes, error) {
	tr, err := pgquery.Parse(s)
	if err != nil {
		return nil, err
	}
	for _, stmt := range tr.Stmts {
		switch stmt := stmt.GetStmt(); {
		case stmt.GetRenameStmt() != nil &&
			stmt.GetRenameStmt().GetRenameType() == pgquery.ObjectType_OBJECT_COLUMN:
			modify, err := expectModify(changes)
			if err != nil {
				return nil, err
			}
			rename := stmt.GetRenameStmt()
			parseutil.RenameColumn(modify, &parseutil.Rename{
				From: rename.GetSubname(),
				To:   rename.GetNewname(),
			})
		case stmt.GetRenameStmt() != nil &&
			stmt.GetRenameStmt().GetRenameType() == pgquery.ObjectType_OBJECT_INDEX:
			modify, err := expectModify(changes)
			if err != nil {
				return nil, err
			}
			rename := stmt.GetRenameStmt()
			parseutil.RenameIndex(modify, &parseutil.Rename{
				From: rename.GetRelation().GetRelname(),
				To:   rename.GetNewname(),
			})
		case stmt.GetRenameStmt() != nil &&
			stmt.GetRenameStmt().GetRenameType() == pgquery.ObjectType_OBJECT_TABLE:
			rename := stmt.GetRenameStmt()
			changes = parseutil.RenameTable(changes, &parseutil.Rename{
				From: rename.GetRelation().GetRelname(),
				To:   rename.GetNewname(),
			})
		case stmt.GetIndexStmt() != nil &&
			stmt.GetIndexStmt().GetConcurrent():
			modify, err := expectModify(changes)
			if err != nil {
				return nil, err
			}
			name := stmt.GetIndexStmt().GetIdxname()
			i := schema.Changes(modify.Changes).IndexAddIndex(name)
			if i == -1 {
				return nil, fmt.Errorf("AddIndex %q command not found", name)
			}
			add := modify.Changes[i].(*schema.AddIndex)
			if !slices.ContainsFunc(add.Extra, func(c schema.Clause) bool {
				_, ok := c.(*postgres.Concurrently)
				return ok
			}) {
				add.Extra = append(add.Extra, &postgres.Concurrently{})
			}
		case stmt.GetDropStmt() != nil && stmt.GetDropStmt().GetConcurrent():
			modify, err := expectModify(changes)
			if err != nil {
				return nil, err
			}
			for _, p := range stmt.GetDropStmt().GetObjects() {
				items := p.GetList().GetItems()
				var name string
				switch {
				// Match DROP INDEX <name>.
				case len(items) == 1 && items[0].GetString_().GetSval() != "":
					name = items[0].GetString_().GetSval()
				// Match DROP INDEX <schema>.<name>.
				case len(items) == 2 && modify.T.Schema != nil &&
					items[0].GetString_().GetSval() == modify.T.Schema.Name &&
					items[1].GetString_().GetSval() != "":
					name = items[1].GetString_().GetSval()
				default:
					continue
				}
				i := schema.Changes(modify.Changes).IndexDropIndex(name)
				if i == -1 {
					return nil, fmt.Errorf("DropIndex %q command not found", name)
				}
				drop := modify.Changes[i].(*schema.DropIndex)
				if !slices.ContainsFunc(drop.Extra, func(c schema.Clause) bool {
					_, ok := c.(*postgres.Concurrently)
					return ok
				}) {
					drop.Extra = append(drop.Extra, &postgres.Concurrently{})
				}
			}
		case stmt.GetAlterTableStmt() != nil:
			if fixed, err := FixAlterTable(s, stmt.GetAlterTableStmt(), changes); err == nil {
				changes = fixed // Make ALTER fixes optional.
			}
		}
	}
	return changes, nil
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
func matchTable(n *pgquery.RangeVar, t *schema.Table) bool {
	return n.GetRelname() == t.Name && (n.GetSchemaname() == "" || n.GetSchemaname() == t.Schema.Name)
}
