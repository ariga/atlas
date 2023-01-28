// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/cmdlog"
	"ariga.io/atlas/cmd/atlas/internal/lint"
	cmdmigrate "ariga.io/atlas/cmd/atlas/internal/migrate"
	"ariga.io/atlas/cmd/atlas/internal/migrate/ent"
	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/revision"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"
	"ariga.io/atlas/sql/sqltool"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/spf13/cobra"
)

func init() {
	migrateCmd := migrateCmd()
	migrateCmd.AddCommand(
		migrateApplyCmd(),
		migrateDiffCmd(),
		migrateHashCmd(),
		migrateImportCmd(),
		migrateLintCmd(),
		migrateNewCmd(),
		migrateSetCmd(),
		migrateStatusCmd(),
		migrateValidateCmd(),
	)
	Root.AddCommand(migrateCmd)
}

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
	revisionSchema  string
	dryRun          bool
	logFormat       string
	lockTimeout     time.Duration
	allowDirty      bool   // allow working on a database that already has resources
	fromVersion     string // compute pending files based on this version
	baselineVersion string // apply with this version as baseline
	txMode          string // (none, file, all)
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
			Example: `  atlas migrate apply -u mysql://user:pass@localhost:3306/dbname
  atlas migrate apply --dir file:///path/to/migration/directory --url mysql://user:pass@localhost:3306/dbname 1
  atlas migrate apply --env dev 1
  atlas migrate apply --dry-run --env dev 1`,
			Args: cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if GlobalFlags.SelectedEnv == "" {
					return migrateApplyRun(cmd, args, flags)
				}
				return cmdEnvsRun(migrateApplyRun, setMigrateEnvFlags, cmd, args, &flags)
			},
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURL(cmd.Flags(), &flags.url)
	addFlagDirURL(cmd.Flags(), &flags.dirURL)
	addFlagLog(cmd.Flags(), &flags.logFormat)
	addFlagRevisionSchema(cmd.Flags(), &flags.revisionSchema)
	addFlagDryRun(cmd.Flags(), &flags.dryRun)
	addFlagLockTimeout(cmd.Flags(), &flags.lockTimeout)
	cmd.Flags().StringVarP(&flags.fromVersion, flagFrom, "", "", "calculate pending files from the given version (including it)")
	cmd.Flags().StringVarP(&flags.baselineVersion, flagBaseline, "", "", "start the first migration after the given baseline version")
	cmd.Flags().StringVarP(&flags.txMode, flagTxMode, "", txModeFile, "set transaction mode [none, file, all]")
	cmd.Flags().BoolVarP(&flags.allowDirty, flagAllowDirty, "", false, "allow start working on a non-clean database")
	cmd.MarkFlagsMutuallyExclusive(flagFrom, flagBaseline)
	return cmd
}

