// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/spf13/cobra"
)

const (
	migrateFlagDir        = "dir"
	migrateFlagSchema     = "schema"
	migrateFlagForce      = "force"
	migrateDiffFlagDevURL = "dev-url"
	migrateDiffFlagTo     = "to"
)

var (
	// MigrateFlags are the flags used in MigrateCmd (and sub-commands).
	MigrateFlags struct {
		DirURL  string
		DevURL  string
		ToURL   string
		Schemas []string
		Force   bool
	}
	// MigrateCmd represents the migrate command. It wraps several other sub-commands.
	MigrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Manage versioned migration files",
		Long:  "'atlas migrate' wraps several sub-commands for migration management.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Migrate commands will not run on a broken migration directory, unless the force flag is given.
			if !MigrateFlags.Force {
				dir, err := dir()
				if err != nil {
					return err
				}
				if err := migrate.Validate(dir); err != nil {
					fmt.Fprintf(
						cmd.ErrOrStderr(),
						"Error: %s\n\nYou have a checksum error in your migration directory.\n"+
							"This happens if you manually create or edit a migration file.\n"+
							"Please check your migration files and run\n\n"+
							"'atlas migrate hash --force'\n\nto re-hash the contents and resolve the error.\n\n",
						err,
					)
					os.Exit(1)
				}
			}
			return nil
		},
	}
	// MigrateDiffCmd represents the migrate diff command.
	MigrateDiffCmd = &cobra.Command{
		Use:   "diff",
		Short: "Compute the diff between the migration directory and a connected database and create a new migration file.",
		Long: `'atlas migrate diff' uses the dev-database to re-run all migration files in the migration
directory and compares it to a given desired state and create a new migration file containing SQL statements to migrate 
the migration directory state to the desired schema. The desired state can be another connected database or an HCL file.`,
		Example: `  atlas migrate diff --dev-url mysql://user:pass@localhost:3306/dev --to mysql://user:pass@localhost:3306/dbname
  atlas migrate diff --dev-url mysql://user:pass@localhost:3306/dev --to file://atlas.hcl`,
		Args: cobra.MaximumNArgs(1),
		RunE: CmdMigrateDiffRun,
	}
	// MigrateHashCmd represents the migrate hash command.
	MigrateHashCmd = &cobra.Command{
		Use:   "hash",
		Short: "Hash creates an integrity hash file for the migration directories.",
		Long: `'atlas migrate hash' computes the integrity hash sum of the migration directory and stores it in the atlas.sum file.
This command should be used whenever a manual change in the migration directory was made.`,
		Example: `  atlas migrate hash --force`,
		RunE:    CmdMigrateHashRun,
	}
	// MigrateValidateCmd represents the migrate validate command.
	MigrateValidateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validates the migration directories checksum.",
		Long: `'atlas migrate validate' computes the integrity hash sum of the migration directory and compares it to 
the atlas.sum file. If there is a mismatch it will be reported.`,
		Example: `  atlas migrate validate
  atlas migrate validate --dir /path/to/migration/directory`,
		Run: func(*cobra.Command, []string) {},
	}
)

func init() {
	// Add sub-commands.
	RootCmd.AddCommand(MigrateCmd)
	MigrateCmd.AddCommand(MigrateDiffCmd)
	MigrateCmd.AddCommand(MigrateValidateCmd)
	MigrateCmd.AddCommand(MigrateHashCmd)
	// Global flags.
	MigrateCmd.PersistentFlags().StringVarP(&MigrateFlags.DirURL, migrateFlagDir, "", "file://migrations", "select migration directory using DSN format")
	MigrateCmd.PersistentFlags().StringSliceVarP(&MigrateFlags.Schemas, migrateFlagSchema, "", nil, "set schema names")
	MigrateCmd.PersistentFlags().BoolVarP(&MigrateFlags.Force, migrateFlagForce, "", false, "force a command to run on a broken migration directory state")
	MigrateCmd.PersistentFlags().SortFlags = false
	// Diff flags.
	MigrateDiffCmd.Flags().StringVarP(&MigrateFlags.DevURL, migrateDiffFlagDevURL, "", "", "[driver://username:password@address/dbname?param=value] select a data source using the DSN format")
	MigrateDiffCmd.Flags().StringVarP(&MigrateFlags.ToURL, migrateDiffFlagTo, "", "", "[driver://username:password@address/dbname?param=value] select a data source using the DSN format")
	MigrateDiffCmd.Flags().SortFlags = false
	cobra.CheckErr(MigrateDiffCmd.MarkFlagRequired(migrateDiffFlagDevURL))
	cobra.CheckErr(MigrateDiffCmd.MarkFlagRequired(migrateDiffFlagTo))
}

