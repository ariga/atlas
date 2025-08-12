// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package atlasexec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type (
	// MigrateApplyParams are the parameters for the `migrate apply` command.
	MigrateApplyParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		Context   *DeployRunContext
		DirURL    string

		URL             string
		RevisionsSchema string
		BaselineVersion string
		TxMode          string
		ExecOrder       MigrateExecOrder
		Amount          uint64
		ToVersion       string
		AllowDirty      bool
		DryRun          bool
	}
	// MigrateApply contains a summary of a migration applying attempt on a database.
	MigrateApply struct {
		Env
		Pending []File         `json:"Pending,omitempty"` // Pending migration files
		Applied []*AppliedFile `json:"Applied,omitempty"` // Applied files
		Current string         `json:"Current,omitempty"` // Current migration version
		Target  string         `json:"Target,omitempty"`  // Target migration version
		Start   time.Time
		End     time.Time
		// Error is set even then, if it was not caused by a statement in a migration file,
		// but by Atlas, e.g. when committing or rolling back a transaction.
		Error string `json:"Error,omitempty"`
	}
	// MigrateApplyError is returned when an error occurred
	// during a migration applying attempt.
	MigrateApplyError struct {
		Result []*MigrateApply
		Stderr string
	}
	// MigrateExecOrder define how Atlas computes and executes pending migration files to the database.
	// See: https://atlasgo.io/versioned/apply#execution-order
	MigrateExecOrder string
	// MigrateDownParams are the parameters for the `migrate down` command.
	MigrateDownParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		Context   *DeployRunContext
		DevURL    string

		DirURL          string
		URL             string
		RevisionsSchema string
		Amount          uint64
		ToVersion       string
		ToTag           string

		// Not yet supported
		// DryRun          bool
		// TxMode          string
	}
	// MigrateDown contains a summary of a migration down attempt on a database.
	MigrateDown struct {
		Planned  []File          `json:"Planned,omitempty"`  // Planned migration files
		Reverted []*RevertedFile `json:"Reverted,omitempty"` // Reverted files
		Current  string          `json:"Current,omitempty"`  // Current migration version
		Target   string          `json:"Target,omitempty"`   // Target migration version
		Total    int             `json:"Total,omitempty"`    // Total number of migrations to revert
		Start    time.Time
		End      time.Time
		// URL and Status are set only when the migration is planned or executed in the cloud.
		URL    string `json:"URL,omitempty"`
		Status string `json:"Status,omitempty"`
		// Error is set even then, if it was not caused by a statement in a migration file,
		// but by Atlas, e.g. when committing or rolling back a transaction.
		Error string `json:"Error,omitempty"`
	}
	// MigratePushParams are the parameters for the `migrate push` command.
	MigratePushParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		Context   *RunContext
		DevURL    string

		Name        string
		Tag         string
		DirURL      string
		DirFormat   string
		LockTimeout string
	}
	// MigrateLintParams are the parameters for the `migrate lint` command.
	MigrateLintParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		Context   *RunContext
		Format    string
		DevURL    string
		GitBase   string
		GitDir    string

		DirURL string
		Latest uint64
		Writer io.Writer
		Base   string
		Web    bool
	}
	// MigrateHashParams are the parameters for the `migrate hash` command.
	MigrateHashParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs

		DirURL    string
		DirFormat string
	}
	// MigrateRebaseParams are the parameters for the `migrate rebase` command.
	MigrateRebaseParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs

		DirURL string
		Files  []string
	}
	// MigrateTestParams are the parameters for the `migrate test` command.
	MigrateTestParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		Context   *RunContext
		DevURL    string

		DirURL          string
		DirFormat       string
		Run             string
		RevisionsSchema string
		Paths           []string
	}
	// MigrateStatusParams are the parameters for the `migrate status` command.
	MigrateStatusParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs

		DirURL          string
		URL             string
		RevisionsSchema string
	}
	// MigrateDiffParams are the parameters for the `migrate diff` command.
	MigrateDiffParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs

		Name        string
		ToURL       string
		DevURL      string
		DirURL      string
		DirFormat   string
		Schema      []string
		LockTimeout string
		Format      string
		Qualifier   string
	}
	// MigrateDiff contains the result of the `migrate diff` command.
	MigrateDiff struct {
		Files []File `json:"Files,omitempty"` // Generated migration files
		Dir   string `json:"Dir,omitempty"`   // Path to migration directory
	}
	// MigrateStatus contains a summary of the migration status of a database.
	MigrateStatus struct {
		Env       Env         `json:"Env,omitempty"`       // Environment info.
		Available []File      `json:"Available,omitempty"` // Available migration files
		Pending   []File      `json:"Pending,omitempty"`   // Pending migration files
		Applied   []*Revision `json:"Applied,omitempty"`   // Applied migration files
		Current   string      `json:"Current,omitempty"`   // Current migration version
		Next      string      `json:"Next,omitempty"`      // Next migration version
		Count     int         `json:"Count,omitempty"`     // Count of applied statements of the last revision
		Total     int         `json:"Total,omitempty"`     // Total statements of the last migration
		Status    string      `json:"Status,omitempty"`    // Status of migration (OK, PENDING)
		Error     string      `json:"Error,omitempty"`     // Last Error that occurred
		SQL       string      `json:"SQL,omitempty"`       // SQL that caused the last Error
	}
)

