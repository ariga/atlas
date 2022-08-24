// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package pgparse

import (
	"fmt"

	"ariga.io/atlas/cmd/atlas/internal/sqlparse/parsefix"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
)

// FixChange fixes the changes according to the given statement.
func FixChange(_ migrate.Driver, s string, changes schema.Changes) (schema.Changes, error) {
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
			parsefix.RenameColumn(modify, r)
		}
	case *tree.RenameIndex:
		modify, err := expectModify(changes)
		if err != nil {
			return nil, err
		}
		parsefix.RenameIndex(modify, &parsefix.Rename{
			From: stmt.Index.String(),
			To:   stmt.NewName.String(),
		})
	case *tree.RenameTable:
		changes = parsefix.RenameTable(changes, &parsefix.Rename{
			From: stmt.Name.String(),
			To:   stmt.NewName.String(),
		})
	}
	return changes, nil
}

// renameColumn returns the renamed column exists in the statement, is any.
func renameColumn(stmt *tree.AlterTable) (*parsefix.Rename, bool) {
	for _, c := range stmt.Cmds {
		if r, ok := c.(*tree.AlterTableRenameColumn); ok {
			return &parsefix.Rename{
				From: r.Column.String(),
				To:   r.NewName.String(),
			}, true
		}
	}
	return nil, false
}

func expectModify(changes schema.Changes) (*schema.ModifyTable, error) {
	if len(changes) != 1 {
		return nil, fmt.Errorf("unexected number fo changes: %d", len(changes))
	}
	modify, ok := changes[0].(*schema.ModifyTable)
	if !ok {
		return nil, fmt.Errorf("expected modify-table change for alter-table statement, but got: %T", changes[0])
	}
	return modify, nil
}
