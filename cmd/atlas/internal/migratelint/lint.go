// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migratelint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/sqlparse"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/fatih/color"
)

type (
	// A ChangeDetector takes a migration directory and splits it into the "base" files (already merged) and new ones.
	ChangeDetector interface {
		// DetectChanges splits the files of a migration directory into the "base" files (already merged) and new ones.
		DetectChanges(context.Context) ([]migrate.File, []migrate.File, error)
	}

	// A ChangeLoader takes a set of migration files and will create multiple schema.Changes out of it.
	ChangeLoader interface {
		// LoadChanges converts each of the given migration files into one Changes.
		LoadChanges(context.Context, []migrate.File) (*Changes, error)
	}

	// Changes holds schema changes information returned by the loader.
	Changes struct {
		From, To *schema.Realm    // Current and desired schema.
		Files    []*sqlcheck.File // Files for moving from current to desired state.
	}
)

type (
	// GitChangeDetector implements the ChangeDetector interface by utilizing a git repository.
	GitChangeDetector struct {
		work string      // path to the git working directory (i.e. -C)
		base string      // name of the base branch (e.g. master)
		path string      // path of the migration directory relative to the repository root (in slash notation)
		dir  migrate.Dir // the migration directory to load migration files from
	}

	// GitChangeDetectorOption allows configuring GitChangeDetector with functional arguments.
	GitChangeDetectorOption func(*GitChangeDetector) error
)

// NewGitChangeDetector configures a new GitChangeDetector.
func NewGitChangeDetector(dir migrate.Dir, opts ...GitChangeDetectorOption) (*GitChangeDetector, error) {
	if dir == nil {
		return nil, errors.New("internal/ci: dir cannot be nil")
	}
	d := &GitChangeDetector{dir: dir}
	for _, opt := range opts {
		if err := opt(d); err != nil {
			return nil, err
		}
	}
	if d.base == "" {
		d.base = "master"
	}
	if d.path == "" {
		d.path = "migrations"
	}
	return d, nil
}

// WithWorkDir configures the git working directory for a GitChangeDetector.
func WithWorkDir(work string) GitChangeDetectorOption {
	return func(d *GitChangeDetector) error {
		d.work = work
		return nil
	}
}

// WithBase configures the git base branch name for a GitChangeDetector.
func WithBase(base string) GitChangeDetectorOption {
	return func(d *GitChangeDetector) error {
		d.base = base
		return nil
	}
}

// WithMigrationsPath configures the path for the migration directory.
func WithMigrationsPath(path string) GitChangeDetectorOption {
	return func(d *GitChangeDetector) error {
		d.path = filepath.ToSlash(path)
		return nil
	}
}

// DetectChanges implements the ChangeDetector interface.
func (d *GitChangeDetector) DetectChanges(ctx context.Context) ([]migrate.File, []migrate.File, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, nil, fmt.Errorf("lookup git: %w", err)
	}
	var args []string
	if d.work != "" {
		args = append(args, "-C", d.work)
	}
	args = append(args, "--no-pager", "diff", "--name-only", "--diff-filter=A", d.base, "HEAD", d.path)
	buf, err := exec.CommandContext(ctx, "git", args...).CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("git diff: %w", err)
	}
	diff := strings.Split(string(buf), "\n")
	names := make(map[string]struct{}, len(diff))
	for i := range diff {
		names[filepath.Base(diff[i])] = struct{}{}
	}
	files, err := d.dir.Files()
	if err != nil {
		return nil, nil, fmt.Errorf("reading migration directory: %w", err)
	}
	// Iterate over the migration files. If we find a file, that has been added in the diff between base and head,
	// every migration file preceding it can be considered old, the file itself and everything thereafter new,
	// since Atlas assumes a linear migration history.
	for i, f := range files {
		if _, ok := names[f.Name()]; ok {
			return files[:i], files[i:], nil
		}
	}
	return files, nil, nil
}

var (
	_ ChangeDetector = (*GitChangeDetector)(nil)
	_ ChangeDetector = (*DirChangeDetector)(nil)
)

// A DirChangeDetector implements the ChangeDetector
// interface by comparing two migration directories.
type DirChangeDetector struct {
	// Base and Head are the migration directories to compare.
	// Base represents the current state, Head the desired state.
	Base, Head migrate.Dir
}

// DetectChanges implements migratelint.ChangeDetector.
func (d DirChangeDetector) DetectChanges(context.Context) ([]migrate.File, []migrate.File, error) {
	baseS, err := d.Base.Checksum()
	if err != nil {
		return nil, nil, err
	}
	headS, err := d.Head.Checksum()
	if err != nil {
		return nil, nil, err
	}
	files, err := d.Head.Files()
	if err != nil {
		return nil, nil, err
	}
	for i := range headS {
		if len(baseS)-1 < i || baseS[i] != headS[i] {
			return files[:i], files[i:], nil
		}
	}
	return files, nil, nil
}

