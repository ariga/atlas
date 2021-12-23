package update

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckForUpdate(t *testing.T) {
	currentVersion := "v1.2.3"
	LatestVersion := "v2.2.3"
	u, err := shouldUpdate(currentVersion)
	require.NoError(t, err)
	require.Equal(t, &update{version: LatestVersion, shouldNotify: true, reason: reasonVersionUpdate}, u)
}