// migrateApplyCmd represents the 'atlas migrate apply' subcommand.
func migrateApplyRun(cmd *cobra.Command, args []string, flags migrateApplyFlags) error {
	var (
		count int
		err   error
	)
	if len(args) > 0 {
		if count, err = strconv.Atoi(args[0]); err != nil {
			return err
		}
		if count < 1 {
			return fmt.Errorf("cannot apply '%d' migration files", count)
		}
	}
	// Open and validate the migration directory.
	migrationDir, err := dir(flags.dirURL, false)
	if err != nil {
		return err
	}
	if err := migrate.Validate(migrationDir); err != nil {
		printChecksumError(cmd)
		return err
	}
	// Open a client to the database.
	if flags.url == "" {
		return errors.New(`required flag "url" not set`)
	}
	client, err := sqlclient.Open(cmd.Context(), flags.url)
	if err != nil {
		return err
	}
	defer client.Close()
	// Acquire a lock.
	if l, ok := client.Driver.(schema.Locker); ok {
		unlock, err := l.Lock(cmd.Context(), applyLockValue, flags.lockTimeout)
		if err != nil {
			return fmt.Errorf("acquiring database lock: %w", err)
		}
		// If unlocking fails notify the user about it.
		defer func() { cobra.CheckErr(unlock()) }()
	}
	if err := checkRevisionSchemaClarity(cmd, client, flags.revisionSchema); err != nil {
		return err
	}
	var rrw migrate.RevisionReadWriter
	rrw, err = entRevisions(cmd.Context(), client, flags.revisionSchema)
	if err != nil {
		return err
	}
	if err := rrw.(*cmdmigrate.EntRevisions).Migrate(cmd.Context()); err != nil {
		return err
	}
	// Determine pending files.
	opts := []migrate.ExecutorOption{
		migrate.WithOperatorVersion(operatorVersion()),
	}
	if flags.allowDirty {
		opts = append(opts, migrate.WithAllowDirty(true))
	}
	if v := flags.baselineVersion; v != "" {
		opts = append(opts, migrate.WithBaselineVersion(v))
	}
	if v := flags.fromVersion; v != "" {
		opts = append(opts, migrate.WithFromVersion(v))
	}
	report := cmdlog.NewMigrateApply(client, migrationDir)
	ex, err := migrate.NewExecutor(client.Driver, migrationDir, rrw, opts...)
	if err != nil {
		return err
	}
	pending, err := ex.Pending(cmd.Context())
	if err != nil && !errors.Is(err, migrate.ErrNoPendingFiles) {
		return err
	}
	if errors.Is(err, migrate.ErrNoPendingFiles) {
		return reportApply(cmd, flags, report)
	}
	if l := len(pending); count == 0 || count >= l {
		// Cannot apply more than len(pending) migration files.
		count = l
	}
	pending = pending[:count]
	applied, err := rrw.ReadRevisions(cmd.Context())
	if err != nil {
		return err
	}
	opts = append(opts, migrate.WithLogger(report))
	if err := migrate.LogIntro(report, applied, pending); err != nil {
		return err
	}
	var (
		mux = tx{
			dryRun: flags.dryRun,
			mode:   flags.txMode,
			schema: flags.revisionSchema,
			c:      client,
			rrw:    rrw,
		}
		drv migrate.Driver
	)
	for _, f := range pending {
		drv, rrw, err = mux.driverFor(cmd.Context(), f)
		if err != nil {
			return err
		}
		ex, err = migrate.NewExecutor(drv, migrationDir, rrw, opts...)
		if err != nil {
			return err
		}
		if err = mux.mayRollback(ex.Execute(cmd.Context(), f)); err != nil {
			report.Error = err.Error()
			break
		}
		if err = mux.mayCommit(); err != nil {
			report.Error = err.Error()
			break
		}
	}
	if err == nil {
		if err = mux.commit(); err != nil {
			report.Error = err.Error()
		} else {
			report.Log(migrate.LogDone{})
		}
	}
	if err2 := reportApply(cmd, flags, report); err2 != nil {
		if err != nil {
			err2 = fmt.Errorf("%w: %v", err, err2)
		}
		err = err2
	}
	return err
}

