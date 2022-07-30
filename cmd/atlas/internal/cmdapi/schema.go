// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"bytes"
	"errors"
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

const (
	urlFlag         = "url"
	schemaFlag      = "schema"
	excludeFlag     = "exclude"
	devURLFlag      = "dev-url"
	fileFlag        = "file"
	dsnFlag         = "dsn"
	varFlag         = "var"
	autoApproveFlag = "auto-approve"
)

var (
	// schemaCmd represents the subcommand 'atlas schema'.
	schemaCmd = &cobra.Command{
		Use:   "schema",
		Short: "Work with atlas schemas.",
		Long:  "The `atlas schema` command groups subcommands for working with Atlas schemas.",
	}

	// SchemaFlags are common flags used by schema commands.
	SchemaFlags struct {
		URL     string
		Schemas []string
		Exclude []string

		// Deprecated: DSN is an alias for URL.
		DSN string
	}

	// ApplyFlags are the flags used in SchemaApply command.
	ApplyFlags struct {
		DevURL      string
		Paths       []string
		DryRun      bool
		AutoApprove bool
	}

	// CleanFlags are the flags used in SchemaClean command.
	CleanFlags struct {
		URL         string
		AutoApprove bool
	}

	// SchemaApply represents the 'atlas schema apply' subcommand command.
	SchemaApply = &cobra.Command{
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
		PreRunE: schemaFlagsFromEnv,
		RunE:    CmdApplyRun,
		Example: `  atlas schema apply -u "mysql://user:pass@localhost/dbname" -f atlas.hcl
  atlas schema apply -u "mysql://localhost" -f schema.hcl --schema prod --schema staging
  atlas schema apply -u "mysql://user:pass@localhost:3306/dbname" -f schema.hcl --dry-run
  atlas schema apply -u "mariadb://user:pass@localhost:3306/dbname" -f schema.hcl
  atlas schema apply --url "postgres://user:pass@host:port/dbname?sslmode=disable" -f schema.hcl
  atlas schema apply -u "sqlite://file:ex1.db?_fk=1" -f schema.hcl
  atlas schema apply -u "spanner://projects/PROJECT/instances/INSTANCE/databases/DATABASE" -f atlas.hcl"`,
	}

	// SchemaClean represents the 'atlas schema clean' subcommand.
	SchemaClean = &cobra.Command{
		Use:   "clean [flags]",
		Short: "Removes all objects from the connected database.",
		Long: `'atlas schema clean' drops all objects in the connected database and leaves it in an empty state.
As a safety feature, 'atlas schema clean' will ask for confirmation before attempting to execute any SQL.`,
		Example: `  atlas schema clean -u mysql://user:pass@localhost:3306/dbname
  atlas schema clean -u mysql://user:pass@localhost:3306/`,
		PreRunE: schemaFlagsFromEnv,
		RunE:    CmdCleanRun,
	}

	// SchemaInspect represents the 'atlas schema inspect' subcommand.
	SchemaInspect = &cobra.Command{
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
		PreRunE: schemaFlagsFromEnv,
		RunE:    CmdInspectRun,
		Example: `  atlas schema inspect -u "mysql://user:pass@localhost:3306/dbname"
  atlas schema inspect -u "mariadb://user:pass@localhost:3306/" --schema=schemaA,schemaB -s schemaC
  atlas schema inspect --url "postgres://user:pass@host:port/dbname?sslmode=disable"
  atlas schema inspect -u "sqlite://file:ex1.db?_fk=1"
  atlas schema inspect -u "spanner://projects/PROJECT/instances/INSTANCE/databases/DATABASE"`,
	}

	// SchemaFmt represents the 'atlas schema fmt' subcommand.
	SchemaFmt = &cobra.Command{
		Use:   "fmt [path ...]",
		Short: "Formats Atlas HCL files",
		Long: `'atlas schema fmt' formats all ".hcl" files under the given path using
canonical HCL layout style as defined by the github.com/hashicorp/hcl/v2/hclwrite package.
Unless stated otherwise, the fmt command will use the current directory.

After running, the command will print the names of the files it has formatted. If all
files in the directory are formatted, no input will be printed out.
`,
		Run: CmdFmtRun,
	}
)

const (
	answerApply = "Apply"
	answerAbort = "Abort"
)