// MigratePush runs the 'migrate push' command.
func (c *Client) MigratePush(ctx context.Context, params *MigratePushParams) (string, error) {
	args := []string{"migrate", "push"}
	if params.DevURL != "" {
		args = append(args, "--dev-url", params.DevURL)
	}
	if params.DirURL != "" {
		args = append(args, "--dir", params.DirURL)
	}
	if params.DirFormat != "" {
		args = append(args, "--dir-format", params.DirFormat)
	}
	if params.LockTimeout != "" {
		args = append(args, "--lock-timeout", params.LockTimeout)
	}
	if params.Context != nil {
		buf, err := json.Marshal(params.Context)
		if err != nil {
			return "", err
		}
		args = append(args, "--context", string(buf))
	}
	if params.ConfigURL != "" {
		args = append(args, "--config", params.ConfigURL)
	}
	if params.Env != "" {
		args = append(args, "--env", params.Env)
	}
	if params.Vars != nil {
		args = append(args, params.Vars.AsArgs()...)
	}
	if params.Name == "" {
		return "", errors.New("directory name cannot be empty")
	}
	if params.Tag != "" {
		args = append(args, fmt.Sprintf("%s:%s", params.Name, params.Tag))
	} else {
		args = append(args, params.Name)
	}
	resp, err := stringVal(c.runCommand(ctx, args))
	return strings.TrimSpace(resp), err
}

// MigrateApply runs the 'migrate apply' command.
func (c *Client) MigrateApply(ctx context.Context, params *MigrateApplyParams) (*MigrateApply, error) {
	return firstResult(c.MigrateApplySlice(ctx, params))
}