func reportApply(cmd *cobra.Command, flags migrateApplyFlags, r *cmdlog.MigrateApply) error {
	var (
		err error
		f   = cmdlog.MigrateApplyTemplate
	)
	if v := flags.logFormat; v != "" {
		f, err = template.New("format").Funcs(cmdlog.ApplyTemplateFuncs).Parse(v)
		if err != nil {
			return fmt.Errorf("parse log format: %w", err)
		}
	}
	if err = f.Execute(cmd.OutOrStdout(), r); err != nil {
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
	revisionSchema    string // revision schema name
	qualifier         string // optional table qualifier
}

// migrateDiffCmd represents the 'atlas migrate diff' subcommand.
func migrateDiffCmd() *cobra.Command {
	var (
		flags migrateDiffFlags
		cmd   = &cobra.Command{
			Use:   "diff [flags] [name]",
			Short: "Compute the diff between the migration directory and a desired state and create a new migration file.",
			Long: `'atlas migrate diff' uses the dev-database to re-run all migration files in the migration directory, compares
it to a given desired state and create a new migration file containing SQL statements to migrate the migration
directory state to the desired schema. The desired state can be another connected database or an HCL file.`,
			Example: `  atlas migrate diff --dev-url mysql://user:pass@localhost:3306/dev --to file://schema.hcl
  atlas migrate diff --dev-url mysql://user:pass@localhost:3306/dev --to file://atlas.hcl add_users_table
  atlas migrate diff --dev-url mysql://user:pass@localhost:3306/dev --to mysql://user:pass@localhost:3306/dbname
  atlas migrate diff --env dev`,
			Args: cobra.MaximumNArgs(1),
			PreRunE: func(cmd *cobra.Command, args []string) error {
				if err := migrateFlagsFromEnv(cmd); err != nil {
					return err
				}
				if err := dirFormatBC(flags.dirFormat, &flags.dirURL); err != nil {
					return err
				}
				return checkDir(cmd, flags.dirURL, true)
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				return migrateDiffRun(cmd, args, flags)
			},
		}
	)
	cmd.Flags().SortFlags = false
	addFlagToURLs(cmd.Flags(), &flags.desiredURLs)
	addFlagDevURL(cmd.Flags(), &flags.devURL)
	addFlagDirURL(cmd.Flags(), &flags.dirURL)
	addFlagDirFormat(cmd.Flags(), &flags.dirFormat)
	addFlagRevisionSchema(cmd.Flags(), &flags.revisionSchema)
	addFlagSchemas(cmd.Flags(), &flags.schemas)
	addFlagLockTimeout(cmd.Flags(), &flags.lockTimeout)
	cmd.Flags().StringVar(&flags.qualifier, flagQualifier, "", "qualify tables with custom qualifier when working on a single schema")
	cmd.Flags().BoolVarP(&flags.edit, flagEdit, "", false, "edit the generated migration file(s)")
	cobra.CheckErr(cmd.MarkFlagRequired(flagTo))
	cobra.CheckErr(cmd.MarkFlagRequired(flagDevURL))
	return cmd
}

func migrateDiffRun(cmd *cobra.Command, args []string, flags migrateDiffFlags) error {
	// Open a dev driver.
	dev, err := sqlclient.Open(cmd.Context(), flags.devURL)
	if err != nil {
		return err
	}
	defer dev.Close()
	// Acquire a lock.
	if l, ok := dev.Driver.(schema.Locker); ok {
		unlock, err := l.Lock(cmd.Context(), "atlas_migrate_diff", flags.lockTimeout)
		if err != nil {
			return fmt.Errorf("acquiring database lock: %w", err)
		}
		// If unlocking fails notify the user about it.
		defer func() { cobra.CheckErr(unlock()) }()
	}
	// Open the migration directory.
	u, err := url.Parse(flags.dirURL)
	if err != nil {
		return err
	}
	dir, err := dirURL(u, false)
	if err != nil {
		return err
	}
	if flags.edit {
		dir = &editDir{dir}
	}
	f, err := formatter(u)
	if err != nil {
		return err
	}
	// Get a state reader for the desired state.
	desired, err := stateReader(cmd.Context(), &stateReaderConfig{
		urls:    flags.desiredURLs,
		dev:     dev,
		client:  dev,
		schemas: flags.schemas,
		vars:    GlobalFlags.Vars,
	})
	if err != nil {
		return err
	}
	defer desired.Close()
	opts := []migrate.PlannerOption{migrate.PlanFormat(f)}
	if dev.URL.Schema != "" {
		// Disable tables qualifier in schema-mode.
		opts = append(opts, migrate.PlanWithSchemaQualifier(flags.qualifier))
	}
	// Plan the changes and create a new migration file.
	pl := migrate.NewPlanner(dev.Driver, dir, opts...)
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	plan, err := func() (*migrate.Plan, error) {
		if dev.URL.Schema != "" {
			return pl.PlanSchema(cmd.Context(), name, desired.StateReader)
		}
		return pl.Plan(cmd.Context(), name, desired.StateReader)
	}()
	var cerr migrate.NotCleanError
	switch {
	case errors.Is(err, migrate.ErrNoPlan):
		cmd.Println("The migration directory is synced with the desired state, no changes to be made")
		return nil
	case errors.As(err, &cerr) && dev.URL.Schema == "" && desired.schema != "":
		return fmt.Errorf("dev database is not clean (%s). Add a schema to the URL to limit the scope of the connection", cerr.Reason)
	case err != nil:
		return err
	default:
		// Write the plan to a new file.
		return pl.WritePlan(plan)
	}
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
				if err := migrateFlagsFromEnv(cmd); err != nil {
					return err
				}
				return dirFormatBC(flags.dirFormat, &flags.dirURL)
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				dir, err := dir(flags.dirURL, false)
				if err != nil {
					return err
				}
				sum, err := dir.Checksum()
				if err != nil {
					return err
				}
				return migrate.WriteSumFile(dir, sum)
			},
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
			Example: `  atlas migrate import --from file:///path/to/source/directory?format=liquibase --to file:///path/to/migration/directory`,
			// Validate the source directory. Consider a directory with no sum file
			// valid, since it might be an import from an existing project.
			PreRunE: func(cmd *cobra.Command, _ []string) error {
				if err := migrateFlagsFromEnv(cmd); err != nil {
					return err
				}
				if err := dirFormatBC(flags.dirFormat, &flags.fromURL); err != nil {
					return err
				}
				d, err := dir(flags.fromURL, false)
				if err != nil {
					return err
				}
				if err = migrate.Validate(d); err != nil && !errors.Is(err, migrate.ErrChecksumNotFound) {
					printChecksumError(cmd)
					return err
				}
				return nil
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				return migrateImportRun(cmd, args, flags)
			},
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
	if f := p.Query().Get("format"); f == "" || f == formatAtlas {
		return fmt.Errorf("cannot import a migration directory already in %q format", formatAtlas)
	}
	src, err := dir(flags.fromURL, false)
	if err != nil {
		return err
	}
	trgt, err := dir(flags.toURL, true)
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
	// Extract the statements for each of the migration files, add them to a plan to format with the
	// migrate.DefaultFormatter.
	for _, f := range ff {
		stmts, err := f.StmtDecls()
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
	latest            uint
	gitBase, gitDir   string
}

// migrateLintCmd represents the 'atlas migrate lint' subcommand.
func migrateLintCmd() *cobra.Command {
	var (
		flags migrateLintFlags
		cmd   = &cobra.Command{
			Use:   "lint [flags]",
			Short: "Run analysis on the migration directory",
			Example: `  atlas migrate lint --env dev
  atlas migrate lint --dir file:///path/to/migration/directory --dev-url mysql://root:pass@localhost:3306 --latest 1
  atlas migrate lint --dir file:///path/to/migration/directory --dev-url mysql://root:pass@localhost:3306 --git-base master
  atlas migrate lint --dir file:///path/to/migration/directory --dev-url mysql://root:pass@localhost:3306 --log '{{ json .Files }}'`,
			PreRunE: func(cmd *cobra.Command, args []string) error {
				if err := migrateFlagsFromEnv(cmd); err != nil {
					return err
				}
				return dirFormatBC(flags.dirFormat, &flags.dirURL)
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				return migrateLintRun(cmd, args, flags)
			},
		}
	)
	cmd.Flags().SortFlags = false
	addFlagDevURL(cmd.Flags(), &flags.devURL)
	addFlagDirURL(cmd.Flags(), &flags.dirURL)
	addFlagDirFormat(cmd.Flags(), &flags.dirFormat)
	cmd.Flags().StringVarP(&flags.logFormat, flagLog, "", "", "custom logging using a Go template")
	cmd.Flags().UintVarP(&flags.latest, flagLatest, "", 0, "run analysis on the latest N migration files")
	cmd.Flags().StringVarP(&flags.gitBase, flagGitBase, "", "", "run analysis against the base Git branch")
	cmd.Flags().StringVarP(&flags.gitDir, flagGitDir, "", ".", "path to the repository working directory")
	cobra.CheckErr(cmd.MarkFlagRequired(flagDevURL))
	return cmd
}

func migrateLintRun(cmd *cobra.Command, _ []string, flags migrateLintFlags) error {
	dev, err := sqlclient.Open(cmd.Context(), flags.devURL)
	if err != nil {
		return err
	}
	defer dev.Close()
	dir, err := dir(flags.dirURL, false)
	if err != nil {
		return err
	}
	var detect lint.ChangeDetector
	switch {
	case flags.latest == 0 && flags.gitBase == "":
		return fmt.Errorf("--%s or --%s is required", flagLatest, flagGitBase)
	case flags.latest > 0 && flags.gitBase != "":
		return fmt.Errorf("--%s and --%s are mutually exclusive", flagLatest, flagGitBase)
	case flags.latest > 0:
		detect = lint.LatestChanges(dir, int(flags.latest))
	case flags.gitBase != "":
		detect, err = lint.NewGitChangeDetector(
			dir,
			lint.WithWorkDir(flags.gitDir),
			lint.WithBase(flags.gitBase),
			lint.WithMigrationsPath(dir.(interface{ Path() string }).Path()),
		)
		if err != nil {
			return err
		}
	}
	format := lint.DefaultTemplate
	if f := flags.logFormat; f != "" {
		format, err = template.New("format").Funcs(lint.TemplateFuncs).Parse(f)
		if err != nil {
			return fmt.Errorf("parse log format: %w", err)
		}
	}
	env, err := selectEnv(GlobalFlags.SelectedEnv)
	if err != nil {
		return err
	}
	az, err := sqlcheck.AnalyzerFor(dev.Name, env.Lint.Remain())
	if err != nil {
		return err
	}
	r := &lint.Runner{
		Dev:            dev,
		Dir:            dir,
		ChangeDetector: detect,
		ReportWriter: &lint.TemplateWriter{
			T: format,
			W: cmd.OutOrStdout(),
		},
		Analyzers: az,
	}
	err = r.Run(cmd.Context())
	// Print the error in case it was not printed before.
	cmd.SilenceErrors = errors.As(err, &lint.SilentError{})
	cmd.SilenceUsage = cmd.SilenceErrors
	return err
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
				if err := migrateFlagsFromEnv(cmd); err != nil {
					return err
				}
				if err := dirFormatBC(flags.dirFormat, &flags.dirURL); err != nil {
					return err
				}
				return checkDir(cmd, flags.dirURL, false)
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				return migrateNewRun(cmd, args, flags)
			},
		}
	)
	cmd.Flags().SortFlags = false
	addFlagDirURL(cmd.Flags(), &flags.dirURL)
	addFlagDirFormat(cmd.Flags(), &flags.dirFormat)
	cmd.Flags().BoolVarP(&flags.edit, flagEdit, "", false, "edit the created migration file(s)")
	return cmd
}

