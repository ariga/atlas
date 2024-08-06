// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package cmdapi

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"text/template"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/cloudapi"
	"ariga.io/atlas/cmd/atlas/internal/cmdext"
	"ariga.io/atlas/cmd/atlas/internal/cmdlog"
	"ariga.io/atlas/cmd/atlas/internal/cmdstate"
	cmdmigrate "ariga.io/atlas/cmd/atlas/internal/migrate"
	"ariga.io/atlas/cmd/atlas/internal/migratelint"
	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/spf13/cobra"
)

func init() {
	schemaCmd := schemaCmd()
	schemaCmd.AddCommand(
		schemaApplyCmd(),
		schemaCleanCmd(),
		schemaDiffCmd(),
		schemaFmtCmd(),
		schemaInspectCmd(),
		unsupportedCommand("schema", "test"),
	)
	Root.AddCommand(schemaCmd)
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
		unsupportedCommand("migrate", "checkpoint"),
		unsupportedCommand("migrate", "down"),
		unsupportedCommand("migrate", "rebase"),
		unsupportedCommand("migrate", "rm"),
		unsupportedCommand("migrate", "edit"),
		unsupportedCommand("migrate", "push"),
		unsupportedCommand("migrate", "test"),
	)
	Root.AddCommand(migrateCmd)
}

// unsupportedCommand create a stub command that reports
// the command is not supported by this build.
func unsupportedCommand(cmd, sub string) *cobra.Command {
	s := fmt.Sprintf(
		`'atlas %s %s' is not supported by the community version.

To install the non-community version of Atlas, use the following command:

	curl -sSf https://atlasgo.sh | sh

Or, visit the website to see all installation options:

	https://atlasgo.io/docs#installation
`,
		cmd, sub,
	)
	c := &cobra.Command{
		Hidden: true,
		Use:    fmt.Sprintf("%s is not supported by this build", sub),
		Short:  s,
		Long:   s,
		RunE: RunE(func(*cobra.Command, []string) error {
			return AbortErrorf(s)
		}),
	}
	c.SetHelpTemplate(s + "\n")
	return c
}

type (
	// Project represents an atlas.hcl project config file.
	Project struct {
		Envs  []*Env `spec:"env"`  // List of environments
		Lint  *Lint  `spec:"lint"` // Optional global lint policy
		Diff  *Diff  `spec:"diff"` // Optional global diff policy
		Test  *Test  `spec:"test"` // Optional test configuration
		cloud *cmdext.AtlasConfig
	}
)

const (
	envSkipUpgradeSuggestions = "ATLAS_NO_UPGRADE_SUGGESTIONS"
	oneWeek                   = 7 * 24 * time.Hour
)

// maySuggestUpgrade informs the user about the limitations of the community edition to stderr
// at most once a week. The user can disable this message by setting the ATLAS_NO_UPGRADE_SUGGESTIONS
// environment variable.
func maySuggestUpgrade(cmd *cobra.Command) {
	if os.Getenv(envSkipUpgradeSuggestions) != "" {
		return
	}
	state := cmdstate.File[LocalState]{Name: localStateFile}
	prev, err := state.Read()
	if err != nil {
		return
	}
	if time.Since(prev.UpgradeSuggested) < oneWeek {
		return
	}
	s := `Notice: This Atlas edition lacks support for features such as checkpoints,
testing, down migrations, and more. Additionally, advanced database objects such as views, 
triggers, and stored procedures are not supported. To read more: https://atlasgo.io/community-edition

To install the non-community version of Atlas, use the following command:

	curl -sSf https://atlasgo.sh | sh

Or, visit the website to see all installation options:

	https://atlasgo.io/docs#installation

`
	fmt.Fprintf(cmd.ErrOrStderr(), cmdlog.ColorCyan(s))
	prev.UpgradeSuggested = time.Now()
	_ = state.Write(prev)
}

// migrateLintSetFlags allows setting extra flags for the 'migrate lint' command.
func migrateLintSetFlags(*cobra.Command, *migrateLintFlags) {}

