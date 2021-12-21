package build

import (
	"runtime/debug"
)

// Version is dynamically set by the release script.
var Version = "DEV"

// Date is dynamically set at build time in the release script.
var Date = "" // YYYY-MM-DD

func init() {
	if Version == "DEV" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "(devel)" {
			Version = info.Main.Version
		}
	}
}
