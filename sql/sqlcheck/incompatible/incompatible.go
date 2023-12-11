// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package incompatible

import (
	"context"
	"errors"
	"fmt"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
)

// Analyzer checks for backwards-incompatible (breaking) changes.
type Analyzer struct {
	sqlcheck.Options
}

// New creates a new backwards-incompatible changes Analyzer with the given options.
func New(r *schemahcl.Resource) (*Analyzer, error) {
	az := &Analyzer{}
	if r, ok := r.Resource(az.Name()); ok {
		if err := r.As(&az.Options); err != nil {
			return nil, fmt.Errorf("sql/sqlcheck: parsing incompatible check options: %w", err)
		}
	}
	return az, nil
}

// List of codes.
var (
	codeRenameT = sqlcheck.Code("BC101")
	codeRenameC = sqlcheck.Code("BC102")
)

// Name of the analyzer. Implements the sqlcheck.NamedAnalyzer interface.
func (*Analyzer) Name() string {
	return "incompatible"
}

// Analyze implements sqlcheck.Analyzer.
func (a *Analyzer) Analyze(_ context.Context, p *sqlcheck.Pass) error {
	var diags []sqlcheck.Diagnostic
	for i, sc := range p.File.Changes {
		for _, c := range sc.Changes {
			switch c := c.(type) {
			case *schema.RenameTable:
				if p.File.SchemaSpan(c.From.Schema)&sqlcheck.SpanAdded == 0 && p.File.TableSpan(c.From)&sqlcheck.SpanAdded == 0 &&
					!ViewForRenamedT(p, c.From.Name, c.To.Name, sc.Stmt.Pos) {
					diags = append(diags, sqlcheck.Diagnostic{
						Code: codeRenameT,
						Pos:  sc.Stmt.Pos,
						Text: fmt.Sprintf("Renaming table %q to %q", c.From.Name, c.To.Name),
					})
				}
			case *schema.ModifyTable:
				for j := range c.Changes {
					r, ok := c.Changes[j].(*schema.RenameColumn)
					if ok && p.File.TableSpan(c.T)&sqlcheck.SpanAdded == 0 && !wasAddedBack(p.File.Changes[i:], r.From) {
						diags = append(diags, sqlcheck.Diagnostic{
							Code: codeRenameC,
							Pos:  sc.Stmt.Pos,
							Text: fmt.Sprintf("Renaming column %q to %q", r.From.Name, r.To.Name),
						})
					}
				}
			}
		}
	}
	if len(diags) > 0 {
		const reportText = "backward incompatible changes detected"
		p.Reporter.WriteReport(sqlcheck.Report{Text: reportText, Diagnostics: diags})
		if sqlx.V(a.Error) {
			return errors.New(reportText)
		}
	}
	return nil
}

// ViewForRenamedT checks if a view was created was a table that was renamed after the given position.
func ViewForRenamedT(p *sqlcheck.Pass, old, new string, pos int) bool {
	// The parser used for parsing this file can check if the
	// given nullable column was filled before the given position.
	pr, ok := p.File.Parser.(interface {
		CreateViewAfter(stmts []*migrate.Stmt, old, new string, pos int) (bool, error)
	})
	if !ok {
		return false
	}
	stmts, err := migrate.FileStmtDecls(p.Dev, p.File)
	if err != nil {
		return false
	}
	created, _ := pr.CreateViewAfter(stmts, old, new, pos)
	return created
}

func wasAddedBack(changes []*sqlcheck.Change, old *schema.Column) bool {
	for _, sc := range changes {
		for _, c := range sc.Changes {
			m, ok := c.(*schema.ModifyTable)
			if !ok {
				continue
			}
			// Although it is recommended to add the renamed column as generated,
			// adding it as a regular column is considered backwards compatible.
			if schema.Changes(m.Changes).IndexAddColumn(old.Name) != -1 {
				return true
			}
		}
	}
	return false
}
