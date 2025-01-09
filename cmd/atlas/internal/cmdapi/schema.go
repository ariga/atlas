// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/cmdext"
	"ariga.io/atlas/cmd/atlas/internal/cmdlog"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/1lann/promptui"
	"github.com/chzyer/readline"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/spf13/cobra"
)

// schemaCmd represents the subcommand 'atlas schema'.
func schemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Work with atlas schemas.",
		Long:  "The `atlas schema` command groups subcommands working with declarative Atlas schemas.",
	}
	addGlobalFlags(cmd.PersistentFlags())
	return cmd
}

type schemaApplyFlags struct {
	url         string        // URL of database to apply the changes on.
	devURL      string        // URL of the dev database.
	paths       []string      // Paths to HCL files.
	toURLs      []string      // URLs of the desired state.
	planURL     string        // URL to a pre-planned migration.
	schemas     []string      // Schemas to take into account when diffing.
	exclude     []string      // List of glob patterns used to filter resources from applying (see schema.InspectOptions).
	dryRun      bool          // Only show SQL on screen instead of applying it.
	edit        bool          // Open the generated SQL in an editor.
	autoApprove bool          // Don't prompt for approval before applying SQL.
	logFormat   string        // Log format.
	txMode      string        // (none, file)
	lockTimeout time.Duration // Lock timeout.
}

// check that the flags are valid before running the command.
func (f *schemaApplyFlags) check(env *Env) error {
	switch {
	case f.url == "":
		return errors.New(`required flag(s) "url" not set`)
	case len(f.paths) == 0 && len(f.toURLs) == 0:
		if f.planURL != "" {
			return errors.New(`the flag "to" is required to verify the provided plan`)
		}
		return errors.New(`one of flag(s) "file" or "to" is required`)
	case f.txMode != txModeNone && f.txMode != txModeFile:
		return fmt.Errorf("unknown tx-mode %q", f.txMode)
	case f.autoApprove && env.Lint.Review != "":
		return fmt.Errorf("auto-approve is not allowed when a lint policy is set to %q", env.Lint.Review)
	case f.edit && f.devURL == "":
		return errors.New("--edit requires a connection to the dev-database (provided by --dev-url)")
	}
	// If the old -f flag is given convert them to the URL format. If both are given,
	// cobra would throw an error since they are marked as mutually exclusive.
	if len(f.toURLs) == 0 {
		f.toURLs = fixFileURLs(f.paths)
	}
	return nil
}