// latestChange implements the ChangeDetector by selecting the latest N files.
type latestChange struct {
	n   int         // number of (latest) files considered new.
	dir migrate.Dir // migration directory to load migration files from.
}

// LatestChanges implements the ChangeDetector interface by selecting the latest N files as new.
// It is useful for executing analysis on files in development before they are committed or on
// all files in a directory.
func LatestChanges(dir migrate.Dir, n int) ChangeDetector {
	return &latestChange{n: n, dir: dir}
}

// DetectChanges implements the ChangeDetector interface.
func (d *latestChange) DetectChanges(context.Context) ([]migrate.File, []migrate.File, error) {
	files, err := d.dir.Files()
	if err != nil {
		return nil, nil, fmt.Errorf("internal/ci: reading migration directory: %w", err)
	}
	// In case n is -1 or greater than the
	// number of files, return all files.
	if len(files) <= d.n || d.n < 0 {
		return nil, files, nil
	}
	return files[:len(files)-d.n], files[len(files)-d.n:], nil
}

// DevLoader implements the ChangesLoader interface using a dev-driver.
type DevLoader struct {
	// Dev environment used as a sandbox instantiated to the starting point (e.g. base branch).
	Dev *sqlclient.Client
}

// LoadChanges implements the ChangesLoader interface.
func (d *DevLoader) LoadChanges(ctx context.Context, base, files []migrate.File) (diff *Changes, err error) {
	unlock, err := d.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()
	// Clean up after ourselves.
	restore, err := d.Dev.Driver.Snapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("taking database snapshot: %w", err)
	}
	defer func() {
		if err2 := restore(ctx); err2 != nil {
			err = errors.Join(err, fmt.Errorf("restore dev-database snapshot: %w", err2))
		}
	}()
	current, err := d.base(ctx, base)
	if err != nil {
		return nil, err
	}
	diff = &Changes{
		From:  current,
		Files: make([]*sqlcheck.File, len(files)),
	}
	cks := make([]int, 0, len(files))
	for i, f := range files {
		diff.Files[i] = &sqlcheck.File{
			File:   f,
			From:   current,
			Parser: sqlparse.ParserFor(d.Dev.Name),
		}
		// Skip checkpoint files and process them separately at the end.
		if ck, ok := f.(migrate.CheckpointFile); ok && ck.IsCheckpoint() {
			cks = append(cks, i)
			continue
		}
		// A common case is when importing a project to Atlas the baseline
		// migration file might be very long. However, since we execute on
		// a clean database, the per-statement analysis is not needed.
		if len(base) == 0 && i == 0 {
			current, err = d.first(ctx, diff.Files[i], current)
		} else {
			current, err = d.next(ctx, diff.Files[i], current)
		}
		if err != nil {
			return nil, err
		}
		diff.Files[i].To = current
	}
	diff.To = current
	// For each checkpoint file, restore the dev environment
	// to the base point (clean) and load its changes.
	for _, i := range cks {
		if err := restore(ctx); err != nil {
			return nil, err
		}
		current, err := d.inspect(ctx)
		if err != nil {
			return nil, err
		}
		if _, err := d.next(ctx, diff.Files[i], current); err != nil {
			return nil, err
		}
	}
	return diff, nil
}

// base brings the dev environment to the base point and returns its state. It skips to the first checkpoint,
// if there is one, assuming the history is replay-able before that point as this was tested in previous runs.
func (d *DevLoader) base(ctx context.Context, base []migrate.File) (*schema.Realm, error) {
	if i := migrate.FilesLastIndex(base, func(f migrate.File) bool {
		ck, ok := f.(migrate.CheckpointFile)
		return ok && ck.IsCheckpoint()
	}); i != -1 {
		base = base[i:]
	}
	for _, f := range base {
		stmts, err := d.stmts(ctx, f, false)
		if err != nil {
			return nil, err
		}
		for _, s := range stmts {
			if _, err := d.Dev.ExecContext(ctx, s.Text); err != nil {
				return nil, &FileError{File: f.Name(), Err: fmt.Errorf("executing statement: %w", err), Pos: s.Pos}
			}
		}
	}
	return d.inspect(ctx)
}

