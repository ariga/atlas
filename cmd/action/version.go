package action

import (
	"fmt"
	"regexp"
	"strings"

	"ariga.io/atlas/cmd/action/internal/build"
	"github.com/spf13/cobra"
)

var (
	versionCmd = &cobra.Command{
		Use:    "version",
		Hidden: true,
		Run:    cmdVersionRun,
	}
)

func init() {
	RootCmd.AddCommand(versionCmd)
	RootCmd.Version = build.Version

}

func cmdVersionRun(cmd *cobra.Command, args []string) {
	RootCmd.Print(format(build.Version, build.Date))
}

func format(version, buildDate string) string {
	version = strings.TrimPrefix(version, "v")
	var dateStr string
	if buildDate != "" {
		dateStr = fmt.Sprintf(" (%s)", buildDate)
	}
	return fmt.Sprintf("atlas version %s%s\n%s\n", version, dateStr, changelogURL(version))
}

func changelogURL(version string) string {
	path := "https://github.com/ariga/atlas"
	r := regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[\w.]+)?$`)
	if !r.MatchString(version) {
		return fmt.Sprintf("%s/releases/latest", path)
	}
	url := fmt.Sprintf("%s/releases/tag/v%s", path, strings.TrimPrefix(version, "v"))
	return url
}
