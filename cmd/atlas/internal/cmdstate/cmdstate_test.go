// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdstate_test

import (
	"os"
	"path/filepath"
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/cmdstate"

	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/require"
)

func TestFile(t *testing.T) {
	homedir.DisableCache = true
	t.Cleanup(func() { homedir.DisableCache = false })

	type T struct{ V string }
	f := cmdstate.File[T]{Name: "test", Dir: t.TempDir()}
	v, err := f.Read()
	require.NoError(t, err)
	require.Equal(t, T{}, v)
	require.NoError(t, f.Write(T{V: "v"}))
	v, err = f.Read()
	require.NoError(t, err)
	require.Equal(t, T{V: "v"}, v)

	home := t.TempDir()
	t.Setenv("HOME", home)
	f = cmdstate.File[T]{Name: "t"}
	_, err = f.Read()
	require.NoError(t, err)
	dirs, err := os.ReadDir(home)
	require.NoError(t, err)
	require.Empty(t, dirs)

	require.NoError(t, f.Write(T{V: "v"}))
	dirs, err = os.ReadDir(home)
	require.NoError(t, err)
	require.Len(t, dirs, 1)
	require.Equal(t, ".atlas", dirs[0].Name())
	dirs, err = os.ReadDir(filepath.Join(home, ".atlas"))
	require.NoError(t, err)
	require.Len(t, dirs, 1)
	require.Equal(t, "t.json", dirs[0].Name())
	v, err = f.Read()
	require.NoError(t, err)
	require.Equal(t, T{V: "v"}, v)
}
