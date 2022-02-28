// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package update

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchellh/go-homedir"
	"golang.org/x/mod/semver"
)

type (
	// Store is where the latest release data is stored on CLI host.
	Store struct {
		Version   string    `json:"version"`
		URL       string    `json:"url"`
		CheckedAt time.Time `json:"checkedat"`
	}

	// LatestRelease holds the required fields from https://api.github.com/repos/ariga/atlas/releases/latest.
	LatestRelease struct {
		Version string `json:"tag_name"`
		URL     string `json:"html_url"`
	}
)

const (
	// AtlasNoUpdateNotifier when enabled it cancels checking for update
	AtlasNoUpdateNotifier = "ATLAS_NO_UPDATE_NOTIFIER"
)

// CheckForUpdate implements a notification to the user when a later release is available
// 1. Check release file ~/.atlas/release.json for latest known release and poll time
// 2. If last poll was more than 24h, poll GitHub public API https://docs.github.com/en/rest/reference/releases#get-the-latest-release
// 3. Store the latest release metadata
// 4. If current build Version, that is not development, is lower than the latest release, notify user
func CheckForUpdate(version string, logF func(i ...interface{})) {
	if !enabled(version) {
		return
	}
	p, err := homedir.Expand("~/.atlas")
	if err != nil {
		return
	}
	ok, message, err := shouldUpdate(version, p, latestReleaseFromGithub)
	if err != nil || !ok {
		return
	}
	logF(message)
}

func enabled(version string) bool {
	if _, ok := os.LookupEnv(AtlasNoUpdateNotifier); ok {
		return false
	}
	if _, ok := os.LookupEnv("GITHUB_ACTIONS"); ok {
		return false
	}
	if ok := semver.IsValid(version); !ok {
		return false
	}
	return true
}

func shouldUpdate(version string, path string, latestReleaseF func() (LatestRelease, error)) (bool, string, error) {
	r := localStore(path)
	if shouldSkip(r) {
		return false, "", nil
	}
	l, err := latestReleaseF()
	if err != nil {
		return false, "", err
	}
	if err := saveStore(path, l, time.Now()); err != nil {
		return false, "", err
	}
	if semver.Compare(version, l.Version) != -1 {
		return false, "", nil
	}
	return true, fmt.Sprintf("A new version of Atlas is available (%s):\n%s", l.Version, l.URL), nil
}

func latestReleaseFromGithub() (LatestRelease, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/ariga/atlas/releases/latest", nil)
	if err != nil {
		return LatestRelease{}, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	// https://docs.github.com/en/rest/overview/resources-in-the-rest-api#user-agent-required
	req.Header.Set("User-Agent", "Ariga-Atlas-CLI")
	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		return LatestRelease{}, err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return LatestRelease{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return LatestRelease{}, fmt.Errorf("update: failed to fetch latest release version")
	}
	var b LatestRelease
	if err := json.Unmarshal(data, &b); err != nil {
		return LatestRelease{}, err
	}
	return b, nil
}

func shouldSkip(r *Store) bool {
	if r == nil {
		return false
	}
	return time.Since(r.CheckedAt).Hours() < 24
}

func localStore(path string) *Store {
	b, err := ioutil.ReadFile(fileLocation(path))
	if err != nil {
		return nil
	}
	var s Store
	if err := json.Unmarshal(b, &s); err != nil {
		return nil
	}
	return &s
}

func saveStore(path string, l LatestRelease, t time.Time) error {
	s := Store{Version: l.Version, URL: l.URL, CheckedAt: t}
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}
	b, err := json.Marshal(&s)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fileLocation(path), b, 0600)
}

func fileLocation(p string) string {
	return filepath.Join(p, "release.json")
}
