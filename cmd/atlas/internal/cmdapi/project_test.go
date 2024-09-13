// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/cloudapi"
	"ariga.io/atlas/cmd/atlas/internal/cmdext"
	cmdmigrate "ariga.io/atlas/cmd/atlas/internal/migrate"
	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/schema"

	"github.com/spf13/cobra"
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
  review = ERROR
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

test {
  schema {
    src = ["schema.test.hcl"]
    vars = {
      a = "1"
    }
  }
  migrate {
    src = ["migrate.test.hcl"]
    vars = {
      b = "2"
    }
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
    exec_order = LINEAR_SKIP
  }
  lint {
    latest = 1
    review = WARNING
  }
  diff {
    skip {
      drop_column = true
    }
  }
  test {
    schema {
      src = ["env.schema.test.hcl"]
      vars = {
        a = "a"
      }
    }
    migrate {
      src = ["env.migrate.test.hcl"]
      vars = {
        b = "b"
      }
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
    naming {
      match = "^[A-Z]+$"
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
		_, envs, err := EnvByName(&cobra.Command{}, "local", map[string]cty.Value{
			"unused": cty.StringVal("value"),
		})
		require.NoError(t, err)
		require.Len(t, envs, 1)
		env := envs[0]
		sort.Slice(env.Extra.Attrs, func(i, j int) bool {
			return env.Extra.Attrs[i].K < env.Extra.Attrs[j].K
		})
		require.NotNil(t, env.config)
		env.config = nil
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
				ExecOrder:       "LINEAR_SKIP",
			},
			Diff: &Diff{
				SkipChanges: &SkipChanges{
					DropColumn: true,
				},
			},
			Lint: &Lint{
				Latest: 1,
				Review: ReviewWarning,
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
			Test: &Test{
				Schema: struct {
					Src  []string `spec:"src"`
					Vars Vars     `spec:"vars"`
				}{
					Src: []string{"env.schema.test.hcl"},
					Vars: Vars{
						"a": cty.StringVal("a"),
					},
				},
				Migrate: struct {
					Src  []string `spec:"src"`
					Vars Vars     `spec:"vars"`
				}{
					Src: []string{"env.migrate.test.hcl"},
					Vars: Vars{
						"b": cty.StringVal("b"),
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
			cloud: &cmdext.AtlasConfig{
				Project: cloudapi.DefaultProjectName,
			},
		}, env)
		sources, err := env.Sources()
		require.NoError(t, err)
		require.EqualValues(t, []string{"local/app.hcl"}, sources)
	})
	t.Run("multi", func(t *testing.T) {
		_, envs, err := EnvByName(&cobra.Command{}, "multi", nil)
		require.NoError(t, err)
		require.Len(t, envs, 1)
		srcs, err := envs[0].Sources()
		require.NoError(t, err)
		require.EqualValues(t, []string{"./a.hcl", "./b.hcl"}, srcs)
		require.EqualValues(t, ReviewError, envs[0].Lint.Review)
		require.Len(t, envs[0].Lint.Extra.Children, 1)
		require.Equal(t, "naming", envs[0].Lint.Extra.Children[0].Type)
		require.Equal(t, "1", envs[0].Test.Schema.Vars["a"].AsString())
		require.Equal(t, "2", envs[0].Test.Migrate.Vars["b"].AsString())
	})
	t.Run("with input", func(t *testing.T) {
		_, envs, err := EnvByName(&cobra.Command{}, "local", map[string]cty.Value{
			"name": cty.StringVal("goodbye"),
		})
		require.NoError(t, err)
		require.Len(t, envs, 1)
		str, ok := envs[0].Attr("str")
		require.True(t, ok)
		val, err := str.String()
		require.NoError(t, err)
		require.EqualValues(t, "goodbye", val)
	})
	t.Run("wrong env", func(t *testing.T) {
		_, _, err = EnvByName(&cobra.Command{}, "home", nil)
		require.EqualError(t, err, `env "home" not defined in config file`)
	})
	t.Run("wrong dir", func(t *testing.T) {
		GlobalFlags.ConfigURL = defaultConfigPath
		_, _, err = EnvByName(&cobra.Command{}, "home", nil)
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
	_, envs, err := EnvByName(&cobra.Command{}, "local", nil)
	require.NoError(t, err)
	require.Len(t, envs, 1)
	require.Equal(t, "local", envs[0].Name)
	require.Equal(t, "env: local", envs[0].Format.Schema.Apply)
}

func TestEnvCache(t *testing.T) {
	h := `
variable "path" {
  type = string
}

data "template_dir" "name" {
  path = var.path
}

env {
  name = atlas.env
  migration {
    dir = data.template_dir.name.url
  }
}
`
	path := filepath.Join(t.TempDir(), "atlas.hcl")
	err := os.WriteFile(path, []byte(h), 0600)
	require.NoError(t, err)
	GlobalFlags.ConfigURL = "file://" + path

	dir := filepath.Join(t.TempDir(), "migrations")
	require.NoError(t, os.Mkdir(dir, 0700))
	vars := map[string]cty.Value{"path": cty.StringVal(dir)}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	_, envs1, err := EnvByName(cmd, "local", vars)
	require.NoError(t, err)
	require.Len(t, envs1, 1)
	require.Contains(t, envs1[0].Migration.Dir, dir)

	require.NoError(t, os.Remove(dir))
	_, envs1, err = EnvByName(cmd, "local", vars)
	require.NoError(t, err)
	require.Len(t, envs1, 1)
	require.Contains(t, envs1[0].Migration.Dir, dir, "should return the same dir, even if it doesn't exist anymore")
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
  review = WARNING
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
	project, envs, err := EnvByName(&cobra.Command{}, "", nil)
	require.NoError(t, err)
	require.Len(t, envs, 0)
	require.Equal(t, 1, project.Lint.Latest)
	require.NotNil(t, project.Diff.SkipChanges)
	require.True(t, project.Diff.SkipChanges.DropColumn)
	require.Equal(t, ReviewWarning, project.Lint.Review)
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
	_, envs, err := EnvByName(&cobra.Command{}, "unnamed", nil)
	require.NoError(t, err)
	require.Len(t, envs, 1)
	require.Equal(t, "unnamed", envs[0].Name)
	require.Equal(t, "b", envs[0].Format.Schema.Diff)
	require.Equal(t, "env: unnamed", envs[0].Format.Schema.Apply)
	_, envs, err = EnvByName(&cobra.Command{}, "dev", nil)
	require.NoError(t, err)
	require.Len(t, envs, 1)
	attr, ok := envs[0].Extra.Attr("a")
	require.True(t, ok)
	v, err := attr.String()
	require.NoError(t, err)
	require.Equal(t, "b", v)

	h = `
data "template_dir" "unknown" {
  path = "unknown"
}

env {
  name = atlas.env
  a = data.template_dir.unknown.url
}

env "dev" {
  a = "b"
}`
	require.NoError(t, os.WriteFile(path, []byte(h), 0600))
	_, envs, err = EnvByName(&cobra.Command{}, "dev", nil)
	require.NoError(t, err)
	require.Len(t, envs, 1)
	attr, ok = envs[0].Extra.Attr("a")
	require.True(t, ok)
	v, err = attr.String()
	require.NoError(t, err)
	require.Equal(t, "b", v)

	// Loading template directory should fail.
	_, envs, err = EnvByName(&cobra.Command{}, "other", nil)
	require.Error(t, err)
	require.Empty(t, envs)
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
