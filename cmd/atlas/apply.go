package main

import (
	"context"

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
		Run:   func(cmd *cobra.Command, args []string) { runApply(applyFlags.dsn, applyFlags.file) },
		Example: `
atlas schema apply -d "mysql://user:pass@host:port/dbname" -f atlas.hcl
atlas schema apply -d "postgres://user:pass@host:port/dbname" -f atlas.hcl
atlas schema apply --dsn "sqlite3://path/to/dbname.sqlite3" --file atlas.hcl
`,
	}
)

func init() {
	schemaCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVarP(&applyFlags.dsn, "dsn", "d", "", "[driver+transport://user:pass@host/dbname?opt1=a&opt2=b] Select data source using the dsn format")
	applyCmd.Flags().StringVarP(&applyFlags.file, "file", "f", "", "[/path/to/file] file containing schema")
	cobra.CheckErr(applyCmd.MarkFlagRequired("dsn"))
	cobra.CheckErr(applyCmd.MarkFlagRequired("file"))
}

func runApply(dsn string, file string) {
	ctx := context.Background()
	a, err := defaultMux.OpenAtlas(dsn)
	cobra.CheckErr(err)
	name, err := schemaNameFromDSN(dsn)
	cobra.CheckErr(err)
	schema, err := a.Inspector.InspectSchema(ctx, name, nil)
	cobra.CheckErr(err)
	changes, err := a.Differ.SchemaDiff(schema, schema)
	cobra.CheckErr(err)
	err = a.Execer.Exec(ctx, changes)
	cobra.CheckErr(err)
}
