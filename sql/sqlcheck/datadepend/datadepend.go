// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package datadepend

import (
	"context"
	"fmt"

	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
)

type (
	// Options defines the configuration options
	// for the data-dependent changes checker.
	Options struct {
		// UniqueIndex indicates if the analyzer should check for modification or
		// addition of unique indexes to tables that can cause migration to fail.
		UniqueIndex *bool `spec:"drop_schema,omitempty"`

		// NotNull indicates if the analyzer should check for modification
		// or addition of NOT NULL constraints to columns.
		NotNull *bool `spec:"not_null,omitempty"`

		// Underlying driver handlers.
		Handler struct {
			// AddNotNull is applied when a new non-nullable column was
			// added to an existing table.
			AddNotNull ColumnHandler
		}
	}

	// Analyzer checks data-dependent changes.
	Analyzer struct {
		Options
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

// New creates a new data-dependant analyzer with the given options.
func New(opts Options) *Analyzer {
	notnull, unique := true, true
	if opts.NotNull != nil {
		notnull = *opts.NotNull
	}
	if opts.UniqueIndex != nil {
		unique = *opts.UniqueIndex
	}
	return &Analyzer{Options: Options{UniqueIndex: &unique, NotNull: &notnull, Handler: opts.Handler}}
}

// Analyze runs data-depend analysis on MySQL changes.
func (a *Analyzer) Analyze(ctx context.Context, p *sqlcheck.Pass) error {
	a.Report(p, a.Diagnostics(ctx, p))
	return nil
}

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
					if *a.UniqueIndex && c.I.Unique && column != nil {
						diags = append(diags, sqlcheck.Diagnostic{
							Pos:  sc.Pos,
							Text: fmt.Sprintf("Adding a unique index %q on table %q might fail in case column %q contains duplicate entries", c.I.Name, m.T.Name, column.Name),
						})
					}
				case *schema.ModifyIndex:
					if *a.UniqueIndex && c.Change.Is(schema.ChangeUnique) && c.To.Unique && p.File.IndexSpan(m.T, c.To)&sqlcheck.SpanAdded == 0 {
						diags = append(diags, sqlcheck.Diagnostic{
							Pos:  sc.Pos,
							Text: fmt.Sprintf("Modifying an index %q on table %q might fail in case of duplicate entries", c.To.Name, m.T.Name),
						})
					}
				case *schema.AddColumn:
					// In case the column is nullable without default
					// value and the table was not added in this file.
					if *a.NotNull && a.Handler.AddNotNull != nil && !c.C.Type.Null && c.C.Default == nil && p.File.TableSpan(m.T)&sqlcheck.SpanAdded != 1 {
						d, err := a.Handler.AddNotNull(&ColumnPass{Pass: p, Change: sc, Table: m.T, Column: c.C})
						if err != nil {
							return
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
func (a *Analyzer) Report(p *sqlcheck.Pass, diags []sqlcheck.Diagnostic) {
	if len(diags) > 0 {
		p.Reporter.WriteReport(sqlcheck.Report{
			Text:        fmt.Sprintf("Data dependent changes detected in file %s", p.File.Name()),
			Diagnostics: diags,
		})
	}
}
