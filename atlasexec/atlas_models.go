// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package atlasexec

import (
	"errors"
	"time"

	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"
)

type (
	// File wraps migrate.File to implement json.Marshaler.
	File struct {
		Name        string `json:"Name,omitempty"`
		Version     string `json:"Version,omitempty"`
		Description string `json:"Description,omitempty"`
		Content     string `json:"Content,omitempty"`
	}
	// AppliedFile is part of a MigrateApply containing information about an applied file in a migration attempt.
	AppliedFile struct {
		File
		Start   time.Time
		End     time.Time
		Skipped int           // Amount of skipped SQL statements in a partially applied file.
		Applied []string      // SQL statements applied with success
		Checks  []*FileChecks // Assertion checks
		Error   *struct {
			Stmt string // SQL statement that failed.
			Text string // Error returned by the database.
		}
	}
	// RevertedFile is part of a MigrateDown containing information about a reverted file in a downgrade attempt.
	RevertedFile struct {
		File
		Start   time.Time
		End     time.Time
		Skipped int      // Amount of skipped SQL statements in a partially applied file.
		Applied []string // SQL statements applied with success
		Scope   string   // Scope of the revert. e.g., statement, versions, etc.
		Error   *struct {
			Stmt string // SQL statement that failed.
			Text string // Error returned by the database.
		}
	}
	// A SummaryReport contains a summary of the analysis of all files.
	// It is used as an input to templates to report the CI results.
	SummaryReport struct {
		URL string `json:"URL,omitempty"` // URL of the report, if exists.
		// Env holds the environment information.
		Env struct {
			Driver string         `json:"Driver,omitempty"` // Driver name.
			URL    *sqlclient.URL `json:"URL,omitempty"`    // URL to dev database.
			Dir    string         `json:"Dir,omitempty"`    // Path to migration directory.
		}
		// Schema versions found by the runner.
		Schema struct {
			Current string `json:"Current,omitempty"` // Current schema.
			Desired string `json:"Desired,omitempty"` // Desired schema.
		}
		// Steps of the analysis. Added in verbose mode.
		Steps []*StepReport `json:"Steps,omitempty"`
		// Files reports. Non-empty in case there are findings.
		Files []*FileReport `json:"Files,omitempty"`
	}
	// StepReport contains a summary of the analysis of a single step.
	StepReport struct {
		Name   string      `json:"Name,omitempty"`   // Step name.
		Text   string      `json:"Text,omitempty"`   // Step description.
		Error  string      `json:"Error,omitempty"`  // Error that cause the execution to halt.
		Result *FileReport `json:"Result,omitempty"` // Result of the step. For example, a diagnostic.
	}
	// FileReport contains a summary of the analysis of a single file.
	FileReport struct {
		Name    string            `json:"Name,omitempty"`    // Name of the file.
		Text    string            `json:"Text,omitempty"`    // Contents of the file.
		Reports []sqlcheck.Report `json:"Reports,omitempty"` // List of reports.
		Error   string            `json:"Error,omitempty"`   // File specific error.
	}
	// FileChecks represents a set of checks to run before applying a file.
	FileChecks struct {
		Name  string     `json:"Name,omitempty"`  // File/group name.
		Stmts []*Check   `json:"Stmts,omitempty"` // Checks statements executed.
		Error *StmtError `json:"Error,omitempty"` // Assertion error.
		Start time.Time  `json:"Start,omitempty"` // Start assertion time.
		End   time.Time  `json:"End,omitempty"`   // End assertion time.
	}
	// Check represents an assertion and its status.
	Check struct {
		Stmt  string  `json:"Stmt,omitempty"`  // Assertion statement.
		Error *string `json:"Error,omitempty"` // Assertion error, if any.
	}
	// StmtError groups a statement with its execution error.
	StmtError struct {
		Stmt string `json:"Stmt,omitempty"` // SQL statement that failed.
		Text string `json:"Text,omitempty"` // Error message as returned by the database.
	}
	// Env holds the environment information.
	Env struct {
		Driver string         `json:"Driver,omitempty"` // Driver name.
		URL    *sqlclient.URL `json:"URL,omitempty"`    // URL to dev database.
		Dir    string         `json:"Dir,omitempty"`    // Path to migration directory.
	}
	// Changes represents a list of changes that are pending or applied.
	Changes struct {
		Applied []string   `json:"Applied,omitempty"` // SQL changes applied with success
		Pending []string   `json:"Pending,omitempty"` // SQL changes that were not applied
		Error   *StmtError `json:"Error,omitempty"`   // Error that occurred during applying
	}
	// A Revision denotes an applied migration in a deployment. Used to track migration executions state of a database.
	Revision struct {
		Version         string        `json:"Version"`             // Version of the migration.
		Description     string        `json:"Description"`         // Description of this migration.
		Type            string        `json:"Type"`                // Type of the migration.
		Applied         int           `json:"Applied"`             // Applied amount of statements in the migration.
		Total           int           `json:"Total"`               // Total amount of statements in the migration.
		ExecutedAt      time.Time     `json:"ExecutedAt"`          // ExecutedAt is the starting point of execution.
		ExecutionTime   time.Duration `json:"ExecutionTime"`       // ExecutionTime of the migration.
		Error           string        `json:"Error,omitempty"`     // Error of the migration, if any occurred.
		ErrorStmt       string        `json:"ErrorStmt,omitempty"` // ErrorStmt is the statement that raised Error.
		OperatorVersion string        `json:"OperatorVersion"`     // OperatorVersion that executed this migration.
	}
	// A Report describes a schema analysis report with an optional specific diagnostic.
	Report struct {
		Text        string       `json:"Text"`                  // Report text.
		Desc        string       `json:"Desc,omitempty"`        // Optional description (secondary text).
		Error       bool         `json:"Error,omitempty"`       // Report is an error report.
		Diagnostics []Diagnostic `json:"Diagnostics,omitempty"` // Report diagnostics.
	}
	// A Diagnostic is a text associated with a specific position of a definition/element in a file.
	Diagnostic struct {
		Pos  *schema.Pos `json:"Pos,omitempty"`  // Element position.
		Text string      `json:"Text"`           // Diagnostic text.
		Code string      `json:"Code,omitempty"` // Code describes the check (optional).
	}
)

// DiagnosticsCount returns the total number of diagnostics in the report.
func (r *SummaryReport) DiagnosticsCount() int {
	var n int
	for _, f := range r.Files {
		for _, r := range f.Reports {
			n += len(r.Diagnostics)
		}
	}
	return n
}

// Errors returns the errors in the summary report, if exists.
func (r *SummaryReport) Errors() []error {
	var errs []error
	for _, f := range r.Files {
		if f.Error != "" {
			errs = append(errs, errors.New(f.Error))
		}
	}
	return errs
}
