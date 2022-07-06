// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"github.com/stretchr/testify/require"
)

func TestLoadEnv(t *testing.T) {
	d := t.TempDir()
	h := `
variable "name" {
	type = string
	default = "hello"
}

env "local" {
	url = "mysql://root:pass@localhost:3306/"
	dev = "docker://mysql/8"
	src = "./app.hcl"
	schemas = ["hello", "world"]
	
	migration_dir {
		url = "file://migrations"
		format = atlas
	}
	
	bool = true
	integer = 42
	str = var.name
}
`
	err := os.WriteFile(filepath.Join(d, projectFileName), []byte(h), 0600)
	require.NoError(t, err)
	path := filepath.Join(d, projectFileName)
	t.Run("ok", func(t *testing.T) {
		var env *Env
		env, err = LoadEnv(path, "local")
		sort.Slice(env.Extra.Attrs, func(i, j int) bool {
			return env.Extra.Attrs[i].K < env.Extra.Attrs[j].K
		})
		require.NoError(t, err)
		require.EqualValues(t, &Env{
			Name:    "local",
			URL:     "mysql://root:pass@localhost:3306/",
			DevURL:  "docker://mysql/8",
			Source:  "./app.hcl",
			Schemas: []string{"hello", "world"},
			MigrationDir: &MigrationDir{
				URL:    "file://migrations",
				Format: formatAtlas,
			},
			DefaultExtension: schemaspec.DefaultExtension{
				Extra: schemaspec.Resource{
					Attrs: []*schemaspec.Attr{
						{K: "bool", V: &schemaspec.LiteralValue{V: "true"}},
						{K: "integer", V: &schemaspec.LiteralValue{V: "42"}},
						{K: "str", V: &schemaspec.LiteralValue{V: `"hello"`}},
					},
				},
			},
		}, env)
	})
	t.Run("with input", func(t *testing.T) {
		env, err := LoadEnv(path, "local", WithInput(map[string]string{
			"name": "goodbye",
		}))
		require.NoError(t, err)
		str, ok := env.Attr("str")
		require.True(t, ok)
		val, err := str.String()
		require.NoError(t, err)
		require.EqualValues(t, "goodbye", val)
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