func migrateNewRun(_ *cobra.Command, args []string, flags migrateNewFlags) error {
	u, err := url.Parse(flags.dirURL)
	if err != nil {
		return err
	}
	dir, err := dirURL(u, true)
	if err != nil {
		return err
	}
	if flags.edit {
		dir = &editDir{dir}
	}
	f, err := formatter(u)
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
			Example: `  atlas migrate set 3 --url mysql://user:pass@localhost:3306/
  atlas migrate set --env local
  atlas migrate set 1.2.4 --url mysql://user:pass@localhost:3306/my_db --revision-schema my_revisions`,
			PreRunE: func(cmd *cobra.Command, _ []string) error {
				if err := migrateFlagsFromEnv(cmd); err != nil {
					return err
				}
				if err := dirFormatBC(flags.dirFormat, &flags.dirURL); err != nil {
					return err
				}
				return checkDir(cmd, flags.dirURL, false)
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				return migrateSetRun(cmd, args, flags)
			},
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
	dir, err := dir(flags.dirURL, false)
	if err != nil {
		return err
	}
	files, err := dir.Files()
	if err != nil {
		return err
	}
	client, err := sqlclient.Open(ctx, flags.url)
	if err != nil {
		return err
	}
	defer client.Close()
	// Acquire a lock.
	if l, ok := client.Driver.(schema.Locker); ok {
		unlock, err := l.Lock(ctx, applyLockValue, 0)
		if err != nil {
			return fmt.Errorf("acquiring database lock: %w", err)
		}
		// If unlocking fails notify the user about it.
		defer func() { cobra.CheckErr(unlock()) }()
	}
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
	log := &cmdlog.MigrateSet{}
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
	if log.Current, err = rrw.CurrentRevision(ctx); err != nil && !ent.IsNotFound(err) {
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
			Example: `  atlas migrate status --url mysql://user:pass@localhost:3306/
  atlas migrate status --url mysql://user:pass@localhost:3306/ --dir file:///path/to/migration/directory`,
			PreRunE: func(cmd *cobra.Command, _ []string) error {
				if err := migrateFlagsFromEnv(cmd); err != nil {
					return err
				}
				if err := dirFormatBC(flags.dirFormat, &flags.dirURL); err != nil {
					return err
				}
				return checkDir(cmd, flags.dirURL, false)
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				return migrateStatusRun(cmd, args, flags)
			},
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURL(cmd.Flags(), &flags.url)
	addFlagLog(cmd.Flags(), &flags.logFormat)
	addFlagDirURL(cmd.Flags(), &flags.dirURL)
	addFlagDirFormat(cmd.Flags(), &flags.dirFormat)
	addFlagRevisionSchema(cmd.Flags(), &flags.revisionSchema)
	return cmd
}

func migrateStatusRun(cmd *cobra.Command, _ []string, flags migrateStatusFlags) error {
	dir, err := dir(flags.dirURL, false)
	if err != nil {
		return err
	}
	client, err := sqlclient.Open(cmd.Context(), flags.url)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := checkRevisionSchemaClarity(cmd, client, flags.revisionSchema); err != nil {
		return err
	}
	report, err := (&cmdmigrate.StatusReporter{
		Client: client,
		Dir:    dir,
		Schema: revisionSchemaName(client, flags.revisionSchema),
	}).Report(cmd.Context())
	if err != nil {
		return err
	}
	format := cmdlog.MigrateStatusTemplate
	if f := flags.logFormat; f != "" {
		if format, err = template.New("format").Funcs(cmdlog.StatusTemplateFuncs).Parse(f); err != nil {
			return fmt.Errorf("parse log format: %w", err)
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
  atlas migrate validate --dir file:///path/to/migration/directory
  atlas migrate validate --dir file:///path/to/migration/directory --dev-url mysql://user:pass@localhost:3306/dev
  atlas migrate validate --env dev`,
			PreRunE: func(cmd *cobra.Command, _ []string) error {
				if err := migrateFlagsFromEnv(cmd); err != nil {
					return err
				}
				if err := dirFormatBC(flags.dirFormat, &flags.dirURL); err != nil {
					return err
				}
				return checkDir(cmd, flags.dirURL, false)
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				return migrateValidateRun(cmd, args, flags)
			},
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
	dir, err := dir(flags.dirURL, false)
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
		opts := &schema.InspectOptions{Tables: []string{revision.Table}}
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

func entRevisions(ctx context.Context, c *sqlclient.Client, flag string) (*cmdmigrate.EntRevisions, error) {
	return cmdmigrate.NewEntRevisions(ctx, c, cmdmigrate.WithSchema(revisionSchemaName(c, flag)))
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
	txModeNone = "none"
	txModeAll  = "all"
	txModeFile = "file"
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
	switch ds := l.Directive("txmode"); {
	case len(ds) > 1:
		return "", fmt.Errorf("multiple txmode values found in file %q: %q", f.Name(), ds)
	case len(ds) == 0 || ds[0] == tx.mode:
		return tx.mode, nil
	case ds[0] == txModeAll:
		return "", fmt.Errorf("txmode %q is not allowed in file directive %q", txModeAll, f.Name())
	case ds[0] == txModeNone, ds[0] == txModeFile:
		if tx.mode == txModeAll {
			return "", fmt.Errorf("cannot set txmode directive to %q in %q when txmode %q is set globally", ds[0], f.Name(), txModeAll)
		}
		return ds[0], nil
	default:
		return "", fmt.Errorf("unknown txmode %q found in file directive %q", ds[0], f.Name())
	}
}

func operatorVersion() string {
	v, _ := parse(version)
	return "Atlas CLI " + v
}

// dir parses u and calls dirURL.
func dir(u string, create bool) (migrate.Dir, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	return dirURL(parsed, create)
}

// dirURL returns a migrate.Dir to use as migration directory. For now only local directories are supported.
func dirURL(u *url.URL, create bool) (migrate.Dir, error) {
	if u.Scheme != "file" {
		return nil, fmt.Errorf("unsupported driver %q", u.Scheme)
	}
	path := filepath.Join(u.Host, u.Path)
	if path == "" {
		path = "migrations"
	}
	fn := func() (migrate.Dir, error) { return migrate.NewLocalDir(path) }
	switch f := u.Query().Get("format"); f {
	case "", formatAtlas:
		// this is the default
	case formatGolangMigrate:
		fn = func() (migrate.Dir, error) { return sqltool.NewGolangMigrateDir(path) }
	case formatGoose:
		fn = func() (migrate.Dir, error) { return sqltool.NewGooseDir(path) }
	case formatFlyway:
		fn = func() (migrate.Dir, error) { return sqltool.NewFlywayDir(path) }
	case formatLiquibase:
		fn = func() (migrate.Dir, error) { return sqltool.NewLiquibaseDir(path) }
	case formatDBMate:
		fn = func() (migrate.Dir, error) { return sqltool.NewDBMateDir(path) }
	default:
		return nil, fmt.Errorf("unknown dir format %q", f)
	}
	d, err := fn()
	if create && errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, err
		}
		d, err = fn()
		if err != nil {
			return nil, err
		}
	}
	return d, err
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
	d, err := dir(url, create)
	if err != nil {
		return err
	}
	if err = migrate.Validate(d); err != nil {
		printChecksumError(cmd)
		return err
	}
	return nil
}

func printChecksumError(cmd *cobra.Command) {
	fmt.Fprintf(cmd.OutOrStderr(), `You have a checksum error in your migration directory.
This happens if you manually create or edit a migration file.
Please check your migration files and run

'atlas migrate hash'

to re-hash the contents and resolve the error

`)
	cmd.SilenceUsage = true
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
		case scheme == "":
			scheme = current
		case scheme != current:
			return "", fmt.Errorf("got mixed --to url schemes: %q and %q, the desired state must be provided from a single kind of source", scheme, current)
		case current != "file":
			return "", fmt.Errorf("got multiple --to urls of scheme %q, only multiple 'file://' urls are supported", current)
		}
	}
	return scheme, nil
}

// parseHCLPaths parses the HCL files in the given paths. If a path represents a directory,
// its direct descendants will be considered, skipping any subdirectories. If a project file
// is present in the input paths, an error is returned.
func parseHCLPaths(paths ...string) (*hclparse.Parser, error) {
	p := hclparse.NewParser()
	for _, path := range paths {
		switch stat, err := os.Stat(path); {
		case err != nil:
			return nil, err
		case stat.IsDir():
			dir, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}
			for _, f := range dir {
				// Skip nested dirs.
				if f.IsDir() {
					continue
				}
				if err := mayParse(p, filepath.Join(path, f.Name())); err != nil {
					return nil, err
				}
			}
		default:
			if err := mayParse(p, path); err != nil {
				return nil, err
			}
		}
	}
	if len(p.Files()) == 0 {
		return nil, fmt.Errorf("no schema files found in: %s", paths)
	}
	return p, nil
}

// mayParse will parse the file in path if it is an HCL file. If the file is an Atlas
// project file an error is returned.
func mayParse(p *hclparse.Parser, path string) error {
	if n := filepath.Base(path); filepath.Ext(n) != ".hcl" {
		return nil
	}
	switch f, diag := p.ParseHCLFile(path); {
	case diag.HasErrors():
		return diag
	case isProjectFile(f):
		return fmt.Errorf("cannot parse project file %q as a schema file", path)
	default:
		return nil
	}
}

func isProjectFile(f *hcl.File) bool {
	for _, blk := range f.Body.(*hclsyntax.Body).Blocks {
		if blk.Type == "env" {
			return true
		}
	}
	return false
}

const (
	formatAtlas         = "atlas"
	formatGolangMigrate = "golang-migrate"
	formatGoose         = "goose"
	formatFlyway        = "flyway"
	formatLiquibase     = "liquibase"
	formatDBMate        = "dbmate"
)

func formatter(u *url.URL) (migrate.Formatter, error) {
	switch f := u.Query().Get("format"); f {
	case formatAtlas:
		return migrate.DefaultFormatter, nil
	case formatGolangMigrate:
		return sqltool.GolangMigrateFormatter, nil
	case formatGoose:
		return sqltool.GooseFormatter, nil
	case formatFlyway:
		return sqltool.FlywayFormatter, nil
	case formatLiquibase:
		return sqltool.LiquibaseFormatter, nil
	case formatDBMate:
		return sqltool.DBMateFormatter, nil
	default:
		return nil, fmt.Errorf("unknown format %q", f)
	}
}

func migrateFlagsFromEnv(cmd *cobra.Command) error {
	activeEnv, err := selectEnv(GlobalFlags.SelectedEnv)
	if err != nil {
		return err
	}
	return setMigrateEnvFlags(cmd, activeEnv)
}

func setMigrateEnvFlags(cmd *cobra.Command, env *Env) error {
	if err := inputValuesFromEnv(cmd, env); err != nil {
		return err
	}
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
		if err := maySetFlag(cmd, flagLog, env.Log.Migrate.Apply); err != nil {
			return err
		}
		if err := maySetFlag(cmd, flagLockTimeout, env.Migration.LockTimeout); err != nil {
			return err
		}
	case "diff":
		if err := maySetFlag(cmd, flagLockTimeout, env.Migration.LockTimeout); err != nil {
			return err
		}
	case "lint":
		if err := maySetFlag(cmd, flagLog, env.Log.Migrate.Lint); err != nil {
			return err
		}
		if err := maySetFlag(cmd, flagLog, env.Lint.Log); err != nil {
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
		if err := maySetFlag(cmd, flagLog, env.Log.Migrate.Status); err != nil {
			return err
		}
	}
	// Transform "src" to a URL.
	srcs, err := env.Sources()
	if err != nil {
		return err
	}
	for i, s := range srcs {
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

// cmdEnvsRun executes a given command on each of the configured environment.
func cmdEnvsRun[F any](
	run func(*cobra.Command, []string, F) error,
	set func(*cobra.Command, *Env) error,
	cmd *cobra.Command, args []string, flags *F,
) error {
	envs, err := LoadEnv(GlobalFlags.SelectedEnv, WithInput(GlobalFlags.Vars))
	if err != nil {
		return err
	}
	var (
		w     bytes.Buffer
		out   = cmd.OutOrStdout()
		reset = resetFromEnv(cmd)
	)
	cmd.SetOut(io.MultiWriter(out, &w))
	defer cmd.SetOut(out)
	for i, e := range envs {
		if err := set(cmd, e); err != nil {
			return err
		}
		if err := run(cmd, args, *flags); err != nil {
			return err
		}
		b := bytes.TrimLeft(w.Bytes(), " \t\r")
		// In case a custom logging was configured, ensure there is
		// a newline separator between the different environments.
		if cmd.Flags().Changed(flagLog) && bytes.LastIndexByte(b, '\n') != len(b)-1 && i != len(envs)-1 {
			cmd.Println()
		}
		reset()
		w.Reset()
	}
	return nil
}

type editDir struct{ migrate.Dir }

// WriteFile implements the migrate.Dir.WriteFile method.
func (d *editDir) WriteFile(name string, b []byte) (err error) {
	if name != migrate.HashFileName {
		if b, err = edit(name, b); err != nil {
			return err
		}
	}
	return d.Dir.WriteFile(name, b)
}

// edit allows editing the file content using editor.
func edit(name string, src []byte) ([]byte, error) {
	path := filepath.Join(os.TempDir(), name)
	if err := os.WriteFile(path, src, 0644); err != nil {
		return nil, fmt.Errorf("write source content to temp file: %w", err)
	}
	defer os.Remove(path)
	editor := "vi"
	if e := os.Getenv("EDITOR"); e != "" {
		editor = e
	}
	cmd := exec.Command("sh", "-c", editor+" "+path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("exec edit: %w", err)
	}
	b, err := os.ReadFile(path)
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

// QueryContext overrides the wrapped schema.ExecQuerier to not execute any SQL.
func (dryRunDriver) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil
}

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
