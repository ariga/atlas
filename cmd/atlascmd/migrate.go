// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package atlascmd

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/spf13/cobra"
)

const (
	migrateFlagDir         = "dir"
	migrateFlagSchema      = "schema"
	migrateFlagForce       = "force"
	migrateDiffFlagDevURL  = "dev-url"
	migrateDiffFlagTo      = "to"
	migrateDiffFlagVerbose = "verbose"
)

var (
	// MigrateFlags are the flags used in MigrateCmd (and sub-commands).
	MigrateFlags struct {
		DirURL  string
		DevURL  string
		ToURL   string
		Schemas []string
		Force   bool
		Verbose bool
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
	// MigrateDiffCmd represents the 'atlas migrate diff' subcommand.
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
		Short: "Hash (re-)creates an integrity hash file for the migration directory.",
		Long: `'atlas migrate hash' computes the integrity hash sum of the migration directory and stores it in the atlas.sum file.
This command should be used whenever a manual change in the migration directory was made.`,
		Example: `  atlas migrate hash --force`,
		RunE:    CmdMigrateHashRun,
	}
	// MigrateNewCmd represents the migrate new command.
	MigrateNewCmd = &cobra.Command{
		Use:     "new",
		Short:   "Creates a new empty migration file in the migration directory.",
		Long:    `'atlas migrate new' creates a new migration according to the configured formatter without any statements in it.`,
		Example: `  atlas migrate new my-new-migration`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    CmdMigrateNewRun,
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
	Root.AddCommand(MigrateCmd)
	MigrateCmd.AddCommand(MigrateDiffCmd)
	MigrateCmd.AddCommand(MigrateHashCmd)
	MigrateCmd.AddCommand(MigrateNewCmd)
	MigrateCmd.AddCommand(MigrateValidateCmd)
	// Global flags.
	MigrateCmd.PersistentFlags().StringVarP(&MigrateFlags.DirURL, migrateFlagDir, "", "file://migrations", "select migration directory using DSN format")
	MigrateCmd.PersistentFlags().StringSliceVarP(&MigrateFlags.Schemas, migrateFlagSchema, "", nil, "set schema names")
	MigrateCmd.PersistentFlags().BoolVarP(&MigrateFlags.Force, migrateFlagForce, "", false, "force a command to run on a broken migration directory state")
	MigrateCmd.PersistentFlags().SortFlags = false
	// Diff flags.
	MigrateDiffCmd.Flags().StringVarP(&MigrateFlags.DevURL, migrateDiffFlagDevURL, "", "", "[driver://username:password@address/dbname?param=value] select a data source using the DSN format")
	MigrateDiffCmd.Flags().StringVarP(&MigrateFlags.ToURL, migrateDiffFlagTo, "", "", "[driver://username:password@address/dbname?param=value] select a data source using the DSN format")
	MigrateDiffCmd.Flags().BoolVarP(&MigrateFlags.Verbose, migrateDiffFlagVerbose, "", false, "enable verbose logging")
	MigrateDiffCmd.Flags().SortFlags = false
	cobra.CheckErr(MigrateDiffCmd.MarkFlagRequired(migrateDiffFlagDevURL))
	cobra.CheckErr(MigrateDiffCmd.MarkFlagRequired(migrateDiffFlagTo))
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
			return err
		}
		// If unlocking fails notify the user about it.
		defer cobra.CheckErr(unlock())
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
	// Plan the changes and create a new migration file.
	pl := migrate.NewPlanner(dev.Driver, dir)
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
}

// CmdMigrateNewRun is the command executed when running the CLI with 'migrate new' args.
func CmdMigrateNewRun(_ *cobra.Command, args []string) error {
	dir, err := dir()
	if err != nil {
		return err
	}
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	return migrate.NewPlanner(nil, dir).WritePlan(&migrate.Plan{Name: name})
}

// CmdMigrateHashRun is the command executed when running the CLI with 'migrate hash' args.
func CmdMigrateHashRun(*cobra.Command, []string) error {
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
func to(ctx context.Context, client *sqlclient.Client) (migrate.StateReader, error) {
	parts := strings.SplitN(MigrateFlags.ToURL, "://", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid driver url %q", MigrateFlags.ToURL)
	}
	schemas := MigrateFlags.Schemas
	switch parts[0] {
	case "file": // hcl file
		f, err := ioutil.ReadFile(parts[1])
		if err != nil {
			return nil, err
		}
		realm := &schema.Realm{}
		if err := client.Eval(f, realm, nil); err != nil {
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
		if norm, ok := client.Driver.(schema.Normalizer); ok {
			realm, err = norm.NormalizeRealm(ctx, realm)
			if err != nil {
				return nil, err
			}
		}
		return migrate.Realm(realm), nil
	default: // database connection
		client, err := sqlclient.Open(ctx, MigrateFlags.ToURL)
		if err != nil {
			return nil, err
		}
		if client.URL.Schema != "" {
			schemas = append(schemas, client.URL.Schema)
		}
		return struct {
			migrate.StateReader
			io.Closer
		}{
			Closer:      client,
			StateReader: migrate.Conn(client, &schema.InspectRealmOption{Schemas: schemas}),
		}, nil
	}
}
