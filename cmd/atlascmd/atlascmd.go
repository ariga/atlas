// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

// Package atlascmd holds the entire Root commands used to build
// an atlas distribution.
package atlascmd

import (
	"fmt"
	"os"
	"strings"

	"ariga.io/atlas/cmd/atlascmd/update"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

var (
	// Root represents the root command when called without any subcommands.
	Root = &cobra.Command{
		Use:   "atlas",
		Short: "A database toolkit.",
	}

	// version is the atlas CLI build version
	// Should be set by build script "-X 'ariga.io/atlas/cmd/action.version=${version}'"
	version string

	// schemaCmd represents the subcommand 'atlas version'.
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Prints this Atlas CLI version information.",
		Run: func(cmd *cobra.Command, args []string) {
			v, u := parse(version)
			Root.Printf("atlas version %s\n%s\n", v, u)
		},
	}

	// envCmd represents the subcommand 'atlas env'.
	envCmd = &cobra.Command{
		Use:   "env",
		Short: "Print atlas environment variables.",
		Long: `'atlas env' prints atlas environment information.

Every set environment param will be printed in the form of NAME=VALUE.

List of supported environment parameters:
* ATLAS_NO_UPDATE_NOTIFIER: On any command, the CLI will check for new releases using the GitHub API.
  This check will happen at most once every 24 hours. To cancel this behavior, set the environment 
  variable "ATLAS_NO_UPDATE_NOTIFIER".`,
		Run: func(cmd *cobra.Command, args []string) {
			keys := []string{update.AtlasNoUpdateNotifier}
			for _, k := range keys {
				if v, ok := os.LookupEnv(k); ok {
					cmd.Println(fmt.Sprintf("%s=%s", k, v))
				}
			}
		},
	}
)

// CheckForUpdate exposes internal update logic to CLI
func CheckForUpdate() {
	update.CheckForUpdate(version, Root.PrintErrln)
}

func init() {
	Root.AddCommand(envCmd)
	Root.AddCommand(schemaCmd)
	Root.AddCommand(versionCmd)
}

// parse returns a user facing version and release notes url
func parse(version string) (string, string) {
	u := "https://github.com/ariga/atlas/releases/latest"
	if ok := semver.IsValid(version); !ok {
		return "- development", u
	}
	s := strings.Split(version, "-")
	if len(s) != 0 && s[len(s)-1] != "canary" {
		u = fmt.Sprintf("https://github.com/ariga/atlas/releases/tag/%s", version)
	}
	return version, u
}