// schemaApplyCmd represents the 'atlas schema apply' subcommand.
func schemaApplyCmd() *cobra.Command {
	var (
		flags schemaApplyFlags
		cmd   = &cobra.Command{
			Use:   "apply",
			Short: "Apply an atlas schema to a target database.",
			// Use 80-columns as max width.
			Long: `'atlas schema apply' plans and executes a database migration to bring a given
database to the state described in the provided Atlas schema. Before running the
migration, Atlas will print the migration plan and prompt the user for approval.

The schema is provided by one or more URLs (to a HCL file or 
directory, database or migration directory) using the "--to, -t" flag:
  atlas schema apply -u URL --to "file://file1.hcl" --to "file://file2.hcl"
  atlas schema apply -u URL --to "file://schema/" --to "file://override.hcl"

As a convenience, schema URLs may also be provided via an environment definition in
the project file (see: https://atlasgo.io/cli/projects).

If run with the "--dry-run" flag, atlas will exit after printing out the planned
migration.`,
			Example: `  atlas schema apply -u "mysql://user:pass@localhost/dbname" --to "file://atlas.hcl"
  atlas schema apply -u "mysql://localhost" --to "file://schema.sql" --dev-url "docker://mysql/8/dev"
  atlas schema apply --env local --dev-url "docker://postgres/15/dev?search_path=public" --dry-run
  atlas schema apply -u "sqlite://file.db" --to "file://schema.sql" --dev-url "sqlite://dev?mode=memory"`,
			RunE: RunE(func(cmd *cobra.Command, args []string) error {
				return schemaApplyRunE(cmd, args, &flags)
			}),
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURL(cmd.Flags(), &flags.url)
	addFlagToURLs(cmd.Flags(), &flags.toURLs)
	addFlagExclude(cmd.Flags(), &flags.exclude)
	addFlagSchemas(cmd.Flags(), &flags.schemas)
	addFlagDevURL(cmd.Flags(), &flags.devURL)
	addFlagDryRun(cmd.Flags(), &flags.dryRun)
	addFlagAutoApprove(cmd.Flags(), &flags.autoApprove)
	addFlagLog(cmd.Flags(), &flags.logFormat)
	addFlagFormat(cmd.Flags(), &flags.logFormat)
	cmd.Flags().StringVarP(&flags.txMode, flagTxMode, "", txModeFile, "set transaction mode [none, file]")
	cmd.Flags().StringVarP(&flags.planURL, flagPlan, "", "", "URL to a pre-planned migration (e.g., atlas://repo/plans/name)")
	cmd.Flags().BoolVarP(&flags.edit, flagEdit, "", false, "open the generated SQL in an editor")
	addFlagLockTimeout(cmd.Flags(), &flags.lockTimeout)
	// Hidden support for the deprecated -f flag.
	cmd.Flags().StringSliceVarP(&flags.paths, flagFile, "f", nil, "[paths...] file or directory containing HCL or SQL files")
	cobra.CheckErr(cmd.Flags().MarkHidden(flagFile))
	cmd.MarkFlagsMutuallyExclusive(flagFile, flagTo)
	cmd.MarkFlagsMutuallyExclusive(flagLog, flagFormat)
	cmd.MarkFlagsMutuallyExclusive(flagEdit, flagPlan)
	cmd.MarkFlagsMutuallyExclusive(flagDryRun, flagAutoApprove)
	return cmd
}

func applyChanges(ctx context.Context, client *sqlclient.Client, changes []schema.Change, txMode string) error {
	opts := planOptions(client)
	if txMode == txModeNone {
		return client.ApplyChanges(ctx, changes, opts...)
	}
	tx, err := client.Tx(ctx, nil)
	if err != nil {
		return err
	}
	if err := tx.ApplyChanges(ctx, changes, opts...); err != nil {
		// Rollback on error but the underlying error is still
		// returned to make type-assertion in schemaApplyRun pass.
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// planOptions returns the default options for planning declarative changes.
func planOptions(c *sqlclient.Client) []migrate.PlanOption {
	opts := []migrate.PlanOption{
		func(opts *migrate.PlanOptions) {
			opts.Indent = "  "
		},
	}
	// In case the scope is a database schema, generate
	// the plan without the schema qualifier.
	if c.URL.Schema != "" {
		opts = append(opts, func(opt *migrate.PlanOptions) {
			opt.SchemaQualifier = new(string)
		})
	}
	return opts
}

type schemaCleanFlags struct {
	url         string // URL of database to apply the changes on.
	autoApprove bool   // Don't prompt for approval before applying SQL.
	logFormat   string // Log format.
	dryRun      bool   // Only show SQL on screen instead of applying it.
}

// schemaCleanCmd represents the 'atlas schema clean' subcommand.
func schemaCleanCmd() *cobra.Command {
	var (
		flags schemaCleanFlags
		cmd   = &cobra.Command{
			Use:   "clean [flags]",
			Short: "Removes all objects from the connected database.",
			Long: `'atlas schema clean' drops all objects in the connected database and leaves it in an empty state.
As a safety feature, 'atlas schema clean' will ask for confirmation before attempting to execute any SQL.`,
			Example: `  atlas schema clean -u "mysql://user:pass@localhost:3306/dbname"
  atlas schema clean -u "mysql://user:pass@localhost:3306/"`,
			PreRunE: func(cmd *cobra.Command, _ []string) error {
				return schemaFlagsFromConfig(cmd)
			},
			RunE: RunE(func(cmd *cobra.Command, args []string) error {
				return schemaCleanRun(cmd, args, flags)
			}),
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURL(cmd.Flags(), &flags.url)
	addFlagDryRun(cmd.Flags(), &flags.dryRun)
	addFlagFormat(cmd.Flags(), &flags.logFormat)
	addFlagAutoApprove(cmd.Flags(), &flags.autoApprove)
	cobra.CheckErr(cmd.MarkFlagRequired(flagURL))
	return cmd
}

func schemaCleanRun(cmd *cobra.Command, _ []string, flags schemaCleanFlags) error {
	c, err := sqlclient.Open(cmd.Context(), flags.url)
	if err != nil {
		return err
	}
	defer c.Close()
	var drop []schema.Change
	// If the connection is bound to a schema, only drop the resources inside the schema.
	switch c.URL.Schema {
	case "":
		r, err := c.InspectRealm(cmd.Context(), nil)
		if err != nil {
			return err
		}
		drop, err = c.RealmDiff(r, schema.NewRealm())
		if err != nil {
			return err
		}
	default:
		s, err := c.InspectSchema(cmd.Context(), c.URL.Schema, nil)
		if err != nil {
			return err
		}
		drop, err = c.SchemaDiff(s, schema.New(s.Name))
		if err != nil {
			return err
		}
	}
	return applySchemaClean(cmd, c, drop, flags)
}

type schemaDiffFlags struct {
	fromURL []string
	toURL   []string
	devURL  string
	schemas []string
	exclude []string
	format  string
}

// schemaDiffCmd represents the 'atlas schema diff' subcommand.
func schemaDiffCmd() *cobra.Command {
	cmd, _ := schemaDiffCmdWithFlags()
	return cmd
}

func schemaDiffCmdWithFlags() (*cobra.Command, *schemaDiffFlags) {
	var (
		flags schemaDiffFlags
		cmd   = &cobra.Command{
			Use:   "diff",
			Short: "Calculate and print the diff between two schemas.",
			Long: `'atlas schema diff' reads the state of two given schema definitions, 
calculates the difference in their schemas, and prints a plan of
SQL statements to migrate the "from" database to the schema of the "to" database.
The database states can be read from a connected database, an HCL project or a migration directory.`,
			Example: `  atlas schema diff --from "mysql://user:pass@localhost:3306/test" --to "file://schema.hcl"
  atlas schema diff --from "mysql://user:pass@localhost:3306" --to "file://schema_1.hcl" --to "file://schema_2.hcl"
  atlas schema diff --from "mysql://user:pass@localhost:3306" --to "file://migrations" --format '{{ sql . "  " }}'`,
			PreRunE: func(cmd *cobra.Command, _ []string) error {
				return schemaFlagsFromConfig(cmd)
			},
			RunE: RunE(func(cmd *cobra.Command, args []string) error {
				env, err := selectEnv(cmd)
				if err != nil {
					return err
				}
				return schemaDiffRun(cmd, args, flags, env)
			}),
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURLs(cmd.Flags(), &flags.fromURL, flagFrom, flagFromShort)
	addFlagToURLs(cmd.Flags(), &flags.toURL)
	addFlagDevURL(cmd.Flags(), &flags.devURL)
	addFlagSchemas(cmd.Flags(), &flags.schemas)
	addFlagExclude(cmd.Flags(), &flags.exclude)
	addFlagFormat(cmd.Flags(), &flags.format)
	cobra.CheckErr(cmd.MarkFlagRequired(flagFrom))
	cobra.CheckErr(cmd.MarkFlagRequired(flagTo))
	return cmd, &flags
}

func schemaDiffRun(cmd *cobra.Command, _ []string, flags schemaDiffFlags, env *Env) error {
	var (
		ctx = cmd.Context()
		c   *sqlclient.Client
	)
	// We need a driver for diffing and planning. If given, dev database has precedence.
	if flags.devURL != "" {
		var err error
		c, err = sqlclient.Open(ctx, flags.devURL)
		if err != nil {
			return err
		}
		defer c.Close()
	}
	from, err := stateReader(ctx, env, &stateReaderConfig{
		urls:    flags.fromURL,
		dev:     c,
		vars:    env.Vars(),
		schemas: flags.schemas,
		exclude: flags.exclude,
	})
	if err != nil {
		return err
	}
	defer from.Close()
	to, err := stateReader(ctx, env, &stateReaderConfig{
		urls:    flags.toURL,
		dev:     c,
		vars:    env.Vars(),
		schemas: flags.schemas,
		exclude: flags.exclude,
	})
	if err != nil {
		return err
	}
	defer to.Close()
	if c == nil {
		// If not both states are provided by a database connection, the call to state-reader would have returned
		// an error already. If we land in this case, we can assume both states are database connections.
		c = to.Closer.(*sqlclient.Client)
	}
	format := cmdlog.SchemaDiffTemplate
	if v := flags.format; v != "" {
		if format, err = template.New("format").Funcs(cmdlog.SchemaDiffFuncs).Parse(v); err != nil {
			return fmt.Errorf("parse log format: %w", err)
		}
	}
	diff, err := computeDiff(ctx, c, from, to, diffOptions(cmd, env)...)
	if err != nil {
		return err
	}
	maySuggestUpgrade(cmd)
	return format.Execute(cmd.OutOrStdout(),
		cmdlog.NewSchemaDiff(ctx, c, diff.from, diff.to, diff.changes),
	)
}

type schemaInspectFlags struct {
	url       string   // URL of resource to inspect.
	devURL    string   // URL of the dev database.
	logFormat string   // Format of the log output.
	schemas   []string // Schemas to take into account when diffing.
	exclude   []string // List of glob patterns used to filter resources from applying (see schema.InspectOptions).
}

// schemaInspectCmd represents the 'atlas schema inspect' subcommand.
func schemaInspectCmd() *cobra.Command {
	cmd, _ := schemaInspectCmdWithFlags()
	return cmd
}

func schemaInspectCmdWithFlags() (*cobra.Command, *schemaInspectFlags) {
	var (
		env   *Env
		flags schemaInspectFlags
		cmd   = &cobra.Command{
			Use:   "inspect",
			Short: "Inspect a database and print its schema in Atlas DDL syntax.",
			Long: `'atlas schema inspect' connects to the given database and inspects its schema.
It then prints to the screen the schema of that database in Atlas DDL syntax. This output can be
saved to a file, commonly by redirecting the output to a file named with a ".hcl" suffix:

  atlas schema inspect -u "mysql://user:pass@localhost:3306/dbname" > schema.hcl

This file can then be edited and used with the` + " `atlas schema apply` " + `command to plan
and execute schema migrations against the given database. In cases where users wish to inspect
all multiple schemas in a given database (for instance a MySQL server may contain multiple named
databases), omit the relevant part from the url, e.g. "mysql://user:pass@localhost:3306/".
To select specific schemas from the databases, users may use the "--schema" (or "-s" shorthand)
flag.
	`,
			Example: `  atlas schema inspect -u "mysql://user:pass@localhost:3306/dbname"
  atlas schema inspect -u "mariadb://user:pass@localhost:3306/" --schema=schemaA,schemaB -s schemaC
  atlas schema inspect --url "postgres://user:pass@host:port/dbname?sslmode=disable"
  atlas schema inspect -u "sqlite://file:ex1.db?_fk=1"`,
			PreRunE: RunE(func(cmd *cobra.Command, args []string) (err error) {
				if env, err = selectEnv(cmd); err != nil {
					return err
				}
				return setSchemaEnvFlags(cmd, env)
			}),
			RunE: RunE(func(cmd *cobra.Command, args []string) error {
				return schemaInspectRun(cmd, args, flags, env)
			}),
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURL(cmd.Flags(), &flags.url)
	addFlagDevURL(cmd.Flags(), &flags.devURL)
	addFlagSchemas(cmd.Flags(), &flags.schemas)
	addFlagExclude(cmd.Flags(), &flags.exclude)
	addFlagLog(cmd.Flags(), &flags.logFormat)
	addFlagFormat(cmd.Flags(), &flags.logFormat)
	cobra.CheckErr(cmd.MarkFlagRequired(flagURL))
	cmd.MarkFlagsMutuallyExclusive(flagLog, flagFormat)
	return cmd, &flags
}

func schemaInspectRun(cmd *cobra.Command, _ []string, flags schemaInspectFlags, env *Env) error {
	var (
		ctx = cmd.Context()
		dev *sqlclient.Client
	)
	useDev, err := readerUseDev(env, flags.url)
	if err != nil {
		return err
	}
	if flags.devURL != "" && useDev {
		if dev, err = sqlclient.Open(ctx, flags.devURL); err != nil {
			return err
		}
		defer dev.Close()
	}
	r, err := stateReader(ctx, env, &stateReaderConfig{
		urls:    []string{flags.url},
		dev:     dev,
		vars:    env.Vars(),
		schemas: flags.schemas,
		exclude: flags.exclude,
	})
	if err != nil {
		return err
	}
	defer r.Close()
	client, ok := r.Closer.(*sqlclient.Client)
	if !ok && dev != nil {
		client = dev
	}
	format := cmdlog.SchemaInspectTemplate
	if v := flags.logFormat; v != "" {
		if format, err = template.New("format").Funcs(cmdlog.InspectTemplateFuncs).Parse(v); err != nil {
			return fmt.Errorf("parse log format: %w", err)
		}
	}
	s, err := r.ReadState(ctx)
	if err != nil {
		return err
	}
	i := cmdlog.NewSchemaInspect(ctx, client, s)
	i.TargetURL = flags.url
	return format.Execute(cmd.OutOrStdout(), i)
}

// schemaFmtCmd represents the 'atlas schema fmt' subcommand.
func schemaFmtCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fmt [path ...]",
		Short: "Formats Atlas HCL files",
		Long: `'atlas schema fmt' formats all ".hcl" files under the given paths using
canonical HCL layout style as defined by the github.com/hashicorp/hcl/v2/hclwrite package.
Unless stated otherwise, the fmt command will use the current directory.

After running, the command will print the names of the files it has formatted. If all
files in the directory are formatted, no input will be printed out.
`,
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			return schemaFmtRun(cmd, args)
		}),
	}
	return cmd
}

func schemaFmtRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = append(args, "./")
	}
	for _, path := range args {
		tasks, err := tasks(path)
		if err != nil {
			return err
		}
		for _, task := range tasks {
			changed, err := fmtFile(task)
			if err != nil {
				return err
			}
			if changed {
				cmd.Println(task.path)
			}
		}
	}
	return nil
}

// selectEnv returns the Env from the current project file based on the selected
// argument. If selected is "", or no project file exists in the current directory
// a zero-value Env is returned.
func selectEnv(cmd *cobra.Command) (*Env, error) {
	switch name := GlobalFlags.SelectedEnv; {
	// A config file was passed without an env.
	case name == "" && cmd.Flags().Changed(flagConfig):
		p, envs, err := EnvByName(cmd, name, GlobalFlags.Vars)
		if err != nil {
			return nil, err
		}
		if len(envs) != 0 {
			return nil, fmt.Errorf("unexpected number of envs found: %d", len(envs))
		}
		return &Env{Lint: p.Lint, Diff: p.Diff, Migration: &Migration{}, config: p}, nil
	// No config nor env was passed.
	case name == "":
		return &Env{Lint: &Lint{}, Migration: &Migration{}}, nil
	// Env was passed.
	default:
		_, envs, err := EnvByName(cmd, name, GlobalFlags.Vars)
		if err != nil {
			return nil, err
		}
		if len(envs) > 1 {
			return nil, fmt.Errorf("multiple envs found for %q", name)
		}
		return envs[0], nil
	}
}

func schemaFlagsFromConfig(cmd *cobra.Command) error {
	env, err := selectEnv(cmd)
	if err != nil {
		return err
	}
	return setSchemaEnvFlags(cmd, env)
}

func setSchemaEnvFlags(cmd *cobra.Command, env *Env) error {
	if err := maySetFlag(cmd, flagDevURL, env.DevURL); err != nil {
		return err
	}
	srcs, err := env.Sources()
	if err != nil {
		return err
	}
	srcs = fixFileURLs(srcs)
	if err := maySetFlag(cmd, flagFile, strings.Join(srcs, ",")); err != nil {
		return err
	}
	if err := maySetFlag(cmd, flagSchema, strings.Join(env.Schemas, ",")); err != nil {
		return err
	}
	if err := maySetFlag(cmd, flagExclude, strings.Join(env.Exclude, ",")); err != nil {
		return err
	}
	switch cmd.Name() {
	case "clean":
		if err := maySetFlag(cmd, flagURL, env.URL); err != nil {
			return err
		}
		if err := maySetFlag(cmd, flagFormat, env.Format.Schema.Clean); err != nil {
			return err
		}
	case "inspect":
		if err := maySetFlag(cmd, flagURL, env.URL); err != nil {
			return err
		}
		if err := maySetFlag(cmd, flagFormat, env.Format.Schema.Inspect); err != nil {
			return err
		}
	case "apply":
		if err := maySetFlag(cmd, flagURL, env.URL); err != nil {
			return err
		}
		if err := maySetFlag(cmd, flagFormat, env.Format.Schema.Apply); err != nil {
			return err
		}
	case "diff":
		if err := maySetFlag(cmd, flagFormat, env.Format.Schema.Diff); err != nil {
			return err
		}
	case "push":
		if err := maySetFlag(cmd, flagURL, strings.Join(srcs, ",")); err != nil {
			return err
		}
		if err := maySetFlag(cmd, flagFormat, env.Format.Schema.Push); err != nil {
			return err
		}
	case "test":
		// Give the "src" precedence over the "url" argument.
		if len(srcs) > 0 {
			if err := maySetFlag(cmd, flagURL, strings.Join(srcs, ",")); err != nil {
				return err
			}
		} else if err := maySetFlag(cmd, flagURL, env.URL); err != nil {
			return err
		}
	}
	return nil
}

// diff holds the changes between two realms.
type diff struct {
	from, to *schema.Realm
	changes  []schema.Change
}

func computeDiff(ctx context.Context, differ *sqlclient.Client, from, to *cmdext.StateReadCloser, opts ...schema.DiffOption) (*diff, error) {
	current, err := from.ReadState(ctx)
	if err != nil {
		return nil, err
	}
	desired, err := to.ReadState(ctx)
	if err != nil {
		return nil, err
	}
	var changes []schema.Change
	switch {
	// In case an HCL file is compared against a specific database schema (not a realm).
	case to.HCL && len(desired.Schemas) == 1 && from.Schema != "" && desired.Schemas[0].Name != from.Schema:
		return nil, fmt.Errorf("mismatched HCL and database schemas: %q <> %q", from.Schema, desired.Schemas[0].Name)
	// Compare realm if the desired state is an HCL file or both connections are not bound to a schema.
	case from.HCL, to.HCL, from.Schema == "" && to.Schema == "":
		changes, err = differ.RealmDiff(current, desired, opts...)
		if err != nil {
			return nil, err
		}
	case from.Schema == "" && to.Schema != "":
		return nil, fmt.Errorf("cannot diff a schema %q with a database connection. See: https://atlasgo.io/url", to.Schema)
	case from.Schema != "" && to.Schema == "":
		return nil, fmt.Errorf("cannot diff a database connection with a schema %q. See: https://atlasgo.io/url", from.Schema)
	default:
		// SchemaDiff checks for name equality which is irrelevant in the case
		// the user wants to compare their contents, reset them to allow the comparison.
		current.Schemas[0].Name, desired.Schemas[0].Name = "", ""
		changes, err = differ.SchemaDiff(current.Schemas[0], desired.Schemas[0], opts...)
		if err != nil {
			return nil, err
		}
	}
	return &diff{
		changes: changes,
		from:    current,
		to:      desired,
	}, nil
}

const (
	answerApply = "Apply"
	answerAbort = "Abort"
)

// cmdPrompt returns a promptui.Select that uses the given command's input and output.
func cmdPrompt(cmd *cobra.Command) *promptui.Select {
	return &promptui.Select{
		Label:    "Are you sure?",
		HideHelp: true,
		Stdin:    io.NopCloser(cmd.InOrStdin()),
		Stdout:   nopBellCloser{cmd.OutOrStdout()},
	}
}

func promptUser(cmd *cobra.Command) bool {
	prompt := cmdPrompt(cmd)
	prompt.Items = []string{answerApply, answerAbort}
	_, result, err := prompt.Run()
	if err != nil && !errors.Is(err, promptui.ErrInterrupt) {
		// Fail in case of unexpected errors.
		cobra.CheckErr(err)
	}
	return result == answerApply
}

type nopBellCloser struct{ io.Writer }

func (n nopBellCloser) Write(p []byte) (int, error) {
	if len(p) == 1 && p[0] == readline.CharBell {
		return 0, nil // Skip bell noise.
	}
	return n.Writer.Write(p)
}

func (nopBellCloser) Close() error { return nil }

func tasks(path string) ([]fmttask, error) {
	var tasks []fmttask
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		if strings.HasSuffix(path, ".hcl") {
			tasks = append(tasks, fmttask{
				path: path,
				info: stat,
			})
		}
		return tasks, nil
	}
	all, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, f := range all {
		if f.IsDir() {
			continue
		}
		if strings.HasSuffix(f.Name(), ".hcl") {
			i, err := f.Info()
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, fmttask{
				path: filepath.Join(path, f.Name()),
				info: i,
			})
		}
	}
	return tasks, nil
}

type fmttask struct {
	path string
	info fs.FileInfo
}

// fmtFile tries to format a file and reports if formatting occurred.
func fmtFile(task fmttask) (bool, error) {
	orig, err := os.ReadFile(task.path)
	if err != nil {
		return false, err
	}
	formatted := hclwrite.Format(orig)
	if !bytes.Equal(formatted, orig) {
		return true, os.WriteFile(task.path, formatted, task.info.Mode())
	}
	return false, nil
}

// fixFileURLs converts all file paths to a URL format, if not already.
// For example, "schema.hcl" to "file://schema.hcl".
func fixFileURLs(src []string) []string {
	for i, s := range src {
		if !isURL(s) {
			src[i] = "file://" + s
		}
	}
	return src
}
