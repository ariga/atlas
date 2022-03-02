package action

import (
	"os"
	"testing"

	"ariga.io/atlas/cmd/action/internal/update"

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
