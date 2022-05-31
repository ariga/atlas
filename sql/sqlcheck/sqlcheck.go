// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlcheck

import (
	"context"

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

		// Report reports a diagnostic
		Report Reporter
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

	// A Diagnostic is a message associated with a source location or range.
	Diagnostic struct {
		Pos     int    // Diagnostic position.
		Message string // Diagnostic message.
	}

	// Reporter represents a diagnostic reporter.
	Reporter interface {
		Report(Diagnostic)
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

// ReporterFunc is a function that implements Reporter.
type ReporterFunc func(Diagnostic)

// Report calls f(d).
func (f ReporterFunc) Report(d Diagnostic) {
	f(d)
}

// NopReporter is a Reporter that does nothing.
var NopReporter Reporter = ReporterFunc(func(Diagnostic) {})
