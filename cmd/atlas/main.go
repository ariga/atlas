package main

import (
	"os"

	"ariga.io/atlas/cmd/action"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	action.RootCmd.SetOut(os.Stdout)
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
