// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

const projectFileName = "file://atlas.hcl"

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

		// Log of the environment.
		Log Log `spec:"log"`
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

	// Log represents a logging configuration of an environment.
	Log struct {
		Migrate struct {
			// Apply configures the logging for 'migrate apply'.
			Apply string `spec:"apply"`
			// Lint configures the logging for 'migrate lint'.
			Lint string `spec:"lint"`
			// Status configures the logging for 'migrate status'.
			Status string `spec:"status"`
		} `spec:"migrate"`
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
	schemahcl.WithDataSource("sql", func(ctx *hcl.EvalContext, b *hclsyntax.Block) (cty.Value, error) {
		return (&sqlsrc{ctx: ctx, block: b}).exec()
	}),
)

// LoadEnv reads the project file in path, and loads
// the environment instances with the provided name.
func LoadEnv(name string, opts ...LoadOption) ([]*Env, error) {
	cfg := &loadConfig{}
	for _, f := range opts {
		f(cfg)
	}
	u, err := url.Parse(GlobalFlags.ConfigURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "file" {
		return nil, fmt.Errorf("unsupported project file driver %q", u.Scheme)
	}
	path := filepath.Join(u.Host, u.Path)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = fmt.Errorf("project file %q was not found: %w", path, err)
		}
		return nil, err
	}
	project := &Project{Lint: &Lint{}}
	if err := hclState.EvalBytes(b, project, cfg.inputVals); err != nil {
		return nil, err
	}
	envs := make(map[string][]*Env)
	for _, e := range project.Envs {
		if e.Name == "" {
			return nil, fmt.Errorf("all envs must have names on file %q", path)
		}
		if _, err := e.Sources(); err != nil {
			return nil, err
		}
		if e.Migration == nil {
			e.Migration = &Migration{}
		}
		e.Lint = e.Lint.Extend(project.Lint)
		envs[e.Name] = append(envs[e.Name], e)
	}
	selected, ok := envs[name]
	if !ok {
		return nil, fmt.Errorf("env %q not defined in project file", name)
	}
	return selected, nil
}

func init() {
	schemahcl.Register("env", &Env{})
}

// sqlsrc represents an SQL data-source.
type sqlsrc struct {
	ctx   *hcl.EvalContext
	block *hclsyntax.Block
}

// exec executes the source block for getting the data.
func (s *sqlsrc) exec() (cty.Value, error) {
	attrs, diags := s.block.Body.JustAttributes()
	if diags.HasErrors() {
		return cty.NilVal, diags
	}
	u, err := s.stringAttr(attrs, "url")
	if err != nil {
		return cty.NilVal, err
	}
	query, err := s.stringAttr(attrs, "query")
	if err != nil {
		return cty.NilVal, err
	}
	var args []any
	if at, ok := attrs["args"]; ok {
		switch v, diags := at.Expr.Value(s.ctx); {
		case diags.HasErrors():
			return cty.NilVal, s.errorf(`evaluating "args": %w`, err)
		case !v.CanIterateElements():
			return cty.NilVal, s.errorf(`attribute "args" must be a list, got: %s`, v.Type())
		default:
			for it := v.ElementIterator(); it.Next(); {
				switch _, v := it.Element(); v.Type() {
				case cty.String:
					args = append(args, v.AsString())
				case cty.Number:
					f, _ := v.AsBigFloat().Float64()
					args = append(args, f)
				case cty.Bool:
					args = append(args, v.True())
				default:
					return cty.NilVal, fmt.Errorf(`attribute "args" must contain primitive types, got: %s`, v.Type())
				}
			}
		}
	}
	var values []cty.Value
	c, err := sqlclient.Open(context.Background(), u)
	if err != nil {
		return cty.NilVal, s.errorf("opening connection: %w", err)
	}
	defer c.Close()
	rows, err := c.QueryContext(context.Background(), query, args...)
	if err != nil {
		return cty.NilVal, s.errorf("executing query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v any
		if err := rows.Scan(&v); err != nil {
			return cty.NilVal, s.errorf("scanning row: %w", err)
		}
		switch v := v.(type) {
		case bool:
			values = append(values, cty.BoolVal(v))
		case int64:
			values = append(values, cty.NumberIntVal(v))
		case float64:
			values = append(values, cty.NumberFloatVal(v))
		case string:
			values = append(values, cty.StringVal(v))
		case []byte:
			values = append(values, cty.StringVal(string(v)))
		default:
			return cty.NilVal, s.errorf("unsupported row type: %T", v)
		}
	}
	obj := map[string]cty.Value{
		"count":  cty.NumberIntVal(int64(len(values))),
		"values": cty.ListValEmpty(cty.NilType),
		"value":  cty.NilVal,
	}
	if len(values) > 0 {
		obj["value"] = values[0]
		obj["values"] = cty.ListVal(values)
	}
	return cty.ObjectVal(obj), nil
}

func (s *sqlsrc) stringAttr(attrs hcl.Attributes, name string) (string, error) {
	at, ok := attrs[name]
	if !ok {
		return "", s.errorf("missing %q attribute", name)
	}
	u, diags := at.Expr.Value(s.ctx)
	if diags.HasErrors() {
		return "", s.errorf("evaluating %q attribute: %w", name, diags)
	}
	if u.Type() != cty.String {
		return "", s.errorf("attribute %q must be a string, got: %s", name, u.Type())
	}
	return u.AsString(), nil
}

func (s *sqlsrc) errorf(format string, args ...any) error {
	return fmt.Errorf("data.sql.%s: %w", s.block.Labels[1], fmt.Errorf(format, args...))
}
