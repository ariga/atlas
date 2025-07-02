// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package atlasexec

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnv(t *testing.T) {
	// printenv is a simple command that prints all environment variables
	c, err := NewClient(t.TempDir(), "printenv")
	require.NoError(t, err)

	// Should not be able to override the default environment variable
	require.ErrorContains(t, c.SetEnv(map[string]string{
		"FOO":                      "bar",
		"ATLAS_NO_UPDATE_NOTIFIER": "0",
	}), "cannot override the default environment variable")

	// Should be able to set new environment variables
	require.NoError(t, c.SetEnv(map[string]string{
		"FOO": "bar",
		"BAZ": "qux",
	}))

	// Invoke the command and check the environment variables
	v, err := c.runCommand(context.Background(), nil)
	require.NoError(t, err)
	raw, err := io.ReadAll(v)
	require.NoError(t, err)
	require.Equal(t, `ATLAS_NO_UPDATE_NOTIFIER=1
ATLAS_NO_UPGRADE_SUGGESTIONS=1
BAZ=qux
FOO=bar
`, string(raw))
}