// MigrateApplySlice runs the 'migrate apply' command for multiple targets.
func (c *Client) MigrateApplySlice(ctx context.Context, params *MigrateApplyParams) ([]*MigrateApply, error) {
	args := []string{"migrate", "apply", "--format", "{{ json . }}"}
	if params.Env != "" {
		args = append(args, "--env", params.Env)
	}
	if params.ConfigURL != "" {
		args = append(args, "--config", params.ConfigURL)
	}
	if params.Context != nil {
		buf, err := json.Marshal(params.Context)
		if err != nil {
			return nil, err
		}
		args = append(args, "--context", string(buf))
	}
	if params.URL != "" {
		args = append(args, "--url", params.URL)
	}
	if params.DirURL != "" {
		args = append(args, "--dir", params.DirURL)
	}
	if params.AllowDirty {
		args = append(args, "--allow-dirty")
	}
	if params.DryRun {
		args = append(args, "--dry-run")
	}
	if params.RevisionsSchema != "" {
		args = append(args, "--revisions-schema", params.RevisionsSchema)
	}
	if params.BaselineVersion != "" {
		args = append(args, "--baseline", params.BaselineVersion)
	}
	if params.TxMode != "" {
		args = append(args, "--tx-mode", params.TxMode)
	}
	if params.ExecOrder != "" {
		args = append(args, "--exec-order", string(params.ExecOrder))
	}
	if params.ToVersion != "" {
		args = append(args, "--to-version", params.ToVersion)
	}
	if params.Amount > 0 {
		args = append(args, strconv.FormatUint(params.Amount, 10))
	}
	if params.Vars != nil {
		args = append(args, params.Vars.AsArgs()...)
	}
	return jsonDecodeErr(newMigrateApplyError)(c.runCommand(ctx, args))
}

// MigrateDown runs the 'migrate down' command.
func (c *Client) MigrateDown(ctx context.Context, params *MigrateDownParams) (*MigrateDown, error) {
	args := []string{"migrate", "down", "--format", "{{ json . }}"}
	if params.Env != "" {
		args = append(args, "--env", params.Env)
	}
	if params.ConfigURL != "" {
		args = append(args, "--config", params.ConfigURL)
	}
	if params.DevURL != "" {
		args = append(args, "--dev-url", params.DevURL)
	}
	if params.Context != nil {
		buf, err := json.Marshal(params.Context)
		if err != nil {
			return nil, err
		}
		args = append(args, "--context", string(buf))
	}
	if params.URL != "" {
		args = append(args, "--url", params.URL)
	}
	if params.DirURL != "" {
		args = append(args, "--dir", params.DirURL)
	}
	if params.RevisionsSchema != "" {
		args = append(args, "--revisions-schema", params.RevisionsSchema)
	}
	if params.ToVersion != "" {
		args = append(args, "--to-version", params.ToVersion)
	}
	if params.ToTag != "" {
		args = append(args, "--to-tag", params.ToTag)
	}
	if params.Amount > 0 {
		args = append(args, strconv.FormatUint(params.Amount, 10))
	}
	if params.Vars != nil {
		args = append(args, params.Vars.AsArgs()...)
	}
	r, err := c.runCommand(ctx, args)
	if cliErr := (&Error{}); errors.As(err, &cliErr) && cliErr.Stderr == "" {
		r = strings.NewReader(cliErr.Stdout)
		err = nil
	}
	// NOTE: This command only support one result.
	return firstResult(jsonDecode[MigrateDown](r, err))
}

// MigrateTest runs the 'migrate test' command.
func (c *Client) MigrateTest(ctx context.Context, params *MigrateTestParams) (string, error) {
	args := []string{"migrate", "test"}
	if params.Env != "" {
		args = append(args, "--env", params.Env)
	}
	if params.ConfigURL != "" {
		args = append(args, "--config", params.ConfigURL)
	}
	if params.DirURL != "" {
		args = append(args, "--dir", params.DirURL)
	}
	if params.DirFormat != "" {
		args = append(args, "--dir-format", params.DirFormat)
	}
	if params.DevURL != "" {
		args = append(args, "--dev-url", params.DevURL)
	}
	if params.Context != nil {
		buf, err := json.Marshal(params.Context)
		if err != nil {
			return "", err
		}
		args = append(args, "--context", string(buf))
	}
	if params.RevisionsSchema != "" {
		args = append(args, "--revisions-schema", params.RevisionsSchema)
	}
	if params.Run != "" {
		args = append(args, "--run", params.Run)
	}
	if params.Vars != nil {
		args = append(args, params.Vars.AsArgs()...)
	}
	if len(params.Paths) > 0 {
		args = append(args, params.Paths...)
	}
	return stringVal(c.runCommand(ctx, args))
}

