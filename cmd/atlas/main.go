package main

import (
	"context"
	"os"
	"os/signal"

	"ariga.io/atlas/cmd/action"
	_ "ariga.io/atlas/cmd/internal/docker"
	_ "ariga.io/atlas/sql/mysql"
	_ "ariga.io/atlas/sql/postgres"
	_ "ariga.io/atlas/sql/sqlite"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	action.RootCmd.SetOut(os.Stdout)
	err := action.RootCmd.ExecuteContext(ctx)
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
