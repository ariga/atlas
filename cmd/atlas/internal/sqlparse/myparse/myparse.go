// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package myparse

import (
	"fmt"

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
	alter, ok := stmt.(*ast.AlterTableStmt)
	if !ok {
		return changes, nil
	}
	if len(changes) != 1 {
		return nil, fmt.Errorf("unexected number fo changes: %d", len(changes))
	}
	modify, ok := changes[0].(*schema.ModifyTable)
	if !ok {
		return nil, fmt.Errorf("expected modify-table change for alter-table statement, but got: %T", changes[0])
	}
	for _, r := range renameColumns(alter) {
		changes := schema.Changes(modify.Changes)
		i1 := changes.IndexDropColumn(r.From)
		i2 := changes.IndexAddColumn(r.To)
		if i1 == -1 || i2 == -1 {
			continue
		}
		changes = append(changes, &schema.RenameColumn{
			From: changes[i1].(*schema.DropColumn).C,
			To:   changes[i2].(*schema.AddColumn).C,
		})
		changes.RemoveIndex(i1, i2)
		modify.Changes = changes
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
