package main

import (
	"context"
	"os"

	"ariga.io/atlas/schema/schemaspec/schemahcl"
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
		Run:   func(cmd *cobra.Command, args []string) { 
			inspectRun(inspectFlags.dsn)
		},
		Example: `
atlas schema inspect -d mysql://user:pass@host:port/dbname
atlas schema inspect --dsn postgres://user:pass@host:port/dbname`,
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

func inspectRun(dsn string) {
	ctx := context.Background()
	a, err := defaultMux.OpenAtlas(dsn)
	cobra.CheckErr(err)
	name, err := schemaNameFromDSN(dsn)
	cobra.CheckErr(err)
	schema, err := a.Inspector.InspectSchema(ctx, name, nil)
	cobra.CheckErr(err)
	ddl, err := a.MarshalSpec(schema, schemahcl.Marshal)
	cobra.CheckErr(err)
	schemaCmd.Println(string(ddl))
}
