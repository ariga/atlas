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
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/lint"
	cmdmigrate "ariga.io/atlas/cmd/atlas/internal/migrate"
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
				return migrateEnvsRun(migrateApplyRun, cmd, args, &flags)
			},
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURL(cmd.Flags(), &flags.url)
	addFlagDirURL(cmd.Flags(), &flags.dirURL)
	addFlagLog(cmd.Flags(), &flags.logFormat)
	addFlagRevisionSchema(cmd.Flags(), &flags.revisionSchema)
	addFlagDryRun(cmd.Flags(), &flags.dryRun)
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
		count, err = strconv.Atoi(args[0])
		if err != nil {
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
		unlock, err := l.Lock(cmd.Context(), applyLockValue, 0)
		if err != nil {
			return fmt.Errorf("acquiring database lock: %w", err)
		}
		// If unlocking fails notify the user about it.
		defer cobra.CheckErr(unlock())
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
	ex, err := migrate.NewExecutor(client.Driver, migrationDir, rrw, opts...)
	if err != nil {
		return err
	}
	pending, err := ex.Pending(cmd.Context())
	if err != nil && !errors.Is(err, migrate.ErrNoPendingFiles) {
		return err
	}
	if errors.Is(err, migrate.ErrNoPendingFiles) {
		cmd.Println("No migration files to execute")
		return nil
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
	report := cmdmigrate.NewApplyReport(client, migrationDir)
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
		drv, rrw, err = mux.driver(cmd.Context())
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
	f := cmdmigrate.DefaultApplyTemplate
	if v := flags.logFormat; v != "" {
		f, err = template.New("format").Funcs(cmdmigrate.ApplyTemplateFuncs).Parse(v)
		if err != nil {
			return fmt.Errorf("parse log format: %w", err)
		}
	}
	if err2 := (&cmdmigrate.TemplateWriter{T: f, W: cmd.OutOrStdout()}).WriteReport(report); err2 != nil {
		if err != nil {
			return fmt.Errorf("%w: %v", err2, err)
		}
		err = err2
	}
	return err
}

type migrateDiffFlags struct {
	desiredURLs       []string
	dirURL, dirFormat string
	devURL            string
	schemas           []string
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
	addFlagURLs(cmd.Flags(), &flags.desiredURLs, flagTo)
	addFlagDevURL(cmd.Flags(), &flags.devURL)
	addFlagDirURL(cmd.Flags(), &flags.dirURL)
	addFlagDirFormat(cmd.Flags(), &flags.dirFormat)
	addFlagRevisionSchema(cmd.Flags(), &flags.revisionSchema)
	addFlagSchemas(cmd.Flags(), &flags.schemas)
	cmd.Flags().StringVar(&flags.qualifier, flagQualifier, "", "qualify tables with custom qualifier when working on a single schema")
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
		unlock, err := l.Lock(cmd.Context(), "atlas_migrate_diff", 0)
		if err != nil {
			return fmt.Errorf("acquiring database lock: %w", err)
		}
		// If unlocking fails notify the user about it.
		defer cobra.CheckErr(unlock())
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
	f, err := formatter(u)
	if err != nil {
		return err
	}
	// Get a state reader for the desired state.
	desired, err := toTarget(cmd.Context(), dev, flags.desiredURLs, flags.schemas)
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
	case errors.As(err, &cerr) && dev.URL.Schema == "" && desired.Schema != "":
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

type migrateNewFlags struct{ dirURL, dirFormat string }

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
			Use:   "set [flags] <version>",
			Short: "Set the current version of the migration history table.",
			Long: `'atlas migrate set' edits the revision table to consider all migrations up to and including the given version
to be applied. This command is usually used after manually making changes to the managed database.`,
			Example: `  atlas migrate set 3 --url mysql://user:pass@localhost:3306/
  atlas migrate set 4 --env local
  atlas migrate set 1.2.4 --url mysql://user:pass@localhost:3306/my_db --revision-schema my_revisions`,
			Args: cobra.ExactArgs(1),
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

func migrateSetRun(cmd *cobra.Command, args []string, flags migrateSetFlags) error {
	dir, err := dir(flags.dirURL, false)
	if err != nil {
		return err
	}
	avail, err := dir.Files()
	if err != nil {
		return err
	}
	// Check if the target version does exist in the migration directory.
	if idx := migrate.FilesLastIndex(avail, func(f migrate.File) bool {
		return f.Version() == args[0]
	}); idx == -1 {
		return fmt.Errorf("migration with version %q not found", args[0])
	}
	client, err := sqlclient.Open(cmd.Context(), flags.url)
	if err != nil {
		return err
	}
	defer client.Close()
	// Acquire a lock.
	if l, ok := client.Driver.(schema.Locker); ok {
		unlock, err := l.Lock(cmd.Context(), applyLockValue, 0)
		if err != nil {
			return fmt.Errorf("acquiring database lock: %w", err)
		}
		// If unlocking fails notify the user about it.
		defer cobra.CheckErr(unlock())
	}
	if err := checkRevisionSchemaClarity(cmd, client, flags.revisionSchema); err != nil {
		return err
	}
	// Ensure revision table exists.
	rrw, err := entRevisions(cmd.Context(), client, flags.revisionSchema)
	if err != nil {
		return err
	}
	if err := rrw.Migrate(cmd.Context()); err != nil {
		return err
	}
	// Wrap manipulation in a transaction.
	tx, err := client.Tx(cmd.Context(), nil)
	if err != nil {
		return err
	}
	rrw, err = entRevisions(cmd.Context(), tx.Client, flags.revisionSchema)
	if err != nil {
		return err
	}
	revs, err := rrw.ReadRevisions(cmd.Context())
	if err != nil {
		return err
	}
	if err := func() error {
		for _, r := range revs {
			// Check all existing revisions and ensure they precede the given version. If we encounter a partially
			// applied revision, or one with errors, mark them "fixed".
			switch {
			// remove revision to keep linear history
			case r.Version > args[0]:
				if err := rrw.DeleteRevision(cmd.Context(), r.Version); err != nil {
					return err
				}
			// keep, but if with error mark "fixed"
			case r.Version == args[0] && (r.Error != "" || r.Total != r.Applied):
				r.Type = migrate.RevisionTypeExecute | migrate.RevisionTypeResolved
				if err := rrw.WriteRevision(cmd.Context(), r); err != nil {
					return err
				}
			}
		}
		revs, err = rrw.ReadRevisions(cmd.Context())
		if err != nil {
			return err
		}
		// If the target version succeeds the last revision, mark
		// migrations applied, until we reach the target version.
		var pending []migrate.File
		switch {
		case len(revs) == 0:
			// Take every file until we reach target version.
			for _, f := range avail {
				if f.Version() > args[0] {
					break
				}
				pending = append(pending, f)
			}
		case args[0] > revs[len(revs)-1].Version:
		loop:
			// Take every file succeeding the last revision until we reach target version.
			for _, f := range avail {
				switch {
				case f.Version() <= revs[len(revs)-1].Version:
					// Migration precedes last revision.
				case f.Version() > args[0]:
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
			if err := rrw.WriteRevision(cmd.Context(), &migrate.Revision{
				Version:         f.Version(),
				Description:     f.Desc(),
				Type:            migrate.RevisionTypeResolved,
				ExecutedAt:      time.Now(),
				Hash:            h,
				OperatorVersion: operatorVersion(),
			}); err != nil {
				return err
			}
		}
		return nil
	}(); err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			err = fmt.Errorf("%v: %w", err2, err)
		}
		return err
	}
	return tx.Commit()
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
	var format = cmdmigrate.DefaultStatusTemplate
	if f := flags.logFormat; f != "" {
		format, err = template.New("format").Funcs(cmdmigrate.StatusTemplateFuncs).Parse(f)
		if err != nil {
			return fmt.Errorf("parse log format: %w", err)
		}
	}
	return (&cmdmigrate.StatusReporter{
		Client:       client,
		Dir:          dir,
		ReportWriter: &cmdmigrate.TemplateWriter{T: format, W: cmd.OutOrStdout()},
		Schema:       revisionSchemaName(client, flags.revisionSchema),
	}).Report(cmd.Context())
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
	tx           *sqlclient.TxClient
	rrw          migrate.RevisionReadWriter
}

// driver returns the migrate.Driver to use to execute migration statements.
func (tx *tx) driver(ctx context.Context) (migrate.Driver, migrate.RevisionReadWriter, error) {
	if tx.dryRun {
		// If the --dry-run flag is given we don't want to execute any statements on the database.
		return &dryRunDriver{tx.c.Driver}, &dryRunRevisions{tx.rrw}, nil
	}
	switch tx.mode {
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
		tx.rrw, err = entRevisions(ctx, tx.tx.Client, tx.schema)
		if err != nil {
			return nil, nil, err
		}
		return tx.tx.Driver, tx.rrw, nil
	case txModeAll:
		// In file-mode, this function is called each time a new file is executed. Since we wrap all files into one
		// huge transaction, if there already is an opened one, use that.
		if tx.tx == nil {
			var err error
			tx.tx, err = tx.c.Tx(ctx, nil)
			if err != nil {
				return nil, nil, err
			}
			tx.rrw, err = entRevisions(ctx, tx.tx.Client, tx.schema)
			if err != nil {
				return nil, nil, err
			}
		}
		return tx.tx.Driver, tx.rrw, nil
	default:
		return nil, nil, fmt.Errorf("unknown tx-mode %q", tx.mode)
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
	if !tx.dryRun && tx.mode == txModeFile {
		return tx.commit()
	}
	return nil
}

// commit the transaction, if one is active.
func (tx *tx) commit() error {
	if tx.tx == nil {
		return nil
	}
	defer func() { tx.tx = nil }()
	return tx.tx.Commit()
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

type target struct {
	migrate.StateReader        // desired state.
	io.Closer                  // optional close function.
	Schema              string // in case we work on a single schema.
}

// to returns a migrate.StateReader for the given to flag.
func toTarget(ctx context.Context, dev *sqlclient.Client, urls, schemas []string) (*target, error) {
	scheme, err := selectScheme(urls)
	if err != nil {
		return nil, err
	}
	switch scheme {
	case "file": // hcl file
		realm := &schema.Realm{}
		paths := make([]string, 0, len(urls))
		for _, u := range urls {
			paths = append(paths, strings.TrimPrefix(u, "file://"))
		}
		parsed, err := parseHCLPaths(paths...)
		if err != nil {
			return nil, err
		}
		if err := dev.Eval(parsed, realm, nil); err != nil {
			return nil, err
		}
		if len(schemas) > 0 {
			// Validate all schemas in file were selected by user.
			sm := make(map[string]bool, len(schemas))
			for _, s := range schemas {
				sm[s] = true
			}
			for _, s := range realm.Schemas {
				if !sm[s.Name] {
					return nil, fmt.Errorf("schema %q from paths %q is not requested (all schemas in HCL must be requested)", s.Name, paths)
				}
			}
		}
		// In case the dev connection is bound to a specific schema, we require the
		// desired schema to contain only one schema. Thus, executing diff will be
		// done on the content of these two schema and not the whole realm.
		if dev.URL.Schema != "" && len(realm.Schemas) > 1 {
			return nil, fmt.Errorf("cannot use HCL with more than 1 schema when dev-url is limited to schema %q", dev.URL.Schema)
		}
		if norm, ok := dev.Driver.(schema.Normalizer); ok && len(realm.Schemas) > 0 {
			realm, err = norm.NormalizeRealm(ctx, realm)
			if err != nil {
				return nil, err
			}
		}
		t := &target{StateReader: migrate.Realm(realm), Closer: io.NopCloser(nil)}
		if len(realm.Schemas) == 1 {
			t.Schema = realm.Schemas[0].Name
		}
		return t, nil
	default: // database connection
		client, err := sqlclient.Open(ctx, urls[0])
		if err != nil {
			return nil, err
		}
		t := &target{Closer: client}
		switch s := client.URL.Schema; {
		// Connection to a specific schema.
		case s != "":
			if len(schemas) > 1 || len(schemas) == 1 && schemas[0] != s {
				return nil, fmt.Errorf("cannot specify schemas with a schema connection to %q", s)
			}
			t.Schema = s
			t.StateReader = migrate.SchemaConn(client, s, &schema.InspectOptions{})
		// A single schema is selected.
		case len(schemas) == 1:
			t.Schema = schemas[0]
			t.StateReader = migrate.SchemaConn(client, schemas[0], &schema.InspectOptions{})
		// Multiple or all schemas.
		default:
			// In case the dev connection is limited to a single schema,
			// but we compare it to entire database.
			if dev.URL.Schema != "" {
				return nil, fmt.Errorf("cannot use database-url without a schema when dev-url is limited to %q", dev.URL.Schema)
			}
			t.StateReader = migrate.RealmConn(client, &schema.InspectRealmOption{Schemas: schemas})
		}
		return t, nil
	}
}

// selectScheme validates the scheme of the provided to urls and returns the selected
// url scheme. Currently, all URLs must be of the same scheme, and only multiple
// "file://" URLs are allowed.
func selectScheme(urls []string) (string, error) {
	var scheme string
	if len(urls) == 0 {
		return "", errors.New("at least one url is required")
	}
	for _, url := range urls {
		parts := strings.SplitN(url, "://", 2)
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
	return setFlagsFromEnv(cmd, activeEnv)
}

func setFlagsFromEnv(cmd *cobra.Command, env *Env) error {
	if err := inputValsFromEnv(cmd, env); err != nil {
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
	if err := maySetFlag(cmd, flagURL, env.URL); err != nil {
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
	if s := "[" + strings.Join(env.Schemas, "") + "]"; len(env.Schemas) > 0 {
		if err := maySetFlag(cmd, flagSchema, s); err != nil {
			return err
		}
	}
	return nil
}

// migrateEnvsRun executes a given command on each of the configured environment.
func migrateEnvsRun[F any](run func(*cobra.Command, []string, F) error, cmd *cobra.Command, args []string, flags *F) error {
	envs, err := LoadEnv(GlobalFlags.SelectedEnv, WithInput(GlobalFlags.Vars))
	if err != nil {
		return err
	}
	var (
		w     bytes.Buffer
		reset = resetFromEnv(cmd)
	)
	cmd.SetOut(io.MultiWriter(cmd.OutOrStdout(), &w))
	for i, e := range envs {
		if err := setFlagsFromEnv(cmd, e); err != nil {
			return err
		}
		if err := run(cmd, args, *flags); err != nil {
			return err
		}
		out := bytes.TrimLeft(w.Bytes(), " \t\r")
		// In case a custom logging was configured, ensure there is
		// a newline separator between the different environments.
		if cmd.Flags().Changed(flagLog) && bytes.LastIndexByte(out, '\n') != len(out)-1 && i != len(envs)-1 {
			cmd.Println()
		}
		reset()
		w.Reset()
	}
	return nil
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
