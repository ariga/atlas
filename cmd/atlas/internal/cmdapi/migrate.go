// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/lint"
	entmigrate "ariga.io/atlas/cmd/atlas/internal/migrate"
	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/revision"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"
	"ariga.io/atlas/sql/sqltool"
	"github.com/fatih/color"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	migrateFlagURL              = "url"
	migrateFlagDevURL           = "dev-url"
	migrateFlagDir              = "dir"
	migrateFlagDirFormat        = "dir-format"
	migrateFlagForce            = "force"
	migrateFlagLog              = "log"
	migrateFlagRevisionsSchema  = "revisions-schema"
	migrateFlagDryRun           = "dry-run"
	migrateFlagTo               = "to"
	migrateFlagSchema           = "schema"
	migrateLintLatest           = "latest"
	migrateLintGitDir           = "git-dir"
	migrateLintGitBase          = "git-base"
	migrateDiffQualifier        = "qualifier"
	migrateApplyAllowDirty      = "allow-dirty"
	migrateApplyFromVersion     = "from"
	migrateApplyBaselineVersion = "baseline"
	migrateApplyTxMode          = "tx-mode"
)

var (
	// MigrateFlags are the flags used in MigrateCmd (and sub-commands).
	MigrateFlags struct {
		URL            string
		DevURL         string
		ToURLs         []string
		Schemas        []string
		DirURL         string
		DirFormat      string
		RevisionSchema string
		Force          bool
		Apply          struct {
			DryRun          bool
			LogFormat       string
			AllowDirty      bool
			FromVersion     string
			BaselineVersion string
			TxMode          string
		}
		Diff struct {
			Qualifier string // optional table qualifier
		}
		Lint struct {
			Format  string // log formatting
			Latest  uint   // latest N migration files
			GitDir  string // repository working dir
			GitBase string // branch name to compare with
		}
	}
	// MigrateCmd represents the migrate command. It wraps several other sub-commands.
	MigrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Manage versioned migration files",
		Long:  "'atlas migrate' wraps several sub-commands for migration management.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := migrateFlagsFromEnv(cmd, nil); err != nil {
				return err
			}
			// Migrate commands will not run on a broken migration directory, unless the force flag is given.
			if !MigrateFlags.Force {
				dir, err := dir(false)
				if err != nil {
					return err
				}
				if err := migrate.Validate(dir); err != nil {
					printChecksumErr(cmd.OutOrStderr())
					cmd.SilenceUsage = true
					return err
				}
			}
			return nil
		},
	}
	// MigrateApplyCmd represents the 'atlas migrate apply' subcommand.
	MigrateApplyCmd = &cobra.Command{
		Use:   "apply [flags] [count]",
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
		RunE: CmdMigrateApplyRun,
	}
	// MigrateDiffCmd represents the 'atlas migrate diff' subcommand.
	MigrateDiffCmd = &cobra.Command{
		Use:   "diff [flags] [name]",
		Short: "Compute the diff between the migration directory and a desired state and create a new migration file.",
		Long: `'atlas migrate diff' uses the dev-database to re-run all migration files in the migration directory, compares
it to a given desired state and create a new migration file containing SQL statements to migrate the migration
directory state to the desired schema. The desired state can be another connected database or an HCL file.`,
		Example: `  atlas migrate diff --dev-url mysql://user:pass@localhost:3306/dev --to file://atlas.hcl
  atlas migrate diff --dev-url mysql://user:pass@localhost:3306/dev --to file://atlas.hcl add_users_table
  atlas migrate diff --dev-url mysql://user:pass@localhost:3306/dev --to mysql://user:pass@localhost:3306/dbname
  atlas migrate diff --env dev`,
		Args: cobra.MaximumNArgs(1),
		// If the migration directory does not exist on the validation attempt, this command will create it and
		// consider the new migration directory "valid".
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := migrateFlagsFromEnv(cmd, nil); err != nil {
				return err
			}
			// Migrate commands will not run on a broken migration directory, unless the force flag is given.
			if !MigrateFlags.Force {
				dir, err := dir(true)
				if err != nil {
					return err
				}
				if err := migrate.Validate(dir); err != nil {
					printChecksumErr(cmd.OutOrStderr())
					cmd.SilenceUsage = true
					return err
				}
			}
			return nil
		},
		RunE: CmdMigrateDiffRun,
	}
	// MigrateHashCmd represents the 'atlas migrate hash' command.
	MigrateHashCmd = &cobra.Command{
		Use:   "hash",
		Short: "Hash (re-)creates an integrity hash file for the migration directory.",
		Long: `'atlas migrate hash' computes the integrity hash sum of the migration directory and stores it in the atlas.sum file.
This command should be used whenever a manual change in the migration directory was made.`,
		Example: `  atlas migrate hash --force`,
		RunE:    CmdMigrateHashRun,
	}
	// MigrateNewCmd represents the 'atlas migrate new' command.
	MigrateNewCmd = &cobra.Command{
		Use:     "new [name]",
		Short:   "Creates a new empty migration file in the migration directory.",
		Long:    `'atlas migrate new' creates a new migration according to the configured formatter without any statements in it.`,
		Example: `  atlas migrate new my-new-migration`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    CmdMigrateNewRun,
	}
	// MigrateStatusCmd represents the 'atlas migrate status' command.
	MigrateStatusCmd = &cobra.Command{
		Use:   "status [flags]",
		Short: "Get information about the current migration status.",
		Long:  `'atlas migrate status' reports information about the current status of a connected database compared to the migration directory.`,
		Example: `  atlas migrate status --url mysql://user:pass@localhost:3306/
  atlas migrate status --url mysql://user:pass@localhost:3306/ --dir file:///path/to/migration/directory`,
		RunE: CmdMigrateStatusRun,
	}
	// MigrateValidateCmd represents the 'atlas migrate validate' command.
	MigrateValidateCmd = &cobra.Command{
		Use:   "validate [flags]",
		Short: "Validates the migration directories checksum and SQL statements.",
		Long: `'atlas migrate validate' computes the integrity hash sum of the migration directory and compares it to the
atlas.sum file. If there is a mismatch it will be reported. If the --dev-url flag is given, the migration
files are executed on the connected database in order to validate SQL semantics.`,
		Example: `  atlas migrate validate
  atlas migrate validate --dir file:///path/to/migration/directory
  atlas migrate validate --dir file:///path/to/migration/directory --dev-url mysql://user:pass@localhost:3306/dev
  atlas migrate validate --env dev`,
		RunE: CmdMigrateValidateRun,
	}
	// MigrateLintCmd represents the 'atlas migrate Lint' command.
	MigrateLintCmd = &cobra.Command{
		Use:   "lint",
		Short: "Run analysis on the migration directory",
		Example: `  atlas migrate lint --env dev
  atlas migrate lint --dir file:///path/to/migration/directory --dev-url mysql://root:pass@localhost:3306 --latest 1
  atlas migrate lint --dir file:///path/to/migration/directory --dev-url mysql://root:pass@localhost:3306 --git-base master
  atlas migrate lint --dir file:///path/to/migration/directory --dev-url mysql://root:pass@localhost:3306 --log '{{ json .Files }}'`,
		// Override the parent 'migrate' pre-run function to allow executing
		// 'migrate lint' on directories that are not maintained by Atlas.
		PersistentPreRunE: migrateFlagsFromEnv,
		RunE:              CmdMigrateLintRun,
	}
)