func init() {
	// Common flags.
	receivesEnv(schemaCmd)

	// Schema apply flags.
	schemaCmd.AddCommand(SchemaApply)
	SchemaApply.Flags().SortFlags = false
	SchemaApply.Flags().StringSliceVarP(&ApplyFlags.Paths, fileFlag, "f", nil, "[paths...] file or directory containing the HCL files")
	SchemaApply.Flags().StringVarP(&SchemaFlags.URL, urlFlag, "u", "", "URL to the database using the format:\n[driver://username:password@address/dbname?param=value]")
	SchemaApply.Flags().StringSliceVarP(&SchemaFlags.Exclude, excludeFlag, "", nil, "List of glob patterns used to filter resources from applying.")
	SchemaApply.Flags().StringSliceVarP(&SchemaFlags.Schemas, schemaFlag, "s", nil, "Set schema names.")
	SchemaApply.Flags().StringVarP(&ApplyFlags.DevURL, devURLFlag, "", "", "URL for the dev database. Used to validate schemas and calculate diffs\nbefore running migration.")
	SchemaApply.Flags().BoolVarP(&ApplyFlags.DryRun, "dry-run", "", false, "Dry-run. Print SQL plan without prompting for execution.")
	SchemaApply.Flags().BoolVarP(&ApplyFlags.AutoApprove, autoApproveFlag, "", false, "Auto approve. Apply the schema changes without prompting for approval.")
	SchemaApply.Flags().StringVarP(&SchemaFlags.DSN, dsnFlag, "d", "", "")
	cobra.CheckErr(SchemaApply.Flags().MarkHidden(dsnFlag))
	cobra.CheckErr(SchemaApply.MarkFlagRequired(urlFlag))
	cobra.CheckErr(SchemaApply.MarkFlagRequired(fileFlag))

	// Schema clean flags.
	SchemaClean.Flags().StringVarP(&CleanFlags.URL, urlFlag, "u", "", "URL to the database using the format:\n[driver://username:password@address/dbname?param=value]")
	SchemaClean.Flags().BoolVarP(&CleanFlags.AutoApprove, autoApproveFlag, "", false, "Auto approve. Apply the schema changes without prompting for approval.")
	cobra.CheckErr(SchemaClean.MarkFlagRequired(urlFlag))

	// Schema inspect flags.
	schemaCmd.AddCommand(SchemaInspect)
	SchemaInspect.Flags().StringVarP(&SchemaFlags.URL, urlFlag, "u", "", "[driver://username:password@protocol(address)/dbname?param=value] select a database using the URL format")
	SchemaInspect.Flags().StringSliceVarP(&SchemaFlags.Schemas, schemaFlag, "s", nil, "Set schema name")
	SchemaInspect.Flags().StringSliceVarP(&SchemaFlags.Exclude, excludeFlag, "", nil, "List of glob patterns used to filter resources from inspection")
	SchemaInspect.Flags().StringVarP(&SchemaFlags.DSN, dsnFlag, "d", "", "")
	cobra.CheckErr(SchemaInspect.Flags().MarkHidden(dsnFlag))
	cobra.CheckErr(SchemaInspect.MarkFlagRequired(urlFlag))

	// Schema fmt.
	schemaCmd.AddCommand(SchemaFmt)
	schemaCmd.AddCommand(SchemaClean)
}

// selectEnv returns the Env from the current project file based on the selected
// argument. If selected is "", or no project file exists in the current directory
// a zero-value Env is returned.
func selectEnv(selected string) (*Env, error) {
	env := &Env{
		Lint:      &Lint{},
		Migration: &Migration{},
	}
	if selected == "" {
		return env, nil
	}
	if _, err := os.Stat(projectFileName); os.IsNotExist(err) {
		return nil, fmt.Errorf("project file %q was not found", projectFileName)
	}
	return LoadEnv(projectFileName, selected, WithInput(GlobalFlags.Vars))
}

func schemaFlagsFromEnv(cmd *cobra.Command, _ []string) error {
	activeEnv, err := selectEnv(GlobalFlags.SelectedEnv)
	if err != nil {
		return err
	}
	if err := inputValsFromEnv(cmd); err != nil {
		return err
	}
	if err := dsn2url(cmd); err != nil {
		return err
	}
	if err := maySetFlag(cmd, urlFlag, activeEnv.URL); err != nil {
		return err
	}
	if err := maySetFlag(cmd, devURLFlag, activeEnv.DevURL); err != nil {
		return err
	}
	srcs, err := activeEnv.Sources()
	if err != nil {
		return err
	}
	if err := maySetFlag(cmd, fileFlag, strings.Join(srcs, "")); err != nil {
		return err
	}
	if s := strings.Join(activeEnv.Schemas, ","); s != "" {
		if err := maySetFlag(cmd, schemaFlag, s); err != nil {
			return err
		}
	}
	if s := strings.Join(activeEnv.Exclude, ","); s != "" {
		if err := maySetFlag(cmd, excludeFlag, s); err != nil {
			return err
		}
	}
	return nil
}

