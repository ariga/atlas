package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

type tInspectFlags struct {
	dsn string
}

var (
	inspectFlags tInspectFlags

	// inspectCmd represents the inspect command
	inspectCmd = &cobra.Command{
		Use:   "inspect",
		Short: "Inspect atlas schema.",
		Long:  `Inspect atlas schema.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("inspect called")
			fmt.Println(inspectFlags)
		},
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
	_ = inspectCmd.MarkFlagRequired("dsn")
}
