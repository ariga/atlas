// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent && !official

package cmdapi

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLI_Version(t *testing.T) {
	tests := []struct {
		name     string
		cmd      *exec.Cmd
		expected string
	}{
		{
			name: "dev mode",
			cmd: exec.Command("go", "run", "github.com/s-sokolko/atlas/cmd/atlas",
				"version",
			),
			expected: "atlas unofficial version - development\nhttps://github.com/ariga/atlas/releases/latest\n",
		},
		{
			name: "release",
			cmd: exec.Command("go", "run",
				"-ldflags",
				"-X github.com/s-sokolko/atlas/cmd/atlas/internal/cmdapi.version=v1.2.3",
				"github.com/s-sokolko/atlas/cmd/atlas",
				"version",
			),
			expected: "atlas unofficial version v1.2.3\nhttps://github.com/ariga/atlas/releases/tag/v1.2.3\n",
		},
		{
			name: "canary",
			cmd: exec.Command("go", "run",
				"-ldflags",
				"-X github.com/s-sokolko/atlas/cmd/atlas/internal/cmdapi.version=v0.3.0-6539f2704b5d-canary",
				"github.com/s-sokolko/atlas/cmd/atlas",
				"version",
			),
			expected: "atlas unofficial version v0.3.0-6539f2704b5d-canary\nhttps://github.com/ariga/atlas/releases/latest\n",
		},
		{
			name: "flavor",
			cmd: exec.Command("go", "run",
				"-ldflags",
				"-X github.com/s-sokolko/atlas/cmd/atlas/internal/cmdapi.flavor=flavor",
				"github.com/s-sokolko/atlas/cmd/atlas",
				"version",
			),
			expected: "atlas unofficial flavor version - development\nhttps://github.com/ariga/atlas/releases/latest\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ATLAS_NO_UPDATE_NOTIFIER", "true")
			stdout := bytes.NewBuffer(nil)
			tt.cmd.Stdout = stdout
			tt.cmd.Stderr = os.Stderr
			require.NoError(t, tt.cmd.Run())
			require.Equal(t, tt.expected+versionInfo, stdout.String())
		})
	}
}
