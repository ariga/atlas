// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package update

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCheckForUpdate(t *testing.T) {
	url := "https://github.com/ariga/atlas/releases/tag/test"
	tests := []struct {
		name           string
		currentVersion string
		latestVersion  string
		store          *Store
		latestRelease  *LatestRelease
		shouldNotify   bool
		message        string
	}{
		{
			name:           "stale store fetch and update newer version",
			currentVersion: "v1.2.3",
			latestVersion:  "v2.2.3",
			store:          &Store{Version: "v1.2.3", URL: url, CheckedAt: time.Now().Add(-25 * time.Hour)},
			latestRelease:  &LatestRelease{Version: "v2.2.3", URL: url},
			shouldNotify:   true,
			message:        fmt.Sprintf("A new version of Atlas is available (v2.2.3):\n%s", url),
		},
		{
			name:           "fresh store do nothing",
			currentVersion: "v1.2.3",
			latestVersion:  "v2.2.3",
			store:          &Store{Version: "v1.2.3", URL: url, CheckedAt: time.Now().Add(-23 * time.Hour)},
			shouldNotify:   false,
		},
		{
			name:           "no store fetch and update newer version",
			currentVersion: "v1.2.3",
			latestVersion:  "v2.2.3",
			latestRelease:  &LatestRelease{Version: "v2.2.3", URL: url},
			shouldNotify:   true,
			message:        fmt.Sprintf("A new version of Atlas is available (v2.2.3):\n%s", url),
		},
		{
			name:           "no store fetch and update newer minor version",
			currentVersion: "v1.2.3",
			latestVersion:  "v1.3.3",
			latestRelease:  &LatestRelease{Version: "v2.2.3", URL: url},
			shouldNotify:   true,
			message:        fmt.Sprintf("A new version of Atlas is available (v2.2.3):\n%s", url),
		},
		{
			name:           "no store fetch and update newer patch version",
			currentVersion: "v1.2.3",
			latestVersion:  "v1.2.4",
			latestRelease:  &LatestRelease{Version: "v2.2.3", URL: url},
			shouldNotify:   true,
			message:        fmt.Sprintf("A new version of Atlas is available (v2.2.3):\n%s", url),
		},
		{
			name:           "no store fetch and update newer patch version - canary",
			currentVersion: "v1.2.3-6539f2704b5d-canary",
			latestVersion:  "v1.2.4",
			latestRelease:  &LatestRelease{Version: "v2.2.3", URL: url},
			shouldNotify:   true,
			message:        fmt.Sprintf("A new version of Atlas is available (v2.2.3):\n%s", url),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := t.TempDir()
			if tt.store != nil {
				b, err := json.Marshal(tt.store)
				require.NoError(t, err)
				require.NoError(t, ioutil.WriteFile(fileLocation(p), b, 0600))
			}
			var f func() (*LatestRelease, error)
			if tt.latestRelease != nil {
				f = func() (*LatestRelease, error) { return tt.latestRelease, nil }
			}
			ok, m, err := shouldUpdate(tt.currentVersion, p, f)
			require.NoError(t, err)
			require.Equal(t, tt.shouldNotify, ok)
			require.Equal(t, tt.message, m)
		})
	}
}
