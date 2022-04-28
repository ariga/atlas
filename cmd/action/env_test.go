// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"os"
	"testing"

	"ariga.io/atlas/cmd/internal/update"

	"github.com/stretchr/testify/require"
)

func TestEnv(t *testing.T) {
	out, err := runCmd(RootCmd, "env")
	require.NoError(t, err)
	require.Empty(t, out)
}

func TestEnv_Set(t *testing.T) {
	err := os.Setenv(update.AtlasNoUpdateNotifier, "test")
	require.NoError(t, err)
	out, err := runCmd(RootCmd, "env")
	require.NoError(t, err)
	require.Equal(t, "ATLAS_NO_UPDATE_NOTIFIER=test\n", out)
}
