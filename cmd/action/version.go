package action

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

// version is the atlas CLI build version
// Should be set by build script "-X 'ariga.io/atlas/cmd/action.version=${version}'"
var version string

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show atlas CLI version",
	Long: `Show atlas CLI version

On any command, the CLI will check for updates with the GitHub public API once every 24 hours.
To cancel this behavior, set the environment parameter "ATLAS_NO_UPDATE_NOTIFIER"`,
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
		return "- development", "https://github.com/ariga/atlas/releases/tag/latest"
	}
	return version, fmt.Sprintf("https://github.com/ariga/atlas/releases/tag/%s", version)
}