// MigrateStatus runs the 'migrate status' command.
func (c *Client) MigrateStatus(ctx context.Context, params *MigrateStatusParams) (*MigrateStatus, error) {
	args := []string{"migrate", "status", "--format", "{{ json . }}"}
	if params.Env != "" {
		args = append(args, "--env", params.Env)
	}
	if params.ConfigURL != "" {
		args = append(args, "--config", params.ConfigURL)
	}
	if params.URL != "" {
		args = append(args, "--url", params.URL)
	}
	if params.DirURL != "" {
		args = append(args, "--dir", params.DirURL)
	}
	if params.RevisionsSchema != "" {
		args = append(args, "--revisions-schema", params.RevisionsSchema)
	}
	if params.Vars != nil {
		args = append(args, params.Vars.AsArgs()...)
	}
	// NOTE: This command only support one result.
	return firstResult(jsonDecode[MigrateStatus](c.runCommand(ctx, args)))
}

// MigrateDiff runs the 'migrate diff --dry-run' command and returns the generated migration files without changing the filesystem.
// Requires atlas CLI to be logged in to the cloud.
func (c *Client) MigrateDiff(ctx context.Context, params *MigrateDiffParams) (*MigrateDiff, error) {
	args := []string{"migrate", "diff", "--dry-run"}
	if params.Env != "" {
		args = append(args, "--env", params.Env)
	}
	if params.ConfigURL != "" {
		args = append(args, "--config", params.ConfigURL)
	}
	if params.ToURL != "" {
		args = append(args, "--to", params.ToURL)
	}
	if params.DevURL != "" {
		args = append(args, "--dev-url", params.DevURL)
	}
	if params.DirURL != "" {
		args = append(args, "--dir", params.DirURL)
	}
	if params.DirFormat != "" {
		args = append(args, "--dir-format", params.DirFormat)
	}
	if params.LockTimeout != "" {
		args = append(args, "--lock-timeout", params.LockTimeout)
	}
	if params.Qualifier != "" {
		args = append(args, "--qualifier", params.Qualifier)
	}
	if len(params.Schema) > 0 {
		args = append(args, "--schema", strings.Join(params.Schema, ","))
	}
	if params.Format != "" {
		args = append(args, "--format", params.Format)
	}
	if params.Vars != nil {
		args = append(args, params.Vars.AsArgs()...)
	}
	if params.Name != "" {
		args = append(args, params.Name)
	}
	v, err := jsonDecode[MigrateDiff](c.runCommand(ctx, args))
	var e *Error
	switch {
	// if jsonDecode returns an error, and stderr is empty, it means the migration is synced with the desired state.
	case errors.As(err, &e) && e.Stderr == "":
		return &MigrateDiff{}, nil
	case err != nil:
		return nil, err
	}
	return firstResult(v, nil)
}

// MigrateLint runs the 'migrate lint' command.
func (c *Client) MigrateLint(ctx context.Context, params *MigrateLintParams) (*SummaryReport, error) {
	if params.Writer != nil || params.Web {
		return nil, errors.New("atlasexec: Writer or Web reporting are not supported with MigrateLint, use MigrateLintError")
	}
	args, err := params.AsArgs()
	if err != nil {
		return nil, err
	}
	r, err := c.runCommand(ctx, args)
	if cliErr := (&Error{}); errors.As(err, &cliErr) && cliErr.Stderr == "" {
		r = strings.NewReader(cliErr.Stdout)
		err = nil
	}
	// NOTE: This command only support one result.
	return firstResult(jsonDecode[SummaryReport](r, err))
}

