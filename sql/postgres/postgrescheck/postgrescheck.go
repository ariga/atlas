// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgrescheck

import (
	"fmt"

	"github.com/s-sokolko/atlas/schemahcl"
	"github.com/s-sokolko/atlas/sql/postgres"
	"github.com/s-sokolko/atlas/sql/sqlcheck"
	"github.com/s-sokolko/atlas/sql/sqlcheck/condrop"
	"github.com/s-sokolko/atlas/sql/sqlcheck/datadepend"
	"github.com/s-sokolko/atlas/sql/sqlcheck/destructive"
	"github.com/s-sokolko/atlas/sql/sqlcheck/incompatible"
	"github.com/s-sokolko/atlas/sql/sqlcheck/naming"
)

func addNotNull(p *datadepend.ColumnPass) (diags []sqlcheck.Diagnostic, err error) {
	tt, err := postgres.FormatType(p.Column.Type.Type)
	if err != nil {
		return nil, err
	}
	return []sqlcheck.Diagnostic{
		{
			Pos: p.Change.Stmt.Pos,
			Text: fmt.Sprintf(
				"Adding a non-nullable %q column %q will fail in case table %q is not empty",
				tt, p.Column.Name, p.Table.Name,
			),
		},
	}, nil
}

func analyzers(r *schemahcl.Resource) ([]sqlcheck.Analyzer, error) {
	ds, err := destructive.New(r)
	if err != nil {
		return nil, err
	}
	cd, err := condrop.New(r)
	if err != nil {
		return nil, err
	}
	dd, err := datadepend.New(r, datadepend.Handler{
		AddNotNull: addNotNull,
	})
	if err != nil {
		return nil, err
	}
	bc, err := incompatible.New(r)
	if err != nil {
		return nil, err
	}
	nm, err := naming.New(r)
	if err != nil {
		return nil, err
	}
	return []sqlcheck.Analyzer{ds, dd, cd, bc, nm}, nil
}
