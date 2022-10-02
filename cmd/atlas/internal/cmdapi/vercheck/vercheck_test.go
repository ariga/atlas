// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package vercheck

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVerCheck(t *testing.T) {
	var path, ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		output := `{"latest":{"Version":"v0.7.2","Summary":"","Link":"https://github.com/ariga/atlas/releases/tag/v0.7.2"},"advisory":null}`
		path = r.URL.Path
		ua = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(output))
	}))
	defer srv.Close()

	vc := New(srv.URL, "")
	check, err := vc.Check("v0.1.2")

	require.EqualValues(t, "/atlas/v0.1.2", path)
	require.EqualValues(t, "Ariga-Atlas-CLI", ua)
	require.NoError(t, err)
	require.EqualValues(t, &Payload{
		Latest: &Latest{
			Version: "v0.7.2",
			Summary: "",
			Link:    "https://github.com/ariga/atlas/releases/tag/v0.7.2",
		},
	}, check)
}

func TestState(t *testing.T) {
	hrAgo, err := json.Marshal(State{CheckedAt: time.Now().Add(-time.Hour)})
	require.NoError(t, err)
	weekAgo, err := json.Marshal(State{CheckedAt: time.Now().Add(-time.Hour * 24 * 7)})
	require.NoError(t, err)
	for _, tt := range []struct {
		name        string
		state       string
		expectedRun bool
	}{
		{
			name:        "corrupt json",
			state:       "{",
			expectedRun: true,
		},
		{
			name:        "none",
			state:       "", // no file
			expectedRun: true,
		},
		{
			name:        "one hr ago",
			state:       string(hrAgo),
			expectedRun: false,
		},
		{
			name:        "one week ago",
			state:       string(weekAgo),
			expectedRun: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var ran bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ran = true
				_, _ = w.Write([]byte(`{}`))
			}))
			t.Cleanup(srv.Close)
			path := filepath.Join(t.TempDir(), "release.json")
			if tt.state != "" {
				err := os.WriteFile(path, []byte(tt.state), os.ModePerm)
				require.NoError(t, err)
			}
			vc := New(srv.URL, path)
			_, _ = vc.Check("v0.1.2")
			require.EqualValues(t, tt.expectedRun, ran)

			b, err := os.ReadFile(path)
			require.NoError(t, err)
			if tt.expectedRun {
				require.NotEqualValues(t, tt.state, b)
			} else {
				require.EqualValues(t, tt.state, b)
			}
		})
	}
}

func TestTemplate(t *testing.T) {

	for _, tt := range []struct {
		name    string
		payload Payload
		exp     string
	}{
		{
			name:    "empty",
			payload: Payload{},
			exp:     "",
		},
		{
			name: "version with summary",
			payload: Payload{
				Latest: &Latest{
					Version: "v0.7.2",
					Summary: "A great version including amazing features.",
					Link:    "https://atlasgo.io/v0.7.2",
				},
			},
			exp: `A new version of Atlas is available (v0.7.2): https://atlasgo.io/v0.7.2
A great version including amazing features.`,
		},
		{
			name: "version",
			payload: Payload{
				Latest: &Latest{
					Version: "v0.7.2",
					Link:    "https://atlasgo.io/v0.7.2",
				},
			},
			exp: `A new version of Atlas is available (v0.7.2): https://atlasgo.io/v0.7.2`,
		},
		{
			name: "with advisory",
			payload: Payload{
				Advisory: &Advisory{Text: "This version contains a vulnerability, please upgrade."},
			},
			exp: `SECURITY ADVISORY
This version contains a vulnerability, please upgrade.`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			err := Notify.Execute(&b, tt.payload)
			require.NoError(t, err)
			require.EqualValues(t, tt.exp, b.String())
		})
	}
}
