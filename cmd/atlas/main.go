package main

import (
	"os"

	"ariga.io/atlas/cmd/action"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

func main() {
	cobra.OnInitialize(initConfig)
	err := action.RootCmd.Execute()
	// Print error from command
	if err != nil {
		action.RootCmd.PrintErrln("Error:", err)
	}
	// Check for update
	action.CheckForUpdate()
	// Exit code according to command success
	if err != nil {
		os.Exit(1)
	}

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	action.RootCmd.SetOut(os.Stdout)
}