// MigrateHash runs the 'migrate hash' command.
func (c *Client) MigrateHash(ctx context.Context, params *MigrateHashParams) error {
	args := []string{"migrate", "hash"}
	if params.Env != "" {
		args = append(args, "--env", params.Env)
	}
	if params.ConfigURL != "" {
		args = append(args, "--config", params.ConfigURL)
	}
	if params.Vars != nil {
		args = append(args, params.Vars.AsArgs()...)
	}
	if params.DirURL != "" {
		args = append(args, "--dir", params.DirURL)
	}
	if params.DirFormat != "" {
		args = append(args, "--dir-format", params.DirFormat)
	}
	_, err := c.runCommand(ctx, args)
	return err
}

// MigrateRebase runs the 'migrate rebase' command.
func (c *Client) MigrateRebase(ctx context.Context, params *MigrateRebaseParams) error {
	args := []string{"migrate", "rebase"}
	if params.Env != "" {
		args = append(args, "--env", params.Env)
	}
	if params.ConfigURL != "" {
		args = append(args, "--config", params.ConfigURL)
	}
	if params.Vars != nil {
		args = append(args, params.Vars.AsArgs()...)
	}
	if params.DirURL != "" {
		args = append(args, "--dir", params.DirURL)
	}
	args = append(args, strings.Join(params.Files, " "))
	_, err := c.runCommand(ctx, args)
	return err
}

// MigrateLintError runs the 'migrate lint' command, the output is written to params.Writer and reports
// if an error occurred. If the error is a setup error, a Error is returned. If the error is a lint error,
// LintErr is returned.
func (c *Client) MigrateLintError(ctx context.Context, params *MigrateLintParams) error {
	args, err := params.AsArgs()
	if err != nil {
		return err
	}
	r, err := c.runCommand(ctx, args)
	var (
		cliErr *Error
		isCLI  = errors.As(err, &cliErr)
	)
	// Setup errors.
	if isCLI && cliErr.Stderr != "" {
		return cliErr
	}
	// Lint errors.
	if isCLI && cliErr.Stdout != "" {
		err = ErrLint
		r = strings.NewReader(cliErr.Stdout)
	}
	// Unknown errors.
	if err != nil && !isCLI {
		return err
	}
	if params.Writer != nil && r != nil {
		if _, ioErr := io.Copy(params.Writer, r); ioErr != nil {
			err = errors.Join(err, ioErr)
		}
	}
	return err
}

// AsArgs returns the parameters as arguments.
func (p *MigrateLintParams) AsArgs() ([]string, error) {
	args := []string{"migrate", "lint"}
	if p.Web {
		args = append(args, "-w")
	}
	if p.Context != nil {
		buf, err := json.Marshal(p.Context)
		if err != nil {
			return nil, err
		}
		args = append(args, "--context", string(buf))
	}
	if p.Env != "" {
		args = append(args, "--env", p.Env)
	}
	if p.ConfigURL != "" {
		args = append(args, "--config", p.ConfigURL)
	}
	if p.DevURL != "" {
		args = append(args, "--dev-url", p.DevURL)
	}
	if p.DirURL != "" {
		args = append(args, "--dir", p.DirURL)
	}
	if p.Base != "" {
		args = append(args, "--base", p.Base)
	}
	if p.Latest > 0 {
		args = append(args, "--latest", strconv.FormatUint(p.Latest, 10))
	}
	if p.GitBase != "" {
		args = append(args, "--git-base", p.GitBase)
	}
	if p.GitDir != "" {
		args = append(args, "--git-dir", p.GitDir)
	}
	if p.Vars != nil {
		args = append(args, p.Vars.AsArgs()...)
	}
	format := "{{ json . }}"
	if p.Format != "" {
		format = p.Format
	}
	args = append(args, "--format", format)
	return args, nil
}

