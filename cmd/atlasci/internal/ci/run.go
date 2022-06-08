// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ci

import (
	"context"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"
)

// Runner is used to execute CI jobs.
type Runner struct {
	// DevClient configures the "dev driver" to calculate
	// migration changes by the driver.
	Dev *sqlclient.Client

	// RunChangeDetector configures the ChangeDetector to
	// be used by the runner.
	ChangeDetector ChangeDetector

	// Scan is used for scanning the migration directory.
	Scan migrate.Scanner

	// Analyzer defines the analysis to be run in the CI job.
	Analyzer sqlcheck.Analyzer

	// Reporter is used to report diagnostics in the CI.
	Reporter sqlcheck.ReportWriter
}

// Run executes the CI job.
func (r *Runner) Run(ctx context.Context) error {
	base, feat, err := r.ChangeDetector.DetectChanges(ctx)
	if err != nil {
		return err
	}
	// Bring the dev environment to the base point.
	for _, f := range base {
		stmt, err := r.Scan.Stmts(f)
		if err != nil {
			return err
		}
		for _, s := range stmt {
			if _, err := r.Dev.ExecContext(ctx, s); err != nil {
				return err
			}
		}
	}
	l := &DevLoader{Dev: r.Dev, Scan: r.Scan}
	files, err := l.LoadChanges(ctx, feat)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := r.Analyzer.Analyze(ctx, &sqlcheck.Pass{File: f, Dev: r.Dev, Reporter: r.Reporter}); err != nil {
			return err
		}
	}
	return nil
}
