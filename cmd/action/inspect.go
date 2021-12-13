package action

import (
	"context"

	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"github.com/spf13/cobra"
)

var (
	// InspectFlags are the flags used in Inspect command.
	InspectFlags struct {
		DSN string
		Web bool
	}
	// InspectCmd represents the inspect command.
	InspectCmd = &cobra.Command{
		Use:   "inspect",
		Short: "Inspect an atlas schema",
		Run:   CmdInspectRun,
		Example: `
atlas schema inspect -d mysql://user:pass@tcp(localhost:3306)/dbname
atlas schema inspect --dsn postgres://user:pass@host:port/dbname`,
	}
)

func init() {
	schemaCmd.AddCommand(InspectCmd)
	InspectCmd.Flags().StringVarP(
		&InspectFlags.DSN,
		"dsn",
		"d",
		"",
		"[driver://username:password@protocol(address)/dbname?param=value] Select data source using the dsn format",
	)
	InspectCmd.Flags().BoolVarP(&InspectFlags.Web, "web", "w", false, "open in UI server")
	cobra.CheckErr(InspectCmd.MarkFlagRequired("dsn"))
}

// CmdInspectRun is the command used when running CLI.
func CmdInspectRun(cmd *cobra.Command, args []string) {
	d, err := defaultMux.OpenAtlas(InspectFlags.DSN)
	cobra.CheckErr(err)
	m := schemaMarshal{marshalSpec: d.MarshalSpec, marshaler: schemahcl.Marshal}
	inspectRun(d, &m, InspectFlags.DSN)
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