func init() {
	// Add sub-commands.
	Root.AddCommand(MigrateCmd)
	MigrateCmd.AddCommand(MigrateApplyCmd)
	MigrateCmd.AddCommand(MigrateDiffCmd)
	MigrateCmd.AddCommand(MigrateHashCmd)
	MigrateCmd.AddCommand(MigrateNewCmd)
	MigrateCmd.AddCommand(MigrateValidateCmd)
	MigrateCmd.AddCommand(MigrateStatusCmd)
	MigrateCmd.AddCommand(MigrateLintCmd)
	// Reusable flags.
	urlFlag := func(f *string, name, short string, set *pflag.FlagSet) {
		set.StringVarP(f, name, short, "", "[driver://username:password@address/dbname?param=value] select a database using the URL format")
	}
	revisionsFlag := func(set *pflag.FlagSet) {
		set.StringVarP(&MigrateFlags.RevisionSchema, migrateFlagRevisionsSchema, "", entmigrate.DefaultRevisionSchema, "schema name where the revisions table resides")
	}
	// Global flags.
	MigrateCmd.PersistentFlags().StringVarP(&MigrateFlags.DirURL, migrateFlagDir, "", "file://migrations", "select migration directory using URL format")
	MigrateCmd.PersistentFlags().StringSliceVarP(&MigrateFlags.Schemas, migrateFlagSchema, "", nil, "set schema names")
	MigrateCmd.PersistentFlags().StringVarP(&MigrateFlags.DirFormat, migrateFlagDirFormat, "", formatAtlas, "set migration file format")
	MigrateCmd.PersistentFlags().BoolVarP(&MigrateFlags.Force, migrateFlagForce, "", false, "force a command to run on a broken migration directory state")
	MigrateCmd.PersistentFlags().SortFlags = false
	// Apply flags.
	MigrateApplyCmd.Flags().StringVarP(&MigrateFlags.Apply.LogFormat, migrateFlagLog, "", logFormatTTY, "log format to use")
	revisionsFlag(MigrateApplyCmd.Flags())
	MigrateApplyCmd.Flags().BoolVarP(&MigrateFlags.Apply.DryRun, migrateFlagDryRun, "", false, "do not actually execute any SQL but show it on screen")
	MigrateApplyCmd.Flags().StringVarP(&MigrateFlags.Apply.FromVersion, migrateApplyFromVersion, "", "", "calculate pending files from the given version (including it)")
	MigrateApplyCmd.Flags().StringVarP(&MigrateFlags.Apply.BaselineVersion, migrateApplyBaselineVersion, "", "", "start the first migration after the given baseline version")
	MigrateApplyCmd.Flags().StringVarP(&MigrateFlags.Apply.TxMode, migrateApplyTxMode, "", txModeFile, "set transaction mode [none, file, all]")
	MigrateApplyCmd.Flags().BoolVarP(&MigrateFlags.Apply.AllowDirty, migrateApplyAllowDirty, "", false, "allow start working on a non-clean database")
	urlFlag(&MigrateFlags.URL, migrateFlagURL, "u", MigrateApplyCmd.Flags())
	MigrateApplyCmd.Flags().SortFlags = false
	cobra.CheckErr(MigrateApplyCmd.MarkFlagRequired(migrateFlagURL))
	MigrateApplyCmd.MarkFlagsMutuallyExclusive(migrateApplyTxMode, migrateFlagDryRun)
	MigrateApplyCmd.MarkFlagsMutuallyExclusive(migrateApplyFromVersion, migrateApplyBaselineVersion)
	// Diff flags.
	urlFlag(&MigrateFlags.DevURL, migrateFlagDevURL, "", MigrateDiffCmd.Flags())
	MigrateDiffCmd.Flags().StringSliceVarP(&MigrateFlags.ToURLs, migrateFlagTo, "", nil, "[driver://username:password@address/dbname?param=value ...] select a desired state using the URL format")
	MigrateDiffCmd.Flags().StringVarP(&MigrateFlags.Diff.Qualifier, migrateDiffQualifier, "", "", "qualify tables with custom qualifier when working on a single schema")
	MigrateDiffCmd.Flags().SortFlags = false
	cobra.CheckErr(MigrateDiffCmd.MarkFlagRequired(migrateFlagDevURL))
	cobra.CheckErr(MigrateDiffCmd.MarkFlagRequired(migrateFlagTo))
	// Validate flags.
	urlFlag(&MigrateFlags.DevURL, migrateFlagDevURL, "", MigrateValidateCmd.Flags())
	// Status flags.
	urlFlag(&MigrateFlags.URL, migrateFlagURL, "u", MigrateStatusCmd.Flags())
	revisionsFlag(MigrateStatusCmd.Flags())
	// Lint flags.
	urlFlag(&MigrateFlags.DevURL, migrateFlagDevURL, "", MigrateLintCmd.Flags())
	MigrateLintCmd.PersistentFlags().StringVarP(&MigrateFlags.Lint.Format, migrateFlagLog, "", "", "custom logging using a Go template")
	MigrateLintCmd.PersistentFlags().UintVarP(&MigrateFlags.Lint.Latest, migrateLintLatest, "", 0, "run analysis on the latest N migration files")
	MigrateLintCmd.PersistentFlags().StringVarP(&MigrateFlags.Lint.GitBase, migrateLintGitBase, "", "", "run analysis against the base Git branch")
	MigrateLintCmd.PersistentFlags().StringVarP(&MigrateFlags.Lint.GitDir, migrateLintGitDir, "", ".", "path to the repository working directory")
	cobra.CheckErr(MigrateLintCmd.MarkFlagRequired(migrateFlagDevURL))
	receivesEnv(MigrateCmd)
}

