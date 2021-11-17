package base

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
		Short: "Inspect an atlas schema",
		Run: func(cmd *cobra.Command, args []string) {
			d, err := defaultMux.OpenAtlas(inspectFlags.dsn)
			cobra.CheckErr(err)
			m := schemaMarshal{marshalSpec: d.MarshalSpec, marshaler: schemahcl.Marshal}
			inspectRun(d, &m, inspectFlags.dsn)
		},
		Example: `
atlas schema inspect -d mysql://user:pass@tcp(localhost:3306)/dbname
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
		"[driver://username:password@protocol(address)/dbname?param=value] Select data source using the dsn format",
	)
	cobra.CheckErr(inspectCmd.MarkFlagRequired("dsn"))
}

func inspectRun(d *Driver, m schemaMarshaler, dsn string) {
	ctx := context.Background()
	name, err := schemaNameFromDSN(dsn)
	cobra.CheckErr(err)
	s, err := d.InspectSchema(ctx, name, nil)
	cobra.CheckErr(err)
	ddl, err := m.marshal(s)
	cobra.CheckErr(err)
	schemaCmd.Print(string(ddl))
}
