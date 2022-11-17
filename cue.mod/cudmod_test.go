package cuemod

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestCopy(t *testing.T) {
	target, err := os.MkdirTemp("", "atlas")
	require.NoError(t, err)

	defer os.RemoveAll(target)

	err = Copy(target)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(target, "cue.mod/module.cue"))
	require.FileExists(t, filepath.Join(target, "cue.mod/gen/ariga.io/atlas/sql/schema/schema_go_gen.cue"))
}
