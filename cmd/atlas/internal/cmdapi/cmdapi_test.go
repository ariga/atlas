// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLI_Version(t *testing.T) {
	// Required to have a clean "stderr" while running first time.
	tests := []struct {
		name     string
		cmd      *exec.Cmd
		expected string
	}{
		{
			name: "dev mode",
			cmd: exec.Command("go", "run", "ariga.io/atlas/cmd/atlas",
				"version",
			),
			expected: "atlas version - development\nhttps://github.com/ariga/atlas/releases/latest\n",
		},
		{
			name: "release",
			cmd: exec.Command("go", "run",
				"-ldflags",
				"-X ariga.io/atlas/cmd/atlas/internal/cmdapi.version=v1.2.3",
				"ariga.io/atlas/cmd/atlas",
				"version",
			),
			expected: "atlas version v1.2.3\nhttps://github.com/ariga/atlas/releases/tag/v1.2.3\n",
		},
		{
			name: "canary",
			cmd: exec.Command("go", "run",
				"-ldflags",
				"-X ariga.io/atlas/cmd/atlas/internal/cmdapi.version=v0.3.0-6539f2704b5d-canary",
				"ariga.io/atlas/cmd/atlas",
				"version",
			),
			expected: "atlas version v0.3.0-6539f2704b5d-canary\nhttps://github.com/ariga/atlas/releases/latest\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ATLAS_NO_UPDATE_NOTIFIER", "true")
			stdout := bytes.NewBuffer(nil)
			tt.cmd.Stdout = stdout
			tt.cmd.Stderr = os.Stderr
			require.NoError(t, tt.cmd.Run())
			require.Equal(t, tt.expected, stdout.String())
		})
	}
}
