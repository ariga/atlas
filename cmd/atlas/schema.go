package main

import (
	"github.com/spf13/cobra"
)

// schemaCmd represents the schema command
var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Work with atlas schemas",
	Long:  "Interact with the schema and data source",
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}
