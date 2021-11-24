package base

import "github.com/spf13/cobra"

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "atlas",
	Short: "A database toolkit.",
}
