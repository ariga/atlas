// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package naming

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
)

type (
	// Analyzer checks for resource naming.
	Analyzer struct {
		sqlcheck.Options
		// Global or resource-specific options.
		All, Schema, Table, Column,
		Index, ForeignKey, Check Options
	}
	// Options for resource naming.
	Options struct {
		re      *regexp.Regexp
		Match   string `spec:"match"`
		Message string `spec:"message"`
	}
)

// New creates a new destructive changes Analyzer with the given options.
func New(r *schemahcl.Resource) (*Analyzer, error) {
	az := &Analyzer{}
	if r, ok := r.Resource(az.Name()); ok {
		if err := r.As(&az.Options); err != nil {
			return nil, fmt.Errorf("sql/sqlcheck: parsing naming default options: %w", err)
		}
		if err := readOptions(r, &az.All); err != nil {
			return nil, err
		}
		for _, f := range []struct {
			string
			*Options
		}{
			{"schema", &az.Schema},
			{"table", &az.Table},
			{"column", &az.Column},
			{"index", &az.Index},
			{"foreign_key", &az.ForeignKey},
			{"check", &az.Check},
		} {
			if fr, ok := r.Resource(f.string); ok {
				if err := readOptions(fr, f.Options); err != nil {
					return nil, err
				}
			}
		}
	}
	return az, nil
}

func readOptions(r *schemahcl.Resource, opts *Options) error {
	if err := r.As(opts); err != nil {
		return fmt.Errorf("sql/sqlcheck: parsing naming options: %w", err)
	}
	re, err := regexp.Compile(opts.Match)
	if err != nil {
		return fmt.Errorf("sql/sqlcheck: parsing naming regexp: %w", err)
	}
	opts.re = re
	return nil
}

// List of codes.
var (
	codeNameS = sqlcheck.Code("NM101")
	codeNameT = sqlcheck.Code("NM102")
	codeNameC = sqlcheck.Code("NM103")
	codeNameI = sqlcheck.Code("NM104")
	codeNameF = sqlcheck.Code("NM105")
	codeNameK = sqlcheck.Code("NM106")
)

// Name of the analyzer. Implements the sqlcheck.NamedAnalyzer interface.
func (*Analyzer) Name() string {
	return "naming"
}

// Analyze implements sqlcheck.Analyzer.
func (a *Analyzer) Analyze(_ context.Context, p *sqlcheck.Pass) error {
	var diags []sqlcheck.Diagnostic
	for _, sc := range p.File.Changes {
		for _, c := range sc.Changes {
			switch c := c.(type) {
			case *schema.AddSchema:
				diags = append(diags, a.match(sc, codeNameS, c.S.Name, "Schema", a.Schema)...)
			case *schema.AddTable:
				diags = append(diags, a.match(sc, codeNameT, c.T.Name, "Table", a.Table)...)
			case *schema.RenameTable:
				diags = append(diags, a.match(sc, codeNameT, c.To.Name, "Table", a.Table)...)
			case *schema.ModifyTable:
				for i := range c.Changes {
					switch mc := c.Changes[i].(type) {
					case *schema.AddColumn:
						diags = append(diags, a.match(sc, codeNameC, mc.C.Name, "Column", a.Column)...)
					case *schema.RenameColumn:
						diags = append(diags, a.match(sc, codeNameC, mc.To.Name, "Column", a.Column)...)
					case *schema.AddIndex:
						diags = append(diags, a.match(sc, codeNameI, mc.I.Name, "Index", a.Index)...)
					case *schema.RenameIndex:
						diags = append(diags, a.match(sc, codeNameI, mc.To.Name, "Index", a.Index)...)
					case *schema.AddForeignKey:
						diags = append(diags, a.match(sc, codeNameF, mc.F.Symbol, "Foreign-key constraint", a.ForeignKey)...)
					case *schema.AddCheck:
						diags = append(diags, a.match(sc, codeNameK, mc.C.Name, "Check constraint", a.Check)...)
					}
				}
			}
		}
	}
	if len(diags) > 0 {
		const reportText = "naming violations detected"
		p.Reporter.WriteReport(sqlcheck.Report{Text: reportText, Diagnostics: diags})
		if sqlx.V(a.Error) {
			return errors.New(reportText)
		}
	}
	return nil
}

func (a *Analyzer) match(c *sqlcheck.Change, code, name, resource string, opts Options) []sqlcheck.Diagnostic {
	re, msg := opts.re, opts.Message
	if re == nil {
		re, msg = a.All.re, a.All.Message
	}
	if re == nil || re.MatchString(name) {
		return nil
	}
	d := sqlcheck.Diagnostic{
		Pos:  c.Stmt.Pos,
		Text: fmt.Sprintf("%s named %q violates the naming policy", resource, name),
		Code: code,
	}
	if msg != "" {
		d.Text += ": " + msg
	}
	return []sqlcheck.Diagnostic{d}
}
