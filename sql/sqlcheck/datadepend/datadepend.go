// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package datadepend

import (
	"context"
	"errors"
	"fmt"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
)

type (
	// Analyzer checks data-dependent changes.
	Analyzer struct {
		sqlcheck.Options
		Handler
	}

	// Handler holds the underlying driver handlers.
	Handler struct {
		// AddNotNull is applied when a new non-nullable column was
		// added to an existing table.
		AddNotNull ColumnHandler
	}

	// ColumnPass wraps the information needed
	// by the handler below to diagnose columns.
	ColumnPass struct {
		*sqlcheck.Pass
		Change *sqlcheck.Change // Change context (statement).
		Table  *schema.Table    // The table this column belongs to.
		Column *schema.Column   // The diagnosed column.
	}

	// ColumnHandler allows provide custom diagnostic for specific column rules.
	ColumnHandler func(*ColumnPass) ([]sqlcheck.Diagnostic, error)
)

// New creates a new data-dependent analyzer with the given options.
func New(r *schemahcl.Resource, h Handler) (*Analyzer, error) {
	az := &Analyzer{Handler: h}
	if r, ok := r.Resource(az.Name()); ok {
		if err := r.As(&az.Options); err != nil {
			return nil, fmt.Errorf("sql/sqlcheck: parsing datadepend check options: %w", err)
		}
	}
	return az, nil
}

// Name of the analyzer. Implements the sqlcheck.NamedAnalyzer interface.
func (*Analyzer) Name() string {
	return "data_depend"
}

// Analyze runs data-depend analysis on MySQL changes.
func (a *Analyzer) Analyze(ctx context.Context, p *sqlcheck.Pass) error {
	return a.Report(p, a.Diagnostics(ctx, p))
}

// List of codes.
var (
	codeAddUniqueI  = sqlcheck.Code("MF101")
	codeModUniqueI  = sqlcheck.Code("MF102")
	codeAddNotNullC = sqlcheck.Code("MF103")
)

// Diagnostics runs the common analysis on the file and returns its diagnostics.
func (a *Analyzer) Diagnostics(_ context.Context, p *sqlcheck.Pass) (diags []sqlcheck.Diagnostic) {
	for _, sc := range p.File.Changes {
		for _, c := range sc.Changes {
			m, ok := c.(*schema.ModifyTable)
			if !ok {
				continue
			}
			for _, c := range m.Changes {
				switch c := c.(type) {
				case *schema.AddIndex:
					column := func() *schema.Column {
						for i := range c.I.Parts {
							// We consider a column a non-new column if
							// it was not added in this migration file.
							if column := c.I.Parts[i].C; column != nil && p.File.ColumnSpan(m.T, column)&sqlcheck.SpanAdded == 0 {
								return column
							}
						}
						return nil
					}()
					// A unique index was added on an existing column.
					if c.I.Unique && column != nil {
						diags = append(diags, sqlcheck.Diagnostic{
							Code: codeAddUniqueI,
							Pos:  sc.Stmt.Pos,
							Text: fmt.Sprintf("Adding a unique index %q on table %q might fail in case column %q contains duplicate entries", c.I.Name, m.T.Name, column.Name),
						})
					}
				case *schema.ModifyIndex:
					if c.Change.Is(schema.ChangeUnique) && c.To.Unique && p.File.IndexSpan(m.T, c.To)&sqlcheck.SpanAdded == 0 {
						diags = append(diags, sqlcheck.Diagnostic{
							Code: codeModUniqueI,
							Pos:  sc.Stmt.Pos,
							Text: fmt.Sprintf("Modifying an index %q on table %q might fail in case of duplicate entries", c.To.Name, m.T.Name),
						})
					}
				case *schema.AddColumn:
					// In case the column is nullable without default
					// value and the table was not added in this file.
					if a.Handler.AddNotNull != nil && !c.C.Type.Null && c.C.Default == nil && p.File.TableSpan(m.T)&sqlcheck.SpanAdded != 1 {
						d, err := a.Handler.AddNotNull(&ColumnPass{Pass: p, Change: sc, Table: m.T, Column: c.C})
						if err != nil {
							return
						}
						for i := range d {
							// In case there is no driver-specific code.
							if d[i].Code == "" {
								d[i].Code = codeAddNotNullC
							}
						}
						diags = append(diags, d...)
					}
				}
			}
		}
	}
	return
}

// Report provides standard reporting for data-dependent changes. Drivers that
// decorate this Analyzer should call this function to get consistent reporting
// between dialects.
func (a *Analyzer) Report(p *sqlcheck.Pass, diags []sqlcheck.Diagnostic) error {
	if len(diags) > 0 {
		p.Reporter.WriteReport(sqlcheck.Report{Text: reportText, Diagnostics: diags})
		if sqlx.V(a.Error) {
			return errors.New(reportText)
		}
	}
	return nil
}

const reportText = "data dependent changes detected"
