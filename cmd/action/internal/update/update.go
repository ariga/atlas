package update

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"ariga.io/atlas/cmd/action/internal/build"
)

type (
	reason string
	update struct {
		shouldNotify bool
		version      string
		reason       reason
	}
	Store struct {
		Version string `hcl:"version"`
		URL     string `hcl:"url"`
		Time    string `hcl:"time"`
	}
)

const (
	reasonPollingInterval reason = "not enough time between checks - 24 hours"
	reasonNoVersionUpdate reason = "no version update"
	reasonVersionUpdate   reason = "version update"
)

// CheckForUpdate implements a notification to the user when a later release is available
// Logic is based on similar features in other open source CLIs (For example GitHub):
// 1. Check release file ~/.local/atlas/release.hcl for latest known release and when was it polled
// 2. Poll GitHub public CLI for latest release if last poll was less than 24h
// 3. Store a release file in  with the data
// 4. If current build version, that is not development, is lower than the latest release, notify user
func CheckForUpdate() {
	shouldUpdate(build.Version, "myHome")
}

func shouldUpdate(version string, path string) (*update, error) {
	r, err := localStore(path)
	if err != nil {
		return nil, fmt.Errorf("update: failed to load local release file %s", err)
	}
	return nil, nil
}

func localStore(path string) (*Store, error) {
	p := filepath.Join(path, "release.hcl")
	f, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}

}
