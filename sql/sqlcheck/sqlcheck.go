// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

// Package sqlcheck provides interfaces for analyzing the contents of SQL files
// to generate insights on the safety of many kinds of changes to database
// schemas. With this package developers may define an Analyzer that can be used
// to diagnose the impact of SQL statements on the target database. For instance,
// The `destructive` package exposes an Analyzer that detects destructive changes
// to the database schema, such as the dropping of tables or columns.
package sqlcheck

import (
	"context"
	"sync"

	"ariga.io/atlas/schemahcl"
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

	// A NamedAnalyzer describes an Analyzer that has a name.
	NamedAnalyzer interface {
		Analyzer
		// Name of the analyzer. Identifies the analyzer
		// in configuration and linting passes.
		Name() string
	}

	// A Pass provides information to the Analyzer.Analyze function
	// that applies a specific analyzer to an SQL file.
	Pass struct {
		// A migration file and the changes it describes.
		File *File

		// Dev is a driver-specific environment used to execute analysis work.
		Dev *sqlclient.Client

		// Report reports analysis reports.
		Reporter ReportWriter
	}

	// File represents a parsed version of a migration file.
	File struct {
		migrate.File

		// Changes represents the list of changes this file represents.
		Changes []*Change

		// Sum represents a summary of changes this file represents. For example,
		// in case of a file that contains exactly two statements, and the first
		// statement is reverted by the one after it, the Sum is nil.
		Sum schema.Changes

		// A Parser that may be used for parsing this file. It sets to any as the contract
		// between checks and their parsers can vary. For example, in case of running checks
		// from CLI, the injected parser can be found in cmd/atlas/internal/sqlparse.Parser.
		Parser any

		// schema spans. lazily initialized.
		spans map[string]*schemaSpan
	}

	// A Change in a migration file.
	Change struct {
		schema.Changes               // The actual changes.
		Stmt           *migrate.Stmt // The SQL statement generated this change.
	}

	// A Report describes an analysis report with an optional specific diagnostic.
	Report struct {
		Text           string         `json:"Text"`                     // Report text.
		Diagnostics    []Diagnostic   `json:"Diagnostics,omitempty"`    // Report diagnostics.
		SuggestedFixes []SuggestedFix `json:"SuggestedFixes,omitempty"` // Report-level suggested fixes.
	}

	// A Diagnostic is a text associated with a specific position of a statement in a file.
	Diagnostic struct {
		Pos            int            `json:"Pos"`                      // Diagnostic position.
		Text           string         `json:"Text"`                     // Diagnostic text.
		Code           string         `json:"Code"`                     // Code describes the check. For example, DS101
		SuggestedFixes []SuggestedFix `json:"SuggestedFixes,omitempty"` // Fixes to this specific diagnostics (statement-level).
	}

	// A SuggestedFix is a change associated with a diagnostic that can
	// be applied to fix the issue. Both the message and the text edit
	// are optional.
	SuggestedFix struct {
		Message  string    `json:"Message"`
		TextEdit *TextEdit `json:"TextEdit,omitempty"`
	}

	// A TextEdit represents a code changes in a file.
	// The suggested edits are line-based starting from 1.
	TextEdit struct {
		Line    int    `json:"Line"`    // Start line to edit.
		End     int    `json:"End"`     // End line to edit.
		NewText string `json:"NewText"` // New text to replace.
	}

	// ReportWriter represents a writer for analysis reports.
	ReportWriter interface {
		WriteReport(Report)
	}

	// Options defines a generic configuration options for analyzers.
	Options struct {
		// Error indicates if an analyzer should
		// error in case a Diagnostic was found.
		Error *bool `spec:"error"`

		// Allow drivers to extend the configuration.
		schemahcl.DefaultExtension
	}
)

// SuggestFix appends a suggested fix to the diagnostic.
func (d *Diagnostic) SuggestFix(m string, e *TextEdit) {
	d.SuggestedFixes = append(d.SuggestedFixes, SuggestedFix{Message: m, TextEdit: e})
}

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

// AnalyzerFunc allows using ordinary functions as analyzers.
type AnalyzerFunc func(ctx context.Context, p *Pass) error

// Analyze calls f.
func (f AnalyzerFunc) Analyze(ctx context.Context, p *Pass) error {
	return f(ctx, p)
}

// ReportWriterFunc is a function that implements Reporter.
type ReportWriterFunc func(Report)

// WriteReport calls f(r).
func (f ReportWriterFunc) WriteReport(r Report) {
	f(r)
}

// ResourceSpan describes the lifespan of a resource
// in perspective to the migration file.
type ResourceSpan uint

const (
	// SpanUnknown describes unknown lifespan.
	// e.g. resource may exist before this file.
	SpanUnknown ResourceSpan = iota

	// SpanAdded describes that a span of
	// a resource was started in this file.
	SpanAdded

	// SpanDropped describes that a span of
	// a resource was ended in this file.
	SpanDropped

	// SpanTemporary indicates that a resource lifetime
	// was started and ended in this file (CREATE and DROP).
	SpanTemporary = SpanAdded | SpanDropped
)

