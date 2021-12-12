package main

import (
	"os"

	"ariga.io/atlas/cmd/base"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
)

func main() {
	cobra.OnInitialize(initConfig)
	cobra.CheckErr(base.RootCmd.Execute())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	base.RootCmd.SetOut(os.Stdout)
}
