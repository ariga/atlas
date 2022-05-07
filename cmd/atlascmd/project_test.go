// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package atlascmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadEnv(t *testing.T) {
	d := t.TempDir()
	h := `
env "local" {
	url = "mysql://root:pass@localhost:3306/"
	dev = "docker://mysql/8"
	src = "./app.hcl"
	schemas = ["hello", "world"]
}
`
	err := os.WriteFile(filepath.Join(d, projectFileName), []byte(h), 0600)
	require.NoError(t, err)
	path := filepath.Join(d, projectFileName)
	t.Run("ok", func(t *testing.T) {
		env := &Environment{}
		env, err = LoadEnv(path, "local")
		require.NoError(t, err)
		require.EqualValues(t, &Environment{
			Name:    "local",
			URL:     "mysql://root:pass@localhost:3306/",
			DevURL:  "docker://mysql/8",
			Source:  "./app.hcl",
			Schemas: []string{"hello", "world"},
		}, env)
	})
	t.Run("wrong env", func(t *testing.T) {
		_, err = LoadEnv(path, "home")
		require.EqualError(t, err, `env "home" not defined in project file`)
	})
	t.Run("wrong dir", func(t *testing.T) {
		wd, err := os.Getwd()
		require.NoError(t, err)
		_, err = LoadEnv(filepath.Join(wd, projectFileName), "home")
		require.ErrorContains(t, err, `no such file or directory`)
	})
	t.Run("duplicate env", func(t *testing.T) {
		dup := h + "\n" + h
		path := filepath.Join(d, "dup.hcl")
		err = os.WriteFile(path, []byte(dup), 0600)
		require.NoError(t, err)
		_, err = LoadEnv(path, "local")
		require.EqualError(t, err, `duplicate environment name "local"`)
	})
}
