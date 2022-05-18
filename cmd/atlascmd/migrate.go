// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package atlascmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"ariga.io/atlas/sql/sqltool"
	"github.com/spf13/pflag"

	"github.com/spf13/cobra"
)

const (
	migrateFlagDevURL      = "dev-url"
	migrateFlagDir         = "dir"
	migrateFlagForce       = "force"
	migrateFlagFormat      = "format"
	migrateFlagSchema      = "schema"
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
		Format  string
		Force   bool
		Verbose bool
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
				dir, err := dir()
				if err != nil {
					return err
				}
				if err := migrate.Validate(dir); err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), `You have a checksum error in your migration directory.
This happens if you manually create or edit a migration file.
Please check your migration files and run

'atlas migrate hash --force'

to re-hash the contents and resolve the error

`)
					cmd.SilenceUsage = true
					return err
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
		Args:    cobra.MaximumNArgs(1),
		PreRunE: migrateFlagsFromEnv,
		RunE:    CmdMigrateDiffRun,
	}
	// MigrateHashCmd represents the migrate hash command.
	MigrateHashCmd = &cobra.Command{
		Use:   "hash",
		Short: "Hash (re-)creates an integrity hash file for the migration directory.",
		Long: `'atlas migrate hash' computes the integrity hash sum of the migration directory and stores it in the atlas.sum file.
This command should be used whenever a manual change in the migration directory was made.`,
		Example: `  atlas migrate hash --force`,
		PreRunE: migrateFlagsFromEnv,
		RunE:    CmdMigrateHashRun,
	}
	// MigrateNewCmd represents the migrate new command.
	MigrateNewCmd = &cobra.Command{
		Use:     "new",
		Short:   "Creates a new empty migration file in the migration directory.",
		Long:    `'atlas migrate new' creates a new migration according to the configured formatter without any statements in it.`,
		Example: `  atlas migrate new my-new-migration`,
		Args:    cobra.MaximumNArgs(1),
		PreRunE: migrateFlagsFromEnv,
		RunE:    CmdMigrateNewRun,
	}
	// MigrateValidateCmd represents the migrate validate command.
	MigrateValidateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validates the migration directories checksum and SQL statements.",
		Long: `'atlas migrate validate' computes the integrity hash sum of the migration directory and compares it to 
the atlas.sum file. If there is a mismatch it will be reported. If the --dev-url flag is given, the migration files are 
executed on the connected database in order to validate SQL semantics.`,
		Example: `  atlas migrate validate
  atlas migrate validate --dir /path/to/migration/directory
  atlas migrate validate --dir /path/to/migration/directory --dev-url mysql://user:pass@localhost:3306/dev`,
		PreRunE: migrateFlagsFromEnv,
		RunE:    CmdMigrateValidateRun,
	}
)

func init() {
	// Add sub-commands.
	Root.AddCommand(MigrateCmd)
	MigrateCmd.AddCommand(MigrateDiffCmd)
	MigrateCmd.AddCommand(MigrateHashCmd)
	MigrateCmd.AddCommand(MigrateNewCmd)
	MigrateCmd.AddCommand(MigrateValidateCmd)
	// Reusable flags.
	devURL := func(set *pflag.FlagSet) {
		set.StringVarP(&MigrateFlags.DevURL, migrateFlagDevURL, "", "", "[driver://username:password@address/dbname?param=value] select a data source using the URL format")
	}
	// Global flags.
	MigrateCmd.PersistentFlags().StringVarP(&MigrateFlags.DirURL, migrateFlagDir, "", "file://migrations", "select migration directory using URL format")
	MigrateCmd.PersistentFlags().StringSliceVarP(&MigrateFlags.Schemas, migrateFlagSchema, "", nil, "set schema names")
	MigrateCmd.PersistentFlags().StringVarP(&MigrateFlags.Format, migrateFlagFormat, "", formatAtlas, "set migration file format")
	MigrateCmd.PersistentFlags().BoolVarP(&MigrateFlags.Force, migrateFlagForce, "", false, "force a command to run on a broken migration directory state")
	MigrateCmd.PersistentFlags().SortFlags = false
	// Diff flags.
	devURL(MigrateDiffCmd.Flags())
	MigrateDiffCmd.Flags().StringVarP(&MigrateFlags.ToURL, migrateDiffFlagTo, "", "", "[driver://username:password@address/dbname?param=value] select a data source using the URL format")
	MigrateDiffCmd.Flags().BoolVarP(&MigrateFlags.Verbose, migrateDiffFlagVerbose, "", false, "enable verbose logging")
	MigrateDiffCmd.Flags().SortFlags = false
	cobra.CheckErr(MigrateDiffCmd.MarkFlagRequired(migrateFlagDevURL))
	cobra.CheckErr(MigrateDiffCmd.MarkFlagRequired(migrateDiffFlagTo))
	// Validate flags.
	devURL(MigrateValidateCmd.Flags())
	receivesEnv(MigrateCmd)
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
	f, err := formatter()
	if err != nil {
		return err
	}
	// Plan the changes and create a new migration file.
	pl := migrate.NewPlanner(dev.Driver, dir, migrate.WithFormatter(f))
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

// CmdMigrateHashRun is the command executed when running the CLI with 'migrate hash' args.
func CmdMigrateHashRun(_ *cobra.Command, _ []string) error {
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

// CmdMigrateNewRun is the command executed when running the CLI with 'migrate new' args.
func CmdMigrateNewRun(_ *cobra.Command, args []string) error {
	dir, err := dir()
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
	return migrate.NewPlanner(nil, dir, migrate.WithFormatter(f)).WritePlan(&migrate.Plan{Name: name})
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
	dir, err := dir()
	if err != nil {
		return err
	}
	ex, err := migrate.NewExecutor(dev.Driver, dir, migrate.NoopRevisionReadWriter{})
	if err != nil {
		return err
	}
	if _, err := ex.ReadState(cmd.Context()); err != nil && !errors.Is(err, migrate.ErrNoPendingFiles) {
		return fmt.Errorf("replaying the migration directory: %w", err)
	}
	return nil
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

const (
	formatAtlas         = "atlas"
	formatGolangMigrate = "golang-migrate"
	formatGoose         = "goose"
	formatFlyway        = "flyway"
	formatLiquibase     = "liquibase"
)

func formatter() (migrate.Formatter, error) {
	switch MigrateFlags.Format {
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
	default:
		return nil, fmt.Errorf("unknown format %q", MigrateFlags.Format)
	}
}

func migrateFlagsFromEnv(cmd *cobra.Command, _ []string) error {
	activeEnv, err := selectEnv(selectedEnv)
	if err != nil {
		return err
	}
	if err := maySetFlag(cmd, migrateFlagDevURL, activeEnv.DevURL); err != nil {
		return err
	}
	if err := maySetFlag(cmd, migrateFlagFormat, activeEnv.MigrationDir.Format); err != nil {
		return err
	}
	// Transform "src" to a URL.
	toURL := activeEnv.Source
	if toURL != "" {
		if toURL, err = filepath.Abs(activeEnv.Source); err != nil {
			return fmt.Errorf("finding abs path to source: %q: %w", activeEnv.Source, err)
		}
		toURL = "file://" + toURL
	}
	if err := maySetFlag(cmd, migrateDiffFlagTo, toURL); err != nil {
		return err
	}
	if s := "[" + strings.Join(activeEnv.Schemas, "") + "]"; len(activeEnv.Schemas) > 0 {
		if err := maySetFlag(cmd, migrateFlagSchema, s); err != nil {
			return err
		}
	}
	return nil
}
