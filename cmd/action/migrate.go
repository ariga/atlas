package action

import (
	"github.com/spf13/cobra"
)

const (
	migrateFlagDir        = "dir"
	migrateDiffFlagDevURL = "dev-url"
	migrateDiffFlagTo     = "to"
)

var (
	// MigrateFlags are the flags used in MigrateCmd (and sub-commands).
	MigrateFlags struct {
		DirURL string
		DevURL string
		ToURL  string
	}
	// MigrateCmd represents the migrate command. It wraps several other sub-commands.
	MigrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Manage versioned migration files",
		Long:  "`atlas migrate`" + ` wraps several sub-commands for migration management.`,
	}
	// MigrateDiffCmd represents the migrate diff command.
	MigrateDiffCmd = &cobra.Command{
		Use:   "diff",
		Short: "Compute the diff between the migration directory and a connected database and create a new migration file.",
		Long: "`atlas migrate diff`" + ` uses the dev-database to re-run all migration files in the migration
directory and compares it to a given desired state and create a new migration file containing SQL statements to migrate 
the migration directory state to the desired schema. The desired state can be another connected database or an HCL file.`,
		Example: `  atlas migrate diff --dev-db mysql://user:pass@localhost:3306/dev --to mysql://user:pass@localhost:3306/dbname
  atlas migrate diff --dev-db mysql://user:pass@localhost:3306/dev --to file://atlas.hcl`,
		Run: CmdMigrateDiffRun,
	}
)

func init() {
	RootCmd.AddCommand(MigrateCmd)
	MigrateCmd.AddCommand(MigrateDiffCmd)
	MigrateCmd.PersistentFlags().StringVarP(&MigrateFlags.DirURL, migrateFlagDir, "", "file://migrations", "select migration directory using DSN format")
	MigrateDiffCmd.Flags().StringVarP(&MigrateFlags.DevURL, migrateDiffFlagDevURL, "", "", "[driver://username:password@address/dbname?param=value] select a data source using the DSN format")
	MigrateDiffCmd.Flags().StringVarP(&MigrateFlags.ToURL, migrateDiffFlagTo, "", "", "[driver://username:password@address/dbname?param=value] select a data source using the DSN format")
	MigrateDiffCmd.Flags().SortFlags = false
	cobra.CheckErr(MigrateDiffCmd.MarkFlagRequired(migrateDiffFlagDevURL))
	cobra.CheckErr(MigrateDiffCmd.MarkFlagRequired(migrateDiffFlagTo))
}

func CmdMigrateDiffRun(*cobra.Command, []string) {
	panic("unimplemented")
}
