package action

import (
	"fmt"
	"path"
	"strings"

	"ariga.io/atlas/cmd/action/internal/build"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

var versionCmd = &cobra.Command{
	Use:    "version",
	Hidden: true,
	Run:    cmdVersionRun,
}

func init() {
	RootCmd.AddCommand(versionCmd)
	RootCmd.Version = build.Version

}

func cmdVersionRun(_ *cobra.Command, _ []string) {
	var (
		b strings.Builder
		v = strings.TrimPrefix(build.Version, "v")
	)
	fmt.Fprintf(&b, "atlas version %s", v)
	if build.Date != "" {
		fmt.Fprintf(&b, " (%s)", build.Date)
	}
	release := path.Join("tag", build.Version)
	if !semver.IsValid(build.Version) {
		release = "latest"
	}
	fmt.Fprintf(&b, "\nhttps://github.com/ariga/atlas/releases/%s\n", release)
	RootCmd.Print(b.String())
}
