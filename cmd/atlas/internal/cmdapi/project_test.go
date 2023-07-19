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
	cmdmigrate "ariga.io/atlas/cmd/atlas/internal/migrate"
	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestEnvByName(t *testing.T) {
	d := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(d, "local.txt"), []byte("text"), 0600))
	h := `
variable "name" {
  type = string
  default = "hello"
}

locals {
  envName = atlas.env
  emptyEnv = getenv("NOT_SET")
  opened = file("local.txt")
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
  diff {
    skip {
      drop_column = true
    }
  }
  bool = true
  integer = 42
  str = var.name
  token  = getenv("ATLAS_TOKEN")
  token2  = getenv("ATLAS_TOKEN2")
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
	require.NoError(t, os.Setenv("ATLAS_TOKEN", "token_atlas"))
	t.Cleanup(func() { require.NoError(t, os.Unsetenv("ATLAS_TOKEN")) })
	t.Run("ok", func(t *testing.T) {
		_, envs, err := EnvByName("local")
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
				Format:          cmdmigrate.FormatAtlas,
				LockTimeout:     "1s",
				RevisionsSchema: "revisions",
			},
			Diff: &Diff{
				SkipChanges: &SkipChanges{
					DropColumn: true,
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
						schemahcl.StringAttr("token", "token_atlas"),
						schemahcl.StringAttr("token2", ""),
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
		_, envs, err := EnvByName("multi")
		require.NoError(t, err)
		require.Len(t, envs, 1)
		srcs, err := envs[0].Sources()
		require.NoError(t, err)
		require.EqualValues(t, []string{"./a.hcl", "./b.hcl"}, srcs)
	})
	t.Run("with input", func(t *testing.T) {
		_, envs, err := EnvByName("local", WithInput(map[string]cty.Value{
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
		_, _, err = EnvByName("home")
		require.EqualError(t, err, `env "home" not defined in project file`)
	})
	t.Run("wrong dir", func(t *testing.T) {
		GlobalFlags.ConfigURL = defaultConfigPath
		_, _, err = EnvByName("home")
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
	_, envs, err := EnvByName("local")
	require.NoError(t, err)
	require.Len(t, envs, 1)
	require.Equal(t, "local", envs[0].Name)
	require.Equal(t, "env: local", envs[0].Format.Schema.Apply)
}

func TestNoEnv(t *testing.T) {
	h := `
env "dev" {
  log {
    schema {
      apply = "env: ${atlas.env}"
    }
  }
}

env {
  name = atlas.env
  log {
    schema {
      apply = "env: ${atlas.env}"
    }
  }
}

lint {
  latest = 1
}

diff {
  skip {
    drop_column = true
  }
}
`
	path := filepath.Join(t.TempDir(), "atlas.hcl")
	err := os.WriteFile(path, []byte(h), 0600)
	require.NoError(t, err)
	GlobalFlags.ConfigURL = "file://" + path
	project, envs, err := EnvByName("")
	require.NoError(t, err)
	require.Len(t, envs, 0)
	require.Equal(t, 1, project.Lint.Latest)
	require.NotNil(t, project.Diff.SkipChanges)
	require.True(t, project.Diff.SkipChanges.DropColumn)
}

func TestPartialParse(t *testing.T) {
	h := `
data "remote_dir" "ignored" {
  name = "ignored"
}

locals {
  a = "b"
}

env {
  name = atlas.env
  log {
    schema {
      diff  = local.a
      apply = "env: ${atlas.env}"
    }
  }
  a = local.a
}

env "dev" {
  a = local.a
}`
	path := filepath.Join(t.TempDir(), "atlas.hcl")
	err := os.WriteFile(path, []byte(h), 0600)
	require.NoError(t, err)
	GlobalFlags.ConfigURL = "file://" + path
	_, envs, err := EnvByName("unnamed")
	require.NoError(t, err)
	require.Len(t, envs, 1)
	require.Equal(t, "unnamed", envs[0].Name)
	require.Equal(t, "b", envs[0].Format.Schema.Diff)
	require.Equal(t, "env: unnamed", envs[0].Format.Schema.Apply)
	_, envs, err = EnvByName("dev")
	require.NoError(t, err)
	require.Len(t, envs, 2)
	for _, e := range envs {
		attr, ok := e.Extra.Attr("a")
		require.True(t, ok)
		v, err := attr.String()
		require.NoError(t, err)
		require.Equal(t, "b", v)
	}
}

func TestDiff_Options(t *testing.T) {
	d := &Diff{}
	require.Len(t, d.Options(), 1)
	d.SkipChanges = &SkipChanges{}
	require.Len(t, d.Options(), 1)

	d.SkipChanges = &SkipChanges{DropSchema: true}
	require.Len(t, d.Options(), 2)
	opts := schema.NewDiffOptions(d.Options()...)
	require.True(t, opts.Skipped(&schema.DropSchema{}))
	require.False(t, opts.Skipped(&schema.DropTable{}))

	d.SkipChanges = &SkipChanges{DropSchema: true, DropTable: true}
	require.Len(t, d.Options(), 2)
	opts = schema.NewDiffOptions(d.Options()...)
	require.True(t, opts.Skipped(&schema.DropSchema{}))
	require.True(t, opts.Skipped(&schema.DropTable{}))
}
