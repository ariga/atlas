package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply atlas schema to data source.",
	Long:  `Apply atlas schema to data source.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("apply called")
	},
	Example: `
atlas schema apply -d mysql://user:pass@host:port/dbname -f atlas.hcl
atlas schema apply -d postgres://user:pass@host:port/dbname -f atlas.hcl
atlas schema apply --dsn sqlite3://path/to/dbname.sqlite3 --file atlas.hcl
`,
}

func init() {
	schemaCmd.AddCommand(applyCmd)
}
