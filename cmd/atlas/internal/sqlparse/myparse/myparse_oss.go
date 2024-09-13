// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package myparse

import (
	"errors"

	"github.com/s-sokolko/atlas/sql/migrate"
	"github.com/s-sokolko/atlas/sql/schema"
)

// Parser for fixing linting changes.
type FileParser struct{}

// FixChange fixes the changes according to the given statement.
func (*FileParser) FixChange(migrate.Driver, string, schema.Changes) (schema.Changes, error) {
	return nil, errors.New("unimplemented")
}

// ColumnFilledBefore checks if the column was filled with values before the given position in the file.
func (*FileParser) ColumnFilledBefore([]*migrate.Stmt, *schema.Table, *schema.Column, int) (bool, error) {
	return false, errors.New("unimplemented")
}

// CreateViewAfter checks if a view was created after the position with the given name to a table.
func (*FileParser) CreateViewAfter([]*migrate.Stmt, string, string, int) (bool, error) {
	return false, errors.New("unimplemented")
}