// first is a version of "next" but is used when linting the first migration file. In this case we do not
// need to analyze each statement, but the entire result of the file (much faster). For example, a baseline
// file or the first migration when running 'schema apply' might contain thousands of lines.
func (d *DevLoader) first(ctx context.Context, f *sqlcheck.File, start *schema.Realm) (current *schema.Realm, err error) {
	stmts, err := d.stmts(ctx, f.File, true)
	if err != nil {
		return nil, err
	}
	// We define the max number of apply-inspect-diff cycles to 10,
	// to limit our linting time for baseline/first migration files.
	const maxStmtLoop = 10
	if len(stmts) <= maxStmtLoop {
		return d.nextStmts(ctx, f, stmts, start)
	}
	for _, s := range stmts {
		if _, err := d.Dev.ExecContext(ctx, s.Text); err != nil {
			return nil, &FileError{File: f.Name(), Err: fmt.Errorf("executing statement: %s: %w", s.Text, err), Pos: s.Pos}
		}
	}
	if current, err = d.inspect(ctx); err != nil {
		return nil, err
	}
	changes, err := d.Dev.RealmDiff(start, current)
	if err != nil {
		return nil, err
	}
	f.Changes = append(f.Changes, &sqlcheck.Change{
		Changes: changes,
		Stmt: &migrate.Stmt{
			Pos: 0, // Beginning of the file.
		},
	})
	f.Sum = changes
	return current, nil
}

// next returns the next state of the database after executing the statements in
// the file. The changes detected by the statements are attached to the file.
func (d *DevLoader) next(ctx context.Context, f *sqlcheck.File, start *schema.Realm) (*schema.Realm, error) {
	stmts, err := d.stmts(ctx, f.File, true)
	if err != nil {
		return nil, err
	}
	return d.nextStmts(ctx, f, stmts, start)
}

// nextStmts is a version of "next" but accepts the statements to execute.
func (d *DevLoader) nextStmts(ctx context.Context, f *sqlcheck.File, stmts []*migrate.Stmt, start *schema.Realm) (current *schema.Realm, err error) {
	current = start
	for _, s := range stmts {
		if _, err := d.Dev.ExecContext(ctx, s.Text); err != nil {
			return nil, &FileError{File: f.Name(), Err: fmt.Errorf("executing statement: %s: %w", s.Text, err), Pos: s.Pos}
		}
		next, err := d.inspect(ctx)
		if err != nil {
			return nil, err
		}
		changes, err := d.Dev.RealmDiff(current, next)
		if err != nil {
			return nil, err
		}
		current = next
		f.Changes = append(f.Changes, &sqlcheck.Change{
			Stmt:    s,
			Changes: d.mayFix(s.Text, changes),
		})
	}
	if f.Sum, err = d.Dev.RealmDiff(start, current); err != nil {
		return nil, err
	}
	return current, nil
}

// mayFix uses the sqlparse package for fixing or attaching more info to the changes.
func (d *DevLoader) mayFix(stmt string, changes schema.Changes) schema.Changes {
	p := sqlparse.ParserFor(d.Dev.Name)
	if p == nil {
		return changes
	}
	if fixed, err := p.FixChange(d.Dev.Driver, stmt, changes); err == nil {
		return fixed
	}
	return changes
}

// inspect the realm and filter by schema if we are connected to one.
func (d *DevLoader) inspect(ctx context.Context) (*schema.Realm, error) {
	if d.Dev.URL.Schema == "" {
		return d.Dev.InspectRealm(ctx, &schema.InspectRealmOption{})
	}
	ns, err := d.Dev.InspectSchema(ctx, "", &schema.InspectOptions{})
	if err != nil {
		return nil, err
	}
	// Normalize the returned realm to
	// look like InspectRealm output.
	if ns.Name == "" {
		ns.Name = d.Dev.URL.Schema
	}
	if ns.Realm == nil {
		ns.Realm = schema.NewRealm(ns)
	}
	return ns.Realm, nil
}

// lock database so no one else interferes with our change detection.
func (d *DevLoader) lock(ctx context.Context) (schema.UnlockFunc, error) {
	name := "atlas_lint"
	// In case the client is connected to specific schema,
	// minimize the lock resolution to the schema name.
	if s := d.Dev.URL.Schema; s != "" {
		name = fmt.Sprintf("%s_%s", name, s)
	}
	unlock, err := d.Dev.Driver.Lock(ctx, name, 0)
	if err != nil {
		return nil, fmt.Errorf("acquiring database lock: %w", err)
	}
	return unlock, nil
}

// FileError represents an error that occurred while processing a file.
type FileError struct {
	File string
	Err  error // Atlas or database error.
	Pos  int   // Position error, if known.
}

func (e FileError) Error() string { return e.Err.Error() }

func (e FileError) Unwrap() error { return e.Err }

