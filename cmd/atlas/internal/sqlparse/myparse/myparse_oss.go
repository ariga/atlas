// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package myparse

import (
	"errors"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
)

// Parser for fixing linting changes.
type Parser struct{}

// FixChange fixes the changes according to the given statement.
func (*Parser) FixChange(migrate.Driver, string, schema.Changes) (schema.Changes, error) {
	return nil, errors.New("unimplemented")
}

// ColumnFilledBefore checks if the column was filled with values before the given position in the file.
func (*Parser) ColumnFilledBefore([]*migrate.Stmt, *schema.Table, *schema.Column, int) (bool, error) {
	return false, errors.New("unimplemented")
}

// CreateViewAfter checks if a view was created after the position with the given name to a table.
func (*Parser) CreateViewAfter([]*migrate.Stmt, string, string, int) (bool, error) {
	return false, errors.New("unimplemented")
}