// CmdMigrateApplyRun is the command executed when running the CLI with 'migrate apply' args.
func CmdMigrateApplyRun(cmd *cobra.Command, args []string) error {
	var (
		n   int
		err error
	)
	if len(args) > 0 {
		n, err = strconv.Atoi(args[0])
		if err != nil {
			return err
		}
	}
	// Open the migration directory.
	dir, err := dir(false)
	if err != nil {
		return err
	}
	// Open a client to the database.
	c, err := sqlclient.Open(cmd.Context(), MigrateFlags.URL)
	if err != nil {
		return err
	}
	defer c.Close()
	// Acquire a lock.
	if l, ok := c.Driver.(schema.Locker); ok {
		unlock, err := l.Lock(cmd.Context(), "atlas_migrate_execute", 0)
		if err != nil {
			return fmt.Errorf("acquiring database lock: %w", err)
		}
		// If unlocking fails notify the user about it.
		defer cobra.CheckErr(unlock())
	}
	// Get the correct log format and destination. Currently, only os.Stdout is supported.
	l, err := logFormat(cmd.OutOrStdout())
	if err != nil {
		return err
	}
	var rrw migrate.RevisionReadWriter
	rrw, err = entmigrate.NewEntRevisions(c, []entmigrate.Option{entmigrate.WithSchema(MigrateFlags.RevisionSchema)}...)
	if err != nil {
		return err
	}
	if err := rrw.(*entmigrate.EntRevisions).Init(cmd.Context()); err != nil {
		return err
	}
	defer func(rrw *entmigrate.EntRevisions) {
		if err2 := rrw.Flush(cmd.Context()); err2 != nil {
			if err != nil {
				err2 = fmt.Errorf("%w: %v", err, err2)
			}
			err = err2
		}
	}(rrw.(*entmigrate.EntRevisions))
	// Determine pending files and lock the database while working.
	ex, err := migrate.NewExecutor(c.Driver, dir, rrw, executorOptions(l)...)
	if err != nil {
		return err
	}
	pending, err := ex.Pending(cmd.Context())
	if err != nil && !errors.Is(err, migrate.ErrNoPendingFiles) {
		return err
	}
	if errors.Is(err, migrate.ErrNoPendingFiles) {
		cmd.Println("The migration directory is synced with the database, no migration files to execute")
		return nil
	}
	if n > 0 {
		// Cannot apply more than len(pending) files.
		if n >= len(pending) {
			n = len(pending)
		}
		pending = pending[:n]
	}
	revs, err := rrw.ReadRevisions(cmd.Context())
	if err != nil {
		return err
	}
	if err := migrate.LogIntro(l, revs, pending); err != nil {
		return err
	}
	var (
		mux = tx{ctx: cmd.Context(), c: c, rrw: rrw, files: pending}
		drv migrate.Driver
	)
	for _, f := range pending {
		drv, rrw, err = mux.driver(cmd.Context())
		if err != nil {
			return err
		}
		ex, err := migrate.NewExecutor(drv, dir, rrw, executorOptions(l)...)
		if err != nil {
			return err
		}
		if err := mux.mayRollback(ex.Execute(cmd.Context(), f), f); err != nil {
			return err
		}
		if err := mux.mayCommit(); err != nil {
			return err
		}
	}
	if err := mux.commit(); err != nil {
		return err
	}
	l.Log(migrate.LogDone{})
	return mux.commit()
}