// Runner is used to execute migration linting.
type Runner struct {
	// DevClient configures the "dev driver" to calculate
	// migration changes by the driver.
	Dev *sqlclient.Client

	// RunChangeDetector configures the ChangeDetector to
	// be used by the runner.
	ChangeDetector ChangeDetector

	// Dir is used for scanning and validating the migration directory.
	Dir migrate.Dir

	// Analyzers defines the analysis to run on each migration file.
	Analyzers []sqlcheck.Analyzer

	// ReportWriter writes the summary report.
	ReportWriter ReportWriter

	// summary report. reset on each run.
	sum *SummaryReport
}

// Run executes migration linting.
func (r *Runner) Run(ctx context.Context) error {
	switch err := r.summary(ctx); err.(type) {
	case nil:
		if err := r.ReportWriter.WriteReport(r.sum); err != nil {
			return err
		}
		// If any of the analyzers or the steps
		// returns an error, fail silently.
		for _, f := range r.sum.Files {
			if f.Error != "" {
				return SilentError{error: errors.New(f.Error)}
			}
		}
		for _, s := range r.sum.Steps {
			// Currently, we piggyback step errors
			// (such as non-linear) on FileReport.
			if s.Result != nil && s.Error != "" {
				return SilentError{error: errors.New(s.Error)}
			}
		}
		return nil
	case *FileError:
		if err := r.ReportWriter.WriteReport(r.sum); err != nil {
			return err
		}
		return SilentError{error: err}
	default:
		return err
	}
}

// A list of steps in CI report.
const (
	StepIntegrityCheck = "Migration Integrity Check"
	StepDetectChanges  = "Detect New Migration Files"
	StepLoadChanges    = "Replay Migration Files"
	StepAnalyzeFile    = "Analyze %s"
)

func (r *Runner) summary(ctx context.Context) error {
	r.sum = NewSummaryReport(r.Dev, r.Dir)
	defer func() { r.sum.End = time.Now() }()

	// Integrity check.
	switch err := migrate.Validate(r.Dir); {
	case errors.Is(err, migrate.ErrChecksumNotFound):
	case err != nil:
		var (
			err = &FileError{File: migrate.HashFileName, Err: err}
			rep = &FileReport{Name: migrate.HashFileName, Error: err.Error()}
		)
		if csErr := (&migrate.ChecksumError{}); errors.As(err, &csErr) {
			err.Pos = csErr.Pos
			rep = &FileReport{
				Name:  migrate.HashFileName,
				Error: fmt.Sprintf("%s (atlas.sum): L%d: %s was %s", csErr, csErr.Line, csErr.File, csErr.Reason),
			}
		}
		r.sum.Files = append(r.sum.Files, rep)
		return r.sum.StepError(StepIntegrityCheck, fmt.Sprintf("File %s is invalid", migrate.HashFileName), err)
	default:
		// If the hash file exists, it is valid.
		if _, err := fs.Stat(r.Dir, migrate.HashFileName); err == nil {
			r.sum.StepResult(StepIntegrityCheck, fmt.Sprintf("File %s is valid", migrate.HashFileName), nil)
		}
	}

	// Detect new migration files.
	base, feat, err := r.ChangeDetector.DetectChanges(ctx)
	switch err := err.(type) {
	// No error.
	case nil:
		r.sum.StepResult(StepDetectChanges, fmt.Sprintf("Found %d new migration files (from %d total)", len(feat), len(base)+len(feat)), nil)
	// Error that should be reported, but not halt the lint.
	case interface{ StepReport() *StepReport }:
		r.sum.Steps = append(r.sum.Steps, err.StepReport())
	default:
		return r.sum.StepError(StepDetectChanges, "Failed find new migration files", err)
	}
	if len(base) > 0 {
		r.sum.FromV = base[len(base)-1].Version()
	}
	if len(feat) > 0 {
		r.sum.ToV = feat[len(feat)-1].Version()
	}
	r.sum.TotalFiles = len(feat)

	// Load files into changes.
	l := &DevLoader{Dev: r.Dev}
	diff, err := l.LoadChanges(ctx, base, feat)
	if err != nil {
		if fr := (&FileError{}); errors.As(err, &fr) {
			r.sum.Files = append(r.sum.Files, &FileReport{Name: fr.File, Error: err.Error()})
		}
		return r.sum.StepError(StepLoadChanges, "Failed loading changes on dev database", err)
	}
	r.sum.StepResult(StepLoadChanges, fmt.Sprintf("Loaded %d changes on dev database", len(diff.Files)), nil)
	r.sum.WriteSchema(r.Dev, diff)

	// Analyze files.
	return r.analyze(ctx, diff.Files)
}

