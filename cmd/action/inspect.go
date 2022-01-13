package action

import (
	"context"

	"github.com/spf13/cobra"
)

var (
	// InspectFlags are the flags used in Inspect command.
	InspectFlags struct {
		DSN     string
		Web     bool
		Address string
	}
	// InspectCmd represents the inspect command.
	InspectCmd = &cobra.Command{
		Use:   "inspect",
		Short: "Inspect an atlas schema",
		Run:   CmdInspectRun,
		Example: `
atlas schema inspect -d "mysql://user:pass@tcp(localhost:3306)/dbname"
atlas schema inspect -d "mariadb://user:pass@tcp(localhost:3306)/dbname"
atlas schema inspect --dsn "postgres://user:pass@host:port/dbname"
atlas schema inspect -d "sqlite://file:ex1.db?_fk=1"`,
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
	InspectCmd.Flags().BoolVarP(&InspectFlags.Web, "web", "w", false, "Open in a local Atlas UI")
	InspectCmd.Flags().StringVarP(&InspectFlags.Address, "address", "", "127.0.0.1:5800", "used with -w, local address to bind the server to")
	cobra.CheckErr(InspectCmd.MarkFlagRequired("dsn"))
}

// CmdInspectRun is the command used when running CLI.
func CmdInspectRun(_ *cobra.Command, _ []string) {
	if InspectFlags.Web {
		schemaCmd.PrintErrln("The Atlas UI is not available in this release.")
		return
	}
	d, err := defaultMux.OpenAtlas(InspectFlags.DSN)
	cobra.CheckErr(err)
	inspectRun(d, InspectFlags.DSN)
}

func inspectRun(d *Driver, dsn string) {
	ctx := context.Background()
	name, err := SchemaNameFromDSN(dsn)
	cobra.CheckErr(err)
	s, err := d.InspectSchema(ctx, name, nil)
	cobra.CheckErr(err)
	ddl, err := d.MarshalSpec(s)
	cobra.CheckErr(err)
	schemaCmd.Print(string(ddl))
}