// tx handles wrapping migration execution in transactions.
// It is also responsible for saving this information onto the revisions as the core package is transaction agnostic.
type tx struct {
	c     *sqlclient.Client
	tx    *sqlclient.TxClient
	ctx   context.Context
	rrw   migrate.RevisionReadWriter
	files []migrate.File
}

// driver returns the migrate.Driver to use to execute migration statements.
func (tx *tx) driver(ctx context.Context) (migrate.Driver, migrate.RevisionReadWriter, error) {
	if MigrateFlags.Apply.DryRun {
		// If the --dry-run flag is given we don't want to execute any statements on the database.
		return &dryRunDriver{tx.c.Driver}, &dryRunRevisions{tx.rrw}, nil
	}
	switch MigrateFlags.Apply.TxMode {
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
		}
		return tx.tx.Driver, tx.rrw, nil
	default:
		return nil, nil, fmt.Errorf("unknown tx-mode %q", MigrateFlags.Apply.TxMode)
	}
}

// mayRollback may roll back a transaction depending on the given transaction mode.
func (tx *tx) mayRollback(err error, file migrate.File) error {
	if tx.tx != nil && err != nil {
		if err2 := tx.tx.Rollback(); err2 != nil {
			err = fmt.Errorf("%v: %w", err2, err)
		}
		var rollback []migrate.File
		switch MigrateFlags.Apply.TxMode {
		case txModeFile:
			rollback = append(rollback, file)
		case txModeAll:
			for _, f := range tx.files {
				if f.Version() > file.Version() {
					break
				}
				rollback = append(rollback, f)
			}
		}
		for _, f := range rollback {
			rev, err2 := tx.rrw.ReadRevision(tx.ctx, f.Version())
			if err2 != nil {
				err = fmt.Errorf("%v: %w", err2, err)
			}
			rev.Applied = 0
			if err2 := tx.rrw.WriteRevision(tx.ctx, rev); err2 != nil {
				err = fmt.Errorf("%v: %w", err2, err)
			}
		}
	}
	return err
}

