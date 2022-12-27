// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"ariga.io/atlas/cmd/atlas/internal/cmdlog"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/manifoldco/promptui"
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
	)
	Root.AddCommand(schemaCmd)
}

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
	url         string   // URL of database to apply the changes on.
	devURL      string   // URL of the dev database.
	paths       []string // Paths to HCL files.
	toURLs      []string // URLs of the desired state.
	schemas     []string // Schemas to take into account when diffing.
	exclude     []string // List of glob patterns used to filter resources from applying (see schema.InspectOptions).
	dryRun      bool     // Only show SQL on screen instead of applying it.
	autoApprove bool     // Don't prompt for approval before applying SQL.
	logFormat   string   // Log format.
	dsn         string   // Deprecated: DSN is an alias for URL.
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
  atlas schema apply -u URL --to file://file1.hcl --to file://file2.hcl
  atlas schema apply -u URL --to file://schema/ --to file://override.hcl

As a convenience, schema URLs may also be provided via an environment definition in
the project file (see: https://atlasgo.io/cli/projects).

If run with the "--dry-run" flag, atlas will exit after printing out the planned
migration.`,
			Example: `  atlas schema apply -u "mysql://user:pass@localhost/dbname" --to file://atlas.hcl
  atlas schema apply -u "mysql://localhost" --to file://schema.hcl --schema prod --schema staging
  atlas schema apply -u "mysql://user:pass@localhost:3306/dbname" --to file://schema.hcl --dry-run
  atlas schema apply -u "mariadb://user:pass@localhost:3306/dbname" --to file://schema.hcl
  atlas schema apply --url "postgres://user:pass@host:port/dbname?sslmode=disable" --to file://schema.hcl
  atlas schema apply -u "sqlite://file:ex1.db?_fk=1" --to file://schema.hcl`,
			PreRunE: func(cmd *cobra.Command, _ []string) error {
				return dsn2url(cmd)
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				if GlobalFlags.SelectedEnv == "" {
					return schemaApplyRun(cmd, args, flags)
				}
				return cmdEnvsRun(schemaApplyRun, setSchemaEnvFlags, cmd, args, &flags)
			},
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
	cmd.Flags().StringVarP(&flags.logFormat, flagLog, "", "", "custom logging using a Go template")
	addFlagDSN(cmd.Flags(), &flags.dsn)
	// Hidden support for the deprecated -f flag.
	cmd.Flags().StringSliceVarP(&flags.paths, flagFile, "f", nil, "[paths...] file or directory containing HCL or SQL files")
	cobra.CheckErr(cmd.Flags().MarkHidden(flagFile))
	cmd.MarkFlagsMutuallyExclusive(flagFile, flagTo)
	return cmd
}

