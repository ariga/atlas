// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/cmdext"
	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/schema"

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

locals {
	envName = atlas.env
}

lint {
	destructive {
		error = true
	}
	// Backwards compatibility with old attribute.
	log = <<EOS
{{- range $f := .Files }}
	{{- $f.Name }}
{{- end }}
EOS
}

diff {
  skip {
    drop_schema = true
  }
}

env "local" {
	url = "mysql://root:pass@localhost:3306/"
	dev = "docker://mysql/8"
	src = "${local.envName}/app.hcl"
	schemas = ["hello", "world"]
	migration {
		dir = "file://migrations"
		format = atlas
		lock_timeout = "1s"
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
	path := filepath.Join(d, "atlas.hcl")
	err := os.WriteFile(path, []byte(h), 0600)
	require.NoError(t, err)
	GlobalFlags.ConfigURL = "file://" + path
	t.Run("ok", func(t *testing.T) {
		envs, err := LoadEnv("local")
		require.NoError(t, err)
		require.Len(t, envs, 1)
		env := envs[0]
		sort.Slice(env.Extra.Attrs, func(i, j int) bool {
			return env.Extra.Attrs[i].K < env.Extra.Attrs[j].K
		})
		require.EqualValues(t, &Env{
			Name:    "local",
			URL:     "mysql://root:pass@localhost:3306/",
			DevURL:  "docker://mysql/8",
			Schemas: []string{"hello", "world"},
			Migration: &Migration{
				Dir:             "file://migrations",
				Format:          formatAtlas,
				LockTimeout:     "1s",
				RevisionsSchema: "revisions",
			},
			Diff: &Diff{
				SkipChanges: &SkipChanges{
					DropSchema: true,
				},
			},
			Lint: &Lint{
				Latest: 1,
				Format: "{{- range $f := .Files }}\n\t{{- $f.Name }}\n{{- end }}\n",
				DefaultExtension: schemahcl.DefaultExtension{
					Extra: schemahcl.Resource{
						Children: []*schemahcl.Resource{
							{
								Type: "destructive",
								Attrs: []*schemahcl.Attr{
									schemahcl.BoolAttr("error", true),
								},
							},
						},
					},
				},
			},
			DefaultExtension: schemahcl.DefaultExtension{
				Extra: schemahcl.Resource{
					Attrs: []*schemahcl.Attr{
						schemahcl.BoolAttr("bool", true),
						schemahcl.IntAttr("integer", 42),
						schemahcl.StringAttr("src", "local/app.hcl"),
						schemahcl.StringAttr("str", "hello"),
					},
				},
			},
			cfg: &cmdext.AtlasConfig{},
		}, env)
		sources, err := env.Sources()
		require.NoError(t, err)
		require.EqualValues(t, []string{"local/app.hcl"}, sources)
	})
	t.Run("multi", func(t *testing.T) {
		envs, err := LoadEnv("multi")
		require.NoError(t, err)
		require.Len(t, envs, 1)
		srcs, err := envs[0].Sources()
		require.NoError(t, err)
		require.EqualValues(t, []string{"./a.hcl", "./b.hcl"}, srcs)
	})
	t.Run("with input", func(t *testing.T) {
		envs, err := LoadEnv("local", WithInput(map[string]cty.Value{
			"name": cty.StringVal("goodbye"),
		}))
		require.NoError(t, err)
		require.Len(t, envs, 1)
		str, ok := envs[0].Attr("str")
		require.True(t, ok)
		val, err := str.String()
		require.NoError(t, err)
		require.EqualValues(t, "goodbye", val)
	})
	t.Run("wrong env", func(t *testing.T) {
		_, err = LoadEnv("home")
		require.EqualError(t, err, `env "home" not defined in project file`)
	})
	t.Run("wrong dir", func(t *testing.T) {
		GlobalFlags.ConfigURL = defaultConfigPath
		_, err = LoadEnv("home")
		require.ErrorContains(t, err, `no such file or directory`)
	})
}

func TestUnnamedEnv(t *testing.T) {
	h := `
env {
  name = atlas.env
  log {
    schema {
      apply = "env: ${atlas.env}"
    }
  }
}`
	path := filepath.Join(t.TempDir(), "atlas.hcl")
	err := os.WriteFile(path, []byte(h), 0600)
	require.NoError(t, err)
	GlobalFlags.ConfigURL = "file://" + path
	envs, err := LoadEnv("local")
	require.NoError(t, err)
	require.Len(t, envs, 1)
	require.Equal(t, "local", envs[0].Name)
	require.Equal(t, "env: local", envs[0].Format.Schema.Apply)
}

func TestDiff_Options(t *testing.T) {
	d := &Diff{}
	require.Len(t, d.Options(), 0)
	d.SkipChanges = &SkipChanges{}
	require.Len(t, d.Options(), 0)

	d.SkipChanges = &SkipChanges{DropSchema: true}
	require.Len(t, d.Options(), 1)
	opts := schema.NewDiffOptions(d.Options()...)
	require.True(t, opts.Skipped(&schema.DropSchema{}))
	require.False(t, opts.Skipped(&schema.DropTable{}))

	d.SkipChanges = &SkipChanges{DropSchema: true, DropTable: true}
	require.Len(t, d.Options(), 1)
	opts = schema.NewDiffOptions(d.Options()...)
	require.True(t, opts.Skipped(&schema.DropSchema{}))
	require.True(t, opts.Skipped(&schema.DropTable{}))
}
