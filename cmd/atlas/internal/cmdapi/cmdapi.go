// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

// Package cmdapi holds the atlas commands used to build
// an atlas distribution.
package cmdapi

import (
	"fmt"
	"os"
	"strings"

	"ariga.io/atlas/cmd/atlas/internal/update"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

var (
	// Root represents the root command when called without any subcommands.
	Root = &cobra.Command{
		Use:          "atlas",
		Short:        "A database toolkit.",
		SilenceUsage: true,
	}

	// version is the atlas CLI build version
	// Should be set by build script "-X 'ariga.io/atlas/cmd/action.version=${version}'"
	version string

	// GlobalFlags contains flags common to many Atlas sub-commands.
	GlobalFlags struct {
		// SelectedEnv contains the environment selected from the active
		// project via the --env flag.
		SelectedEnv string
		// Vars contains the input variables passed from the CLI to
		// Atlas DDL or project files.
		Vars map[string]string
	}

	// schemaCmd represents the subcommand 'atlas version'.
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Prints this Atlas CLI version information.",
		Run: func(cmd *cobra.Command, args []string) {
			v, u := parse(version)
			Root.Printf("atlas version %s\n%s\n", v, u)
		},
	}

	// EnvCmd represents the subcommand 'atlas env'.
	EnvCmd = &cobra.Command{
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

// CheckForUpdate exposes internal update logic to CLI.
func CheckForUpdate() {
	update.Check(version, Root.PrintErrln)
}

func init() {
	Root.AddCommand(EnvCmd)
	Root.AddCommand(schemaCmd)
	Root.AddCommand(versionCmd)
}

// receivesEnv configures cmd to receive the common '--env' flag.
func receivesEnv(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&GlobalFlags.SelectedEnv, "env", "", "", "set which env from the project file to use")
	cmd.PersistentFlags().StringToStringVarP(&GlobalFlags.Vars, varFlag, "", nil, "input variables")
}

// inputValsFromEnv populates GlobalFlags.Vars from the active environment. If we are working
// inside a project, the "var" flag is not propagated to the schema definition. Instead, it
// is used to evaluate the project file which can pass input values via the "values" block
// to the schema.
func inputValsFromEnv(cmd *cobra.Command) error {
	activeEnv, err := selectEnv(GlobalFlags.SelectedEnv)
	if err != nil {
		return err
	}
	if fl := cmd.Flag(varFlag); fl == nil {
		return nil
	}
	values, err := activeEnv.asMap()
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}
	pairs := make([]string, 0, len(values))
	for k, v := range values {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	vars := strings.Join(pairs, ",")
	if err := cmd.Flags().Set(varFlag, vars); err != nil {
		return err
	}
	return nil
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
