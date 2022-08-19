// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package pgparse

import (
	"fmt"

	"ariga.io/atlas/cmd/atlas/internal/sqlparse/parsefix"
	"ariga.io/atlas/sql/schema"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
)

// FixChange fixes the changes according to the given statement.
func FixChange(s string, changes schema.Changes) (schema.Changes, error) {
	stmt, err := parser.ParseOne(s)
	if err != nil {
		return nil, err
	}
	switch stmt := stmt.AST.(type) {
	case *tree.AlterTable:
		if r, ok := renameColumn(stmt); ok {
			if len(changes) != 1 {
				return nil, fmt.Errorf("unexected number fo changes: %d", len(changes))
			}
			modify, ok := changes[0].(*schema.ModifyTable)
			if !ok {
				return nil, fmt.Errorf("expected modify-table change for alter-table statement, but got: %T", changes[0])
			}
			// ALTER COLUMN cannot be combined with additional commands.
			if len(changes) > 2 {
				return nil, fmt.Errorf("unexpected number of changes found: %d", len(changes))
			}
			parsefix.RenameColumn(modify, r.From, r.To)
		}
	case *tree.RenameTable:
		changes = parsefix.RenameTable(changes, stmt.Name.String(), stmt.NewName.String())
	}
	return changes, nil
}

// renameColumns returns the renamed column exists in the statement, is any.
func renameColumn(stmt *tree.AlterTable) (*struct{ From, To string }, bool) {
	for _, c := range stmt.Cmds {
		if r, ok := c.(*tree.AlterTableRenameColumn); ok {
			return &struct{ From, To string }{
				From: r.Column.String(),
				To:   r.NewName.String(),
			}, true
		}
	}
	return nil, false
}
