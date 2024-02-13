// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package destructive

import (
	"errors"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
)

func (*Analyzer) hasEmptyTableCheck(*sqlcheck.Pass, *schema.Table) bool {
	return false // unimplemented.
}

func (*Analyzer) hasEmptyColumnCheck(*sqlcheck.Pass, *schema.Table, *schema.Column) bool {
	return false // unimplemented.
}

func (*Analyzer) emptyTableCheckStmt(*sqlcheck.Pass, *schema.Table) (*migrate.Stmt, error) {
	return nil, errors.New("unimplemented")
}

func (*Analyzer) emptyColumnCheckStmt(*sqlcheck.Pass, *schema.Table, string) (*migrate.Stmt, error) {
	return nil, errors.New("unimplemented")
}

func suggestFix(*sqlcheck.Pass, []*migrate.Stmt) []sqlcheck.SuggestedFix {
	return nil // unimplemented.
}
