package update

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"golang.org/x/mod/semver"
)

type (
	reason string
	update struct {
		shouldNotify bool
		version      string
		reason       reason
	}
	// Store is where the latest release data is stored on CLI host
	Store struct {
		Version        string    `hcl:"version"`
		URL            string    `hcl:"url"`
		StoreUpdatedAt time.Time `hcl:"updated"`
	}

	LatestRelease struct {
		Version string `json:"tag_name"`
		URL     string `json:"html_url"`
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
func CheckForUpdate(version string) {
	if _, ok := os.LookupEnv("ATLAS_NO_UPDATE_NOTIFIER"); ok {
		return
	}
	shouldUpdate(version, "myHome", latestReleaseFromGithub)
}

func shouldUpdate(version string, path string, latestRelease func() (LatestRelease, error)) (*update, error) {
	r, _ := localStore(path)
	if shouldSkip(r) {
		return &update{reason: reasonPollingInterval, shouldNotify: false}, nil
	}
	l, err := latestRelease()
	if err != nil {
		return nil, err
	}
	if err := saveStore(path, l); err != nil {
		return nil, err
	}
	if semver.Compare(version, l.Version) != -1 {
		return &update{
			shouldNotify: false,
			version:      version,
			reason:       reasonNoVersionUpdate,
		}, nil
	}
	return &update{
		shouldNotify: true,
		version:      l.Version,
		reason:       reasonVersionUpdate,
	}, nil
}

func latestReleaseFromGithub() (LatestRelease, error) {
	return LatestRelease{}, nil
}

func shouldSkip(r *Store) bool {
	if r == nil {
		return false
	}
	return time.Since(r.StoreUpdatedAt).Hours() < 24
}

func localStore(path string) (*Store, error) {
	p := filepath.Join(path, "release.hcl")
	body, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var v Store
	parser := hclparse.NewParser()
	srcHCL, diag := parser.ParseHCL(body, path)
	if diag.HasErrors() {
		return nil, fmt.Errorf("update: failed to parse hcl")
	}
	if srcHCL == nil {
		return nil, fmt.Errorf("update: no HCL syntax found in body")
	}
	err = gohcl.DecodeBody(srcHCL.Body, nil, v)
	if err != nil {
		return nil, fmt.Errorf("update: failed to decode body %s", err)
	}
	return &v, nil
}

func saveStore(path string, l LatestRelease) error {
	f := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(&l, f.Body())
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(path, "release.hcl"), f.Bytes(), 0600)
	return err
}
