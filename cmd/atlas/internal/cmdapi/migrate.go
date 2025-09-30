// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"text/template/parse"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/cloudapi"
	"ariga.io/atlas/cmd/atlas/internal/cmdext"
	"ariga.io/atlas/cmd/atlas/internal/cmdlog"
	cmdmigrate "ariga.io/atlas/cmd/atlas/internal/migrate"
	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/revision"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"ariga.io/atlas/sql/sqltool"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// migrateCmd represents the subcommand 'atlas migrate'.
func migrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Manage versioned migration files",
		Long:  "'atlas migrate' wraps several sub-commands for migration management.",
	}
	addGlobalFlags(cmd.PersistentFlags())
	return cmd
}

type migrateApplyFlags struct {
	url             string
	dirURL          string
	dirFormat       string
	revisionSchema  string
	dryRun          bool
	logFormat       string
	lockTimeout     time.Duration
	allowDirty      bool   // allow working on a database that already has resources
	baselineVersion string // apply with this version as baseline
	txMode          string // (none, file, all)
	execOrder       string // (linear, linear-skip, non-linear)
	context         string // Run context. See cloudapi.DeployContextInput.
}

func (f *migrateApplyFlags) migrateOptions() ([]migrate.ExecutorOption, error) {
	var opts []migrate.ExecutorOption
	if f.allowDirty {
		opts = append(opts, migrate.WithAllowDirty(true))
	}
	if v := f.baselineVersion; v != "" {
		opts = append(opts, migrate.WithBaselineVersion(v))
	}
	if v := f.execOrder; v != "" && v != execOrderLinear {
		switch v {
		case execOrderLinearSkip:
			opts = append(opts, migrate.WithExecOrder(migrate.ExecOrderLinearSkip))
		case execOrderNonLinear:
			opts = append(opts, migrate.WithExecOrder(migrate.ExecOrderNonLinear))
		default:
			return nil, fmt.Errorf("unknown execution order: %q", v)
		}
	}
	return opts, nil
}

