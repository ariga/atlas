// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"ariga.io/atlas/cmd/internal/update"

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
