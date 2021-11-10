package main

import (
	"context"
	"io/ioutil"

	"ariga.io/atlas/sql/schema"

	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"github.com/manifoldco/promptui"
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
			d, err := defaultMux.OpenAtlas(applyFlags.dsn)
			cobra.CheckErr(err)
			u := schemaUnmarshal{unmarshalSpec: d.UnmarshalSpec, unmarshaler: schemahcl.Unmarshal}
			applyRun(d, &u, applyFlags.dsn, applyFlags.file)
		},
		Example: `
atlas schema apply -d mysql://user:pass@tcp(localhost:3306)/dbname -f atlas.hcl
atlas schema apply --dsn postgres://user:pass@tcp(host:port)/dbname -f atlas.hcl`,
	}
)

const (
	answerApply = "Apply"
	answerAbort = "Abort"
)

func init() {
	schemaCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVarP(&applyFlags.dsn, "dsn", "d", "", "[driver://username:password@protocol(address)/dbname?param=value] Select data source using the dsn format")
	applyCmd.Flags().StringVarP(&applyFlags.file, "file", "f", "", "[/path/to/file] file containing schema")
	cobra.CheckErr(applyCmd.MarkFlagRequired("dsn"))
	cobra.CheckErr(applyCmd.MarkFlagRequired("file"))
}

func applyRun(d *Driver, u schemaUnmarshaler, dsn string, file string) {
	ctx := context.Background()
	name, err := schemaNameFromDSN(dsn)
	cobra.CheckErr(err)
	s, err := d.InspectSchema(ctx, name, nil)
	cobra.CheckErr(err)
	f, err := ioutil.ReadFile(file)
	cobra.CheckErr(err)
	var desired schema.Schema
	err = u.unmarshal(f, &desired)
	cobra.CheckErr(err)
	changes, err := d.SchemaDiff(s, &desired)
	cobra.CheckErr(err)
	prompt := promptui.Select{
		Label: "Are you sure?",
		Items: []string{answerApply, answerAbort},
	}
	_, result, err := prompt.Run()
	cobra.CheckErr(err)
	if result == answerApply {
		err = d.Exec(ctx, changes)
		cobra.CheckErr(err)
	}
}
