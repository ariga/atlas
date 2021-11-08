package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

var (
	inspectFlags struct {
		dsn string
	}
	// inspectCmd represents the inspect command.
	inspectCmd = &cobra.Command{
		Use:   "inspect",
		Short: "Inspect atlas schema.",
		Run:   func(cmd *cobra.Command, args []string) { runInspect(inspectFlags.dsn) },
		Example: `
atlas schema inspect -d mysql://user:pass@host:port/dbname
atlas schema inspect -d postgres://user:pass@host:port/dbname
atlas schema inspect --dsn sqlite3://path/to/dbname.sqlite3`,
	}
)

func init() {
	schemaCmd.AddCommand(inspectCmd)
	schemaCmd.SetOut(os.Stdout)
	inspectCmd.Flags().StringVarP(
		&inspectFlags.dsn,
		"dsn",
		"d",
		"",
		"[driver+transport://user:pass@host/dbname?opt1=a&opt2=b] Select data source using the dsn format",
	)
	cobra.CheckErr(inspectCmd.MarkFlagRequired("dsn"))
}

func runInspect(dsn string) {
	ctx := context.Background()
	a, err := defaultMux.OpenAtlas(dsn)
	cobra.CheckErr(err)
	schema, err := a.Inspector.InspectSchema(ctx, "todo", nil)
	cobra.CheckErr(err)
	schemaCmd.Println(schema)
}
