package main

import (
	"github.com/spf13/cobra"
)

var (
	inspectFlags = struct {
		dsn string
	}{}

	// inspectCmd represents the inspect command
	inspectCmd = &cobra.Command{
		Use:   "inspect",
		Short: "Inspect atlas schema.",
		Long:  `Inspect atlas schema.`,
		Run:   func(cmd *cobra.Command, args []string) {},
		Example: `
atlas schema inspect -d mysql://user:pass@host:port/dbname
atlas schema inspect -d postgres://user:pass@host:port/dbname
atlas schema inspect --dsn sqlite3://path/to/dbname.sqlite3`,
	}
)

func init() {
	schemaCmd.AddCommand(inspectCmd)
	inspectCmd.Flags().StringVarP(
		&inspectFlags.dsn,
		"dsn",
		"d",
		"",
		"[driver+transport://user:pass@host/dbname?opt1=a&opt2=b] Select data source using the dsn format",
	)
	cobra.CheckErr(inspectCmd.MarkFlagRequired("dsn"))
}