// SchemaSpan returns the span information for the schema.
func (f *File) SchemaSpan(s *schema.Schema) ResourceSpan {
	return f.schemaSpan(s).state
}

// TableSpan returns the span information for the table.
func (f *File) TableSpan(t *schema.Table) ResourceSpan {
	return f.tableSpan(t).state
}

// ColumnSpan returns the span information for the column.
func (f *File) ColumnSpan(t *schema.Table, c *schema.Column) ResourceSpan {
	return f.tableSpan(t).columns[c.Name]
}

// IndexSpan returns the span information for the index.
func (f *File) IndexSpan(t *schema.Table, i *schema.Index) ResourceSpan {
	return f.tableSpan(t).indexes[i.Name]
}

// ForeignKeySpan returns the span information for the foreign-key constraint.
func (f *File) ForeignKeySpan(t *schema.Table, fk *schema.ForeignKey) ResourceSpan {
	return f.tableSpan(t).forkeys[fk.Symbol]
}

type (
	// schemaSpan holds the span structure of a schema.
	schemaSpan struct {
		state  ResourceSpan
		tables map[string]*tableSpan
	}
	// schemaSpan holds the span structure of a table.
	tableSpan struct {
		state   ResourceSpan
		columns map[string]ResourceSpan
		indexes map[string]ResourceSpan
		forkeys map[string]ResourceSpan
	}
)

func (f *File) loadSpans() {
	f.spans = make(map[string]*schemaSpan)
	for _, sc := range f.Changes {
		for _, c := range sc.Changes {
			switch c := c.(type) {
			case *schema.AddSchema:
				f.schemaSpan(c.S).state = SpanAdded
			case *schema.DropSchema:
				f.schemaSpan(c.S).state |= SpanDropped
			case *schema.AddTable:
				span := f.tableSpan(c.T)
				span.state = SpanAdded
				for _, column := range c.T.Columns {
					span.columns[column.Name] = SpanAdded
				}
				for _, idx := range c.T.Indexes {
					span.indexes[idx.Name] = SpanAdded
				}
				for _, fk := range c.T.ForeignKeys {
					span.forkeys[fk.Symbol] = SpanAdded
				}
			case *schema.DropTable:
				f.tableSpan(c.T).state |= SpanDropped
			case *schema.ModifyTable:
				span := f.tableSpan(c.T)
				for _, c1 := range c.Changes {
					switch c1 := c1.(type) {
					case *schema.AddColumn:
						span.columns[c1.C.Name] = SpanAdded
					case *schema.DropColumn:
						span.columns[c1.C.Name] |= SpanDropped
					case *schema.AddIndex:
						span.indexes[c1.I.Name] = SpanAdded
					case *schema.DropIndex:
						span.indexes[c1.I.Name] |= SpanDropped
					case *schema.AddForeignKey:
						span.forkeys[c1.F.Symbol] = SpanAdded
					case *schema.DropForeignKey:
						span.forkeys[c1.F.Symbol] |= SpanDropped
					}
				}
			}
		}
	}
}

func (f *File) schemaSpan(s *schema.Schema) *schemaSpan {
	if f.spans == nil {
		f.loadSpans()
	}
	if f.spans[s.Name] == nil {
		f.spans[s.Name] = &schemaSpan{tables: make(map[string]*tableSpan)}
	}
	return f.spans[s.Name]
}

func (f *File) tableSpan(t *schema.Table) *tableSpan {
	span := f.schemaSpan(t.Schema)
	if span.tables[t.Name] == nil {
		span.tables[t.Name] = &tableSpan{
			columns: make(map[string]ResourceSpan),
			indexes: make(map[string]ResourceSpan),
			forkeys: make(map[string]ResourceSpan),
		}
	}
	return f.spans[t.Schema.Name].tables[t.Name]
}

// codes registry
var codes sync.Map

// Code stores the given code in the registry.
// It protects from duplicate analyzers' codes.
func Code(code string) string {
	if _, loaded := codes.LoadOrStore(code, struct{}{}); loaded {
		panic("sqlcheck: Code called twice for " + code)
	}
	return code
}

// drivers specific analyzers.
var drivers sync.Map

// Register allows drivers to register a constructor function for creating
// analyzers from the given HCL resource.
func Register(name string, f func(*schemahcl.Resource) ([]Analyzer, error)) {
	drivers.Store(name, f)
}

// AnalyzerFor instantiates a new Analyzer from the given HCL resource
// based on the registered constructor function.
func AnalyzerFor(name string, r *schemahcl.Resource) ([]Analyzer, error) {
	f, ok := drivers.Load(name)
	if ok {
		return f.(func(*schemahcl.Resource) ([]Analyzer, error))(r)
	}
	return nil, nil
}