// migrateLintRun is the run command for 'migrate lint'.
func migrateLintRun(cmd *cobra.Command, _ []string, flags migrateLintFlags, env *Env) error {
	dev, err := sqlclient.Open(cmd.Context(), flags.devURL)
	if err != nil {
		return err
	}
	defer dev.Close()
	dir, err := cmdmigrate.Dir(cmd.Context(), flags.dirURL, false)
	if err != nil {
		return err
	}
	var detect migratelint.ChangeDetector
	switch {
	case flags.latest == 0 && flags.gitBase == "":
		return fmt.Errorf("--%s or --%s is required", flagLatest, flagGitBase)
	case flags.latest > 0 && flags.gitBase != "":
		return fmt.Errorf("--%s and --%s are mutually exclusive", flagLatest, flagGitBase)
	case flags.latest > 0:
		detect = migratelint.LatestChanges(dir, int(flags.latest))
	case flags.gitBase != "":
		detect, err = migratelint.NewGitChangeDetector(
			dir,
			migratelint.WithWorkDir(flags.gitDir),
			migratelint.WithBase(flags.gitBase),
			migratelint.WithMigrationsPath(dir.(interface{ Path() string }).Path()),
		)
		if err != nil {
			return err
		}
	}
	format := migratelint.DefaultTemplate
	if f := flags.logFormat; f != "" {
		format, err = template.New("format").Funcs(migratelint.TemplateFuncs).Parse(f)
		if err != nil {
			return fmt.Errorf("parse format: %w", err)
		}
	}
	az, err := sqlcheck.AnalyzerFor(dev.Name, env.Lint.Remain())
	if err != nil {
		return err
	}
	r := &migratelint.Runner{
		Dev:            dev,
		Dir:            dir,
		ChangeDetector: detect,
		ReportWriter: &migratelint.TemplateWriter{
			T: format,
			W: cmd.OutOrStdout(),
		},
		Analyzers: az,
	}
	err = r.Run(cmd.Context())
	// Print the error in case it was not printed before.
	cmd.SilenceErrors = errors.As(err, &migratelint.SilentError{})
	cmd.SilenceUsage = cmd.SilenceErrors
	return err
}

func migrateDiffRun(cmd *cobra.Command, args []string, flags migrateDiffFlags, env *Env) error {
	ctx := cmd.Context()
	dev, err := sqlclient.Open(ctx, flags.devURL)
	if err != nil {
		return err
	}
	defer dev.Close()
	// Acquire a lock.
	if l, ok := dev.Driver.(schema.Locker); ok {
		unlock, err := l.Lock(ctx, "atlas_migrate_diff", flags.lockTimeout)
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
	dir, err := cmdmigrate.DirURL(ctx, u, false)
	if err != nil {
		return err
	}
	if flags.edit {
		dir = &editDir{dir}
	}
	var name, indent string
	if len(args) > 0 {
		name = args[0]
	}
	f, err := cmdmigrate.Formatter(u)
	if err != nil {
		return err
	}
	if f, indent, err = mayIndent(u, f, flags.format); err != nil {
		return err
	}
	diffOpts := append(diffOptions(cmd, env), schema.DiffNormalized())
	// If there is a state-loader that requires a custom
	// 'migrate diff' handling, offload it the work.
	if d, ok := cmdext.States.Differ(flags.desiredURLs); ok {
		err := d.MigrateDiff(ctx, &cmdext.MigrateDiffOptions{
			To:      flags.desiredURLs,
			Name:    name,
			Indent:  indent,
			Dir:     dir,
			Dev:     dev,
			Options: diffOpts,
		})
		return maskNoPlan(cmd, err)
	}
	// Get a state reader for the desired state.
	desired, err := stateReader(ctx, env, &stateReaderConfig{
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
	opts := []migrate.PlannerOption{
		migrate.PlanFormat(f),
		migrate.PlanWithIndent(indent),
		migrate.PlanWithDiffOptions(diffOpts...),
	}
	if dev.URL.Schema != "" {
		// Disable tables qualifier in schema-mode.
		opts = append(opts, migrate.PlanWithSchemaQualifier(flags.qualifier))
	}
	// Plan the changes and create a new migration file.
	pl := migrate.NewPlanner(dev.Driver, dir, opts...)
	plan, err := func() (*migrate.Plan, error) {
		if dev.URL.Schema != "" {
			return pl.PlanSchema(ctx, name, desired.StateReader)
		}
		return pl.Plan(ctx, name, desired.StateReader)
	}()
	var cerr *migrate.NotCleanError
	switch {
	case errors.As(err, &cerr) && dev.URL.Schema == "" && desired.Schema != "":
		return fmt.Errorf("dev database is not clean (%s). Add a schema to the URL to limit the scope of the connection", cerr.Reason)
	case err != nil:
		return maskNoPlan(cmd, err)
	default:
		return pl.WritePlan(plan)
	}
}

func promptApply(cmd *cobra.Command, flags schemaApplyFlags, diff *diff, client, _ *sqlclient.Client) error {
	if !flags.dryRun && (flags.autoApprove || promptUser(cmd)) {
		return applyChanges(cmd.Context(), client, diff.changes, flags.txMode)
	}
	return nil
}

// withTokenContext allows attaching token to the context.
func withTokenContext(ctx context.Context, _ string, _ *cloudapi.Client) (context.Context, error) {
	return ctx, nil // unimplemented.
}

func setEnvs(context.Context, []*Env) {}

// specOptions are the options for the schema spec.
var specOptions []schemahcl.Option

// diffOptions returns environment-aware diff options.
func diffOptions(_ *cobra.Command, env *Env) []schema.DiffOption {
	return env.DiffOptions()
}

// openClient allows opening environment-aware clients.
func (*Env) openClient(ctx context.Context, u string) (*sqlclient.Client, error) {
	return sqlclient.Open(ctx, u)
}
