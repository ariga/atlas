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
			message:        fmt.Sprintf("new atlas version v2.2.3, %s", url),
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
			message:        fmt.Sprintf("new atlas version v2.2.3, %s", url),
		},
		{
			name:           "no store fetch and update newer minor version",
			currentVersion: "v1.2.3",
			latestVersion:  "v1.3.3",
			latestRelease:  &LatestRelease{Version: "v2.2.3", URL: url},
			shouldNotify:   true,
			message:        fmt.Sprintf("new atlas version v2.2.3, %s", url),
		},
		{
			name:           "no store fetch and update newer patch version",
			currentVersion: "v1.2.3",
			latestVersion:  "v1.2.4",
			latestRelease:  &LatestRelease{Version: "v2.2.3", URL: url},
			shouldNotify:   true,
			message:        fmt.Sprintf("new atlas version v2.2.3, %s", url),
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
			var f func() (LatestRelease, error)
			if tt.latestRelease != nil {
				f = func() (LatestRelease, error) { return *tt.latestRelease, nil }
			}
			ok, m, err := shouldUpdate(tt.currentVersion, p, f)
			require.NoError(t, err)
			require.Equal(t, tt.shouldNotify, ok)
			require.Equal(t, tt.message, m)
		})
	}
}
