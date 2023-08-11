// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

// Package x contains a small set of functions to explore how a public API
// for the CLI might look in the future. Note that this package is intended
// for experimental purposes only and should not be used externally, apart
// from within the project's codebase.
package x

import (
	"context"
	"errors"

	"ariga.io/atlas/cmd/atlas/internal/migratelint"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"
)

// Exposes the lint reporting types.
type (
	FileReport    = migratelint.FileReport
	SummaryReport = migratelint.SummaryReport
)

// ErrEmptyReport is returned when the report is empty.
var ErrEmptyReport = errors.New("empty report")

// lintLatest runs the lint command on the latest changes (files) in the given directory.
func lintLatest(ctx context.Context, dev *sqlclient.Client, dir migrate.Dir, latest int, az []sqlcheck.Analyzer) (report *SummaryReport, err error) {
	r := migratelint.Runner{
		Dev:            dev,
		Dir:            dir,
		Analyzers:      az,
		ChangeDetector: migratelint.LatestChanges(dir, latest),
		ReportWriter: reporterFunc(func(r *SummaryReport) error {
			report = r
			return nil
		}),
	}
	if err = r.Run(ctx); err != nil && !errors.As(err, &migratelint.SilentError{}) {
		return nil, err
	}
	if report == nil {
		return nil, ErrEmptyReport
	}
	return report, nil
}

// reporterFunc implements the lint.ReportWriter interface.
type reporterFunc func(*SummaryReport) error

// WriteReport implements the WriteReport method.
func (f reporterFunc) WriteReport(r *SummaryReport) error {
	return f(r)
}
