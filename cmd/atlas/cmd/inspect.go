/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var dsn string

// inspectCmd represents the inspect command
var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect Atlas schema.",
	Long:  `Inspect Atlas schema.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("inspect called")
		fmt.Println(dsn)
	},
}

func init() {
	schemaCmd.AddCommand(inspectCmd)
	inspectCmd.Flags().StringVarP(&dsn, "dsn", "d", "[driver+transport://user:pass@host/dbname?opt1=a&opt2=b]", "Select database using the dsn format")
}
