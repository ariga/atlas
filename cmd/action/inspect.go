// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"context"

	"ariga.io/atlas/sql/schema"

	"github.com/spf13/cobra"
)

var (
	// InspectFlags are the flags used in Inspect command.
	InspectFlags struct {
		URL    string
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

  atlas schema inspect -u "mysql://user:pass@tcp(localhost:3306)/dbname" > atlas.hcl

This file can then be edited and used with the` + " `atlas schema apply` " + `command to plan
and execute schema migrations against the given database. In cases where users wish to inspect
all multiple schemas in a given database (for instance a MySQL server may contain multiple named
databases), omit the relevant part from the url, e.g. "mysql://user:pass@tcp(localhost:3306)/".
To select specific schemas from the databases, users may use the "--schema" (or "-s" shorthand)
flag.
	`,
		Run: CmdInspectRun,
		Example: `  atlas schema inspect -u "mysql://user:pass@tcp(localhost:3306)/dbname"
  atlas schema inspect -u "mariadb://user:pass@tcp(localhost:3306)/" --schema=schemaA,schemaB -s schemaC
  atlas schema inspect --url "postgres://user:pass@host:port/dbname?sslmode=disable"
  atlas schema inspect -u "sqlite://file:ex1.db?_fk=1"`,
	}
)

func init() {
	schemaCmd.AddCommand(InspectCmd)
	InspectCmd.Flags().StringVarP(&InspectFlags.URL, "url", "u", "", "[driver://username:password@protocol(address)/dbname?param=value] Select data source using the url format")
	InspectCmd.Flags().BoolVarP(&InspectFlags.Web, "web", "w", false, "Open in a local Atlas UI")
	InspectCmd.Flags().StringVarP(&InspectFlags.Addr, "addr", "", ":5800", "Used with -w, local address to bind the server to")
	InspectCmd.Flags().StringSliceVarP(&InspectFlags.Schema, "schema", "s", nil, "Set schema name")
	cobra.CheckErr(InspectCmd.MarkFlagRequired("url"))
	dsn2url(InspectCmd, &InspectFlags.URL)
}

// CmdInspectRun is the command used when running CLI.
func CmdInspectRun(_ *cobra.Command, _ []string) {
	if InspectFlags.Web {
		schemaCmd.PrintErrln("The Atlas UI is not available in this release.")
		return
	}
	d, err := DefaultMux.OpenAtlas(InspectFlags.URL)
	cobra.CheckErr(err)
	inspectRun(d, InspectFlags.URL)
}

func inspectRun(d *Driver, url string) {
	ctx := context.Background()
	schemas := InspectFlags.Schema
	if n, err := SchemaNameFromURL(url); n != "" {
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
