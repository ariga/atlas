// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package migratelint

import (
	"context"
	"fmt"

	"ariga.io/atlas/sql/migrate"
)

func (d *DevLoader) stmts(_ context.Context, f migrate.File, _ bool) ([]*migrate.Stmt, error) {
	stmts, err := migrate.FileStmtDecls(d.Dev, f)
	if err != nil {
		return nil, &FileError{File: f.Name(), Err: fmt.Errorf("scanning statements: %w", err)}
	}
	return stmts, nil
}
