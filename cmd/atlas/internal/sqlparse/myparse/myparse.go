// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package myparse

import (
	"fmt"

	"ariga.io/atlas/cmd/atlas/internal/sqlparse/parsefix"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	_ "github.com/pingcap/tidb/parser/test_driver"
)

// FixChange fixes the changes according to the given statement.
func FixChange(d migrate.Driver, s string, changes schema.Changes) (schema.Changes, error) {
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
			parsefix.RenameColumn(modify, r)
		}
		for _, r := range renameIndexes(stmt) {
			parsefix.RenameIndex(modify, r)
		}
	case *ast.RenameTableStmt:
		for _, t := range stmt.TableToTables {
			changes = parsefix.RenameTable(
				changes,
				&parsefix.Rename{
					From: t.OldTable.Name.O,
					To:   t.NewTable.Name.O,
				})
		}
	}
	return changes, nil
}

// renameColumns returns all renamed columns that exist in the statement.
func renameColumns(stmt *ast.AlterTableStmt) (rename []*parsefix.Rename) {
	for _, s := range stmt.Specs {
		if s.Tp == ast.AlterTableRenameColumn {
			rename = append(rename, &parsefix.Rename{
				From: s.OldColumnName.Name.O,
				To:   s.NewColumnName.Name.O,
			})
		}
	}
	return
}

// renameIndexes returns all renamed indexes that exist in the statement.
func renameIndexes(stmt *ast.AlterTableStmt) (rename []*parsefix.Rename) {
	for _, s := range stmt.Specs {
		if s.Tp == ast.AlterTableRenameIndex {
			rename = append(rename, &parsefix.Rename{
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
		return nil, fmt.Errorf("unexected number fo changes for ALTER command with RENAME clause: %d", len(changes))
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
