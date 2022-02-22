// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"github.com/spf13/cobra"
)

// schemaCmd represents the schema command
var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Work with atlas schemas.",
	Long:  "The `atlas schema` command groups subcommands for working with Atlas schemas.",
}

func init() {
	RootCmd.AddCommand(schemaCmd)
}
