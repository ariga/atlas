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
	if ok := semver.IsValid(version); !ok {
		v := "- development"
		s := strings.Split(version, "_")
		if len(s) != 0 && s[len(s)-1] == "canary" {
			v = version
		}
		return v, "https://github.com/ariga/atlas/releases/tag/latest"
	}
	return version, fmt.Sprintf("https://github.com/ariga/atlas/releases/tag/%s", version)
}
