// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package datadepend

import (
	"context"
	"fmt"
	"sync"

	"ariga.io/atlas/schemahcl"
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

		// Allow drivers to extend the configuration.
		schemahcl.DefaultExtension
	}

	// Analyzer checks data-dependent changes.
	Analyzer struct {
		Options
	}
)

// New creates a new data-dependant analyzer with the given options.
func New(opts Options) *Analyzer {
	unique := true
	if opts.UniqueIndex != nil {
		unique = *opts.UniqueIndex
	}
	return &Analyzer{Options: Options{UniqueIndex: &unique}}
}

// Analyze implements sqlcheck.Analyzer.
func (a *Analyzer) Analyze(ctx context.Context, p *sqlcheck.Pass) error {
	f, ok := drivers.Load(p.Dev.Name)
	if ok {
		return f.(func(context.Context, *Analyzer, *sqlcheck.Pass) error)(ctx, a, p)
	}
	// Fallback to the default implementation.
	a.Report(p, a.Diagnostics(ctx, p))
	return nil
}

// Diagnostics runs the common analysis on the file and returns its diagnostics.
func (a *Analyzer) Diagnostics(_ context.Context, p *sqlcheck.Pass) (diags []sqlcheck.Diagnostic) {
	// Skip running the analysis in case the check is disabled.
	if !*a.UniqueIndex {
		return
	}
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
							Pos:  sc.Pos,
							Text: fmt.Sprintf("Adding a unique index %q on table %q might fail in case column %q contains duplicate entries", c.I.Name, m.T.Name, column.Name),
						})
					}
				case *schema.ModifyIndex:
					if c.Change.Is(schema.ChangeUnique) && c.To.Unique && p.File.IndexSpan(m.T, c.To)&sqlcheck.SpanAdded == 0 {
						diags = append(diags, sqlcheck.Diagnostic{
							Pos:  sc.Pos,
							Text: fmt.Sprintf("Modifying an index %q on table %q might fail in case of duplicate entries", c.To.Name, m.T.Name),
						})
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

// drivers specific analyzers.
var drivers sync.Map

// Register allows drivers to override the
// default analysis with custom behavior.
func Register(name string, f func(context.Context, *Analyzer, *sqlcheck.Pass) error) {
	drivers.Store(name, f)
}
