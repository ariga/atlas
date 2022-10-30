// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package condrop

import (
	"context"
	"errors"
	"fmt"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
)

// Analyzer checks for constraint-dropping changes.
type Analyzer struct {
	sqlcheck.Options
}

// New creates a new constraint-dropping Analyzer with the given options.
func New(r *schemahcl.Resource) (*Analyzer, error) {
	az := &Analyzer{}
	if r, ok := r.Resource(az.Name()); ok {
		if err := r.As(&az.Options); err != nil {
			return nil, fmt.Errorf("sql/sqlcheck: parsing datadepend check options: %w", err)
		}
	}
	return az, nil
}

// List of codes.
var (
	codeDropF = sqlcheck.Code("CD101")
)

// Name of the analyzer. Implements the sqlcheck.NamedAnalyzer interface.
func (*Analyzer) Name() string {
	return "condrop"
}

// Analyze implements sqlcheck.Analyzer.
func (a *Analyzer) Analyze(_ context.Context, p *sqlcheck.Pass) error {
	var diags []sqlcheck.Diagnostic
	for _, sc := range p.File.Changes {
		for _, c := range sc.Changes {
			c, ok := c.(*schema.ModifyTable)
			if !ok {
				continue
			}
			for i := range c.Changes {
				d, ok := c.Changes[i].(*schema.DropForeignKey)
				if !ok {
					continue
				}
				dropC := func() bool {
					for i := range d.F.Columns {
						if schema.Changes(c.Changes).IndexDropColumn(d.F.Columns[i].Name) != -1 {
							return true
						}
					}
					return false
				}()
				// If none of the foreign-key columns were dropped.
				if !dropC {
					diags = append(diags, sqlcheck.Diagnostic{
						Code: codeDropF,
						Pos:  sc.Stmt.Pos,
						Text: fmt.Sprintf("Dropping foreign-key constraint %q", d.F.Symbol),
					})
				}
			}
		}
	}
	if len(diags) > 0 {
		const reportText = "constraint deletion detected"
		p.Reporter.WriteReport(sqlcheck.Report{Text: reportText, Diagnostics: diags})
		if sqlx.V(a.Error) {
			return errors.New(reportText)
		}
	}
	return nil
}
