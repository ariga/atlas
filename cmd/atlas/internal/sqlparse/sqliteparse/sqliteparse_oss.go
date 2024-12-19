// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package sqliteparse

import (
	"errors"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
)

type FileParser struct{}

func (*FileParser) ColumnFilledBefore([]*migrate.Stmt, *schema.Table, *schema.Column, int) (bool, error) {
	return false, errors.New("unimplemented")
}

func (*FileParser) CreateViewAfter([]*migrate.Stmt, string, string, int) (bool, error) {
	return false, errors.New("unimplemented")
}

func (*FileParser) FixChange(_ migrate.Driver, _ string, changes schema.Changes) (schema.Changes, error) {
	return changes, nil // Unimplemented.
}