// maySetFlag sets the flag with the provided name to envVal if such a flag exists
// on the cmd, it was not set by the user via the command line and if envVal is not
// an empty string.
func maySetFlag(cmd *cobra.Command, name, envVal string) error {
	fl := cmd.Flag(name)
	if fl == nil {
		return nil
	}
	if fl.Changed {
		return nil
	}
	if envVal == "" {
		return nil
	}
	return cmd.Flags().Set(name, envVal)
}

func dsn2url(cmd *cobra.Command) error {
	dsnF, urlF := cmd.Flag(dsnFlag), cmd.Flag(urlFlag)
	switch {
	case dsnF == nil:
	case dsnF.Changed && urlF.Changed:
		return errors.New(`both flags "url" and "dsn" were set`)
	case dsnF.Changed && !urlF.Changed:
		return cmd.Flags().Set(urlFlag, dsnF.Value.String())
	}
	return nil
}

// CmdInspectRun is the command used when running CLI.
func CmdInspectRun(cmd *cobra.Command, _ []string) error {
	// Create the client.
	client, err := sqlclient.Open(cmd.Context(), SchemaFlags.URL)
	if err != nil {
		return err
	}
	defer client.Close()
	schemas := SchemaFlags.Schemas
	if client.URL.Schema != "" {
		schemas = append(schemas, client.URL.Schema)
	}
	s, err := client.InspectRealm(cmd.Context(), &schema.InspectRealmOption{
		Schemas: schemas,
		Exclude: SchemaFlags.Exclude,
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

// CmdApplyRun is the command used when running CLI.
func CmdApplyRun(cmd *cobra.Command, _ []string) error {
	c, err := sqlclient.Open(cmd.Context(), SchemaFlags.URL)
	if err != nil {
		return err
	}
	defer c.Close()
	return applyRun(cmd, c, ApplyFlags.DevURL, ApplyFlags.Paths, ApplyFlags.DryRun, ApplyFlags.AutoApprove, GlobalFlags.Vars)
}

// CmdCleanRun is the command executed when running the CLI with 'schema clean' args.
func CmdCleanRun(cmd *cobra.Command, _ []string) error {
	// Open a client to the database.
	c, err := sqlclient.Open(cmd.Context(), CleanFlags.URL)
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
	if CleanFlags.AutoApprove || promptUser() {
		if err := c.ApplyChanges(cmd.Context(), drop); err != nil {
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

// CmdFmtRun formats all HCL files in a given directory using canonical HCL formatting
// rules.
func CmdFmtRun(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		args = append(args, "./")
	}
	for _, path := range args {
		handlePath(cmd, path)
	}
}

func applyRun(cmd *cobra.Command, client *sqlclient.Client, devURL string, paths []string, dryRun, autoApprove bool, input map[string]string) error {
	schemas, ctx := SchemaFlags.Schemas, cmd.Context()
	if client.URL.Schema != "" {
		schemas = append(schemas, client.URL.Schema)
	}
	realm, err := client.InspectRealm(ctx, &schema.InspectRealmOption{
		Schemas: schemas,
		Exclude: SchemaFlags.Exclude,
	})
	if err != nil {
		return err
	}
	desired := &schema.Realm{}
	parsed, err := parseHCLPaths(paths...)
	if err != nil {
		return err
	}
	if err := client.Eval(parsed, desired, input); err != nil {
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
	if _, ok := client.Driver.(schema.Normalizer); ok && devURL != "" {
		dev, err := sqlclient.Open(ctx, ApplyFlags.DevURL)
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
	if !dryRun && (autoApprove || promptUser()) {
		if err := client.ApplyChanges(ctx, changes); err != nil {
			return err
		}
	}
	return nil
}

func promptUser() bool {
	prompt := promptui.Select{
		Label: "Are you sure?",
		Items: []string{answerApply, answerAbort},
	}
	_, result, err := prompt.Run()
	cobra.CheckErr(err)
	return result == answerApply
}

func handlePath(cmd *cobra.Command, path string) {
	tasks, err := tasks(path)
	cobra.CheckErr(err)
	for _, task := range tasks {
		changed, err := fmtFile(task)
		cobra.CheckErr(err)
		if changed {
			cmd.Println(task.path)
		}
	}
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
