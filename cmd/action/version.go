package action

import (
	"fmt"

	"ariga.io/atlas/cmd/action/internal/build"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show atlas CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		v, u := parse(build.Version)
		RootCmd.Println(fmt.Sprintf("atlas CLI version %s\n%s", v, u))
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}

// parse returns a user facing version and release notes url
func parse(version string) (string, string) {
	if ok := semver.IsValid(version); !ok {
		return "- development", "https://github.com/ariga/atlas/releases/tag/latest"
	}
	return version, fmt.Sprintf("https://github.com/ariga/atlas/releases/tag/%s", version)
}
