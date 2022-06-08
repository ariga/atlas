// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlcheck

import (
	"context"
	"fmt"
	"sync"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
)

type (
	// An Analyzer describes a migration file analyzer.
	Analyzer interface {
		// Analyze executes the analysis function.
		Analyze(context.Context, *Pass) error
	}

	// A Pass provides information to the Run function that
	// applies a specific analyzer to an SQL file.
	Pass struct {
		// A migration file and the changes it describes.
		File *File

		// Dev is a driver-specific environment used to execute analysis work.
		Dev *sqlclient.Client

		// Report reports a analysis reports.
		Reporter ReportWriter
	}

	// File represents a parsed version of a migration file.
	File struct {
		migrate.File
		// Changes is the list of changes this file represents.
		Changes []*Change
	}

	// A Change in a migration file.
	Change struct {
		schema.Changes        // The actual changes.
		Stmt           string // The SQL statement generated this change.
		Pos            int    // The position of the statement in the file.
	}

	// A Report describes an analysis report with an optional specific diagnostic.
	Report struct {
		Text        string       // Report text.
		Diagnostics []Diagnostic // Report diagnostics.
	}

	// A Diagnostic is a text associated with a specific position of a statement in a file.
	Diagnostic struct {
		Pos  int    // Diagnostic position.
		Text string // Diagnostic text.
	}

	// ReportWriter represents a writer for analysis reports.
	ReportWriter interface {
		WriteReport(Report)
	}
)

// Analyzers implements Analyzer.
type Analyzers []Analyzer

// Analyze implements Analyzer.
func (a Analyzers) Analyze(ctx context.Context, p *Pass) error {
	for _, a := range a {
		if err := a.Analyze(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

// ReportWriterFunc is a function that implements Reporter.
type ReportWriterFunc func(Report)

// WriteReport calls f(r).
func (f ReportWriterFunc) WriteReport(r Report) {
	f(r)
}

// NopReportWriter is a Reporter that does nothing.
var NopReportWriter ReportWriter = ReportWriterFunc(func(Report) {})

// Destructive checks destructive changes.
var Destructive = &driverAware{
	run: func(ctx context.Context, diags []Diagnostic, p *Pass) error {
		for _, sc := range p.File.Changes {
			for _, c := range sc.Changes {
				switch c := c.(type) {
				case *schema.DropSchema:
					diags = append(diags, Diagnostic{
						Pos:  sc.Pos,
						Text: fmt.Sprintf("Dropping schema %q", c.S.Name),
					})
				case *schema.DropTable:
					diags = append(diags, Diagnostic{
						Pos:  sc.Pos,
						Text: fmt.Sprintf("Dropping table %q", c.T.Name),
					})
				case *schema.ModifyTable:
					for i := range c.Changes {
						d, ok := c.Changes[i].(*schema.DropColumn)
						if !ok {
							continue
						}
						if g := (schema.GeneratedExpr{}); !sqlx.Has(d.C.Attrs, &g) || g.Type != "VIRTUAL" {
							diags = append(diags, Diagnostic{
								Pos:  sc.Pos,
								Text: fmt.Sprintf("Dropping non-virtual column %q", d.C.Name),
							})
						}
					}
				}
			}
		}
		if len(diags) > 0 {
			p.Reporter.WriteReport(Report{
				Text:        fmt.Sprintf("Destructive changes detected in file %q", p.File.Name()),
				Diagnostics: diags,
			})
		}
		return nil
	},
}

// driverAware is a type of analyzer that allows registering driver-level diagnostic functions.
type driverAware struct {
	run     func(context.Context, []Diagnostic, *Pass) error
	mu      sync.RWMutex
	drivers map[string]func(context.Context, *Pass) ([]Diagnostic, error)
}

// Register registers driver-level run function to extend the analyzer.
func (a *driverAware) Register(name string, run func(context.Context, *Pass) ([]Diagnostic, error)) {
	a.mu.Lock()
	if a.drivers == nil {
		a.drivers = make(map[string]func(context.Context, *Pass) ([]Diagnostic, error))
	}
	a.drivers[name] = run
	a.mu.Unlock()
}

// Analyze implements the Analyzer interface.
func (a *driverAware) Analyze(ctx context.Context, p *Pass) error {
	var diags []Diagnostic
	if run, ok := a.drivers[p.Dev.Name]; ok {
		d, err := run(ctx, p)
		if err != nil {
			return err
		}
		diags = d
	}
	return a.run(ctx, diags, p)
}