func migrateApplyCmd() *cobra.Command {
	var (
		flags migrateApplyFlags
		cmd   = &cobra.Command{
			Use:   "apply [flags] [amount]",
			Short: "Applies pending migration files on the connected database.",
			Long: `'atlas migrate apply' reads the migration state of the connected database and computes what migrations are pending.
It then attempts to apply the pending migration files in the correct order onto the database. 
The first argument denotes the maximum number of migration files to apply.
As a safety measure 'atlas migrate apply' will abort with an error, if:
  - the migration directory is not in sync with the 'atlas.sum' file
  - the migration and database history do not match each other

If run with the "--dry-run" flag, atlas will not execute any SQL.`,
			Example: `  atlas migrate apply -u "mysql://user:pass@localhost:3306/dbname"
  atlas migrate apply --dir "file:///path/to/migration/directory" --url "mysql://user:pass@localhost:3306/dbname" 1
  atlas migrate apply --env dev 1
  atlas migrate apply --dry-run --env dev 1`,
			Args: cobra.MaximumNArgs(1),
			RunE: RunE(func(cmd *cobra.Command, args []string) (cmdErr error) {
				switch {
				case GlobalFlags.SelectedEnv == "":
					// Env not selected, but the
					// -c flag might be set.
					env, err := selectEnv(cmd)
					if err != nil {
						return err
					}
					if err := setMigrateEnvFlags(cmd, env); err != nil {
						return err
					}
					return migrateApplyRun(cmd, args, flags, env, &MigrateReport{}) // nop reporter
				default:
					project, envs, err := EnvByName(cmd, GlobalFlags.SelectedEnv, GlobalFlags.Vars)
					if err != nil {
						return err
					}
					set, err := NewReportProvider(cmd.Context(), project, envs, &flags)
					if err != nil {
						return err
					}
					var hasRemote bool
					defer func() {
						if hasRemote {
							set.Flush(cmd, cmdErr)
						}
					}()
					return cmdEnvsRun(envs, setMigrateEnvFlags, cmd, func(env *Env) error {
						// Report deployments only if one of the migration directories is a cloud directory.
						if u, err := url.Parse(flags.dirURL); err == nil && u.Scheme == cmdmigrate.DirTypeAtlas {
							hasRemote = true
						}
						return migrateApplyRun(cmd, args, flags, env, set.ReportFor(flags, env))
					})
				}
			}),
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURL(cmd.Flags(), &flags.url)
	addFlagDirURL(cmd.Flags(), &flags.dirURL)
	addFlagDirFormat(cmd.Flags(), &flags.dirFormat)
	addFlagLog(cmd.Flags(), &flags.logFormat)
	addFlagFormat(cmd.Flags(), &flags.logFormat)
	addFlagRevisionSchema(cmd.Flags(), &flags.revisionSchema)
	addFlagDryRun(cmd.Flags(), &flags.dryRun)
	addFlagLockTimeout(cmd.Flags(), &flags.lockTimeout)
	cmd.Flags().StringVarP(&flags.baselineVersion, flagBaseline, "", "", "start the first migration after the given baseline version")
	cmd.Flags().StringVarP(&flags.txMode, flagTxMode, "", txModeFile, "set transaction mode [none, file, all]")
	cmd.Flags().StringVarP(&flags.execOrder, flagExecOrder, "", execOrderLinear, "set file execution order [linear, linear-skip, non-linear]")
	cmd.Flags().StringVar(&flags.context, flagContext, "", "describes what triggered this command (e.g., GitHub Action)")
	cobra.CheckErr(cmd.Flags().MarkHidden(flagContext))
	cmd.Flags().BoolVarP(&flags.allowDirty, flagAllowDirty, "", false, "allow start working on a non-clean database")
	cmd.MarkFlagsMutuallyExclusive(flagLog, flagFormat)
	return cmd
}

type (
	// MigrateReport responsible for reporting 'migrate apply' reports.
	MigrateReport struct {
		id     string // target id
		env    *Env   // nil, if no env set
		client *sqlclient.Client
		log    *cmdlog.MigrateApply
		rrw    cmdmigrate.RevisionReadWriter
		done   func(*cloudapi.ReportMigrationInput)
	}
	// MigrateReportSet is a set of reports.
	MigrateReportSet struct {
		cloudapi.ReportMigrationSetInput
		client *cloudapi.Client
		done   int // number of done migrations
	}
)

// NewReportProvider returns a new ReporterProvider.
func NewReportProvider(ctx context.Context, p *Project, envs []*Env, flags *migrateApplyFlags) (*MigrateReportSet, error) {
	c := cloudapi.FromContext(ctx)
	if p.cloud.Client != nil {
		c = p.cloud.Client
	}
	s := &MigrateReportSet{
		client: c,
		ReportMigrationSetInput: cloudapi.ReportMigrationSetInput{
			ID:        uuid.NewString(),
			StartTime: time.Now(),
			Planned:   len(envs),
		},
	}
	if flags.context != "" {
		if err := json.Unmarshal([]byte(flags.context), &s.Context); err != nil {
			return nil, fmt.Errorf("invalid --context: %w", err)
		}
	}
	s.Step("Start migration for %d targets", len(envs))
	for _, e := range envs {
		s.StepLog(s.RedactedURL(e.URL))
	}
	return s, nil
}

// RedactedURL returns the redacted URL of the given environment at index i.
func (*MigrateReportSet) RedactedURL(u string) string {
	u, err := cloudapi.RedactedURL(u)
	if err != nil {
		return fmt.Sprintf("Error: redacting URL: %v", err)
	}
	return u
}

// Step starts a new reporting step.
func (s *MigrateReportSet) Step(format string, args ...interface{}) {
	if len(s.Log) > 0 && s.Log[len(s.Log)-1].EndTime.IsZero() {
		s.Log[len(s.Log)-1].EndTime = time.Now()
	}
	s.Log = append(s.Log, cloudapi.ReportStep{
		StartTime: time.Now(),
		Text:      fmt.Sprintf(format, args...),
	})
}

// StepLog logs a line to the current reporting step.
func (s *MigrateReportSet) StepLog(text string) {
	if len(s.Log) == 0 {
		s.Step("Unnamed step") // Unexpected.
	}
	s.Log[len(s.Log)-1].Log = append(s.Log[len(s.Log)-1].Log, cloudapi.ReportStepLog{
		Text: text,
	})
}

// StepLogf logs a line to the current reporting step with formatting.
func (s *MigrateReportSet) StepLogf(format string, args ...interface{}) {
	s.StepLog(fmt.Sprintf(format, args...))
}

// StepLogError logs a line to the current reporting step.
func (s *MigrateReportSet) StepLogError(text string) {
	if !strings.HasPrefix(text, "Error") {
		text = "Error: " + text
	}
	s.StepLog(text)
	s.Error = &text
	s.Log[len(s.Log)-1].Error = true
}

// ReportFor returns a new MigrateReport for the given environment.
func (s *MigrateReportSet) ReportFor(flags migrateApplyFlags, e *Env) *MigrateReport {
	s.Step("Run migration: %d", s.done+1)
	s.StepLogf("Target URL: %s", s.RedactedURL(e.URL))
	s.StepLogf("Migration directory: %s", s.RedactedURL(flags.dirURL))
	return &MigrateReport{
		env: e,
		done: func(r *cloudapi.ReportMigrationInput) {
			s.done++
			r.DryRun = flags.dryRun
			s.Log[len(s.Log)-1].EndTime = time.Now()
			if r.Error != nil && *r.Error != "" {
				s.StepLogError(*r.Error)
			}
			s.Completed = append(s.Completed, *r)
		},
	}
}

// Flush report the migration deployment to the cloud.
// The current implementation is simplistic and sends each
// report separately without marking them as part of a group.
//
// Note that reporting errors are logged, but not cause Atlas to fail.
func (s *MigrateReportSet) Flush(cmd *cobra.Command, cmdErr error) {
	if cmdErr != nil && s.Error == nil {
		var uerr *url.Error
		if errors.As(cmdErr, &uerr) {
			uerr.URL = ""
			cmdErr = uerr
		}
		s.StepLogError(cmdErr.Error())
	}
	var (
		err  error
		link string
	)
	switch {
	// Skip reporting if set is empty,
	// or there is no cloud connectivity.
	case s.Planned == 0, s.client == nil:
		return
	// Single migration that was completed.
	case s.Planned == 1 && len(s.Completed) == 1:
		s.Completed[0].Context = s.Context
		link, err = s.client.ReportMigration(cmd.Context(), s.Completed[0])
	// Single migration that failed to start.
	case s.Planned == 1 && len(s.Completed) == 0:
		s.EndTime = time.Now()
		link, err = s.client.ReportMigrationSet(cmd.Context(), s.ReportMigrationSetInput)
	// Multi environment migration (e.g., multi-tenancy).
	case s.Planned > 1:
		s.EndTime = time.Now()
		link, err = s.client.ReportMigrationSet(cmd.Context(), s.ReportMigrationSetInput)
	}
	switch {
	case err != nil:
		txt := fmt.Sprintf("Error: %s", strings.TrimRight(err.Error(), "\n"))
		// Ensure errors are printed in new lines.
		if cmd.Flags().Changed(flagFormat) {
			txt = "\n" + txt
		}
		cmd.PrintErrln(txt)
	// Unlike errors that are printed to stderr, links are printed to stdout.
	// We do it only if the format was not customized by the user (e.g., JSON).
	case link != "" && !cmd.Flags().Changed(flagFormat):
		cmd.Println(link)
	}
}

// Init the report if the necessary dependencies.
func (r *MigrateReport) Init(c *sqlclient.Client, l *cmdlog.MigrateApply, rrw cmdmigrate.RevisionReadWriter) {
	r.client, r.log, r.rrw = c, l, rrw
}

// RecordTargetID asks the revisions-table to allow or provide
// the target identifier if cloud reporting is enabled.
func (r *MigrateReport) RecordTargetID(ctx context.Context) error {
	if r.CloudEnabled(ctx) {
		id, err := r.rrw.ID(ctx, operatorVersion())
		if err != nil {
			return err
		}
		r.id = id
	}
	return nil
}

// RecordPlanError records any errors that occurred during the planning phase. i.e., when calling to ex.Pending.
func (r *MigrateReport) RecordPlanError(cmd *cobra.Command, flags migrateApplyFlags, planerr string) {
	if !r.CloudEnabled(cmd.Context()) {
		return
	}
	var ver string
	if rev, err := r.rrw.CurrentRevision(cmd.Context()); err == nil {
		ver = rev.Version
	}
	r.done(&cloudapi.ReportMigrationInput{
		ProjectName:  r.env.config.cloud.Project,
		EnvName:      r.env.Name,
		DirName:      r.DirName(flags),
		AtlasVersion: operatorVersion(),
		Target: cloudapi.DeployedTargetInput{
			ID:     r.id,
			Schema: r.client.URL.Schema,
			URL:    r.client.URL.Redacted(),
		},
		StartTime:      r.log.Start,
		EndTime:        r.log.End,
		FromVersion:    r.log.Current,
		ToVersion:      r.log.Target,
		CurrentVersion: ver,
		Error:          &planerr,
		Log:            planerr,
	})
}

// Done closes and flushes this report.
func (r *MigrateReport) Done(cmd *cobra.Command, flags migrateApplyFlags) error {
	if !r.CloudEnabled(cmd.Context()) {
		return logApply(cmd, cmd.OutOrStdout(), flags, r.log)
	}
	var (
		ver  string
		clog bytes.Buffer
		err  = logApply(cmd, io.MultiWriter(cmd.OutOrStdout(), &clog), flags, r.log)
	)
	switch rev, err1 := r.rrw.CurrentRevision(cmd.Context()); {
	case errors.Is(err1, migrate.ErrRevisionNotExist):
	case err1 != nil:
		return errors.Join(err, err1)
	default:
		ver = rev.Version
	}
	r.done(&cloudapi.ReportMigrationInput{
		ProjectName:  r.env.config.cloud.Project,
		EnvName:      r.env.Name,
		DirName:      r.DirName(flags),
		AtlasVersion: operatorVersion(),
		Target: cloudapi.DeployedTargetInput{
			ID:     r.id,
			Schema: r.client.URL.Schema,
			URL:    r.client.URL.Redacted(),
		},
		StartTime:      r.log.Start,
		EndTime:        r.log.End,
		FromVersion:    r.log.Current,
		ToVersion:      r.log.Target,
		CurrentVersion: ver,
		Error: func() *string {
			if r.log.Error != "" {
				return &r.log.Error
			}
			return nil
		}(),
		Files: func() []cloudapi.DeployedFileInput {
			files := make([]cloudapi.DeployedFileInput, len(r.log.Applied))
			for i, f := range r.log.Applied {
				f1 := cloudapi.DeployedFileInput{
					Name:      f.Name(),
					Content:   string(f.Bytes()),
					StartTime: f.Start,
					EndTime:   f.End,
					Skipped:   f.Skipped,
					Applied:   len(f.Applied),
					Error:     (*cloudapi.StmtErrorInput)(f.Error),
					Checks:    make([]cloudapi.FileChecksInput, 0, len(f.Checks)),
				}
				for _, c := range f.Checks {
					stmts := make([]cloudapi.CheckStmtInput, 0, len(c.Stmts))
					for _, s := range c.Stmts {
						stmts = append(stmts, cloudapi.CheckStmtInput{
							Stmt:  s.Stmt,
							Error: s.Error,
						})
					}
					f1.Checks = append(f1.Checks, cloudapi.FileChecksInput{
						Name:   c.Name,
						Start:  c.Start,
						End:    c.End,
						Checks: stmts,
						Error:  (*cloudapi.StmtErrorInput)(c.Error),
					})
				}
				files[i] = f1
			}
			return files
		}(),
		Log: clog.String(),
	})
	return err
}

// DirName returns the directory name for the report.
func (r *MigrateReport) DirName(flags migrateApplyFlags) string {
	dirName := flags.dirURL
	switch u, err := url.Parse(flags.dirURL); {
	case err != nil:
	// Local directories are reported as (dangling)
	// deployments without a directory.
	case u.Scheme == cmdmigrate.DirTypeFile:
		dirName = cloudapi.DefaultDirName
	// Directory slug.
	default:
		dirName = path.Join(u.Host, u.Path)
	}
	return dirName
}

// CloudEnabled reports if cloud reporting is enabled.
func (r *MigrateReport) CloudEnabled(ctx context.Context) bool {
	if r.env == nil || r.env.cloud == nil {
		return false // The --env was not set.
	}
	cloud := r.env.cloud
	// Cloud reporting is enabled only if there is a cloud connection.
	return cloud.Project != "" && (cloud.Client != nil || cloudapi.FromContext(ctx) != nil)
}

func logApply(cmd *cobra.Command, w io.Writer, flags migrateApplyFlags, r *cmdlog.MigrateApply) error {
	var (
		err error
		f   = cmdlog.MigrateApplyTemplate
	)
	if v := flags.logFormat; v != "" {
		f, err = template.New("format").Funcs(cmdlog.ApplyTemplateFuncs).Parse(v)
		if err != nil {
			return fmt.Errorf("parse format: %w", err)
		}
	}
	if err = f.Execute(w, r); err != nil {
		return fmt.Errorf("execute log template: %w", err)
	}
	// In case a custom logging was configured, avoid reporting errors twice.
	// For example, printing error lines may break parsing the JSON output.
	cmd.SilenceErrors = flags.logFormat != ""
	return nil
}

type migrateDiffFlags struct {
	edit              bool
	desiredURLs       []string
	dirURL, dirFormat string
	devURL            string
	schemas           []string
	lockTimeout       time.Duration
	format            string
	qualifier         string // optional table qualifier
	dryRun            bool
}

// migrateDiffCmd represents the 'atlas migrate diff' subcommand.
func migrateDiffCmd() *cobra.Command {
	var (
		flags migrateDiffFlags
		cmd   = &cobra.Command{
			Use:   "diff [flags] [name]",
			Short: "Compute the diff between the migration directory and a desired state and create a new migration file.",
			Long: `The 'atlas migrate diff' command uses the dev-database to calculate the current state of the migration directory
by executing its files. It then compares its state to the desired state and create a new migration file containing
SQL statements for moving from the current to the desired state. The desired state can be another another database,
an HCL, SQL, or ORM schema. See: https://atlasgo.io/versioned/diff`,
			Example: `  atlas migrate diff --dev-url "docker://mysql/8/dev" --to "file://schema.hcl"
  atlas migrate diff --dev-url "docker://postgres/15/dev?search_path=public" --to "file://atlas.hcl" add_users_table
  atlas migrate diff --dev-url "mysql://user:pass@localhost:3306/dev" --to "mysql://user:pass@localhost:3306/dbname"
  atlas migrate diff --env dev --format '{{ sql . "  " }}'`,
			Args: cobra.MaximumNArgs(1),
			PreRunE: func(cmd *cobra.Command, args []string) error {
				if err := migrateFlagsFromConfig(cmd); err != nil {
					return err
				}
				if err := dirFormatBC(flags.dirFormat, &flags.dirURL); err != nil {
					return err
				}
				return checkDir(cmd, flags.dirURL, true)
			},
			RunE: RunE(func(cmd *cobra.Command, args []string) error {
				env, err := selectEnv(cmd)
				if err != nil {
					return err
				}
				return migrateDiffRun(cmd, args, flags, env)
			}),
		}
	)
	cmd.Flags().SortFlags = false
	addFlagToURLs(cmd.Flags(), &flags.desiredURLs)
	addFlagDevURL(cmd.Flags(), &flags.devURL)
	addFlagDirURL(cmd.Flags(), &flags.dirURL)
	addFlagDirFormat(cmd.Flags(), &flags.dirFormat)
	addFlagSchemas(cmd.Flags(), &flags.schemas)
	addFlagLockTimeout(cmd.Flags(), &flags.lockTimeout)
	addFlagFormat(cmd.Flags(), &flags.format)
	cmd.Flags().StringVar(&flags.qualifier, flagQualifier, "", "qualify tables with custom qualifier when working on a single schema")
	cmd.Flags().BoolVarP(&flags.edit, flagEdit, "", false, "edit the generated migration file(s)")
	cmd.Flags().BoolVar(&flags.dryRun, flagDryRun, false, "print the generated file to stdout instead of writing it to the migration directory")
	cobra.CheckErr(cmd.Flags().MarkHidden(flagDryRun))
	cmd.MarkFlagsMutuallyExclusive(flagEdit, flagDryRun)
	cobra.CheckErr(cmd.MarkFlagRequired(flagTo))
	cobra.CheckErr(cmd.MarkFlagRequired(flagDevURL))
	return cmd
}

func mayIndent(dir *url.URL, f migrate.Formatter, format string) (migrate.Formatter, string, error) {
	if format == "" {
		return f, "", nil
	}
	reject := errors.New(`'sql' can only be used to indent statements`)
	t, err := template.New("format").
		// The "sql" is a dummy function to detect if the
		// template was used to indent the SQL statements.
		Funcs(template.FuncMap{"sql": func(...any) (string, error) { return "", reject }}).
		Parse(format)
	if err != nil {
		return nil, "", fmt.Errorf("parse format: %w", err)
	}
	indent, ok := func() (string, bool) {
		if len(t.Tree.Root.Nodes) != 1 {
			return "", false
		}
		n, ok := t.Tree.Root.Nodes[0].(*parse.ActionNode)
		if !ok || len(n.Pipe.Cmds) != 1 || len(n.Pipe.Cmds[0].Args) < 2 || len(n.Pipe.Cmds[0].Args) > 3 {
			return "", false
		}
		args := n.Pipe.Cmds[0].Args
		if args[0].String() != "sql" || args[1].String() != "." && args[1].String() != "$" {
			return "", false
		}
		d := `""` // empty string as arg.
		if len(args) == 3 {
			d = args[2].String()
		}
		return d, true
	}()
	if ok {
		if indent, err = strconv.Unquote(indent); err != nil {
			return nil, "", fmt.Errorf("parse indent: %w", err)
		}
		return f, indent, nil
	}
	// If the template is not an indent, it cannot contain the "sql" function.
	if err := t.Execute(io.Discard, &migrate.Plan{}); err != nil && errors.Is(err, reject) {
		return nil, "", fmt.Errorf("%v. got: %v", reject, t.Root.String())
	}
	tfs := f.(migrate.TemplateFormatter)
	if len(tfs) != 1 {
		return nil, "", fmt.Errorf("cannot use format with: %q", dir.Query().Get("format"))
	}
	return migrate.TemplateFormatter{{N: tfs[0].N, C: t}}, "", nil
}

// maskNoPlan masks ErrNoPlan errors.
func maskNoPlan(cmd *cobra.Command, err error) error {
	if errors.Is(err, migrate.ErrNoPlan) {
		cmd.Println("The migration directory is synced with the desired state, no changes to be made")
		return nil
	}
	return err
}

type migrateHashFlags struct{ dirURL, dirFormat string }

// migrateHashCmd represents the 'atlas migrate hash' subcommand.
func migrateHashCmd() *cobra.Command {
	var (
		flags migrateHashFlags
		cmd   = &cobra.Command{
			Use:   "hash [flags]",
			Short: "Hash (re-)creates an integrity hash file for the migration directory.",
			Long: `'atlas migrate hash' computes the integrity hash sum of the migration directory and stores it in the atlas.sum file.
This command should be used whenever a manual change in the migration directory was made.`,
			Example: `  atlas migrate hash`,
			PreRunE: func(cmd *cobra.Command, args []string) error {
				if err := migrateFlagsFromConfig(cmd); err != nil {
					return err
				}
				return dirFormatBC(flags.dirFormat, &flags.dirURL)
			},
			RunE: RunE(func(cmd *cobra.Command, args []string) error {
				dir, err := cmdmigrate.Dir(cmd.Context(), flags.dirURL, false)
				if err != nil {
					return err
				}
				sum, err := dir.Checksum()
				if err != nil {
					return err
				}
				return migrate.WriteSumFile(dir, sum)
			}),
		}
	)
	addFlagDirURL(cmd.Flags(), &flags.dirURL)
	addFlagDirFormat(cmd.Flags(), &flags.dirFormat)
	cmd.Flags().Bool("force", false, "")
	cobra.CheckErr(cmd.Flags().MarkDeprecated("force", "you can safely omit it."))
	return cmd
}

type migrateImportFlags struct{ fromURL, toURL, dirFormat string }

// migrateImportCmd represents the 'atlas migrate import' subcommand.
func migrateImportCmd() *cobra.Command {
	var (
		flags migrateImportFlags
		cmd   = &cobra.Command{
			Use:     "import [flags]",
			Short:   "Import a migration directory from another migration management tool to the Atlas format.",
			Example: `  atlas migrate import --from "file:///path/to/source/directory?format=liquibase" --to "file:///path/to/migration/directory"`,
			// Validate the source directory. Consider a directory with no sum file
			// valid, since it might be an import from an existing project.
			PreRunE: func(cmd *cobra.Command, _ []string) error {
				if err := migrateFlagsFromConfig(cmd); err != nil {
					return err
				}
				if err := dirFormatBC(flags.dirFormat, &flags.fromURL); err != nil {
					return err
				}
				d, err := cmdmigrate.Dir(cmd.Context(), flags.fromURL, false)
				if err != nil {
					return err
				}
				if err = migrate.Validate(d); err != nil && !errors.Is(err, migrate.ErrChecksumNotFound) {
					printChecksumError(cmd, err)
					return err
				}
				return nil
			},
			RunE: RunE(func(cmd *cobra.Command, args []string) error {
				return migrateImportRun(cmd, args, flags)
			}),
		}
	)
	cmd.Flags().SortFlags = false
	addFlagDirURL(cmd.Flags(), &flags.fromURL, flagFrom)
	addFlagDirURL(cmd.Flags(), &flags.toURL, flagTo)
	addFlagDirFormat(cmd.Flags(), &flags.dirFormat)
	return cmd
}

func migrateImportRun(cmd *cobra.Command, _ []string, flags migrateImportFlags) error {
	p, err := url.Parse(flags.fromURL)
	if err != nil {
		return err
	}
	if f := p.Query().Get("format"); f == "" || f == cmdmigrate.FormatAtlas {
		return fmt.Errorf("cannot import a migration directory already in %q format", cmdmigrate.FormatAtlas)
	}
	src, err := cmdmigrate.Dir(cmd.Context(), flags.fromURL, false)
	if err != nil {
		return err
	}
	trgt, err := cmdmigrate.Dir(cmd.Context(), flags.toURL, true)
	if err != nil {
		return err
	}
	// Target must be empty.
	ff, err := trgt.Files()
	switch {
	case err != nil:
		return err
	case len(ff) != 0:
		return errors.New("target migration directory must be empty")
	}
	ff, err = src.Files()
	switch {
	case err != nil:
		return err
	case len(ff) == 0:
		fmt.Fprint(cmd.OutOrStderr(), "nothing to import")
		cmd.SilenceUsage = true
		return nil
	}
	// Fix version numbers for Flyway repeatable migrations.
	if _, ok := src.(*sqltool.FlywayDir); ok {
		sqltool.SetRepeatableVersion(ff)
	}
	// Extract the statements for each of the migration files,
	// add them to a plan to format with the DefaultFormatter.
	for _, f := range ff {
		stmts, err := f.StmtDecls() // Not driver aware.
		if err != nil {
			return err
		}
		plan := &migrate.Plan{
			Version: f.Version(),
			Name:    f.Desc(),
			Changes: make([]*migrate.Change, len(stmts)),
		}
		var buf strings.Builder
		for i, s := range stmts {
			for _, c := range s.Comments {
				buf.WriteString(c)
				if !strings.HasSuffix(c, "\n") {
					buf.WriteString("\n")
				}
			}
			buf.WriteString(strings.TrimSuffix(s.Text, ";"))
			plan.Changes[i] = &migrate.Change{Cmd: buf.String()}
			buf.Reset()
		}
		files, err := migrate.DefaultFormatter.Format(plan)
		if err != nil {
			return err
		}
		for _, f := range files {
			if err := trgt.WriteFile(f.Name(), f.Bytes()); err != nil {
				return err
			}
		}
	}
	sum, err := trgt.Checksum()
	if err != nil {
		return err
	}
	return migrate.WriteSumFile(trgt, sum)
}

type migrateLintFlags struct {
	dirURL, dirFormat string
	devURL            string
	logFormat         string
	latest            uint   // --latest 1
	gitBase, gitDir   string // --git-base master --git-dir /path/to/git/repo
	// Not enabled by default.
	dirBase string // --base atlas://myapp
	web     bool   // Open the web browser
	context string // Run context. See cloudapi.ContextInput.
}

// migrateLintCmd represents the 'atlas migrate lint' subcommand.
func migrateLintCmd() *cobra.Command {
	var (
		env   *Env
		flags migrateLintFlags
		cmd   = &cobra.Command{
			Use:   "lint [flags]",
			Short: "Run analysis on the migration directory",
			Example: `  atlas migrate lint --env dev
  atlas migrate lint --dir "file:///path/to/migrations" --dev-url "docker://mysql/8/dev" --latest 1
  atlas migrate lint --dir "file:///path/to/migrations" --dev-url "mysql://root:pass@localhost:3306" --git-base master
  atlas migrate lint --dir "file:///path/to/migrations" --dev-url "mysql://root:pass@localhost:3306" --format '{{ json .Files }}'`,
			PreRunE: func(cmd *cobra.Command, args []string) (err error) {
				if env, err = selectEnv(cmd); err != nil {
					return err
				}
				if err := setMigrateEnvFlags(cmd, env); err != nil {
					return err
				}
				return dirFormatBC(flags.dirFormat, &flags.dirURL)
			},
			RunE: RunE(func(cmd *cobra.Command, args []string) error {
				return migrateLintRun(cmd, args, flags, env)
			}),
		}
	)
	cmd.Flags().SortFlags = false
	addFlagDevURL(cmd.Flags(), &flags.devURL)
	addFlagDirURL(cmd.Flags(), &flags.dirURL)
	addFlagDirFormat(cmd.Flags(), &flags.dirFormat)
	addFlagLog(cmd.Flags(), &flags.logFormat)
	addFlagFormat(cmd.Flags(), &flags.logFormat)
	cmd.Flags().UintVarP(&flags.latest, flagLatest, "", 0, "run analysis on the latest N migration files")
	cmd.Flags().StringVarP(&flags.gitBase, flagGitBase, "", "", "run analysis against the base Git branch")
	cmd.Flags().StringVarP(&flags.gitDir, flagGitDir, "", ".", "path to the repository working directory")
	cobra.CheckErr(cmd.MarkFlagRequired(flagDevURL))
	cmd.MarkFlagsMutuallyExclusive(flagLog, flagFormat)
	migrateLintSetFlags(cmd, &flags)
	return cmd
}

type migrateNewFlags struct {
	edit      bool
	dirURL    string
	dirFormat string
}

// migrateNewCmd represents the 'atlas migrate new' subcommand.
func migrateNewCmd() *cobra.Command {
	var (
		flags migrateNewFlags
		cmd   = &cobra.Command{
			Use:     "new [flags] [name]",
			Short:   "Creates a new empty migration file in the migration directory.",
			Long:    `'atlas migrate new' creates a new migration according to the configured formatter without any statements in it.`,
			Example: `  atlas migrate new my-new-migration`,
			Args:    cobra.MaximumNArgs(1),
			PreRunE: func(cmd *cobra.Command, _ []string) error {
				if err := migrateFlagsFromConfig(cmd); err != nil {
					return err
				}
				if err := dirFormatBC(flags.dirFormat, &flags.dirURL); err != nil {
					return err
				}
				return checkDir(cmd, flags.dirURL, true)
			},
			RunE: RunE(func(cmd *cobra.Command, args []string) error {
				return migrateNewRun(cmd, args, flags)
			}),
		}
	)
	cmd.Flags().SortFlags = false
	addFlagDirURL(cmd.Flags(), &flags.dirURL)
	addFlagDirFormat(cmd.Flags(), &flags.dirFormat)
	cmd.Flags().BoolVarP(&flags.edit, flagEdit, "", false, "edit the created migration file(s)")
	return cmd
}

func migrateNewRun(cmd *cobra.Command, args []string, flags migrateNewFlags) error {
	u, err := url.Parse(flags.dirURL)
	if err != nil {
		return err
	}
	dir, err := cmdmigrate.DirURL(cmd.Context(), u, true)
	if err != nil {
		return err
	}
	if flags.edit {
		l, ok := dir.(*migrate.LocalDir)
		if !ok {
			return fmt.Errorf("--edit flag supports only atlas directories, but got: %T", dir)
		}
		dir = &editDir{l}
	}
	f, err := cmdmigrate.Formatter(u)
	if err != nil {
		return err
	}
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	return migrate.NewPlanner(nil, dir, migrate.PlanFormat(f)).WritePlan(&migrate.Plan{Name: name})
}

type migrateSetFlags struct {
	url               string
	dirURL, dirFormat string
	revisionSchema    string
}

// migrateSetCmd represents the 'atlas migrate set' subcommand.
func migrateSetCmd() *cobra.Command {
	var (
		flags migrateSetFlags
		cmd   = &cobra.Command{
			Use:   "set [flags] [version]",
			Short: "Set the current version of the migration history table.",
			Long: `'atlas migrate set' edits the revision table to consider all migrations up to and including the given version
to be applied. This command is usually used after manually making changes to the managed database.`,
			Example: `  atlas migrate set 3 --url "mysql://user:pass@localhost:3306/"
  atlas migrate set --env local
  atlas migrate set 1.2.4 --url "mysql://user:pass@localhost:3306/my_db" --revision-schema my_revisions`,
			PreRunE: func(cmd *cobra.Command, _ []string) error {
				if err := migrateFlagsFromConfig(cmd); err != nil {
					return err
				}
				if err := dirFormatBC(flags.dirFormat, &flags.dirURL); err != nil {
					return err
				}
				return checkDir(cmd, flags.dirURL, false)
			},
			RunE: RunE(func(cmd *cobra.Command, args []string) error {
				return migrateSetRun(cmd, args, flags)
			}),
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURL(cmd.Flags(), &flags.url)
	addFlagDirURL(cmd.Flags(), &flags.dirURL)
	addFlagDirFormat(cmd.Flags(), &flags.dirFormat)
	addFlagRevisionSchema(cmd.Flags(), &flags.revisionSchema)
	return cmd
}

func migrateSetRun(cmd *cobra.Command, args []string, flags migrateSetFlags) (rerr error) {
	ctx := cmd.Context()
	dir, err := cmdmigrate.Dir(ctx, flags.dirURL, false)
	if err != nil {
		return err
	}
	client, err := sqlclient.Open(ctx, flags.url)
	if err != nil {
		return err
	}
	defer client.Close()
	// Acquire a lock.
	unlock, err := client.Driver.Lock(ctx, applyLockValue, 0)
	if err != nil {
		return fmt.Errorf("acquiring database lock: %w", err)
	}
	// If unlocking fails notify the user about it.
	defer func() { cobra.CheckErr(unlock()) }()
	if err := checkRevisionSchemaClarity(cmd, client, flags.revisionSchema); err != nil {
		return err
	}
	// Ensure revision table exists.
	rrw, err := entRevisions(ctx, client, flags.revisionSchema)
	if err != nil {
		return err
	}
	if err := rrw.Migrate(ctx); err != nil {
		return err
	}
	// Wrap manipulation in a transaction.
	tx, err := client.Tx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if rerr == nil {
			rerr = tx.Commit()
		} else if err2 := tx.Rollback(); err2 != nil {
			rerr = fmt.Errorf("%v: %w", err2, err)
		}
	}()
	rrw, err = entRevisions(ctx, tx.Client, flags.revisionSchema)
	if err != nil {
		return err
	}
	revs, err := rrw.ReadRevisions(ctx)
	if err != nil {
		return err
	}
	files, err := dir.Files()
	if err != nil {
		return err
	}
	var version string
	switch n := len(args); {
	// Prevent the case where 'migrate set' is called without a version on
	// a clean database. i.e., we allow only removing or syncing revisions.
	case n == 0 && len(revs) > 0:
		// Calling set without a version and an empty
		// migration directory purges the revision table.
		if len(files) > 0 {
			version = files[len(files)-1].Version()
		}
	case n == 1:
		// Check if the target version does exist in the migration directory.
		if idx := migrate.FilesLastIndex(files, func(f migrate.File) bool {
			return f.Version() == args[0]
		}); idx == -1 {
			return fmt.Errorf("migration with version %q not found", args[0])
		}
		version = args[0]
	default:
		return fmt.Errorf("accepts 1 arg(s), received %d", n)
	}
	log := cmdlog.NewMigrateSet(ctx)
	for _, r := range revs {
		// Check all existing revisions and ensure they precede the given version. If we encounter a partially
		// applied revision, or one with errors, mark them "fixed".
		switch {
		// remove revision to keep linear history
		case r.Version > version:
			log.Removed(r)
			if err := rrw.DeleteRevision(ctx, r.Version); err != nil {
				return err
			}
		// keep, but if with error mark "fixed"
		case r.Version == version && (r.Error != "" || r.Total != r.Applied):
			log.Set(r)
			r.Type = migrate.RevisionTypeExecute | migrate.RevisionTypeResolved
			if err := rrw.WriteRevision(ctx, r); err != nil {
				return err
			}
		}
	}
	revs, err = rrw.ReadRevisions(ctx)
	if err != nil {
		return err
	}
	// If the target version succeeds the last revision, mark
	// migrations applied, until we reach the target version.
	var pending []migrate.File
	switch {
	case len(revs) == 0:
		// Take every file until we reach target version.
		for _, f := range files {
			if f.Version() > version {
				break
			}
			pending = append(pending, f)
		}
	case version > revs[len(revs)-1].Version:
	loop:
		// Take every file succeeding the last revision until we reach target version.
		for _, f := range files {
			switch {
			case f.Version() <= revs[len(revs)-1].Version:
				// Migration precedes last revision.
			case f.Version() > version:
				// Migration succeeds target revision.
				break loop
			default: // between last revision and target
				pending = append(pending, f)
			}
		}
	}
	// Mark every pending file as applied.
	sum, err := dir.Checksum()
	if err != nil {
		return err
	}
	for _, f := range pending {
		h, err := sum.SumByName(f.Name())
		if err != nil {
			return err
		}
		rev := &migrate.Revision{
			Version:         f.Version(),
			Description:     f.Desc(),
			Type:            migrate.RevisionTypeResolved,
			ExecutedAt:      time.Now(),
			Hash:            h,
			OperatorVersion: operatorVersion(),
		}
		log.Set(rev)
		if err := rrw.WriteRevision(ctx, rev); err != nil {
			return err
		}
	}
	if log.Current, err = rrw.CurrentRevision(ctx); err != nil && !errors.Is(err, migrate.ErrRevisionNotExist) {
		return err
	}
	return cmdlog.MigrateSetTemplate.Execute(cmd.OutOrStdout(), log)
}

type migrateStatusFlags struct {
	url               string
	dirURL, dirFormat string
	revisionSchema    string
	logFormat         string
}

// migrateStatusCmd represents the 'atlas migrate status' subcommand.
func migrateStatusCmd() *cobra.Command {
	var (
		flags migrateStatusFlags
		cmd   = &cobra.Command{
			Use:   "status [flags]",
			Short: "Get information about the current migration status.",
			Long:  `'atlas migrate status' reports information about the current status of a connected database compared to the migration directory.`,
			Example: `  atlas migrate status --url "mysql://user:pass@localhost:3306/"
  atlas migrate status --url "mysql://user:pass@localhost:3306/" --dir "file:///path/to/migration/directory"`,
			PreRunE: func(cmd *cobra.Command, _ []string) error {
				if err := migrateFlagsFromConfig(cmd); err != nil {
					return err
				}
				if err := dirFormatBC(flags.dirFormat, &flags.dirURL); err != nil {
					return err
				}
				return checkDir(cmd, flags.dirURL, false)
			},
			RunE: RunE(func(cmd *cobra.Command, args []string) error {
				return migrateStatusRun(cmd, args, flags)
			}),
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURL(cmd.Flags(), &flags.url)
	addFlagLog(cmd.Flags(), &flags.logFormat)
	addFlagFormat(cmd.Flags(), &flags.logFormat)
	addFlagDirURL(cmd.Flags(), &flags.dirURL)
	addFlagDirFormat(cmd.Flags(), &flags.dirFormat)
	addFlagRevisionSchema(cmd.Flags(), &flags.revisionSchema)
	cmd.MarkFlagsMutuallyExclusive(flagLog, flagFormat)
	return cmd
}

func migrateStatusRun(cmd *cobra.Command, _ []string, flags migrateStatusFlags) error {
	ctx := cmd.Context()
	dirURL, err := url.Parse(flags.dirURL)
	if err != nil {
		return fmt.Errorf("parse dir-url: %w", err)
	}
	dir, err := cmdmigrate.DirURL(ctx, dirURL, false)
	if err != nil {
		return err
	}
	client, err := sqlclient.Open(ctx, flags.url)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := checkRevisionSchemaClarity(cmd, client, flags.revisionSchema); err != nil {
		return err
	}
	report, err := (&cmdlog.StatusReporter{
		Client: client,
		Dir:    dir,
		DirURL: dirURL,
		Schema: revisionSchemaName(client, flags.revisionSchema),
	}).Report(ctx)
	if err != nil {
		return err
	}
	format := cmdlog.MigrateStatusTemplate
	if f := flags.logFormat; f != "" {
		if format, err = template.New("format").Funcs(cmdlog.StatusTemplateFuncs).Parse(f); err != nil {
			return fmt.Errorf("parse format: %w", err)
		}
	}
	return format.Execute(cmd.OutOrStdout(), report)
}

type migrateValidateFlags struct {
	devURL            string
	dirURL, dirFormat string
}

// migrateValidateCmd represents the 'atlas migrate validate' subcommand.
func migrateValidateCmd() *cobra.Command {
	var (
		flags migrateValidateFlags
		cmd   = &cobra.Command{
			Use:   "validate [flags]",
			Short: "Validates the migration directories checksum and SQL statements.",
			Long: `'atlas migrate validate' computes the integrity hash sum of the migration directory and compares it to the
atlas.sum file. If there is a mismatch it will be reported. If the --dev-url flag is given, the migration
files are executed on the connected database in order to validate SQL semantics.`,
			Example: `  atlas migrate validate
  atlas migrate validate --dir "file:///path/to/migration/directory"
  atlas migrate validate --dir "file:///path/to/migration/directory" --dev-url "docker://mysql/8/dev"
  atlas migrate validate --env dev --dev-url "docker://postgres/15/dev?search_path=public"`,
			PreRunE: func(cmd *cobra.Command, _ []string) error {
				if err := migrateFlagsFromConfig(cmd); err != nil {
					return err
				}
				if err := dirFormatBC(flags.dirFormat, &flags.dirURL); err != nil {
					return err
				}
				err := checkDir(cmd, flags.dirURL, false)
				return err
			},
			RunE: RunE(func(cmd *cobra.Command, args []string) error {
				return migrateValidateRun(cmd, args, flags)
			}),
		}
	)
	cmd.Flags().SortFlags = false
	addFlagDevURL(cmd.Flags(), &flags.devURL)
	addFlagDirURL(cmd.Flags(), &flags.dirURL)
	addFlagDirFormat(cmd.Flags(), &flags.dirFormat)
	return cmd
}

func migrateValidateRun(cmd *cobra.Command, _ []string, flags migrateValidateFlags) error {
	// Validating the integrity is done by the PersistentPreRun already.
	if flags.devURL == "" {
		// If there is no --dev-url given do not attempt to replay the migration directory.
		return nil
	}
	// Open a client for the dev-db.
	dev, err := sqlclient.Open(cmd.Context(), flags.devURL)
	if err != nil {
		return err
	}
	defer dev.Close()
	// Currently, only our own migration file format is supported.
	dir, err := cmdmigrate.Dir(cmd.Context(), flags.dirURL, false)
	if err != nil {
		return err
	}
	ex, err := migrate.NewExecutor(dev.Driver, dir, migrate.NopRevisionReadWriter{})
	if err != nil {
		return err
	}
	if _, err := ex.Replay(cmd.Context(), func() migrate.StateReader {
		if dev.URL.Schema != "" {
			return migrate.SchemaConn(dev, "", nil)
		}
		return migrate.RealmConn(dev, nil)
	}()); err != nil && !errors.Is(err, migrate.ErrNoPendingFiles) {
		return fmt.Errorf("replaying the migration directory: %w", err)
	}
	return nil
}

const applyLockValue = "atlas_migrate_execute"

func checkRevisionSchemaClarity(cmd *cobra.Command, c *sqlclient.Client, revisionSchemaFlag string) error {
	// The "old" default  behavior for the revision schema location was to store the revision table in its own schema.
	// Now, the table is saved in the connected schema, if any. To keep the backwards compatability, we now require
	// for schema bound connections to have the schema-revision flag present if there is no revision table in the schema
	// but the old default schema does have one.
	if c.URL.Schema != "" && revisionSchemaFlag == "" {
		// If the schema does not contain a revision table, but we can find a table in the previous default schema,
		// abort and tell the user to specify the intention.
		opts := &schema.InspectOptions{Tables: []string{revision.Table}, Mode: schema.InspectTables}
		s, err := c.InspectSchema(cmd.Context(), "", opts)
		var ok bool
		switch {
		case schema.IsNotExistError(err):
			// If the schema does not exist, the table does not as well.
		case err != nil:
			return err
		default:
			// Connected schema does exist, check if the table does.
			_, ok = s.Table(revision.Table)
		}
		if !ok { // Either schema or table does not exist.
			// Check for the old default schema. If it does not exist, we have no problem.
			s, err := c.InspectSchema(cmd.Context(), defaultRevisionSchema, opts)
			switch {
			case schema.IsNotExistError(err):
				// Schema does not exist, we can proceed.
			case err != nil:
				return err
			default:
				if _, ok := s.Table(revision.Table); ok {
					fmt.Fprintf(cmd.OutOrStderr(),
						`We couldn't find a revision table in the connected schema but found one in 
the schema 'atlas_schema_revisions' and cannot determine the desired behavior.

As a safety guard, we require you to specify whether to use the existing
table in 'atlas_schema_revisions' or create a new one in the connected schema
by providing the '--revisions-schema' flag or deleting the 'atlas_schema_revisions'
schema if it is unused.

`)
					cmd.SilenceUsage = true
					cmd.SilenceErrors = true
					return errors.New("ambiguous revision table")
				}
			}
		}
	}
	return nil
}

func entRevisions(ctx context.Context, c *sqlclient.Client, flag string) (cmdmigrate.RevisionReadWriter, error) {
	return cmdmigrate.RevisionsForClient(ctx, c, revisionSchemaName(c, flag))
}

// defaultRevisionSchema is the default schema for storing revisions table.
const defaultRevisionSchema = "atlas_schema_revisions"

func revisionSchemaName(c *sqlclient.Client, flag string) string {
	switch {
	case flag != "":
		return flag
	case c.URL.Schema != "":
		return c.URL.Schema
	default:
		return defaultRevisionSchema
	}
}

const (
	txModeNone      = "none"
	txModeAll       = "all"
	txModeFile      = "file"
	txModeDirective = "txmode"

	execOrderLinear     = "linear"
	execOrderLinearSkip = "linear-skip"
	execOrderNonLinear  = "non-linear"
)

// tx handles wrapping migration execution in transactions.
type tx struct {
	dryRun       bool
	mode, schema string
	c            *sqlclient.Client
	rrw          migrate.RevisionReadWriter
	// current transaction context.
	tx    *sqlclient.TxClient
	txrrw migrate.RevisionReadWriter
}

// driverFor returns the migrate.Driver to use to execute migration statements.
func (tx *tx) driverFor(ctx context.Context, f migrate.File) (migrate.Driver, migrate.RevisionReadWriter, error) {
	if tx.dryRun {
		// If the --dry-run flag is given we don't want to execute any statements on the database.
		return &dryRunDriver{tx.c.Driver}, &dryRunRevisions{tx.rrw}, nil
	}
	mode, err := tx.modeFor(f)
	if err != nil {
		return nil, nil, err
	}
	switch mode {
	case txModeNone:
		return tx.c.Driver, tx.rrw, nil
	case txModeFile:
		// In file-mode, this function is called each time a new file is executed. Open a transaction.
		if tx.tx != nil {
			return nil, nil, errors.New("unexpected active transaction")
		}
		var err error
		tx.tx, err = tx.c.Tx(ctx, nil)
		if err != nil {
			return nil, nil, err
		}
		if tx.txrrw, err = entRevisions(ctx, tx.tx.Client, tx.schema); err != nil {
			return nil, nil, err
		}
		return tx.tx.Driver, tx.txrrw, nil
	case txModeAll:
		// In file-mode, this function is called each time a new file is executed. Since we wrap all files into one
		// huge transaction, if there already is an opened one, use that.
		if tx.tx == nil {
			var err error
			tx.tx, err = tx.c.Tx(ctx, nil)
			if err != nil {
				return nil, nil, err
			}
			if tx.txrrw, err = entRevisions(ctx, tx.tx.Client, tx.schema); err != nil {
				return nil, nil, err
			}
		}
		return tx.tx.Driver, tx.txrrw, nil
	default:
		return nil, nil, fmt.Errorf("unknown tx-mode %q", mode)
	}
}

// mayRollback may roll back a transaction depending on the given transaction mode.
func (tx *tx) mayRollback(err error) error {
	if tx.tx != nil && err != nil {
		if err2 := tx.tx.Rollback(); err2 != nil {
			err = fmt.Errorf("%v: %w", err2, err)
		}
	}
	return err
}

// mayCommit may commit a transaction depending on the given transaction mode.
func (tx *tx) mayCommit() error {
	// Only commit if each file is wrapped in a transaction.
	if tx.tx != nil && !tx.dryRun && tx.mode == txModeFile {
		return tx.commit()
	}
	return nil
}

// commit the transaction, if one is active.
func (tx *tx) commit() error {
	if tx.tx == nil {
		return nil
	}
	defer func() { tx.tx, tx.txrrw = nil, nil }()
	return tx.tx.Commit()
}

func (tx *tx) modeFor(f migrate.File) (string, error) {
	l, ok := f.(*migrate.LocalFile)
	if !ok {
		return tx.mode, nil
	}
	switch m, err := txmodeFor(l); {
	case err != nil:
		return "", err
	case m == "", m == tx.mode:
		return tx.mode, nil
	default: // m == txModeNone, m == txModeFile
		if tx.mode == txModeAll {
			return "", fmt.Errorf("cannot set txmode directive to %q in %q when txmode %q is set globally", m, l.Name(), txModeAll)
		}
		return m, nil
	}
}

// txmodeFor returns the transaction mode for the given file.
func txmodeFor(f *migrate.LocalFile) (string, error) {
	switch ds := f.Directive(txModeDirective); {
	case len(ds) == 0:
		return "", nil
	case len(ds) > 1:
		return "", fmt.Errorf("multiple txmode values found in file %q: %q", f.Name(), ds)
	case ds[0] == txModeAll:
		return "", fmt.Errorf("txmode %q is not allowed in file directive %q. Use %q instead", txModeAll, f.Name(), txModeFile)
	case ds[0] == txModeNone, ds[0] == txModeFile:
		return ds[0], nil
	default:
		return "", fmt.Errorf("unknown txmode %q found in file directive %q", ds[0], f.Name())
	}
}

func operatorVersion() string {
	v, _ := parseV(version)
	return "Atlas CLI " + v
}

// dirFormatBC ensures the soon-to-be deprecated --dir-format flag gets set on all migration directory URLs.
func dirFormatBC(flag string, urls ...*string) error {
	for _, s := range urls {
		u, err := url.Parse(*s)
		if err != nil {
			return err
		}
		if !u.Query().Has("format") && flag != "" {
			q := u.Query()
			q.Set("format", flag)
			u.RawQuery = q.Encode()
			*s = u.String()
		}
	}
	return nil
}

func checkDir(cmd *cobra.Command, url string, create bool) error {
	d, err := cmdmigrate.Dir(cmd.Context(), url, create)
	if err != nil {
		return err
	}
	if err = migrate.Validate(d); err != nil {
		printChecksumError(cmd, err)
		return err
	}
	return nil
}

func printChecksumError(cmd *cobra.Command, err error) {
	cmd.SilenceUsage = true
	out := cmd.OutOrStderr()
	fmt.Fprintln(out, "You have a checksum error in your migration directory.")
	if csErr := (&migrate.ChecksumError{}); errors.As(err, &csErr) {
		fmt.Fprintf(out, "\n\tL%d: %s was %s\n\n", csErr.Line, csErr.File, csErr.Reason)
	}
	fmt.Fprintf(
		out,
		"Please check your migration files and run %v to re-hash the contents\n\n",
		cmdlog.ColorCyan("'atlas migrate hash'"),
	)
}

// selectScheme validates the scheme of the provided to urls and returns the selected
// url scheme. Currently, all URLs must be of the same scheme, and only multiple
// "file://" URLs are allowed.
func selectScheme(urls []string) (string, error) {
	var scheme string
	if len(urls) == 0 {
		return "", errors.New("at least one url is required")
	}
	for _, u := range urls {
		parts := strings.SplitN(u, "://", 2)
		switch current := parts[0]; {
		case len(parts) == 1:
			ex := filepath.Ext(u)
			switch f, err := os.Stat(u); {
			case err != nil:
			case f.IsDir(), ex == cmdext.FileTypeSQL, ex == cmdext.FileTypeHCL:
				return "", fmt.Errorf("missing scheme. Did you mean file://%s?", u)
			}
			return "", errors.New("missing scheme. See: https://atlasgo.io/url")
		case scheme == "":
			scheme = current
		case scheme != current:
			return "", fmt.Errorf("got mixed --to url schemes: %q and %q, the desired state must be provided from a single kind of source", scheme, current)
		case current != cmdext.SchemaTypeFile:
			return "", fmt.Errorf("got multiple --to urls of scheme %q, only multiple 'file://' urls are supported", current)
		}
	}
	return scheme, nil
}

func migrateFlagsFromConfig(cmd *cobra.Command) error {
	env, err := selectEnv(cmd)
	if err != nil {
		return err
	}
	return setMigrateEnvFlags(cmd, env)
}

func setMigrateEnvFlags(cmd *cobra.Command, env *Env) error {
	if err := maySetFlag(cmd, flagURL, env.URL); err != nil {
		return err
	}
	if err := maySetFlag(cmd, flagDevURL, env.DevURL); err != nil {
		return err
	}
	if err := maySetFlag(cmd, flagDirURL, env.Migration.Dir); err != nil {
		return err
	}
	if err := maySetFlag(cmd, flagDirFormat, env.Migration.Format); err != nil {
		return err
	}
	if err := maySetFlag(cmd, flagBaseline, env.Migration.Baseline); err != nil {
		return err
	}
	if err := maySetFlag(cmd, flagRevisionSchema, env.Migration.RevisionsSchema); err != nil {
		return err
	}
	switch cmd.Name() {
	case "apply":
		if err := maySetFlag(cmd, flagFormat, env.Format.Migrate.Apply); err != nil {
			return err
		}
		if err := maySetFlag(cmd, flagLockTimeout, env.Migration.LockTimeout); err != nil {
			return err
		}
		if err := maySetFlag(cmd, flagExecOrder, strings.ReplaceAll(strings.ToLower(env.Migration.ExecOrder), "_", "-")); err != nil {
			return err
		}
	case "down":
		if err := maySetFlag(cmd, flagFormat, env.Format.Migrate.Down); err != nil {
			return err
		}
		if err := maySetFlag(cmd, flagLockTimeout, env.Migration.LockTimeout); err != nil {
			return err
		}
	case "diff", "checkpoint":
		if err := maySetFlag(cmd, flagLockTimeout, env.Migration.LockTimeout); err != nil {
			return err
		}
		if err := maySetFlag(cmd, flagFormat, env.Format.Migrate.Diff); err != nil {
			return err
		}
	case "lint":
		if err := maySetFlag(cmd, flagFormat, env.Format.Migrate.Lint); err != nil {
			return err
		}
		if err := maySetFlag(cmd, flagFormat, env.Lint.Format); err != nil {
			return err
		}
		if err := maySetFlag(cmd, flagLatest, strconv.Itoa(env.Lint.Latest)); err != nil {
			return err
		}
		if err := maySetFlag(cmd, flagGitDir, env.Lint.Git.Dir); err != nil {
			return err
		}
		if err := maySetFlag(cmd, flagGitBase, env.Lint.Git.Base); err != nil {
			return err
		}
	case "status":
		if err := maySetFlag(cmd, flagFormat, env.Format.Migrate.Status); err != nil {
			return err
		}
	}
	// Transform "src" to a URL.
	srcs, err := env.Sources()
	if err != nil {
		return err
	}
	for i, s := range srcs {
		if isURL(s) {
			continue
		}
		if s, err = filepath.Abs(s); err != nil {
			return fmt.Errorf("finding abs path to source: %q: %w", s, err)
		}
		srcs[i] = "file://" + s
	}
	if err := maySetFlag(cmd, flagTo, strings.Join(srcs, ",")); err != nil {
		return err
	}
	if err := maySetFlag(cmd, flagSchema, strings.Join(env.Schemas, ",")); err != nil {
		return err
	}
	return nil
}

// isURL returns true if the given string
// is an Atlas URL with a scheme.
func isURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme != ""
}

// cmdEnvsRun executes a given command on each of the configured environment.
func cmdEnvsRun(
	envs []*Env,
	setFlags func(*cobra.Command, *Env) error,
	cmd *cobra.Command,
	runCmd func(*Env) error,
) error {
	var (
		w     bytes.Buffer
		out   = cmd.OutOrStdout()
		reset = resetFromEnv(cmd)
	)
	cmd.SetOut(io.MultiWriter(out, &w))
	defer cmd.SetOut(out)
	for i, e := range envs {
		if err := setFlags(cmd, e); err != nil {
			return err
		}
		if err := runCmd(e); err != nil {
			return err
		}
		b := bytes.TrimLeft(w.Bytes(), " \t\r")
		// In case a custom logging was configured, ensure there is
		// a newline separator between the different environments.
		if cmd.Flags().Changed(flagFormat) && bytes.LastIndexByte(b, '\n') != len(b)-1 && i != len(envs)-1 {
			cmd.Println()
		}
		reset()
		w.Reset()
	}
	return nil
}

type editDir struct{ *migrate.LocalDir }

// WriteFile implements the migrate.Dir.WriteFile method.
func (d *editDir) WriteFile(name string, b []byte) (err error) {
	if name != migrate.HashFileName {
		if b, err = edit(name, b); err != nil {
			return err
		}
	}
	return d.LocalDir.WriteFile(name, b)
}

// edit allows editing the file content using editor.
func edit(name string, src []byte) ([]byte, error) {
	p := filepath.Join(os.TempDir(), name)
	if err := os.WriteFile(p, src, 0644); err != nil {
		return nil, fmt.Errorf("write source content to temp file: %w", err)
	}
	defer os.Remove(p)
	editor := "vi"
	if e := os.Getenv("EDITOR"); e != "" {
		editor = e
	}
	cmd := exec.Command("sh", "-c", editor+" "+p)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("exec edit: %w", err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read edited temp file: %w", err)
	}
	return b, nil
}

type (
	// dryRunDriver wraps a migrate.Driver without executing any SQL statements.
	dryRunDriver struct{ migrate.Driver }

	// dryRunRevisions wraps a migrate.RevisionReadWriter without executing any SQL statements.
	dryRunRevisions struct{ migrate.RevisionReadWriter }
)

// ExecContext overrides the wrapped schema.ExecQuerier to not execute any SQL.
func (dryRunDriver) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, nil
}

// Lock implements the schema.Locker interface.
func (dryRunDriver) Lock(context.Context, string, time.Duration) (schema.UnlockFunc, error) {
	// We dry-run, we don't execute anything. Locking is not required.
	return func() error { return nil }, nil
}

// CheckClean implements the migrate.CleanChecker interface.
func (dryRunDriver) CheckClean(context.Context, *migrate.TableIdent) error {
	return nil
}

// Snapshot implements the migrate.Snapshoter interface.
func (dryRunDriver) Snapshot(context.Context) (migrate.RestoreFunc, error) {
	// We dry-run, we don't execute anything. Snapshotting not required.
	return func(context.Context) error { return nil }, nil
}

// WriteRevision overrides the wrapped migrate.RevisionReadWriter to not saved any changes to revisions.
func (dryRunRevisions) WriteRevision(context.Context, *migrate.Revision) error {
	return nil
}
