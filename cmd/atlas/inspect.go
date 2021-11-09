package main

import (
	"context"

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
		Run: func(cmd *cobra.Command, args []string) {
			a, err := defaultMux.OpenAtlas(inspectFlags.dsn)
			cobra.CheckErr(err)
			m := schemaMarshal{a, schemahcl.Marshal}
			inspectRun(a, &m, inspectFlags.dsn)
		},
		Example: `
atlas schema inspect -d mysql://user:pass@host:port/dbname
atlas schema inspect --dsn postgres://user:pass@host:port/dbname`,
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

func inspectRun(a *Driver, m schemaMarshaler, dsn string) {
	ctx := context.Background()
	name, err := schemaNameFromDSN(dsn)
	cobra.CheckErr(err)
	s, err := a.Inspector.InspectSchema(ctx, name, nil)
	cobra.CheckErr(err)
	ddl, err := m.marshal(s)
	cobra.CheckErr(err)
	schemaCmd.Print(string(ddl))
}
