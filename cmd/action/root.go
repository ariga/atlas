package action

import (
	"ariga.io/atlas/cmd/action/internal/update"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "atlas",
	Short: "A database toolkit.",
}

// CheckForUpdate exposes internal update logic to CLI
func CheckForUpdate() {
	update.CheckForUpdate(version, RootCmd.PrintErrln)
}
