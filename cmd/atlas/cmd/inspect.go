package cmd

import (
	"fmt"
	"log"

	"ariga.io/atlas/cmd/atlas/cmd/internal/driver"
	"github.com/spf13/cobra"
)

var dsn string

// inspectCmd represents the inspect command
var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect atlas schema.",
	Long:  `Inspect atlas schema.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("inspect called")
		fmt.Println(dsn)
		a, err := driver.NewAtlas(dsn)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(a)
	},
	Example: `
atlas schema inspect -d mysql://user:pass@host:port/dbname
atlas schema inspect -d postgres://user:pass@host:port/dbname
atlas schema inspect --dsn sqlite3://path/to/dbname.sqlite3`,
}

func init() {
	schemaCmd.AddCommand(inspectCmd)
	inspectCmd.Flags().StringVarP(&dsn, "dsn", "d", "", "Select data source using the dsn format [driver+transport://user:pass@host/dbname?opt1=a&opt2=b]")
	_ = inspectCmd.MarkFlagRequired("dsn")
}
