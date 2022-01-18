package action

import (
	"github.com/spf13/cobra"
)

// schemaCmd represents the schema command
var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Work with atlas schemas.",
	Long:  "The `atlas schema` subcommand groups commands for working with Atlas schemas.",
}

func init() {
	RootCmd.AddCommand(schemaCmd)
}
