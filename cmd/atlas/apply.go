package main

import (
	"github.com/spf13/cobra"
)

type tApplyFlags struct {
	dsn  string
	file string
}

var (
	applyFlags tApplyFlags
	// applyCmd represents the apply command
	applyCmd = &cobra.Command{
		Use:   "apply",
		Short: "Apply atlas schema to data source.",
		Long:  `Apply atlas schema to data source.`,
		Run:   func(cmd *cobra.Command, args []string) {},
		Example: `
atlas schema apply -d mysql://user:pass@host:port/dbname -f atlas.hcl
atlas schema apply -d postgres://user:pass@host:port/dbname -f atlas.hcl
atlas schema apply --dsn sqlite3://path/to/dbname.sqlite3 --file atlas.hcl
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
