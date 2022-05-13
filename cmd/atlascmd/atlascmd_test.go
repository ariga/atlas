// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package atlascmd

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"ariga.io/atlas/cmd/atlascmd/update"

	"github.com/stretchr/testify/require"
)

func TestEnv(t *testing.T) {
	out, err := runCmd(Root, "env")
	require.NoError(t, err)
	require.Empty(t, out)
}

func TestEnv_Set(t *testing.T) {
	err := os.Setenv(update.AtlasNoUpdateNotifier, "test")
	require.NoError(t, err)
	out, err := runCmd(Root, "env")
	require.NoError(t, err)
	require.Equal(t, "ATLAS_NO_UPDATE_NOTIFIER=test\n", out)
}

func TestCLI_Version(t *testing.T) {
	// Required to have a clean "stderr" while running first time.
	require.NoError(t, exec.Command("go", "run", "-mod=mod", "ariga.io/atlas/cmd/atlas").Run())
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
				"-X ariga.io/atlas/cmd/atlascmd.version=v1.2.3",
				"ariga.io/atlas/cmd/atlas",
				"version",
			),
			expected: "atlas version v1.2.3\nhttps://github.com/ariga/atlas/releases/tag/v1.2.3\n",
		},
		{
			name: "canary",
			cmd: exec.Command("go", "run",
				"-ldflags",
				"-X ariga.io/atlas/cmd/atlascmd.version=v0.3.0-6539f2704b5d-canary",
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
			require.NoError(t, tt.cmd.Run())
			require.Equal(t, tt.expected, stdout.String())
		})
	}
}
