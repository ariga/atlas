
package cmd

import (
	"github.com/spf13/cobra"
)

// schemaCmd represents the schema command
var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Work with atlas schemas.",
	Long: `Provides ability to interact with schema and datasource.`,
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}
