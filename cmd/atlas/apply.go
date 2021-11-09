package main

import (
	"context"
	"io/ioutil"

	"ariga.io/atlas/sql/schema"

	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/mysql"
	"github.com/spf13/cobra"
)

var (
	applyFlags struct {
		dsn  string
		file string
	}
	// applyCmd represents the apply command.
	applyCmd = &cobra.Command{
		Use:   "apply",
		Short: "Apply atlas schema to data source.",
		Run: func(cmd *cobra.Command, args []string) {
			a, err := defaultMux.OpenAtlas(applyFlags.dsn)
			cobra.CheckErr(err)
			applyRun(a, applyFlags.dsn, applyFlags.file)
		},
		Example: `
atlas schema apply -d mysql://user:pass@host:port/dbname -f atlas.hcl
atlas schema apply --dsn "postgres://user:pass@host:port/dbname" -f atlas.hcl`,
	}
)

func init() {
	schemaCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVarP(&applyFlags.dsn, "dsn", "d", "", "[driver+transport://user:pass@host/dbname?opt1=a&opt2=b] Select data source using the dsn format")
	applyCmd.Flags().StringVarP(&applyFlags.file, "file", "f", "", "[/path/to/file] file containing schema")
	cobra.CheckErr(applyCmd.MarkFlagRequired("dsn"))
	cobra.CheckErr(applyCmd.MarkFlagRequired("file"))
}

func applyRun(d *Driver, dsn string, file string) {
	ctx := context.Background()
	name, err := schemaNameFromDSN(dsn)
	cobra.CheckErr(err)
	s, err := d.Inspector.InspectSchema(ctx, name, nil)
	cobra.CheckErr(err)
	f, err := ioutil.ReadFile(file)
	cobra.CheckErr(err)
	var desired schema.Schema
	err = mysql.UnmarshalSpec(f, schemahcl.Unmarshal, &desired)
	changes, err := d.Differ.SchemaDiff(s, s)
	cobra.CheckErr(err)
	err = d.Execer.Exec(ctx, changes)
	cobra.CheckErr(err)
}
