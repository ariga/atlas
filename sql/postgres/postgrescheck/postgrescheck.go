// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlitecheck

import (
	"fmt"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlcheck/datadepend"
	"ariga.io/atlas/sql/sqlcheck/destructive"
)

// NewDataDepend creates new data-depend analyzer.
func NewDataDepend(*schemahcl.Resource) *datadepend.Analyzer {
	var opts datadepend.Options
	opts.Handler.AddNotNull = func(p *datadepend.ColumnPass) ([]sqlcheck.Diagnostic, error) {
		tt, err := postgres.FormatType(p.Column.Type.Type)
		if err != nil {
			return nil, err
		}
		return []sqlcheck.Diagnostic{
			{
				Pos: p.Change.Pos,
				Text: fmt.Sprintf(
					"Adding a non-nullable %q column %q will fail in case table %q is not empty",
					tt, p.Column.Name, p.Table.Name,
				),
			},
		}, nil
	}
	return datadepend.New(opts)
}

func init() {
	sqlcheck.Register(postgres.DriverName, func(*schemahcl.Resource) (sqlcheck.Analyzer, error) {
		return sqlcheck.Analyzers{
			destructive.New(destructive.Options{}),
			NewDataDepend(nil),
		}, nil
	})
}
