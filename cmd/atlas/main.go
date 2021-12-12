package main

import (
	"os"

	"ariga.io/atlas/cmd/action"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
)

func main() {
	cobra.OnInitialize(initConfig)
	cobra.CheckErr(action.RootCmd.Execute())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	action.RootCmd.SetOut(os.Stdout)
}
