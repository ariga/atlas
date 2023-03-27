// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgrescheck

import (
	"context"
	"errors"
	"fmt"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlcheck/condrop"
	"ariga.io/atlas/sql/sqlcheck/datadepend"
	"ariga.io/atlas/sql/sqlcheck/destructive"
	"ariga.io/atlas/sql/sqlcheck/incompatible"
	"ariga.io/atlas/sql/sqlcheck/naming"
)

func addNotNull(p *datadepend.ColumnPass) (diags []sqlcheck.Diagnostic, err error) {
	tt, err := postgres.FormatType(p.Column.Type.Type)
	if err != nil {
		return nil, err
	}
	return []sqlcheck.Diagnostic{
		{
			Pos: p.Change.Stmt.Pos,
			Text: fmt.Sprintf(
				"Adding a non-nullable %q column %q will fail in case table %q is not empty",
				tt, p.Column.Name, p.Table.Name,
			),
		},
	}, nil
}

func init() {
	sqlcheck.Register(postgres.DriverName, func(r *schemahcl.Resource) ([]sqlcheck.Analyzer, error) {
		ds, err := destructive.New(r)
		if err != nil {
			return nil, err
		}
		cd, err := condrop.New(r)
		if err != nil {
			return nil, err
		}
		dd, err := datadepend.New(r, datadepend.Handler{
			AddNotNull: addNotNull,
		})
		bc, err := incompatible.New(r)
		if err != nil {
			return nil, err
		}
		nm, err := naming.New(r)
		if err != nil {
			return nil, err
		}
		ci, err := NewConcurrentIndex(r)
		if err != nil {
			return nil, err
		}
		return []sqlcheck.Analyzer{ds, dd, cd, bc, nm, ci}, nil
	})
}

type (
	// ConcurrentIndex checks for concurrent index creation,
	// dropping, and its transactional safety.
	ConcurrentIndex struct {
		ConcurrentOptions
	}
	// ConcurrentOptions for concurrent index creation.
	ConcurrentOptions struct {
		sqlcheck.Options
		CheckCreate *bool `spec:"check_create"`
		CheckDrop   *bool `spec:"check_drop"`
		CheckTxMode *bool `spec:"check_txmode"`
	}
)

// NewConcurrentIndex creates a new concurrent-index Analyzer with the given options.
func NewConcurrentIndex(r *schemahcl.Resource) (*ConcurrentIndex, error) {
	az := &ConcurrentIndex{}
	az.CheckCreate = sqlx.P(true)
	az.CheckDrop = sqlx.P(true)
	az.CheckTxMode = sqlx.P(true)
	if r, ok := r.Resource(az.Name()); ok {
		if err := r.As(&az.Options); err != nil {
			return nil, fmt.Errorf("sql/sqlcheck: parsing concurrent_index error option: %w", err)
		}
		if err := r.As(&az.ConcurrentOptions); err != nil {
			return nil, fmt.Errorf("sql/sqlcheck: parsing concurrent_index check options: %w", err)
		}
	}
	return az, nil
}

// Name of the analyzer. Implements the sqlcheck.NamedAnalyzer interface.
func (*ConcurrentIndex) Name() string {
	return "concurrent_index"
}

var (
	// codeCreateNoCon is a PostgreSQL specific code for reporting
	// indexes creation without the CONCURRENTLY option.
	codeCreateNoCon = sqlcheck.Code("PG101")
	// codeDropNoCon is a PostgreSQL specific code for reporting
	// indexes deletion without the CONCURRENTLY option.
	codeDropNoCon = sqlcheck.Code("PG102")
	// codeNoTxNone is a PostgreSQL specific code for reporting indexes
	// creation or deletion concurrently without the txmode directive set
	// to none.
	codeNoTxNone = sqlcheck.Code("PG103")
)

// Analyze implements sqlcheck.Analyzer.
func (a *ConcurrentIndex) Analyze(_ context.Context, p *sqlcheck.Pass) error {
	var (
		notx  bool
		notxC int
		diags []sqlcheck.Diagnostic
	)
	// The txmode directive, currently defined in cmd/atlas,
	// might be moved to sql/migrate in the future.
	if l, ok := p.File.File.(*migrate.LocalFile); ok {
		mode := l.Directive("txmode")
		notx = len(mode) == 1 && mode[0] == "none"
	}
	for _, sc := range p.File.Changes {
		for _, c := range sc.Changes {
			m, ok := c.(*schema.ModifyTable)
			// Skip modifications for tables that have been created in this file.
			if !ok || p.File.TableSpan(m.T)&sqlcheck.SpanAdded == 1 {
				continue
			}
			for i := range m.Changes {
				switch mc := m.Changes[i].(type) {
				case *schema.AddIndex:
					switch hasC := sqlx.Has(mc.Extra, &postgres.Concurrently{}); {
					case !sqlx.V(a.CheckCreate):
					case hasC && !notx && sqlx.V(a.CheckTxMode):
						notxC++
					case !hasC:
						diags = append(diags, sqlcheck.Diagnostic{
							Pos:  sc.Stmt.Pos,
							Code: codeCreateNoCon,
							Text: fmt.Sprintf(
								"Creating index %q non-concurrently causes write locks on the %q table",
								mc.I.Name, m.T.Name,
							),
						})
					}
				case *schema.DropIndex:
					switch hasC := sqlx.Has(mc.Extra, &postgres.Concurrently{}); {
					case !sqlx.V(a.CheckDrop):
					case hasC && !notx && sqlx.V(a.CheckTxMode):
						notxC++
					case !hasC:
						diags = append(diags, sqlcheck.Diagnostic{
							Pos:  sc.Stmt.Pos,
							Code: codeDropNoCon,
							Text: fmt.Sprintf(
								"Dropping index %q non-concurrently causes write locks on the %q table",
								mc.I.Name, m.T.Name,
							),
						})
					}
				}
			}
		}
	}
	if notxC > 0 {
		diags = append([]sqlcheck.Diagnostic{{
			Pos:  0,
			Code: codeNoTxNone,
			Text: "Indexes cannot be created or deleted concurrently within a transaction. Add the `atlas:txmode none` " +
				"directive to the header to prevent this file from running in a transaction",
		}}, diags...)
	}
	if len(diags) > 0 {
		const reportText = "concurrent index violations detected"
		p.Reporter.WriteReport(sqlcheck.Report{Text: reportText, Diagnostics: diags})
		// Report an error only if it is configured this way and the
		// diagnostics include non-concurrent creation or deletion.
		if sqlx.V(a.Error) && (notxC == 0 || len(diags) > 1) {
			return errors.New(reportText)
		}
	}
	return nil
}
