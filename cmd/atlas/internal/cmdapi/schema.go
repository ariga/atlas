// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
	URL         string   // URL of database to apply the changes on.
	DevURL      string   // URL of the dev database.
	Paths       []string // Paths to HCL files.
	Schemas     []string // Schemas to take into account when diffing.
	Exclude     []string // List of glob patterns used to filter resources from applying (see schema.InspectOptions).
	DryRun      bool     // Only show SQL on screen instead of applying it.
	AutoApprove bool     // Don't prompt for approval before applying SQL.
	DSN         string   // Deprecated: DSN is an alias for URL.
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

The schema is provided by one or more paths (to a file or directory) using the "-f" flag:
  atlas schema apply -u URL -f file1.hcl -f file2.hcl
  atlas schema apply -u URL -f schema/ -f override.hcl

As a convenience, schemas may also be provided via an environment definition in
the project file (see: https://atlasgo.io/cli/projects).

If run with the "--dry-run" flag, atlas will exit after printing out the planned
migration.`,
			Example: `  atlas schema apply -u "mysql://user:pass@localhost/dbname" -f atlas.hcl
  atlas schema apply -u "mysql://localhost" -f schema.hcl --schema prod --schema staging
  atlas schema apply -u "mysql://user:pass@localhost:3306/dbname" -f schema.hcl --dry-run
  atlas schema apply -u "mariadb://user:pass@localhost:3306/dbname" -f schema.hcl
  atlas schema apply --url "postgres://user:pass@host:port/dbname?sslmode=disable" -f schema.hcl
  atlas schema apply -u "sqlite://file:ex1.db?_fk=1" -f schema.hcl`,
			PreRunE: schemaFlagsFromEnv,
			RunE: func(cmd *cobra.Command, args []string) error {
				return schemaApplyRun(cmd, args, flags)
			},
		}
	)
	cmd.Flags().SortFlags = false
	cmd.Flags().StringSliceVarP(&flags.Paths, flagFile, "f", nil, "[paths...] file or directory containing the HCL files")
	addFlagURL(cmd.Flags(), &flags.URL)
	addFlagExclude(cmd.Flags(), &flags.Exclude)
	addFlagSchemas(cmd.Flags(), &flags.Schemas)
	addFlagDevURL(cmd.Flags(), &flags.DevURL)
	addFlagDryRun(cmd.Flags(), &flags.DryRun)
	addFlagAutoApprove(cmd.Flags(), &flags.AutoApprove)
	addFlagDSN(cmd.Flags(), &flags.DSN)
	cobra.CheckErr(cmd.MarkFlagRequired(flagURL))
	cobra.CheckErr(cmd.MarkFlagRequired(flagFile))
	return cmd
}

func schemaApplyRun(cmd *cobra.Command, _ []string, flags schemaApplyFlags) error {
	client, err := sqlclient.Open(cmd.Context(), flags.URL)
	if err != nil {
		return err
	}
	defer client.Close()
	schemas, ctx := flags.Schemas, cmd.Context()
	if client.URL.Schema != "" {
		schemas = append(schemas, client.URL.Schema)
	}
	realm, err := client.InspectRealm(ctx, &schema.InspectRealmOption{
		Schemas: schemas,
		Exclude: flags.Exclude,
	})
	if err != nil {
		return err
	}
	desired := &schema.Realm{}
	parsed, err := parseHCLPaths(flags.Paths...)
	if err != nil {
		return err
	}
	if err := client.Eval(parsed, desired, GlobalFlags.Vars); err != nil {
		return err
	}
	if len(schemas) > 0 {
		// Validate all schemas in file were selected by user.
		sm := make(map[string]bool, len(schemas))
		for _, s := range schemas {
			sm[s] = true
		}
		for _, s := range desired.Schemas {
			if !sm[s.Name] {
				return fmt.Errorf("schema %q was not selected %q, all schemas defined in file must be selected", s.Name, schemas)
			}
		}
	}
	if _, ok := client.Driver.(schema.Normalizer); ok && flags.DevURL != "" {
		dev, err := sqlclient.Open(ctx, flags.DevURL)
		if err != nil {
			return err
		}
		defer dev.Close()
		desired, err = dev.Driver.(schema.Normalizer).NormalizeRealm(ctx, desired)
		if err != nil {
			return err
		}
	}
	changes, err := client.RealmDiff(realm, desired)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		cmd.Println("Schema is synced, no changes to be made")
		return nil
	}
	if err := summary(cmd, client, changes); err != nil {
		return err
	}
	if !flags.DryRun && (flags.AutoApprove || promptUser()) {
		if err := client.ApplyChanges(ctx, changes); err != nil {
			return err
		}
	}
	return nil
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
			PreRunE: schemaFlagsFromEnv,
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
		drop, err = c.RealmDiff(r, nil)
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
	if err := summary(cmd, c, drop); err != nil {
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
	fromURL string // URL of the current state.
	toURL   string // URL of the desired state.
	devURL  string // URL of a dev database.
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
			RunE: func(cmd *cobra.Command, args []string) error {
				return schemaDiffRun(cmd, args, flags)
			},
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURL(cmd.Flags(), &flags.fromURL, flagFrom, "")
	addFlagURL(cmd.Flags(), &flags.toURL, flagTo, "")
	addFlagDevURL(cmd.Flags(), &flags.devURL)
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
	from, err := stateReader(ctx, &stateReaderConfig{urls: []string{flags.fromURL}, dev: c})
	if err != nil {
		return err
	}
	defer from.Close()
	to, err := stateReader(ctx, &stateReaderConfig{urls: []string{flags.toURL}, dev: c})
	if err != nil {
		return err
	}
	defer to.Close()
	if c == nil {
		// If not both states are provided by a database connection, the call to state-reader would have returned
		// an error already. If we land in this case, we can assume both states are database connections.
		c = to.Closer.(*sqlclient.Client)
	}
	current, err := from.ReadState(ctx)
	if err != nil {
		return err
	}
	desired, err := to.ReadState(ctx)
	if err != nil {
		return err
	}
	var diff []schema.Change
	switch {
	// compare realm
	case from.Schema == "" && to.Schema == "":
		diff, err = c.RealmDiff(current, desired)
		if err != nil {
			return err
		}
	case from.Schema == "":
		return fmt.Errorf("cannot diff schema %q with a database connection", from.Schema)
	case to.Schema == "":
		return fmt.Errorf("cannot diff database connection with a schema %q", to.Schema)
	default:
		// SchemaDiff checks for name equality which is irrelevant in the case
		// the user wants to compare their contents, reset them to allow the comparison.
		current.Schemas[0].Name, desired.Schemas[0].Name = "", ""
		diff, err = c.SchemaDiff(current.Schemas[0], desired.Schemas[0])
		if err != nil {
			return err
		}
	}
	p, err := c.PlanChanges(ctx, "plan", diff)
	if err != nil {
		return err
	}
	if len(p.Changes) == 0 {
		cmd.Println("Schemas are synced, no changes to be made.")
	}
	for _, c := range p.Changes {
		if c.Comment != "" {
			cmd.Println("--", strings.ToUpper(c.Comment[:1])+c.Comment[1:])
		}
		cmd.Println(c.Cmd)
	}
	return nil
}

type schemaInspectFlags struct {
	URL     string   // URL of database to apply the changes on.
	Schemas []string // Schemas to take into account when diffing.
	Exclude []string // List of glob patterns used to filter resources from applying (see schema.InspectOptions).
	DSN     string   // Deprecated: DSN is an alias for URL.
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
			PreRunE: schemaFlagsFromEnv,
			RunE: func(cmd *cobra.Command, args []string) error {
				return schemaInspectRun(cmd, args, flags)
			},
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURL(cmd.Flags(), &flags.URL)
	addFlagSchemas(cmd.Flags(), &flags.Schemas)
	addFlagExclude(cmd.Flags(), &flags.Exclude)
	addFlagDSN(cmd.Flags(), &flags.DSN)
	cobra.CheckErr(cmd.MarkFlagRequired(flagURL))
	return cmd
}

func schemaInspectRun(cmd *cobra.Command, _ []string, flags schemaInspectFlags) error {
	client, err := sqlclient.Open(cmd.Context(), flags.URL)
	if err != nil {
		return err
	}
	defer client.Close()
	schemas := flags.Schemas
	if client.URL.Schema != "" {
		schemas = append(schemas, client.URL.Schema)
	}
	s, err := client.InspectRealm(cmd.Context(), &schema.InspectRealmOption{
		Schemas: schemas,
		Exclude: flags.Exclude,
	})
	if err != nil {
		return err
	}
	ddl, err := client.MarshalSpec(s)
	if err != nil {
		return err
	}
	cmd.Print(string(ddl))
	return nil
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

func schemaFlagsFromEnv(cmd *cobra.Command, _ []string) error {
	activeEnv, err := selectEnv(GlobalFlags.SelectedEnv)
	if err != nil {
		return err
	}
	if err := inputValsFromEnv(cmd, activeEnv); err != nil {
		return err
	}
	if err := dsn2url(cmd); err != nil {
		return err
	}
	if err := maySetFlag(cmd, flagURL, activeEnv.URL); err != nil {
		return err
	}
	if err := maySetFlag(cmd, flagDevURL, activeEnv.DevURL); err != nil {
		return err
	}
	srcs, err := activeEnv.Sources()
	if err != nil {
		return err
	}
	if err := maySetFlag(cmd, flagFile, strings.Join(srcs, "")); err != nil {
		return err
	}
	if s := strings.Join(activeEnv.Schemas, ","); s != "" {
		if err := maySetFlag(cmd, flagSchema, s); err != nil {
			return err
		}
	}
	if s := strings.Join(activeEnv.Exclude, ","); s != "" {
		if err := maySetFlag(cmd, flagExclude, s); err != nil {
			return err
		}
	}
	return nil
}

func summary(cmd *cobra.Command, drv migrate.Driver, changes []schema.Change) error {
	p, err := drv.PlanChanges(cmd.Context(), "", changes)
	if err != nil {
		return err
	}
	cmd.Println("-- Planned Changes:")
	for _, c := range p.Changes {
		if c.Comment != "" {
			cmd.Println("--", strings.ToUpper(c.Comment[:1])+c.Comment[1:])
		}
		cmd.Println(c.Cmd)
	}
	return nil
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
	cobra.CheckErr(err)
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
