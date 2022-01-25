package action

import (
	"context"

	"ariga.io/atlas/sql/schema"
	"github.com/spf13/cobra"
)

var (
	// InspectFlags are the flags used in Inspect command.
	InspectFlags struct {
		DSN    string
		Web    bool
		Addr   string
		Schema []string
	}
	// InspectCmd represents the inspect command.
	InspectCmd = &cobra.Command{
		Use:   "inspect",
		Short: "Inspect an a database's and print its schema in Atlas DDL syntax.",
		Long: "`atlas schema inspect`" + ` connects to the given database and inspects its schema.
It then prints to the screen the schema of that database in Atlas DDL syntax. This output can be 
saved to a file, commonly by redirecting the output to a file named with a ".hcl" suffix:

	atlas schema inspect -d "mysql://user:pass@tcp(localhost:3306)/dbname" > atlas.hcl

This file can then be edited and used with the` + " `atlas schema apply` " + `command to plan
and execute schema migrations against the given database. 
	`,
		Run: CmdInspectRun,
		Example: `
atlas schema inspect -d "mysql://user:pass@tcp(localhost:3306)/dbname"
atlas schema inspect -d "mariadb://user:pass@tcp(localhost:3306)/" --schema=dbnameA,dbnameB -s dbnameC
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
	InspectCmd.Flags().StringVarP(&InspectFlags.Addr, "addr", "", "127.0.0.1:5800", "Used with -w, local address to bind the server to")
	InspectCmd.Flags().StringSliceVarP(&InspectFlags.Schema, "schema", "s", nil, "Set schema name")
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
	schemas := InspectFlags.Schema
	if n, err := SchemaNameFromDSN(dsn); n != "" {
		cobra.CheckErr(err)
		schemas = append(schemas, n)
	}
	s, err := d.InspectRealm(ctx, &schema.InspectRealmOption{
		Schemas: schemas,
	})
	cobra.CheckErr(err)
	ddl, err := d.MarshalSpec(s)
	cobra.CheckErr(err)
	schemaCmd.Print(string(ddl))
}