// analyze runs the analysis on the given files.
func (r *Runner) analyze(ctx context.Context, files []*sqlcheck.File) error {
	for _, f := range files {
		var (
			es []string
			nl = nolintRules(f)
			fr = NewFileReport(f)
		)
		if nl.ignored {
			continue
		}
		for _, az := range r.Analyzers {
			err := func(az sqlcheck.Analyzer) (rerr error) {
				defer func() {
					if rc := recover(); rc != nil {
						var name string
						if n, ok := az.(sqlcheck.NamedAnalyzer); ok {
							name = fmt.Sprintf(" (%s)", n.Name())
						}
						rerr = fmt.Errorf("skip crashed analyzer %s: %v", name, rc)
					}
				}()
				return az.Analyze(ctx, &sqlcheck.Pass{
					File:     f,
					Dev:      r.Dev,
					Reporter: nl.reporterFor(fr, az),
				})
			}(az)
			// If the last report was skipped,
			// skip emitting its error.
			if err != nil && !nl.skipped {
				es = append(es, err.Error())
			}
		}
		fr.Error = strings.Join(es, "; ")
		r.sum.Files = append(r.sum.Files, fr)
		r.sum.StepResult(
			fmt.Sprintf(StepAnalyzeFile, f.Name()),
			fmt.Sprintf("%d reports were found in analysis", len(fr.Reports)),
			fr,
		)
	}
	return nil
}

