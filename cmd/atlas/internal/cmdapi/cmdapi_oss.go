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
	"testing"
	"text/template"
	"time"

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
		unsupportedCommand("schema", "plan"),
		unsupportedCommand("schema", "push"),
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
	s := unsupportedMessage(cmd, sub)
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

// unsupportedMessage returns a message informing the user that the command
// or one of its options are not supported. For example:
//
// unsupportedMessage("migrate", "checkpoint")
// unsupportedMessage("schema", "apply --plan")
func unsupportedMessage(cmd, sub string) string {
	return fmt.Sprintf(
		`'atlas %s %s' is not supported by the community version.

To install the non-community version of Atlas, use the following command:

	curl -sSf https://atlasgo.sh | sh

Or, visit the website to see all installation options:

	https://atlasgo.io/docs#installation
`,
		cmd, sub,
	)
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
	if os.Getenv(envSkipUpgradeSuggestions) != "" || testing.Testing() {
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
	_ = cmdlog.WarnOnce(cmd.ErrOrStderr(), cmdlog.ColorCyan(s))
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
	if flags.dryRun {
		return errors.New("'--dry-run' is not supported in the community version")
	}
	ctx := cmd.Context()
	dev, err := sqlclient.Open(ctx, flags.devURL)
	if err != nil {
		return err
	}
	defer dev.Close()
	// Acquire a lock.
	unlock, err := dev.Lock(ctx, "atlas_migrate_diff", flags.lockTimeout)
	if err != nil {
		return fmt.Errorf("acquiring database lock: %w", err)
	}
	// If unlocking fails notify the user about it.
	defer func() { cobra.CheckErr(unlock()) }()
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
	diffOpts := diffOptions(cmd, env)
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
		vars:    env.Vars(),
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

// schemaApplyRunE is the community version of the 'atlas schema apply' command.
func schemaApplyRunE(cmd *cobra.Command, _ []string, flags *schemaApplyFlags) error {
	switch {
	case flags.edit:
		return AbortErrorf(unsupportedMessage("schema", "apply --edit"))
	case flags.planURL != "":
		return AbortErrorf(unsupportedMessage("schema", "apply --plan"))
	case GlobalFlags.SelectedEnv == "":
		env, err := selectEnv(cmd)
		if err != nil {
			return err
		}
		return schemaApplyRun(cmd, *flags, env)
	default:
		_, envs, err := EnvByName(cmd, GlobalFlags.SelectedEnv, GlobalFlags.Vars)
		if err != nil {
			return err
		}
		if len(envs) != 1 {
			return fmt.Errorf("multi-environment %q is not supported", GlobalFlags.SelectedEnv)
		}
		if err := setSchemaEnvFlags(cmd, envs[0]); err != nil {
			return err
		}
		return schemaApplyRun(cmd, *flags, envs[0])
	}
}

func schemaApplyRun(cmd *cobra.Command, flags schemaApplyFlags, env *Env) error {
	var (
		err    error
		ctx    = cmd.Context()
		dev    *sqlclient.Client
		format = cmdlog.SchemaPlanTemplate
	)
	if err = flags.check(env); err != nil {
		return err
	}
	if v := flags.logFormat; v != "" {
		if !flags.dryRun && !flags.autoApprove {
			return errors.New(`--log and --format can only be used with --dry-run or --auto-approve`)
		}
		if format, err = template.New("format").Funcs(cmdlog.ApplyTemplateFuncs).Parse(v); err != nil {
			return fmt.Errorf("parse log format: %w", err)
		}
	}
	if flags.devURL != "" {
		if dev, err = sqlclient.Open(ctx, flags.devURL); err != nil {
			return err
		}
		defer dev.Close()
	}
	from, err := stateReader(ctx, env, &stateReaderConfig{
		urls:    []string{flags.url},
		schemas: flags.schemas,
		exclude: flags.exclude,
	})
	if err != nil {
		return err
	}
	defer from.Close()
	client, ok := from.Closer.(*sqlclient.Client)
	if !ok {
		return errors.New("--url must be a database connection")
	}
	to, err := stateReader(ctx, env, &stateReaderConfig{
		urls:    flags.toURLs,
		dev:     dev,
		client:  client,
		schemas: flags.schemas,
		exclude: flags.exclude,
		vars:    env.Vars(),
	})
	if err != nil {
		return err
	}
	defer to.Close()
	diff, err := computeDiff(ctx, client, from, to, diffOptions(cmd, env)...)
	if err != nil {
		return err
	}
	maySuggestUpgrade(cmd)
	// Returning at this stage should
	// not trigger the help message.
	cmd.SilenceUsage = true
	switch changes := diff.changes; {
	case len(changes) == 0:
		return format.Execute(cmd.OutOrStdout(), &cmdlog.SchemaApply{})
	case flags.logFormat != "" && flags.autoApprove:
		var (
			applied int
			plan    *migrate.Plan
			cause   *cmdlog.StmtError
			out     = cmd.OutOrStdout()
		)
		if plan, err = client.PlanChanges(ctx, "", changes, planOptions(client)...); err != nil {
			return err
		}
		if err = applyChanges(ctx, client, changes, flags.txMode); err == nil {
			applied = len(plan.Changes)
		} else if i, ok := err.(interface{ Applied() int }); ok && i.Applied() < len(plan.Changes) {
			applied, cause = i.Applied(), &cmdlog.StmtError{Stmt: plan.Changes[i.Applied()].Cmd, Text: err.Error()}
		} else {
			cause = &cmdlog.StmtError{Text: err.Error()}
		}
		err1 := format.Execute(out, cmdlog.NewSchemaApply(ctx, cmdlog.NewEnv(client, nil), plan.Changes[:applied], plan.Changes[applied:], cause))
		return errors.Join(err, err1)
	default:
		switch err := summary(cmd, client, changes, format); {
		case err != nil:
			return err
		case flags.dryRun:
			return nil
		case flags.autoApprove:
			return applyChanges(ctx, client, changes, flags.txMode)
		default:
			return promptApply(cmd, flags, diff, client, dev)
		}
	}
}

// applySchemaClean is the community-version of the 'atlas schema clean' handler.
func applySchemaClean(cmd *cobra.Command, client *sqlclient.Client, drop []schema.Change, flags schemaCleanFlags) error {
	if flags.dryRun {
		return AbortErrorf(unsupportedMessage("schema", "clean --dry-run"))
	}
	if flags.logFormat != "" {
		return AbortErrorf(unsupportedMessage("schema", "clean --format"))
	}
	if len(drop) == 0 {
		cmd.Println("Nothing to drop")
		return nil
	}
	if err := summary(cmd, client, drop, cmdlog.SchemaPlanTemplate); err != nil {
		return err
	}
	if flags.autoApprove || promptUser(cmd) {
		if err := client.ApplyChanges(cmd.Context(), drop); err != nil {
			return err
		}
	}
	return nil
}

func summary(cmd *cobra.Command, c *sqlclient.Client, changes []schema.Change, t *template.Template) error {
	p, err := c.PlanChanges(cmd.Context(), "", changes, planOptions(c)...)
	if err != nil {
		return err
	}
	return t.Execute(
		cmd.OutOrStdout(),
		cmdlog.NewSchemaPlan(cmd.Context(), cmdlog.NewEnv(c, nil), p.Changes, nil),
	)
}

func promptApply(cmd *cobra.Command, flags schemaApplyFlags, diff *diff, client, _ *sqlclient.Client) error {
	if !flags.dryRun && (flags.autoApprove || promptUser(cmd)) {
		return applyChanges(cmd.Context(), client, diff.changes, flags.txMode)
	}
	return nil
}

func maySetLoginContext(*cobra.Command, *Project) error {
	return nil
}

func setEnvs(context.Context, []*Env) {}

// specOptions are the options for the schema spec.
var specOptions []schemahcl.Option

// diffOptions returns environment-aware diff options.
func diffOptions(_ *cobra.Command, env *Env) []schema.DiffOption {
	return append(env.DiffOptions(), schema.DiffNormalized())
}

// openClient allows opening environment-aware clients.
func (*Env) openClient(ctx context.Context, u string) (*sqlclient.Client, error) {
	return sqlclient.Open(ctx, u)
}