func schemaApplyRun(cmd *cobra.Command, _ []string, flags schemaApplyFlags) error {
	switch {
	case flags.url == "":
		return errors.New(`required flag(s) "url" not set`)
	case len(flags.paths) == 0 && len(flags.toURLs) == 0:
		return errors.New(`one of flag(s) "file" or "to" is required`)
	case flags.logFormat != "" && !flags.dryRun && !flags.autoApprove:
		return errors.New(`--log can only be used with --dry-run or --auto-approve`)
	}
	// If the old -f flag is given convert them to the URL format. If both are given,
	// cobra would throw an error since they are marked as mutually exclusive.
	for _, p := range flags.paths {
		if !strings.Contains(p, "://") {
			p = "file://" + p
		}
		flags.toURLs = append(flags.toURLs, p)
	}
	var (
		err    error
		dev    *sqlclient.Client
		ctx    = cmd.Context()
		format = cmdlog.SchemaPlanTemplate
	)
	if v := flags.logFormat; v != "" {
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
	from, err := stateReader(ctx, &stateReaderConfig{
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
	to, err := stateReader(ctx, &stateReaderConfig{
		urls:    flags.toURLs,
		dev:     dev,
		client:  client,
		schemas: flags.schemas,
		exclude: flags.exclude,
		vars:    GlobalFlags.Vars,
	})
	if err != nil {
		return err
	}
	defer to.Close()
	changes, err := computeDiff(ctx, client, from, to)
	if err != nil {
		return err
	}
	// Returning at this stage should
	// not trigger the help message.
	cmd.SilenceUsage = true
	switch {
	case len(changes) == 0:
		return format.Execute(cmd.OutOrStderr(), &cmdlog.SchemaApply{})
	case flags.logFormat != "" && flags.autoApprove:
		var (
			applied int
			plan    *migrate.Plan
			cause   *cmdlog.StmtError
		)
		if plan, err = client.PlanChanges(cmd.Context(), "", changes); err != nil {
			return err
		}
		if err = client.ApplyChanges(ctx, changes); err == nil {
			applied = len(plan.Changes)
		} else if i, ok := err.(interface{ Applied() int }); ok && i.Applied() < len(plan.Changes) {
			applied, cause = i.Applied(), &cmdlog.StmtError{Stmt: plan.Changes[i.Applied()].Cmd, Text: err.Error()}
		} else {
			cause = &cmdlog.StmtError{Text: err.Error()}
		}
		apply := cmdlog.NewSchemaApply(cmdlog.NewEnv(client, nil), plan.Changes[:applied], plan.Changes[applied:], cause)
		if err1 := format.Execute(cmd.OutOrStderr(), apply); err1 != nil {
			if err != nil {
				err1 = fmt.Errorf("%w: %v", err, err1)
			}
			err = err1
		}
		return err
	default:
		if err := summary(cmd, client, changes, format); err != nil {
			return err
		}
		if !flags.dryRun && (flags.autoApprove || promptUser()) {
			return client.ApplyChanges(ctx, changes)
		}
		return nil
	}
}

type schemeCleanFlags struct {
	URL         string // URL of database to apply the changes on.
	AutoApprove bool   // Don't prompt for approval before applying SQL.
}

// schemaCleanCmd represents the 'atlas schema clean' subcommand.
func schemaCleanCmd() *cobra.Command {
	var (
		flags schemeCleanFlags
		cmd   = &cobra.Command{
			Use:   "clean [flags]",
			Short: "Removes all objects from the connected database.",
			Long: `'atlas schema clean' drops all objects in the connected database and leaves it in an empty state.
As a safety feature, 'atlas schema clean' will ask for confirmation before attempting to execute any SQL.`,
			Example: `  atlas schema clean -u mysql://user:pass@localhost:3306/dbname
  atlas schema clean -u mysql://user:pass@localhost:3306/`,
			PreRunE: func(cmd *cobra.Command, _ []string) error {
				if err := schemaFlagsFromEnv(cmd); err != nil {
					return err
				}
				return dsn2url(cmd)
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				return schemaCleanRun(cmd, args, flags)
			},
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURL(cmd.Flags(), &flags.URL)
	addFlagAutoApprove(cmd.Flags(), &flags.AutoApprove)
	cobra.CheckErr(cmd.MarkFlagRequired(flagURL))
	return cmd
}

func schemaCleanRun(cmd *cobra.Command, _ []string, flags schemeCleanFlags) error {
	// Open a client to the database.
	c, err := sqlclient.Open(cmd.Context(), flags.URL)
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
	if len(drop) == 0 {
		cmd.Println("Nothing to drop")
		return nil
	}
	if err := summary(cmd, c, drop, cmdlog.SchemaPlanTemplate); err != nil {
		return err
	}
	if flags.AutoApprove || promptUser() {
		if err := c.ApplyChanges(cmd.Context(), drop); err != nil {
			return err
		}
	}
	return nil
}

type schemaDiffFlags struct {
	fromURL []string
	toURL   []string
	devURL  string
	schemas []string
	exclude []string
}

// schemaDiffCmd represents the 'atlas schema diff' subcommand.
func schemaDiffCmd() *cobra.Command {
	var (
		flags schemaDiffFlags
		cmd   = &cobra.Command{
			Use:   "diff",
			Short: "Calculate and print the diff between two schemas.",
			Long: `'atlas schema diff' reads the state of two given schema definitions, 
calculates the difference in their schemas, and prints a plan of
SQL statements to migrate the "from" database to the schema of the "to" database.
The database states can be read from a connected database, an HCL project or a migration directory.`,
			Example: `  atlas schema diff --from mysql://user:pass@localhost:3306/test --to file://schema.hcl
  atlas schema diff --from mysql://user:pass@localhost:3306 --to file://schema_1.hcl --to file://schema_2.hcl
  atlas schema diff --from mysql://user:pass@localhost:3306 --to file://migrations`,
			PreRunE: func(cmd *cobra.Command, _ []string) error {
				if err := schemaFlagsFromEnv(cmd); err != nil {
					return err
				}
				return dsn2url(cmd)
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				return schemaDiffRun(cmd, args, flags)
			},
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURLs(cmd.Flags(), &flags.fromURL, flagFrom, flagFromShort)
	addFlagToURLs(cmd.Flags(), &flags.toURL)
	addFlagDevURL(cmd.Flags(), &flags.devURL)
	addFlagSchemas(cmd.Flags(), &flags.schemas)
	addFlagExclude(cmd.Flags(), &flags.exclude)
	cobra.CheckErr(cmd.MarkFlagRequired(flagFrom))
	cobra.CheckErr(cmd.MarkFlagRequired(flagTo))
	return cmd
}

func schemaDiffRun(cmd *cobra.Command, _ []string, flags schemaDiffFlags) error {
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
	from, err := stateReader(ctx, &stateReaderConfig{
		urls:    flags.fromURL,
		dev:     c,
		vars:    GlobalFlags.Vars,
		schemas: flags.schemas,
		exclude: flags.exclude,
	})
	if err != nil {
		return err
	}
	defer from.Close()
	to, err := stateReader(ctx, &stateReaderConfig{
		urls:    flags.toURL,
		dev:     c,
		vars:    GlobalFlags.Vars,
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
	diff, err := computeDiff(ctx, c, from, to)
	if err != nil {
		return err
	}
	return summary(cmd, c, diff, cmdlog.SchemaDiffTemplate)
}

type schemaInspectFlags struct {
	url       string   // URL of database to apply the changes on.
	logFormat string   // Format of the log output.
	schemas   []string // Schemas to take into account when diffing.
	exclude   []string // List of glob patterns used to filter resources from applying (see schema.InspectOptions).
	dsn       string   // Deprecated: DSN is an alias for URL.
}

// schemaInspectCmd represents the 'atlas schema inspect' subcommand.
func schemaInspectCmd() *cobra.Command {
	var (
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
			PreRunE: func(cmd *cobra.Command, _ []string) error {
				if err := schemaFlagsFromEnv(cmd); err != nil {
					return err
				}
				return dsn2url(cmd)
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				return schemaInspectRun(cmd, args, flags)
			},
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURL(cmd.Flags(), &flags.url)
	addFlagSchemas(cmd.Flags(), &flags.schemas)
	addFlagExclude(cmd.Flags(), &flags.exclude)
	addFlagLog(cmd.Flags(), &flags.logFormat)
	addFlagDSN(cmd.Flags(), &flags.dsn)
	cobra.CheckErr(cmd.MarkFlagRequired(flagURL))
	return cmd
}

func schemaInspectRun(cmd *cobra.Command, _ []string, flags schemaInspectFlags) error {
	client, err := sqlclient.Open(cmd.Context(), flags.url)
	if err != nil {
		return err
	}
	defer client.Close()
	schemas := flags.schemas
	if client.URL.Schema != "" {
		schemas = append(schemas, client.URL.Schema)
	}
	s, err := client.InspectRealm(cmd.Context(), &schema.InspectRealmOption{
		Schemas: schemas,
		Exclude: flags.exclude,
	})
	format := cmdlog.SchemaInspectTemplate
	if v := flags.logFormat; v != "" {
		if format, err = template.New("format").Funcs(cmdlog.InspectTemplateFuncs).Parse(v); err != nil {
			return fmt.Errorf("parse log format: %w", err)
		}
	}
	return format.Execute(cmd.OutOrStdout(), &cmdlog.SchemaInspect{
		Client: client,
		Realm:  s,
		Error:  err,
	})
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return schemaFmtRun(cmd, args)
		},
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
func selectEnv(name string) (*Env, error) {
	env := &Env{
		Lint:      &Lint{},
		Migration: &Migration{},
	}
	if name == "" {
		return env, nil
	}
	envs, err := LoadEnv(name, WithInput(GlobalFlags.Vars))
	if err != nil {
		return nil, err
	}
	if len(envs) > 1 {
		return nil, fmt.Errorf("multiple envs found for %q", name)
	}
	return envs[0], nil
}

func schemaFlagsFromEnv(cmd *cobra.Command) error {
	activeEnv, err := selectEnv(GlobalFlags.SelectedEnv)
	if err != nil {
		return err
	}
	return setSchemaEnvFlags(cmd, activeEnv)
}

func setSchemaEnvFlags(cmd *cobra.Command, env *Env) error {
	if err := inputValuesFromEnv(cmd, env); err != nil {
		return err
	}
	if err := maySetFlag(cmd, flagURL, env.URL); err != nil {
		return err
	}
	if err := maySetFlag(cmd, flagDevURL, env.DevURL); err != nil {
		return err
	}
	srcs, err := env.Sources()
	if err != nil {
		return err
	}
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
	case "apply":
		if err := maySetFlag(cmd, flagLog, env.Log.Schema.Apply); err != nil {
			return err
		}
	}
	return nil
}

func computeDiff(ctx context.Context, differ *sqlclient.Client, from, to *stateReadCloser) ([]schema.Change, error) {
	current, err := from.ReadState(ctx)
	if err != nil {
		return nil, err
	}
	desired, err := to.ReadState(ctx)
	if err != nil {
		return nil, err
	}
	var diff []schema.Change
	switch {
	// Compare realm if the desired state is an HCL file or both connections are not bound to a schema.
	case from.hcl, to.hcl, from.schema == "" && to.schema == "":
		diff, err = differ.RealmDiff(current, desired)
		if err != nil {
			return nil, err
		}
	case from.schema == "", to.schema == "":
		return nil, fmt.Errorf("cannot diff a schema with a database connection: %q <> %q", from.schema, to.schema)
	default:
		// SchemaDiff checks for name equality which is irrelevant in the case
		// the user wants to compare their contents, reset them to allow the comparison.
		current.Schemas[0].Name, desired.Schemas[0].Name = "", ""
		diff, err = differ.SchemaDiff(current.Schemas[0], desired.Schemas[0])
		if err != nil {
			return nil, err
		}
	}
	return diff, nil
}

func summary(cmd *cobra.Command, c *sqlclient.Client, changes []schema.Change, t *template.Template) error {
	p, err := c.PlanChanges(cmd.Context(), "", changes)
	if err != nil {
		return err
	}
	return t.Execute(
		cmd.OutOrStdout(),
		cmdlog.NewSchemaPlan(cmdlog.NewEnv(c, nil), p.Changes, nil),
	)
}

const (
	answerApply = "Apply"
	answerAbort = "Abort"
)

func promptUser() bool {
	prompt := promptui.Select{
		Label: "Are you sure?",
		Items: []string{answerApply, answerAbort},
	}
	_, result, err := prompt.Run()
	if err != nil && err != promptui.ErrInterrupt {
		// Fail in case of unexpected errors.
		cobra.CheckErr(err)
	}
	return result == answerApply
}

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