var (
	// TemplateFuncs are global functions available in templates.
	TemplateFuncs = template.FuncMap{
		"json": func(v any, args ...string) (string, error) {
			var (
				b   []byte
				err error
			)
			switch len(args) {
			case 0:
				b, err = json.Marshal(v)
			case 1:
				b, err = json.MarshalIndent(v, "", args[0])
			default:
				b, err = json.MarshalIndent(v, args[0], args[1])
			}
			return string(b), err
		},
		"sub":       func(i, j int) int { return i - j },
		"add":       func(i, j int) int { return i + j },
		"repeat":    strings.Repeat,
		"join":      strings.Join,
		"underline": color.New(color.Underline, color.Attribute(90)).Sprint,
		"gray":      color.New(color.Reset, color.Attribute(90)).Sprint,
		"lower":     strings.ToLower,
		"maxWidth": func(s string, n int) []string {
			var (
				j, k  int
				words = strings.Fields(s)
				lines = make([]string, 0, len(words))
			)
			for i := 0; i < len(words); i++ {
				if k+len(words[i]) > n {
					lines = append(lines, strings.Join(words[j:i], " "))
					k, j = 0, i
				}
				k += len(words[i])
			}
			return append(lines, strings.Join(words[j:], " "))
		},
		"cyan":         color.CyanString,
		"green":        color.HiGreenString,
		"red":          color.HiRedString,
		"redBgWhiteFg": color.New(color.FgHiWhite, color.BgHiRed).SprintFunc(),
		"yellow":       color.YellowString,
		"colorize": func(cc, text string) string {
			switch cc {
			case "cyan":
				return color.CyanString(text)
			case "green":
				return color.HiGreenString(text)
			case "red":
				return color.HiRedString(text)
			case "yellow":
				return color.YellowString(text)
			default:
				return text
			}
		},
	}
	// DefaultTemplate is the default template used by the CI job.
	DefaultTemplate = template.Must(template.New("report").
		Funcs(TemplateFuncs).
		Parse(`
{{- if or .Files .NonFileReports }}
  {{- $total := len .Files }}{{- with .TotalFiles }}{{- $total = . }}{{ end }}
  {{- $s := "s" }}{{ if eq $total 1 }}{{ $s = "" }}{{ end }}
  {{- if and .FromV .ToV }}
    {{- printf "Analyzing changes from version %s to %s" (cyan .FromV) (cyan .ToV) }}
  {{- else if .ToV }}
    {{- printf "Analyzing changes until version %s" (cyan .ToV) }}
  {{- else }}
    {{- printf "Analyzing changes" }}
  {{- end }}
  {{- if $total }}
    {{- printf " (%d migration%s in total):\n" $total $s }}
  {{- else }}
    {{- println ":" }}
  {{- end }}
  {{- println }}
  {{- with .NonFileReports }}
    {{- range $i, $s := . }}
      {{- println (yellow "  --") (lower $s.Name) }}
      {{- range $i, $r := $s.Result.Reports }}
        {{- if $r.Text }}
           {{- printf "    %s %s:\n" (yellow "--") $r.Text }}
        {{- end }}
        {{- range $d := $r.Diagnostics }}
          {{- $prefix := printf "    %s " (cyan "--") }}
          {{- print $prefix }}
          {{- $lines := maxWidth $d.Text (sub 85 (len $prefix)) }}
          {{- range $i, $line := $lines }}{{- if $i }}{{- print "       " }}{{- end }}{{- println $line }}{{- end }}
        {{- end }}
        {{- $fixes := $s.Result.SuggestedFixes }}
        {{- if $fixes }}
          {{- $s := "es" }}{{- if eq (len $fixes) 1 }}{{ $s = "" }}{{ end }}
          {{- printf "    %s suggested fix%s:\n" (yellow "--") $s }}
          {{- range $f := $fixes }}
            {{- $prefix := printf "      %s " (cyan "->") }}
            {{- print $prefix }}
            {{- $lines := maxWidth $f.Message (sub 85 (len $prefix)) }}
            {{- range $i, $line := $lines }}{{- if $i }}{{- print "         " }}{{- end }}{{- println $line }}{{- end }}
          {{- end }}
        {{- end }}
      {{- end }}
    {{- end }}
    {{- println }}
  {{- end }}
  {{- range $i, $f := .Files }}
    {{- /* Replay or checksum errors. */ -}}
    {{- if and $f.Error (eq $f.File nil) (eq $i (sub (len $.Files) 1)) }}
      {{- printf "  %s\n\n" (redBgWhiteFg (printf "Error: %s" $f.Error)) }}
      {{- break }}
    {{- end }}
    {{- $heading := printf "analyzing version %s" (cyan $f.Version) }}
    {{- $headinglen := len (printf "analyzing version %s" $f.Version) }}
    {{- println (yellow "  --") $heading }}
    {{- if and $f.Error (not $f.Reports) }}
       {{- printf "Error: %s\n" $f.Name $f.Error }}
       {{- continue }}
    {{- end }}
    {{- range $i, $r := $f.Reports }}
      {{- if $r.Text }}
         {{- printf "    %s %s:\n" (yellow "--") $r.Text }}
      {{- else if $r.Diagnostics }}
         {{- printf "    %s Unnamed diagnostics detected:\n" (yellow "--") }}
      {{- end }}
      {{- range $d := $r.Diagnostics }}
        {{- $prefix := printf "      %s L%d: " (cyan "--") ($f.Line $d.Pos) }}
        {{- print $prefix }}
        {{- $link := (underline (print "https://atlasgo.io/lint/analyzers#" $d.Code)) }}{{ if not $d.Code }}{{ $link = "" }}{{ end }}
        {{- $text := printf "%s %s" $d.Text $link }}
        {{- $lines := maxWidth $text (sub 85 (len $prefix)) }}
        {{- range $i, $line := $lines }}{{- if $i }}{{- print "         " }}{{- end }}{{- println $line }}{{- end }}
      {{- end }}
    {{- else }}
      {{- printf "    %s no diagnostics found\n" (cyan "--") }}
    {{- end }}
    {{- $fixes := $f.SuggestedFixes }}
    {{- if $fixes }}
      {{- $s := "es" }}{{- if eq (len $fixes) 1 }}{{ $s = "" }}{{ end }}
      {{- printf "    %s suggested fix%s:\n" (yellow "--") $s }}
      {{- range $f := $fixes }}
        {{- $prefix := printf "      %s " (cyan "->") }}
        {{- print $prefix }}
        {{- $lines := maxWidth $f.Message (sub 85 (len $prefix)) }}
        {{- range $i, $line := $lines }}{{- if $i }}{{- print "         " }}{{- end }}{{- println $line }}{{- end }}
      {{- end }}
    {{- end }}
    {{- if or (not $f.Error) $f.Reports }}
      {{- printf "  %s ok (%s)\n" (yellow "--") (yellow (.End.Sub .Start).String) }}
    {{- end }}
    {{- println }}
  {{- end }}
  {{- println (cyan "  -------------------------") }}
  {{- printf "  %s %s\n" (yellow "--") (.End.Sub .Start).String }}
  {{- with .VersionStatuses }}
	{{- printf "  %s %s\n" (yellow "--") . }}
  {{- end }}
  {{- with .TotalChanges }}
    {{- $s := "s" }}{{ if eq . 1 }}{{ $s = "" }}{{ end }}
	{{- printf "  %s %d schema change%s\n" (yellow "--") . $s }}
  {{- end }}
  {{- with .DiagnosticsCount }}
    {{- $s := "s" }}{{ if eq . 1 }}{{ $s = "" }}{{ end }}
	{{- printf "  %s %d diagnostic%s\n" (yellow "--") . $s }}
  {{- end }}
{{- end -}}
`))
	// JSONTemplate is the JSON template used by CI wrappers.
	JSONTemplate = template.Must(template.New("json").
		Funcs(TemplateFuncs).
		Parse("{{ json . }}"))
)

