// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package destructive

import (
	"context"
	"fmt"
	"strings"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
)

type (
	// Options defines the configuration options
	// for the destructive changes checker.
	Options struct {
		// DropSchema indicates if the analyzer should check for schema dropping.
		DropSchema *bool `spec:"drop_schema,omitempty"`

		// DropTable indicates if the analyzer should check for table dropping.
		DropTable *bool `spec:"drop_schema,omitempty"`

		// DropColumn indicates if the analyzer should check for column dropping.
		DropColumn *bool `spec:"drop_column,omitempty"`

		// Allow drivers to extend the configuration.
		schemahcl.DefaultExtension
	}

	// Analyzer checks for destructive changes.
	Analyzer struct {
		Options
	}
)

// New creates a new destructive changes Analyzer with the given options.
func New(opts Options) *Analyzer {
	d := &Analyzer{}
	d.DropSchema = abool(opts.DropSchema, true)
	d.DropTable = abool(opts.DropTable, true)
	d.DropColumn = abool(opts.DropColumn, true)
	return d
}

// Analyze implements sqlcheck.Analyzer.
func (a *Analyzer) Analyze(_ context.Context, p *sqlcheck.Pass) error {
	var diags []sqlcheck.Diagnostic
	for _, sc := range p.File.Changes {
		for _, c := range sc.Changes {
			switch c := c.(type) {
			case *schema.DropSchema:
				if *a.DropSchema && p.File.SchemaSpan(c.S) != sqlcheck.SpanTemporary {
					var text string
					switch n := len(c.S.Tables); {
					case n == 0:
						text = fmt.Sprintf("Dropping schema %q", c.S.Name)
					case n == 1:
						text = fmt.Sprintf("Dropping non-empty schema %q with 1 table", c.S.Name)
					case n > 1:
						text = fmt.Sprintf("Dropping non-empty schema %q with %d tables", c.S.Name, n)
					}
					diags = append(diags, sqlcheck.Diagnostic{Pos: sc.Pos, Text: text})
				}
			case *schema.DropTable:
				if *a.DropTable && p.File.SchemaSpan(c.T.Schema) != sqlcheck.SpanDropped && p.File.TableSpan(c.T) != sqlcheck.SpanTemporary {
					diags = append(diags, sqlcheck.Diagnostic{
						Pos:  sc.Pos,
						Text: fmt.Sprintf("Dropping table %q", c.T.Name),
					})
				}
			case *schema.ModifyTable:
				if !*a.DropColumn {
					continue
				}
				for i := range c.Changes {
					d, ok := c.Changes[i].(*schema.DropColumn)
					if !ok || p.File.ColumnSpan(c.T, d.C) == sqlcheck.SpanTemporary {
						continue
					}
					if g := (schema.GeneratedExpr{}); !sqlx.Has(d.C.Attrs, &g) || strings.ToUpper(g.Type) != "VIRTUAL" {
						diags = append(diags, sqlcheck.Diagnostic{
							Pos:  sc.Pos,
							Text: fmt.Sprintf("Dropping non-virtual column %q", d.C.Name),
						})
					}
				}
			}
		}
	}
	if len(diags) > 0 {
		p.Reporter.WriteReport(sqlcheck.Report{
			Text:        fmt.Sprintf("Destructive changes detected in file %s", p.File.Name()),
			Diagnostics: diags,
		})
	}
	return nil
}

func abool(p *bool, v bool) *bool {
	if p != nil {
		return p
	}
	return &v
}