// mayCommit may commit a transaction depending on the given transaction mode.
func (tx *tx) mayCommit() error {
	// Only commit if each file is wrapped in a transaction.
	if !MigrateFlags.Apply.DryRun && MigrateFlags.Apply.TxMode == txModeFile {
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

func executorOptions(l migrate.Logger) []migrate.ExecutorOption {
	opts := []migrate.ExecutorOption{migrate.WithLogger(l)}
	if MigrateFlags.Apply.AllowDirty {
		opts = append(opts, migrate.WithAllowDirty(true))
	}
	if v := MigrateFlags.Apply.BaselineVersion; v != "" {
		opts = append(opts, migrate.WithBaselineVersion(v))
	}
	if v := MigrateFlags.Apply.FromVersion; v != "" {
		opts = append(opts, migrate.WithFromVersion(v))
	}
	return opts
}

// CmdMigrateDiffRun is the command executed when running the CLI with 'migrate diff' args.
func CmdMigrateDiffRun(cmd *cobra.Command, args []string) error {
	// Open a dev driver.
	dev, err := sqlclient.Open(cmd.Context(), MigrateFlags.DevURL)
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
	dir, err := dir(false)
	if err != nil {
		return err
	}
	// Get a state reader for the desired state.
	desired, err := to(cmd.Context(), dev)
	if err != nil {
		return err
	}
	defer desired.Close()
	f, err := formatter()
	if err != nil {
		return err
	}
	opts := []migrate.PlannerOption{migrate.PlanFormat(f)}
	if dev.URL.Schema != "" {
		// Disable tables qualifier in schema-mode.
		opts = append(opts, migrate.PlanWithSchemaQualifier(MigrateFlags.Diff.Qualifier))
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

// CmdMigrateHashRun is the command executed when running the CLI with 'migrate hash' args.
func CmdMigrateHashRun(*cobra.Command, []string) error {
	dir, err := dir(false)
	if err != nil {
		return err
	}
	sum, err := dir.Checksum()
	if err != nil {
		return err
	}
	return migrate.WriteSumFile(dir, sum)
}

// CmdMigrateNewRun is the command executed when running the CLI with 'migrate new' args.
func CmdMigrateNewRun(_ *cobra.Command, args []string) error {
	dir, err := dir(false)
	if err != nil {
		return err
	}
	f, err := formatter()
	if err != nil {
		return err
	}
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	return migrate.NewPlanner(nil, dir, migrate.PlanFormat(f)).WritePlan(&migrate.Plan{Name: name})
}

// CmdMigrateStatusRun is the command executed when running the CLI with 'migrate status' args.
func CmdMigrateStatusRun(cmd *cobra.Command, _ []string) error {
	// Open the migration directory.
	dir, err := dir(false)
	if err != nil {
		return err
	}
	avail, err := dir.Files()
	if err != nil {
		return err
	}
	// Open a client to the database.
	client, err := sqlclient.Open(cmd.Context(), MigrateFlags.URL)
	if err != nil {
		return err
	}
	defer client.Close()
	if ok, err := revisionsTableExists(cmd.Context(), client); !ok || err != nil {
		if err != nil {
			return err
		}
		return statusPrint(cmd.OutOrStdout(), avail, avail, nil)
	}
	// Currently, only in DB revisions are supported.
	opts := []entmigrate.Option{entmigrate.WithSchema(MigrateFlags.RevisionSchema)}
	rrw, err := entmigrate.NewEntRevisions(client, opts...)
	if err != nil {
		return err
	}
	if err := rrw.Init(cmd.Context()); err != nil {
		return err
	}
	// Executor can give us insights on the revision state.
	ex, err := migrate.NewExecutor(client.Driver, dir, rrw)
	if err != nil {
		return err
	}
	pending, err := ex.Pending(cmd.Context())
	if err != nil && !errors.Is(err, migrate.ErrNoPendingFiles) {
		return err
	}
	revs, err := rrw.ReadRevisions(cmd.Context())
	if err != nil {
		return err
	}
	return statusPrint(cmd.OutOrStdout(), avail, pending, revs)
}

func statusPrint(out io.Writer, avail, pending []migrate.File, revs migrate.Revisions) (err error) {
	var (
		cur, next, state string
		applied          = avail[: len(avail)-len(pending) : len(avail)-len(pending)]
		partial          = len(revs) != 0 && revs[len(revs)-1].Applied < revs[len(revs)-1].Total
	)
	switch len(pending) {
	case len(avail):
		cur = "No version applied yet"
	default:
		cur = cyan(applied[len(applied)-1].Version())
		// If the last pending version is partially applied, tell so.
		if partial {
			cur += fmt.Sprintf(" (%d statements applied)", revs[len(revs)-1].Applied)
		}
	}
	if len(pending) == 0 {
		state = green("OK")
		next = "Already at latest version"
	} else {
		state = yellow("PENDING")
		next = cyan(pending[0].Version())
		if partial {
			next += fmt.Sprintf(" (%d statements left)", revs[len(revs)-1].Total-revs[len(revs)-1].Applied)
		}
	}
	exec := cyan(strconv.Itoa(len(applied)))
	if partial {
		exec += " + 1 partially"
	}
	c := cyan
	if len(pending) == 0 {
		c = green
	}
	fmt.Fprintf(out, "Migration Status: %s\n", state)
	fmt.Fprintf(out, "%s%s Current Version:  %s\n", indent2, dash, cur)
	fmt.Fprintf(out, "%s%s Next Version:     %s\n", indent2, dash, next)
	fmt.Fprintf(out, "%s%s Executed Files:   %s\n", indent2, dash, exec)
	fmt.Fprintf(out, "%s%s Pending Files:    %s", indent2, dash, c(strconv.Itoa(len(pending))))
	if partial {
		fmt.Fprintf(out, " (partially)")
	}
	fmt.Fprintf(out, "\n")
	return nil
}

// CmdMigrateValidateRun is the command executed when running the CLI with 'migrate validate' args.
func CmdMigrateValidateRun(cmd *cobra.Command, _ []string) error {
	// Validating the integrity is done by the PersistentPreRun already.
	if MigrateFlags.DevURL == "" {
		// If there is no --dev-url given do not attempt to replay the migration directory.
		return nil
	}
	// Open a client for the dev-db.
	dev, err := sqlclient.Open(cmd.Context(), MigrateFlags.DevURL)
	if err != nil {
		return err
	}
	defer dev.Close()
	// Currently, only our own migration file format is supported.
	dir, err := dir(false)
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

// CmdMigrateLintRun is the command executed when running the CLI with 'migrate lint' args.
func CmdMigrateLintRun(cmd *cobra.Command, _ []string) error {
	dev, err := sqlclient.Open(cmd.Context(), MigrateFlags.DevURL)
	if err != nil {
		return err
	}
	defer dev.Close()
	dir, err := dir(false)
	if err != nil {
		return err
	}
	var detect lint.ChangeDetector
	switch {
	case MigrateFlags.Lint.Latest == 0 && MigrateFlags.Lint.GitBase == "":
		return fmt.Errorf("--%s or --%s is required", migrateLintLatest, migrateLintGitBase)
	case MigrateFlags.Lint.Latest > 0 && MigrateFlags.Lint.GitBase != "":
		return fmt.Errorf("--%s and --%s are mutually exclusive", migrateLintLatest, migrateLintGitBase)
	case MigrateFlags.Lint.Latest > 0:
		detect = lint.LatestChanges(dir, int(MigrateFlags.Lint.Latest))
	case MigrateFlags.Lint.GitBase != "":
		detect, err = lint.NewGitChangeDetector(
			dir,
			lint.WithWorkDir(MigrateFlags.Lint.GitDir),
			lint.WithBase(MigrateFlags.Lint.GitBase),
			lint.WithMigrationsPath(dir.(interface{ Path() string }).Path()),
		)
		if err != nil {
			return err
		}
	}
	format := lint.DefaultTemplate
	if f := MigrateFlags.Lint.Format; f != "" {
		format, err = template.New("format").Funcs(lint.TemplateFuncs).Parse(f)
		if err != nil {
			return fmt.Errorf("parse log format: %w", err)
		}
	}
	az, err := sqlcheck.AnalyzerFor(dev.Name, nil)
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
		Analyzer: az,
	}
	err = r.Run(cmd.Context())
	// Print the error in case it was not printed before.
	cmd.SilenceErrors = errors.As(err, &lint.SilentError{})
	return err
}

const (
	txModeNone = "none"
	txModeAll  = "all"
	txModeFile = "file"
)

func printChecksumErr(out io.Writer) {
	fmt.Fprintf(out, `You have a checksum error in your migration directory.
This happens if you manually create or edit a migration file.
Please check your migration files and run

'atlas migrate hash --force'

to re-hash the contents and resolve the error

`)
}

func revisionsTableExists(ctx context.Context, c *sqlclient.Client) (bool, error) {
	// Connect to the given schema name.
	sc, err := sqlclient.Open(ctx, MigrateFlags.URL, sqlclient.OpenSchema(MigrateFlags.RevisionSchema))
	switch {
	case err != nil && !errors.Is(err, sqlclient.ErrUnsupported):
		return false, err
	case errors.Is(err, sqlclient.ErrUnsupported):
		// If the driver does not support changing the schema use the existing connection.
		sc = c
	case err == nil:
		// Connecting attempt to the schema was successful, make sure to close it.
		defer sc.Close()
	}
	// Inspect schema and check if the table does already exist.
	s, err := sc.InspectSchema(ctx, "", &schema.InspectOptions{Tables: []string{revision.Table}})
	switch {
	case err != nil && !schema.IsNotExistError(err):
		return false, err
	case schema.IsNotExistError(err):
		// Schema does not exist.
		return false, nil
	}
	if _, ok := s.Table(revision.Table); !ok {
		// Table does not exist.
		return false, nil
	}
	// Schema and Table are present.
	return true, nil
}

// dir returns a migrate.Dir to use as migration directory. For now only local directories are supported.
func dir(create bool) (migrate.Dir, error) {
	parts := strings.SplitN(MigrateFlags.DirURL, "://", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid dir url %q", MigrateFlags.DirURL)
	}
	if parts[0] != "file" {
		return nil, fmt.Errorf("unsupported driver %q", parts[0])
	}
	f := func() (migrate.Dir, error) { return migrate.NewLocalDir(parts[1]) }
	switch MigrateFlags.DirFormat {
	case formatAtlas:
	case formatGolangMigrate:
		f = func() (migrate.Dir, error) { return sqltool.NewGolangMigrateDir(parts[1]) }
	case formatGoose:
		f = func() (migrate.Dir, error) { return sqltool.NewGooseDir(parts[1]) }
	case formatFlyway:
		f = func() (migrate.Dir, error) { return sqltool.NewFlywayDir(parts[1]) }
	case formatLiquibase:
		f = func() (migrate.Dir, error) { return sqltool.NewLiquibaseDir(parts[1]) }
	case formatDBMate:
		f = func() (migrate.Dir, error) { return sqltool.NewDBMateDir(parts[1]) }
	default:
		return nil, fmt.Errorf("unknown dir format %q", MigrateFlags.DirFormat)
	}
	d, err := f()
	if create && errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(parts[1], 0755); err != nil {
			return nil, err
		}
		d, err = f()
	}
	return d, err
}

type target struct {
	migrate.StateReader        // desired state.
	io.Closer                  // optional close function.
	Schema              string // in case we work on a single schema.
}

// to returns a migrate.StateReader for the given to flag.
func to(ctx context.Context, dev *sqlclient.Client) (*target, error) {
	scheme, err := selectScheme(MigrateFlags.ToURLs)
	if err != nil {
		return nil, err
	}
	schemas := MigrateFlags.Schemas
	switch scheme {
	case "file": // hcl file
		realm := &schema.Realm{}
		paths := make([]string, 0, len(MigrateFlags.ToURLs))
		for _, u := range MigrateFlags.ToURLs {
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
		client, err := sqlclient.Open(ctx, MigrateFlags.ToURLs[0])
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
		return "", errors.New("at least one --to url is required")
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

func formatter() (migrate.Formatter, error) {
	switch MigrateFlags.DirFormat {
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
		return nil, fmt.Errorf("unknown format %q", MigrateFlags.DirFormat)
	}
}

const logFormatTTY = "tty"

// LogTTY is a migrate.Logger that pretty prints execution progress.
// If the connected out is not a tty, it will fall back to a non-colorful output.
type LogTTY struct {
	out         io.Writer
	start       time.Time
	fileStart   time.Time
	fileCounter int
	stmtCounter int
}

var (
	cyan         = color.CyanString
	green        = color.HiGreenString
	red          = color.HiRedString
	redBgWhiteFg = color.New(color.FgHiWhite, color.BgHiRed).SprintFunc()
	yellow       = color.YellowString
	dash         = yellow("--")
	arr          = cyan("->")
	indent2      = "  "
	indent4      = indent2 + indent2
)

// Log implements the migrate.Logger interface.
func (l *LogTTY) Log(e migrate.LogEntry) {
	switch e := e.(type) {
	case migrate.LogExecution:
		l.start = time.Now()
		fmt.Fprintf(l.out, "Migrating to version %v", cyan(e.To))
		if e.From != "" {
			fmt.Fprintf(l.out, " from %v", cyan(e.From))
		}
		fmt.Fprintf(l.out, " (%d migrations in total):\n", len(e.Files))
	case migrate.LogFile:
		l.fileCounter++
		if !l.fileStart.IsZero() {
			l.reportFileEnd()
		}
		l.fileStart = time.Now()
		fmt.Fprintf(l.out, "\n%s%v migrating version %v", indent2, dash, cyan(e.Version))
		if e.Skip > 0 {
			fmt.Fprintf(l.out, " (partially applied - skipping %s statements)", yellow("%d", e.Skip))
		}
		fmt.Fprint(l.out, "\n")
	case migrate.LogStmt:
		l.stmtCounter++
		fmt.Fprintf(l.out, "%s%v %s\n", indent4, arr, e.SQL)
	case migrate.LogDone:
		l.reportFileEnd()
		fmt.Fprintf(l.out, "\n%s%v\n", indent2, cyan(strings.Repeat("-", 25)))
		fmt.Fprintf(l.out, "%s%v %v\n", indent2, dash, time.Since(l.start))
		fmt.Fprintf(l.out, "%s%v %v migrations\n", indent2, dash, l.fileCounter)
		fmt.Fprintf(l.out, "%s%v %v sql statements\n", indent2, dash, l.stmtCounter)
	case migrate.LogError:
		fmt.Fprintf(l.out, "%s %s\n", indent4, redBgWhiteFg(e.Error.Error()))
		fmt.Fprintf(l.out, "\n%s%v\n", indent2, cyan(strings.Repeat("-", 25)))
		fmt.Fprintf(l.out, "%s%v %v\n", indent2, dash, time.Since(l.start))
		fmt.Fprintf(l.out, "%s%v %v migrations ok (%s)\n", indent2, dash, zero(l.fileCounter-1), red("1 with errors"))
		fmt.Fprintf(l.out, "%s%v %v sql statements ok (%s)\n", indent2, dash, zero(l.stmtCounter-1), red("1 with errors"))
		fmt.Fprintf(l.out, "\n%s\n%v\n\n", red("Error: Execution had errors:"), redBgWhiteFg(e.Error.Error()))
	default:
		fmt.Fprintf(l.out, "%v", e)
	}
}

func (l *LogTTY) reportFileEnd() {
	fmt.Fprintf(l.out, "%s%v ok (%v)\n", indent2, dash, yellow("%s", time.Since(l.fileStart)))
}

func zero(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func logFormat(out io.Writer) (migrate.Logger, error) {
	switch l := MigrateFlags.Apply.LogFormat; l {
	case logFormatTTY:
		return &LogTTY{out: out}, nil
	default:
		return nil, fmt.Errorf("unknown log-format %q", l)
	}
}

func migrateFlagsFromEnv(cmd *cobra.Command, _ []string) error {
	activeEnv, err := selectEnv(GlobalFlags.SelectedEnv)
	if err != nil {
		return err
	}
	if err := inputValsFromEnv(cmd); err != nil {
		return err
	}
	if err := maySetFlag(cmd, migrateFlagURL, activeEnv.URL); err != nil {
		return err
	}
	if err := maySetFlag(cmd, migrateFlagDevURL, activeEnv.DevURL); err != nil {
		return err
	}
	if err := maySetFlag(cmd, migrateFlagRevisionsSchema, activeEnv.Migration.RevisionsSchema); err != nil {
		return err
	}
	if err := maySetFlag(cmd, migrateFlagDirFormat, activeEnv.Migration.Format); err != nil {
		return err
	}
	// Transform "src" to a URL.
	srcs, err := activeEnv.Sources()
	if err != nil {
		return err
	}
	for i, s := range srcs {
		if s, err = filepath.Abs(s); err != nil {
			return fmt.Errorf("finding abs path to source: %q: %w", s, err)
		}
		srcs[i] = "file://" + s
	}
	if err := maySetFlag(cmd, migrateFlagTo, strings.Join(srcs, ",")); err != nil {
		return err
	}
	if s := "[" + strings.Join(activeEnv.Schemas, "") + "]"; len(activeEnv.Schemas) > 0 {
		if err := maySetFlag(cmd, migrateFlagSchema, s); err != nil {
			return err
		}
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
