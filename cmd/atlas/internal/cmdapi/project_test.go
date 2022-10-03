// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"ariga.io/atlas/schemahcl"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestLoadEnv(t *testing.T) {
	d := t.TempDir()
	h := `
variable "name" {
	type = string
	default = "hello"
}

lint {
	destructive {
		error = true
	}
	log = <<EOS
{{- range $f := .Files }}
	{{- $f.Name }}
{{- end }}
EOS
}

env "local" {
	url = "mysql://root:pass@localhost:3306/"
	dev = "docker://mysql/8"
	src = "./app.hcl"
	schemas = ["hello", "world"]
	migration {
		dir = "file://migrations"
		format = atlas
		revisions_schema = "revisions"
	}
	lint {
		latest = 1
	}
	
	bool = true
	integer = 42
	str = var.name
}

env "multi" {
	url = "mysql://root:pass@localhost:3306/"
	src = [
		"./a.hcl",
		"./b.hcl",
	]
	lint {
		git {
			dir  = "./path"
			base = "master"
		}
	}
}
`
	err := os.WriteFile(filepath.Join(d, projectFileName), []byte(h), 0600)
	require.NoError(t, err)
	path := filepath.Join(d, projectFileName)
	t.Run("ok", func(t *testing.T) {
		var env *Env
		env, err = LoadEnv(path, "local")
		require.NoError(t, err)
		sort.Slice(env.Extra.Attrs, func(i, j int) bool {
			return env.Extra.Attrs[i].K < env.Extra.Attrs[j].K
		})
		require.NoError(t, err)
		require.EqualValues(t, &Env{
			Name:    "local",
			URL:     "mysql://root:pass@localhost:3306/",
			DevURL:  "docker://mysql/8",
			Schemas: []string{"hello", "world"},
			Migration: &Migration{
				Dir:             "file://migrations",
				Format:          formatAtlas,
				RevisionsSchema: "revisions",
			},
			Lint: &Lint{
				Latest: 1,
				Log:    "{{- range $f := .Files }}\n\t{{- $f.Name }}\n{{- end }}\n",
				DefaultExtension: schemahcl.DefaultExtension{
					Extra: schemahcl.Resource{
						Children: []*schemahcl.Resource{
							{
								Type: "destructive",
								Attrs: []*schemahcl.Attr{
									{K: "error", V: &schemahcl.LiteralValue{V: "true"}},
								},
							},
						},
					},
				},
			},
			DefaultExtension: schemahcl.DefaultExtension{
				Extra: schemahcl.Resource{
					Attrs: []*schemahcl.Attr{
						{K: "bool", V: &schemahcl.LiteralValue{V: "true"}},
						{K: "integer", V: &schemahcl.LiteralValue{V: "42"}},
						{K: "src", V: &schemahcl.LiteralValue{V: `"./app.hcl"`}},
						{K: "str", V: &schemahcl.LiteralValue{V: `"hello"`}},
					},
				},
			},
		}, env)
		sources, err := env.Sources()
		require.NoError(t, err)
		require.EqualValues(t, []string{"./app.hcl"}, sources)
	})
	t.Run("multi", func(t *testing.T) {
		env, err := LoadEnv(path, "multi")
		require.NoError(t, err)
		srcs, err := env.Sources()
		require.NoError(t, err)
		require.EqualValues(t, []string{"./a.hcl", "./b.hcl"}, srcs)
	})
	t.Run("with input", func(t *testing.T) {
		env, err := LoadEnv(path, "local", WithInput(map[string]cty.Value{
			"name": cty.StringVal("goodbye"),
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
