// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

// version is the atlas CLI build version
// Should be set by build script "-X 'ariga.io/atlas/cmd/action.version=${version}'"
var version string

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints this Atlas CLI version information.",
	Run: func(cmd *cobra.Command, args []string) {
		v, u := parse(version)
		RootCmd.Println(fmt.Sprintf("atlas CLI version %s\n%s", v, u))
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
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
