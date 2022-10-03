// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"errors"
	"fmt"
	"os"

	"ariga.io/atlas/schemahcl"

	"github.com/zclconf/go-cty/cty"
)

const projectFileName = "atlas.hcl"

type loadConfig struct {
	inputVals map[string]cty.Value
}

// LoadOption configures the LoadEnv function.
type LoadOption func(*loadConfig)

// WithInput is a LoadOption that sets the input values for the LoadEnv function.
func WithInput(vals map[string]cty.Value) LoadOption {
	return func(config *loadConfig) {
		config.inputVals = vals
	}
}

type (
	// Project represents an atlas.hcl project file.
	Project struct {
		Envs []*Env `spec:"env"`  // List of environments
		Lint *Lint  `spec:"lint"` // Optional global lint config
	}

	// Env represents an Atlas environment.
	Env struct {
		// Name for this environment.
		Name string `spec:"name,name"`

		// URL of the database.
		URL string `spec:"url"`

		// URL of the dev-database for this environment.
		// See: https://atlasgo.io/dev-database
		DevURL string `spec:"dev"`

		// List of schemas in this database that are managed by Atlas.
		Schemas []string `spec:"schemas"`

		// Exclude defines a list of glob patterns used to filter
		// resources on inspection.
		Exclude []string `spec:"exclude"`

		// Migration containing the migration configuration of the env.
		Migration *Migration `spec:"migration"`

		// Lint of the environment.
		Lint *Lint `spec:"lint"`
		schemahcl.DefaultExtension
	}

	// Migration represents the migration directory for the Env.
	Migration struct {
		Dir             string `spec:"dir"`
		Format          string `spec:"format"`
		RevisionsSchema string `spec:"revisions_schema"`
	}

	// Lint represents the configuration of migration linting.
	Lint struct {
		// Log configures the --log option.
		Log string `spec:"log"`
		// Latest configures the --latest option.
		Latest int `spec:"latest"`
		Git    struct {
			// Dir configures the --git-dir option.
			Dir string `spec:"dir"`
			// Base configures the --git-base option.
			Base string `spec:"base"`
		} `spec:"git"`
		schemahcl.DefaultExtension
	}
)

// Extend allows extending environment blocks with
// a global one. For example:
//
//	lint {
//	  log = <<EOS
//	    ...
//	  EOS
//	}
//
//	env "local" {
//	  ...
//	  lint {
//	    latest = 1
//	  }
//	}
//
//	env "ci" {
//	  ...
//	  lint {
//	    git {
//	      dir = "../"
//	      base = "master"
//	    }
//	  }
//	}
func (l *Lint) Extend(global *Lint) *Lint {
	if l == nil {
		return global
	}
	if l.Log == "" {
		l.Log = global.Log
	}
	l.Extra = global.Extra
	switch {
	// Changes detector was configured on the env.
	case l.Git.Dir != "" && l.Git.Base != "" || l.Latest != 0:
	// Inherit global git detection.
	case global.Git.Dir != "" || global.Git.Base != "":
		if global.Git.Dir != "" {
			l.Git.Dir = global.Git.Dir
		}
		if global.Git.Base != "" {
			l.Git.Base = global.Git.Base
		}
	// Inherit latest files configuration.
	case global.Latest != 0:
		l.Latest = global.Latest
	}
	return l
}

// Sources returns the paths containing the Atlas schema.
func (e *Env) Sources() ([]string, error) {
	attr, exists := e.Attr("src")
	if !exists {
		return nil, nil
	}
	if s, err := attr.String(); err == nil {
		return []string{s}, nil
	}
	if s, err := attr.Strings(); err == nil {
		return s, nil
	}
	return nil, errors.New("expected src to be either a string or a string array")
}

// asMap returns the extra attributes stored in the Env as a map[string]string.
func (e *Env) asMap() (map[string]string, error) {
	m := make(map[string]string, len(e.Extra.Attrs))
	for _, attr := range e.Extra.Attrs {
		if attr.K == "src" {
			continue
		}
		if v, err := attr.String(); err == nil {
			m[attr.K] = v
			continue
		}
		if lv, ok := attr.V.(*schemahcl.LiteralValue); ok {
			m[attr.K] = lv.V
		}
		return nil, fmt.Errorf("expecting attr %q to be a literal, got: %T", attr.K, attr.V)
	}
	return m, nil
}

var hclState = schemahcl.New(
	schemahcl.WithScopedEnums("env.migration.format", formatAtlas, formatFlyway, formatLiquibase, formatGoose, formatGolangMigrate),
)

// LoadEnv reads the project file in path, and loads the environment
// with the provided name into env.
func LoadEnv(path string, name string, opts ...LoadOption) (*Env, error) {
	cfg := &loadConfig{}
	for _, f := range opts {
		f(cfg)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	project := &Project{Lint: &Lint{}}
	if err := hclState.EvalBytes(b, project, cfg.inputVals); err != nil {
		return nil, err
	}
	envs := make(map[string]*Env)
	for _, e := range project.Envs {
		if _, ok := envs[e.Name]; ok {
			return nil, fmt.Errorf("duplicate environment name %q", e.Name)
		}
		if e.Name == "" {
			return nil, fmt.Errorf("all envs must have names on file %q", path)
		}
		if _, err := e.Sources(); err != nil {
			return nil, err
		}
		envs[e.Name] = e
	}
	selected, ok := envs[name]
	if !ok {
		return nil, fmt.Errorf("env %q not defined in project file", name)
	}
	if selected.Migration == nil {
		selected.Migration = &Migration{}
	}
	selected.Lint = selected.Lint.Extend(project.Lint)
	return selected, nil
}

func init() {
	schemahcl.Register("env", &Env{})
}