type (
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

		// Logging only info.
		Start      time.Time `json:"-"` // Start time of the analysis.
		End        time.Time `json:"-"` // End time of the analysis.
		FromV, ToV string    `json:"-"` // From and to versions.
		TotalFiles int       `json:"-"` // Total number of files to analyze.

		// A warning to be printed to the terminal, such as a license warning.
		Warning *struct {
			Title string
			Text  string
		} `json:"-"`
	}

	// FileChange specifies whether the file was added, deleted or changed.
	FileChange string

	// StepReport contains a summary of the analysis of a single step.
	StepReport struct {
		Name   string      `json:"Name,omitempty"`   // Step name.
		Text   string      `json:"Text,omitempty"`   // Step description.
		Error  string      `json:"Error,omitempty"`  // Error that cause the execution to halt.
		Result *FileReport `json:"Result,omitempty"` // Result of the step. For example, a diagnostic.

		// A warning to be printed to the terminal, such as a license warning.
		Warning struct {
			Title string `json:"Title,omitempty"`
			Text  string `json:"Text,omitempty"`
		} `json:"-"`
	}

	// FileReport contains a summary of the analysis of a single file.
	FileReport struct {
		Name    string            `json:"Name,omitempty"`    // Name of the file.
		Text    string            `json:"Text,omitempty"`    // Contents of the file.
		Reports []sqlcheck.Report `json:"Reports,omitempty"` // List of reports.
		Error   string            `json:"Error,omitempty"`   // File specific error.
		Change  FileChange        `json:"Change,omitempty"`  // Change of the file.

		// Logging only info.
		Start          time.Time `json:"-"` // Start time of the analysis.
		End            time.Time `json:"-"` // End time of the analysis.
		*sqlcheck.File `json:"-"`           // Underlying file.
	}

	// ReportWriter is a type of report writer that writes a summary of analysis reports.
	ReportWriter interface {
		WriteReport(*SummaryReport) error
	}

	// A TemplateWriter is a type of writer that writes output according to a template.
	TemplateWriter struct {
		T *template.Template
		W io.Writer
	}

	// SilentError is returned in case the wrapped error is already
	// printed by the runner and should not be printed by its caller
	SilentError struct{ error }
)

const (
	FileChangeAdded    FileChange = "ADDED"
	FileChangeDeleted  FileChange = "DELETED"
	FileChangeModified FileChange = "MODIFIED"
)

// NewSummaryReport returns a new SummaryReport.
func NewSummaryReport(c *sqlclient.Client, dir migrate.Dir) *SummaryReport {
	sum := &SummaryReport{
		Start: time.Now(),
		Env: struct {
			Driver string         `json:"Driver,omitempty"`
			URL    *sqlclient.URL `json:"URL,omitempty"`
			Dir    string         `json:"Dir,omitempty"`
		}{
			Driver: c.Name,
			URL:    c.URL,
		},
		Files: make([]*FileReport, 0),
	}
	if p, ok := dir.(interface{ Path() string }); ok {
		sum.Env.Dir = p.Path()
	}
	return sum
}

// StepResult appends step result to the summary.
func (r *SummaryReport) StepResult(name, text string, result *FileReport) {
	if result != nil {
		result.End = time.Now()
	}
	r.Steps = append(r.Steps, &StepReport{
		Name:   name,
		Text:   text,
		Result: result,
	})
}

// StepError appends step error to the summary.
func (r *SummaryReport) StepError(name, text string, err error) error {
	r.Steps = append(r.Steps, &StepReport{
		Name:  name,
		Text:  text,
		Error: err.Error(),
	})
	return err
}

// WriteSchema writes the current and desired schema to the summary.
func (r *SummaryReport) WriteSchema(c *sqlclient.Client, diff *Changes) {
	if curr, err := c.MarshalSpec(diff.From); err == nil {
		r.Schema.Current = string(curr)
	}
	if desired, err := c.MarshalSpec(diff.To); err == nil {
		r.Schema.Desired = string(desired)
	}
}

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

// VersionStatuses returns statuses description of all versions (migration files).
func (r *SummaryReport) VersionStatuses() string {
	var ok, errs, warns int
	for _, f := range r.Files {
		switch {
		case f.Error != "":
			errs++
		case len(f.Reports) > 0:
			warns++
		default:
			ok++
		}
	}
	parts := make([]string, 0, 3)
	for _, s := range []struct {
		n int
		s string
	}{
		{ok, "ok"},
		{warns, "with warnings"},
		{errs, "with errors"},
	} {
		switch {
		case s.n == 0:
		case s.n == 1 && len(parts) == 0:
			parts = append(parts, fmt.Sprintf("1 version %s", s.s))
		case s.n > 1 && len(parts) == 0:
			parts = append(parts, fmt.Sprintf("%d versions %s", s.n, s.s))
		default:
			parts = append(parts, fmt.Sprintf("%d %s", s.n, s.s))
		}
	}
	return strings.Join(parts, ", ")
}

// TotalChanges returns the total number of changes that were analyzed.
func (r *SummaryReport) TotalChanges() int {
	var n int
	for _, f := range r.Files {
		if f.File != nil {
			for _, c := range f.File.Changes {
				n += len(c.Changes)
			}
		}
	}
	return n
}

