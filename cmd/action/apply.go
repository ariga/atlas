package action

import (
	"context"
	"io/ioutil"

	"ariga.io/atlas/sql/schema"

	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var (
	// ApplyFlags are the flags used in Apply command.
	ApplyFlags struct {
		DSN  string
		File string
		Web  bool
	}
	// ApplyCmd represents the apply command.
	ApplyCmd = &cobra.Command{
		Use:   "apply",
		Short: "Apply an atlas schema to a data source",
		Run:   CmdApplyRun,
		Example: `
atlas schema apply -d mysql://user:pass@tcp(localhost:3306)/dbname -f atlas.hcl
atlas schema apply --dsn postgres://user:pass@host:port/dbname -f atlas.hcl`,
	}
)

const (
	answerApply = "Apply"
	answerAbort = "Abort"
)

func init() {
	schemaCmd.AddCommand(ApplyCmd)
	ApplyCmd.Flags().StringVarP(&ApplyFlags.DSN, "dsn", "d", "", "[driver://username:password@protocol(address)/dbname?param=value] Select data source using the dsn format")
	ApplyCmd.Flags().StringVarP(&ApplyFlags.File, "file", "f", "", "[/path/to/file] file containing schema")
	ApplyCmd.Flags().BoolVarP(&ApplyFlags.Web, "web", "w", false, "open in UI server")
	cobra.CheckErr(ApplyCmd.MarkFlagRequired("dsn"))
	cobra.CheckErr(ApplyCmd.MarkFlagRequired("file"))
}

// CmdApplyRun is the command used when running CLI.
func CmdApplyRun(cmd *cobra.Command, args []string) {
	d, err := defaultMux.OpenAtlas(ApplyFlags.DSN)
	cobra.CheckErr(err)
	u := schemaUnmarshal{unmarshalSpec: d.UnmarshalSpec, unmarshaler: schemahcl.Unmarshal}
	applyRun(d, &u, ApplyFlags.DSN, ApplyFlags.File)
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
	if len(changes) == 0 {
		schemaCmd.Println("Schema is synced, no changes to be made")
		return
	}
	schemaCmd.Println("-- Planned Changes:")
	for _, ch := range changes {
		desc, err := changeDescriptor(ctx, ch, d)
		cobra.CheckErr(err)
		schemaCmd.Println("--", desc.typ, ":", desc.subject)
		for _, q := range desc.queries {
			schemaCmd.Println(q, ";")
		}
	}
	prompt := promptui.Select{
		Label: "Are you sure?",
		Items: []string{answerApply, answerAbort},
	}
	_, result, err := prompt.Run()
	cobra.CheckErr(err)
	if result == answerApply {
		err = d.ApplyChanges(ctx, changes)
		cobra.CheckErr(err)
	}
}
