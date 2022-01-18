package action

import (
	"fmt"
	"os"

	"ariga.io/atlas/cmd/action/internal/update"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Print atlas environment variables.",
	Long: "`atlas env`" + `prints atlas environment information.

Every set environment param will be printed in the form of NAME=VALUE.

List of supported environment parameters:
* *ATLAS_NO_UPDATE_NOTIFIER*: On any command, the CLI will check for new releases using the GitHub API.
  This check will happen at most once every 24 hours. To cancel this behavior, set the environment 
  variable "ATLAS_NO_UPDATE_NOTIFIER".`,
	Run: func(cmd *cobra.Command, args []string) {
		keys := []string{update.AtlasNoUpdateNotifier}
		for _, k := range keys {
			if v, ok := os.LookupEnv(k); ok {
				RootCmd.Println(fmt.Sprintf("%s=%s", k, v))
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(envCmd)
}