// NonFileReports returns reports that are not related to a file,
// but more general, like non-linear/additive changes.
func (r *SummaryReport) NonFileReports() (rs []*StepReport) {
	for _, s := range r.Steps {
		if r1 := s.Result; r1 != nil && r1.File == nil && len(r1.Reports) > 0 {
			rs = append(rs, s)
		}
	}
	return rs
}

// NewFileReport returns a new FileReport.
func NewFileReport(f *sqlcheck.File) *FileReport {
	return &FileReport{Name: f.Name(), Text: string(f.Bytes()), Start: time.Now(), File: f}
}

// Line returns the line number from a position.
func (f *FileReport) Line(pos int) int {
	return strings.Count(f.Text[:pos], "\n") + 1
}

// SuggestedFixes returns the list of suggested fixes for a specific report.
func (f *FileReport) SuggestedFixes() []sqlcheck.SuggestedFix {
	var fixes []sqlcheck.SuggestedFix
	for _, r := range f.Reports {
		// Report-level fixes.
		for _, x := range r.SuggestedFixes {
			if x.Message != "" {
				fixes = append(fixes, x)
			}
		}
		// Diagnostic-level fixes.
		for _, d := range r.Diagnostics {
			for _, x := range d.SuggestedFixes {
				if x.Message != "" {
					fixes = append(fixes, x)
				}
			}
		}
	}
	return fixes
}

// WriteReport implements sqlcheck.ReportWriter.
func (f *FileReport) WriteReport(r sqlcheck.Report) {
	f.Reports = append(f.Reports, r)
}

// WriteReport implements ReportWriter.
func (w *TemplateWriter) WriteReport(r *SummaryReport) error {
	return w.T.Execute(w.W, r)
}

func (err SilentError) Unwrap() error { return err.error }

func nolintRules(f *sqlcheck.File) *skipRules {
	s := &skipRules{pos2rules: make(map[int][]string)}
	if l, ok := f.File.(*migrate.LocalFile); ok {
		ds := l.Directive("nolint")
		// A file directive without specific classes/codes
		// (e.g. atlas:nolint) ignores the entire file.
		if s.ignored = len(ds) == 1 && ds[0] == ""; s.ignored {
			return s
		}
		// A file directive with specific classes/codes applies these
		// rules on all statements (e.g., atlas:nolint destructive).
		for _, d := range ds {
			for _, c := range f.Changes {
				s.pos2rules[c.Stmt.Pos] = append(s.pos2rules[c.Stmt.Pos], strings.Split(d, " ")...)
			}
		}
	}
	for _, c := range f.Changes {
		// A list of changes that were loaded in a batch (no statements per change).
		if c.Stmt != nil {
			for _, d := range c.Stmt.Directive("nolint") {
				s.pos2rules[c.Stmt.Pos] = append(s.pos2rules[c.Stmt.Pos], strings.Split(d, " ")...)
			}
		}
	}
	return s
}

type skipRules struct {
	pos2rules map[int][]string // statement positions to rules
	ignored   bool             // file is ignored. i.e., no analysis is performed
	skipped   bool             // if the last report was skipped by the rules
}

func (s *skipRules) reporterFor(rw sqlcheck.ReportWriter, az sqlcheck.Analyzer) sqlcheck.ReportWriter {
	return sqlcheck.ReportWriterFunc(func(r sqlcheck.Report) {
		var (
			ds     = make([]sqlcheck.Diagnostic, 0, len(r.Diagnostics))
			az, ok = az.(sqlcheck.NamedAnalyzer)
		)
		for _, d := range r.Diagnostics {
			switch rules := s.pos2rules[d.Pos]; {
			case
				// A directive without specific classes/codes
				// (e.g. atlas:nolint) ignore all diagnostics.
				len(rules) == 1 && rules[0] == "",
				// Match a specific code/diagnostic. e.g. atlas:nolint DS101.
				slices.Contains(rules, d.Code),
				// Skip the entire analyzer (class of changes).
				ok && slices.Contains(rules, az.Name()):
			default:
				ds = append(ds, d)
			}
		}
		if s.skipped = len(ds) == 0; !s.skipped {
			rw.WriteReport(sqlcheck.Report{Text: r.Text, Diagnostics: ds})
		}
	})
}

func (d *DevLoader) stmts(_ context.Context, f migrate.File, _ bool) ([]*migrate.Stmt, error) {
	stmts, err := migrate.FileStmtDecls(d.Dev.Driver, f)
	if err != nil {
		return nil, &FileError{File: f.Name(), Err: fmt.Errorf("scanning statements: %w", err)}
	}
	return stmts, nil
}
