// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package datadepend

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
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

		// ModifyNotNull is an optional handler applied when
		// a nullable column was changed to non-nullable.
		ModifyNotNull ColumnHandler
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
	codeModNotNullC = sqlcheck.Code("MF104")
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
					names := func() []string {
						var names []string
						for i := range c.I.Parts {
							// We consider a column a non-new column if
							// it was not added in this migration file.
							if column := c.I.Parts[i].C; column != nil && p.File.ColumnSpan(m.T, column)&sqlcheck.SpanAdded == 0 {
								names = append(names, fmt.Sprintf("%q", column.Name))
							}
						}
						return names
					}()
					// A unique index was added on an existing columns.
					if c.I.Unique && len(names) > 0 {
						s := fmt.Sprintf("columns %s contain", strings.Join(names, ", "))
						if len(names) == 1 {
							s = fmt.Sprintf("column %s contains", names[0])
						}
						diags = append(diags, sqlcheck.Diagnostic{
							Code: codeAddUniqueI,
							Pos:  sc.Stmt.Pos,
							Text: fmt.Sprintf("Adding a unique index %q on table %q might fail in case %s duplicate entries", c.I.Name, m.T.Name, s),
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
				case *schema.ModifyColumn:
					switch {
					case p.File.TableSpan(m.T)&sqlcheck.SpanAdded == 1 || !(c.From.Type.Null && !c.To.Type.Null):
					case a.ModifyNotNull != nil:
						d, err := a.Handler.ModifyNotNull(&ColumnPass{Pass: p, Change: sc, Table: m.T, Column: c.To})
						if err != nil {
							return
						}
						for i := range d {
							// In case there is no driver-specific code.
							if d[i].Code == "" {
								d[i].Code = codeModNotNullC
							}
						}
						diags = append(diags, d...)
					// In case the altered column was not added in this file, and the column
					// was changed nullable to non-nullable without back filling it with values.
					case !ColumnFilled(p, m.T, c.From, sc.Stmt.Pos):
						diags = append(diags, sqlcheck.Diagnostic{
							Code: codeModNotNullC,
							Pos:  sc.Stmt.Pos,
							Text: fmt.Sprintf("Modifying nullable column %q to non-nullable might fail in case it contains NULL values", c.To.Name),
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
func (a *Analyzer) Report(p *sqlcheck.Pass, diags []sqlcheck.Diagnostic) error {
	const reportText = "data dependent changes detected"
	if len(diags) > 0 {
		p.Reporter.WriteReport(sqlcheck.Report{Text: reportText, Diagnostics: diags})
		if sqlx.V(a.Error) {
			return errors.New(reportText)
		}
	}
	return nil
}

// ColumnFilled checks if the column was filled with values before the given position.
func ColumnFilled(p *sqlcheck.Pass, t *schema.Table, c *schema.Column, pos int) bool {
	// The parser used for parsing this file can check if the
	// given nullable column was filled before the given position.
	pr, ok := p.File.Parser.(interface {
		ColumnFilledBefore([]*migrate.Stmt, *schema.Table, *schema.Column, int) (bool, error)
	})
	if !ok {
		return false
	}
	stmts, err := migrate.FileStmtDecls(p.Dev, p.File)
	if err != nil {
		return false
	}
	filled, _ := pr.ColumnFilledBefore(stmts, t, c, pos)
	return filled
}