// Summary of the migration attempt.
func (a *MigrateApply) Summary(ident string) string {
	var (
		passedC, failedC int
		passedS, failedS int
		passedF, failedF int
		lines            = make([]string, 0, 3)
	)
	for _, f := range a.Applied {
		// For each check file, count the
		// number of failed assertions.
		for _, cf := range f.Checks {
			for _, s := range cf.Stmts {
				if s.Error != nil {
					failedC++
				} else {
					passedC++
				}
			}
		}
		passedS += len(f.Applied)
		if f.Error != nil {
			failedF++
			// Last statement failed (not an assertion).
			if len(f.Checks) == 0 || f.Checks[len(f.Checks)-1].Error == nil {
				passedS--
				failedS++
			}
		} else {
			passedF++
		}
	}
	// Execution time.
	lines = append(lines, a.End.Sub(a.Start).String())
	// Executed files.
	switch {
	case passedF > 0 && failedF > 0:
		lines = append(lines, fmt.Sprintf("%d migration%s ok, %d with errors", passedF, plural(passedF), failedF))
	case passedF > 0:
		lines = append(lines, fmt.Sprintf("%d migration%s", passedF, plural(passedF)))
	case failedF > 0:
		lines = append(lines, fmt.Sprintf("%d migration%s with errors", failedF, plural(failedF)))
	}
	// Executed checks.
	switch {
	case passedC > 0 && failedC > 0:
		lines = append(lines, fmt.Sprintf("%d check%s ok, %d failure%s", passedC, plural(passedC), failedC, plural(failedC)))
	case passedC > 0:
		lines = append(lines, fmt.Sprintf("%d check%s", passedC, plural(passedC)))
	case failedC > 0:
		lines = append(lines, fmt.Sprintf("%d check error%s", failedC, plural(failedC)))
	}
	// Executed statements.
	switch {
	case passedS > 0 && failedS > 0:
		lines = append(lines, fmt.Sprintf("%d sql statement%s ok, %d with errors", passedS, plural(passedS), failedS))
	case passedS > 0:
		lines = append(lines, fmt.Sprintf("%d sql statement%s", passedS, plural(passedS)))
	case failedS > 0:
		lines = append(lines, fmt.Sprintf("%d sql statement%s with errors", failedS, plural(failedS)))
	}
	var b strings.Builder
	for i, l := range lines {
		b.WriteString("-")
		b.WriteByte(' ')
		b.WriteString(fmt.Sprintf("**%s**", l))
		if i < len(lines)-1 {
			b.WriteByte('\n')
			b.WriteString(ident)
		}
	}
	return b.String()
}

var (
	// ErrLint is returned when the 'migrate lint' finds a diagnostic that is configured to
	// be reported as an error, such as destructive changes by default.
	ErrLint = errors.New("lint error")
	// Deprecated: Use ErrLint instead.
	LintErr = ErrLint
)

// LatestVersion returns the latest version of the migration directory.
func (r MigrateStatus) LatestVersion() string {
	if l := len(r.Available); l > 0 {
		return r.Available[l-1].Version
	}
	return ""
}

// Amount returns the number of migrations need to apply
// for the given version.
//
// The second return value is true if the version is found
// and the database is up-to-date.
//
// If the version is not found, it returns 0 and the second
// return value is false.
func (r MigrateStatus) Amount(version string) (amount uint64, ok bool) {
	if version == "" {
		amount := uint64(len(r.Pending))
		return amount, amount == 0
	}
	if r.Current == version {
		return amount, true
	}
	for idx, v := range r.Pending {
		if v.Version == version {
			amount = uint64(idx + 1) //nolint:gosec //G115: Safe conversion as idx is from range
			break
		}
	}
	return amount, false
}

func newMigrateApplyError(r []*MigrateApply, stderr string) error {
	return &MigrateApplyError{Result: r, Stderr: stderr}
}

// Error implements the error interface.
func (e *MigrateApplyError) Error() string {
	if e.Stderr != "" {
		return e.Stderr
	}
	return last(e.Result).Error
}

func plural(n int) (s string) {
	if n > 1 {
		s += "s"
	}
	return
}
