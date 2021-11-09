package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	cobra.OnInitialize(initConfig)
	cobra.CheckErr(rootCmd.Execute())
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "atlas",
	Short: "Work with any data source from the command line.",
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	rootCmd.SetOut(os.Stdout)
}