// CmdMigrateDiffRun is the command executed when running the CLI with 'migrate diff' args.
func CmdMigrateDiffRun(cmd *cobra.Command, args []string) error {
	// Open a dev driver.
	dev, err := DefaultMux.OpenAtlas(MigrateFlags.DevURL)
	if err != nil {
		return err
	}
	defer dev.Close()
	if err := checkClean(cmd.Context(), dev); err != nil {
		return err
	}
	// Open the migration directory.
	dir, err := dir()
	if err != nil {
		return err
	}
	// Get a state reader for the desired state.
	desired, err := to(cmd.Context(), dev)
	if src, ok := desired.(io.Closer); ok {
		defer src.Close()
	}
	if err != nil {
		return err
	}
	// Only create one file per plan.
	tf, err := migrate.NewTemplateFormatter(nameTmpl, contentTmpl)
	if err != nil {
		return err
	}
	// Plan the changes and create a new migration file.
	pl := migrate.NewPlanner(dev, dir, migrate.WithFormatter(tf))
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	plan, err := pl.Plan(cmd.Context(), name, desired)
	if err != nil {
		return err
	}
	// Write the plan to a new file.
	return pl.WritePlan(plan)
	// TODO(masseelch): clean up dev after reading the state from migration dir.
}

// CmdMigrateHashRun is the command executed when running the CLI with 'migrate hash' args.
func CmdMigrateHashRun(*cobra.Command, []string) error {
	// Open the migration directory.
	dir, err := dir()
	if err != nil {
		return err
	}
	sum, err := migrate.HashSum(dir)
	if err != nil {
		return err
	}
	return migrate.WriteSumFile(dir, sum)
}

// dir returns a migrate.Dir to use as migration directory. For now only local directories are supported.
func dir() (migrate.Dir, error) {
	parts := strings.SplitN(MigrateFlags.DirURL, "://", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid dir url %q", MigrateFlags.DirURL)
	}
	switch parts[0] {
	case "file":
		return migrate.NewLocalDir(parts[1])
	default:
		return nil, fmt.Errorf("unsupported driver %q", parts[0])
	}
}

// to returns a migrate.StateReader for the given to flag.
func to(ctx context.Context, d *Driver) (migrate.StateReader, error) {
	parts := strings.SplitN(MigrateFlags.ToURL, "://", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid driver url %q", MigrateFlags.ToURL)
	}
	schemas := MigrateFlags.Schemas
	if n, err := SchemaNameFromURL(ctx, parts[0]); n != "" {
		if err != nil {
			return nil, err
		}
		schemas = append(schemas, n)
	}
	switch parts[0] {
	case "file": // hcl file
		f, err := ioutil.ReadFile(parts[1])
		if err != nil {
			return nil, err
		}
		realm := &schema.Realm{}
		if err := d.UnmarshalSpec(f, realm); err != nil {
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
					return nil, fmt.Errorf("schema %q from file %q is not requested (all schemas in HCL must be requested)", s.Name, parts[1])
				}
			}
		}
		if norm, ok := d.Driver.(schema.Normalizer); ok {
			realm, err = norm.NormalizeRealm(ctx, realm)
			if err != nil {
				return nil, err
			}
		}
		return migrate.Realm(realm), nil
	default: // database connection
		drv, err := DefaultMux.OpenAtlas(MigrateFlags.ToURL)
		if err != nil {
			return nil, err
		}
		return &stateReadCloser{
			StateReader: migrate.Conn(drv, &schema.InspectRealmOption{Schemas: schemas}),
			drv:         drv,
		}, nil
	}
}

func checkClean(ctx context.Context, drv *Driver) error {
	realm, err := drv.InspectRealm(ctx, nil)
	if err != nil {
		return err
	}
	if len(realm.Schemas) == 0 {
		return nil
	}
	// If this is an SQLite database it is considered clean if the "main" schema has no tables.
	if strings.HasPrefix(MigrateFlags.DevURL, "sqlite") && realm.Schemas[0].Name == "main" && len(realm.Schemas[0].Tables) == 0 {
		return nil
	}
	return errors.New("dev database must be clean")
}

var (
	funcMap = template.FuncMap{
		"now": func() string { return time.Now().Format("20060102150405") },
		"sem": func(s string) string {
			if !strings.HasSuffix(s, ";") {
				return s + ";"
			}
			return s
		},
	}
	nameTmpl = template.Must(template.New("name").Funcs(funcMap).Parse(
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.sql",
	))
	contentTmpl = template.Must(template.New("content").Funcs(funcMap).Parse(
		"{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ println (sem .Cmd) }}{{ end }}",
	))
)

type stateReadCloser struct {
	migrate.StateReader
	drv *Driver
}

func (s *stateReadCloser) Close() error {
	return s.drv.Close()
}
