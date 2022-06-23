// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlcheck

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// Destructive checks destructive changes.
var Destructive = &driverAware{
	run: func(ctx context.Context, diags []Diagnostic, p *Pass) error {
		for _, sc := range p.File.Changes {
			for _, c := range sc.Changes {
				switch c := c.(type) {
				case *schema.DropSchema:
					if p.File.SchemaSpan(c.S) != SpanTemporary {
						var text string
						switch n := len(c.S.Tables); {
						case n == 0:
							text = fmt.Sprintf("Dropping schema %q", c.S.Name)
						case n == 1:
							text = fmt.Sprintf("Dropping non-empty schema %q with 1 table", c.S.Name)
						case n > 1:
							text = fmt.Sprintf("Dropping non-empty schema %q with %d tables", c.S.Name, n)
						}
						diags = append(diags, Diagnostic{Pos: sc.Pos, Text: text})
					}
				case *schema.DropTable:
					if p.File.SchemaSpan(c.T.Schema) != SpanDropped && p.File.TableSpan(c.T) != SpanTemporary {
						diags = append(diags, Diagnostic{
							Pos:  sc.Pos,
							Text: fmt.Sprintf("Dropping table %q", c.T.Name),
						})
					}
				case *schema.ModifyTable:
					for i := range c.Changes {
						d, ok := c.Changes[i].(*schema.DropColumn)
						if !ok || p.File.ColumnSpan(c.T, d.C) == SpanTemporary {
							continue
						}
						if g := (schema.GeneratedExpr{}); !sqlx.Has(d.C.Attrs, &g) || strings.ToUpper(g.Type) != "VIRTUAL" {
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
				Text:        fmt.Sprintf("Destructive changes detected in file %s", p.File.Name()),
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
