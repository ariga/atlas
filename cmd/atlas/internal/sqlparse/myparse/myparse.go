// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package myparse

import (
	"fmt"

	"ariga.io/atlas/cmd/atlas/internal/sqlparse/parsefix"
	"ariga.io/atlas/sql/schema"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	_ "github.com/pingcap/tidb/parser/test_driver"
)

// FixChange fixes the changes according to the given statement.
func FixChange(s string, changes schema.Changes) (schema.Changes, error) {
	stmt, err := parser.New().ParseOneStmt(s, "", "")
	if err != nil {
		return nil, err
	}
	switch stmt := stmt.(type) {
	case *ast.AlterTableStmt:
		if len(changes) != 1 {
			return nil, fmt.Errorf("unexected number fo changes: %d", len(changes))
		}
		modify, ok := changes[0].(*schema.ModifyTable)
		if !ok {
			return nil, fmt.Errorf("expected modify-table change for alter-table statement, but got: %T", changes[0])
		}
		for _, r := range renameColumns(stmt) {
			parsefix.RenameColumn(modify, r.From, r.To)
		}
	case *ast.RenameTableStmt:
		for _, t := range stmt.TableToTables {
			changes = parsefix.RenameTable(
				changes,
				t.OldTable.Name.String(),
				t.NewTable.Name.String(),
			)
		}
	}
	return changes, nil
}

// renameColumns returns all renamed columns exist in the statement.
func renameColumns(stmt *ast.AlterTableStmt) (rename []struct{ From, To string }) {
	for _, s := range stmt.Specs {
		if s.Tp == ast.AlterTableRenameColumn {
			rename = append(rename, struct{ From, To string }{
				From: s.OldColumnName.Name.O,
				To:   s.NewColumnName.Name.O,
			})
		}
	}
	return
}
