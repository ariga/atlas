// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"ariga.io/atlas/cmd/atlas/internal/cloudapi"
	"ariga.io/atlas/cmd/atlas/internal/cmdext"
	cmdmigrate "ariga.io/atlas/cmd/atlas/internal/migrate"
	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/schema"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/spf13/cobra"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type (
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

		// Schema containing the schema configuration of the env.
		Schema *Schema `spec:"schema"`

		// Migration containing the migration configuration of the env.
		Migration *Migration `spec:"migration"`

		// Diff policy of the environment.
		Diff *Diff `spec:"diff"`

		// Lint policy of the environment.
		Lint *Lint `spec:"lint"`

		// Format of the environment.
		Format Format `spec:"format"`

		// Test configuration of the environment.
		Test *Test `spec:"test"`

		schemahcl.DefaultExtension
		cloud  *cmdext.AtlasConfig
		config *Project
	}

	// Migration represents the migration directory for the Env.
	Migration struct {
		Dir             string   `spec:"dir"`
		Exclude         []string `spec:"exclude"`
		Format          string   `spec:"format"`
		Baseline        string   `spec:"baseline"`
		ExecOrder       string   `spec:"exec_order"`
		LockTimeout     string   `spec:"lock_timeout"`
		RevisionsSchema string   `spec:"revisions_schema"`
		Repo            *Repo    `spec:"repo"`
	}

	// Schema represents a schema in the registry.
	Schema struct {
		// The extension holds the "src" attribute.
		// It can be a string, or a list of strings.
		schemahcl.DefaultExtension
		Repo *Repo `spec:"repo"`
	}

	// Repo represents a repository in the schema registry
	// for a schema or migrations directory.
	Repo struct {
		Name string `spec:"name"` // Name of the repository.
	}

	// Lint represents the configuration of migration linting.
	Lint struct {
		// Format configures the --format option.
		Format string `spec:"log"`
		// Latest configures the --latest option.
		Latest int `spec:"latest"`
		Git    struct {
			// Dir configures the --git-dir option.
			Dir string `spec:"dir"`
			// Base configures the --git-base option.
			Base string `spec:"base"`
		} `spec:"git"`
		// Review defines when Atlas will ask the user to review and approve the changes.
		Review string `spec:"review"`
		schemahcl.DefaultExtension
	}

	// Diff represents the schema diffing policy.
	Diff struct {
		// SkipChanges configures the skip changes policy.
		SkipChanges *SkipChanges `spec:"skip"`
		schemahcl.DefaultExtension
	}

	// Test represents the test configuration of a project or environment.
	Test struct {
		// Schema represents the 'schema test' configuration.
		Schema struct {
			Src  []string `spec:"src"`
			Vars Vars     `spec:"vars"`
		} `spec:"schema"`
		// Migrate represents the 'migrate test' configuration.
		Migrate struct {
			Src  []string `spec:"src"`
			Vars Vars     `spec:"vars"`
		} `spec:"migrate"`
	}

	// SkipChanges represents the skip changes policy.
	SkipChanges struct {
		AddSchema        bool `spec:"add_schema"`
		DropSchema       bool `spec:"drop_schema"`
		ModifySchema     bool `spec:"modify_schema"`
		AddTable         bool `spec:"add_table"`
		DropTable        bool `spec:"drop_table"`
		ModifyTable      bool `spec:"modify_table"`
		RenameTable      bool `spec:"rename_table"`
		AddColumn        bool `spec:"add_column"`
		DropColumn       bool `spec:"drop_column"`
		ModifyColumn     bool `spec:"modify_column"`
		AddIndex         bool `spec:"add_index"`
		DropIndex        bool `spec:"drop_index"`
		ModifyIndex      bool `spec:"modify_index"`
		AddForeignKey    bool `spec:"add_foreign_key"`
		DropForeignKey   bool `spec:"drop_foreign_key"`
		ModifyForeignKey bool `spec:"modify_foreign_key"`
		AddView          bool `spec:"add_view"`
		DropView         bool `spec:"drop_view"`
		ModifyView       bool `spec:"modify_view"`
		RenameView       bool `spec:"rename_view"`
		AddFunc          bool `spec:"add_func"`
		DropFunc         bool `spec:"drop_func"`
		ModifyFunc       bool `spec:"modify_func"`
		RenameFunc       bool `spec:"rename_func"`
		AddProc          bool `spec:"add_proc"`
		DropProc         bool `spec:"drop_proc"`
		ModifyProc       bool `spec:"modify_proc"`
		RenameProc       bool `spec:"rename_proc"`
		AddTrigger       bool `spec:"add_trigger"`
		DropTrigger      bool `spec:"drop_trigger"`
		ModifyTrigger    bool `spec:"modify_trigger"`
		RenameTrigger    bool `spec:"rename_trigger"`
	}

	// Format represents the output formatting configuration of an environment.
	Format struct {
		Migrate struct {
			// Apply configures the formatting for 'migrate apply'.
			Apply string `spec:"apply"`
			// Down configures the formatting for 'migrate down'.
			Down string `spec:"down"`
			// Lint configures the formatting for 'migrate lint'.
			Lint string `spec:"lint"`
			// Status configures the formatting for 'migrate status'.
			Status string `spec:"status"`
			// Apply configures the formatting for 'migrate diff'.
			Diff string `spec:"diff"`
		} `spec:"migrate"`
		Schema struct {
			// Clean configures the formatting for 'schema clean'.
			Clean string `spec:"clean"`
			// Inspect configures the formatting for 'schema inspect'.
			Inspect string `spec:"inspect"`
			// Apply configures the formatting for 'schema apply'.
			Apply string `spec:"apply"`
			// Apply configures the formatting for 'schema diff'.
			Diff string `spec:"diff"`
			// Push configures the formatting for 'schema push'.
			Push string `spec:"push"`
		} `spec:"schema"`
		schemahcl.DefaultExtension
	}
)

// envScheme defines the scheme that can be used to reference env attributes.
const envAttrScheme = "env"

// MigrationRepo returns the migration repository name, if set.
func (e *Env) MigrationRepo() (s string) {
	if e != nil && e.Migration != nil && e.Migration.Repo != nil {
		s = e.Migration.Repo.Name
	}
	return
}

// MigrationExclude returns the exclusion patterns of the migration directory.
func (e *Env) MigrationExclude() []string {
	if e != nil && e.Migration != nil {
		return e.Migration.Exclude
	}
	return nil
}

// SchemaRepo returns the desired schema repository name, if set.
func (e *Env) SchemaRepo() (s string) {
	if e != nil && e.Schema != nil && e.Schema.Repo != nil {
		s = e.Schema.Repo.Name
	}
	return
}

// LintReview returns the review mode for the lint command.
func (e *Env) LintReview() string {
	if e != nil && e.Lint != nil && e.Lint.Review != "" {
		return e.Lint.Review
	}
	return ReviewAlways
}

// VarFromURL returns the string variable (env attribute) from the URL.
func (e *Env) VarFromURL(s string) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}
	if u.Host == "" || u.Path != "" || u.RawQuery != "" {
		return "", fmt.Errorf("invalid env:// variable %q", s)
	}
	var sv string
	switch u.Host {
	case "url":
		sv = e.URL
	case "dev":
		sv = e.DevURL
	case "src", "schema.src":
		var (
			ok   bool
			attr *schemahcl.Attr
		)
		switch {
		case u.Host == "src":
			attr, ok = e.Attr("src")
		case e.Schema != nil:
			attr, ok = e.Schema.Attr("src")
		}
		if !ok {
			return "", fmt.Errorf("env://%s: no src attribute defined in env %q", u.Host, e.Name)
		}
		switch attr.V.Type() {
		case cty.String:
			s, err := attr.String()
			if err != nil {
				return "", fmt.Errorf("env://%s: %w", u.Host, err)
			}
			return s, nil
		case cty.List(cty.String):
			vs, err := attr.Strings()
			if err != nil {
				return "", fmt.Errorf("env://%s: %w", u.Host, err)
			}
			if len(vs) != 0 {
				return "", fmt.Errorf("env://%s: expect one schema in env %q, got %d", u.Host, e.Name, len(vs))
			}
			return vs[0], nil
		default:
			return "", fmt.Errorf("env://%s: src attribute must be a string or list of strings, got %s", u.Host, attr.V.Type().FriendlyName())
		}
	case "migration.dir":
		if e.Migration == nil || e.Migration.Dir == "" {
			return "", fmt.Errorf("env://%s: no migration dir defined in env %q", u.Host, e.Name)
		}
		sv = e.Migration.Dir
	default:
		attr, ok := e.Attr(u.Host)
		if !ok {
			return "", fmt.Errorf("env://%s (attribute) not found in env.%s", u.Host, e.Name)
		}
		if sv, err = attr.String(); err != nil {
			return "", fmt.Errorf("env://%s: %w", u.Host, err)
		}
	}
	if strings.HasPrefix(sv, envAttrScheme+"://") {
		return "", fmt.Errorf("env://%s (attribute) cannot reference another env://", s)
	}
	return sv, nil
}

// support backward compatibility with the 'log' attribute.
func (e *Env) remainedLog() error {
	r, ok := e.Remain().Resource("log")
	if ok {
		return r.As(&e.Format)
	}
	return nil
}

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
	if l.Review == "" {
		l.Review = global.Review
	}
	if l.Format == "" {
		l.Format = global.Format
	}
	if len(l.Extra.Children) == 0 && len(l.Extra.Attrs) == 0 {
		l.Extra = global.Extra
	}
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

// support backward compatibility with the 'log' attribute.
func (l *Lint) remainedLog() error {
	at, ok := l.Remain().Attr("log")
	if !ok {
		return nil
	}
	if l.Format != "" {
		return fmt.Errorf("cannot use both 'log' and 'format' in the same lint block")
	}
	s, err := at.String()
	if err != nil {
		return err
	}
	l.Format = s
	return nil
}

// Extend allows extending environment blocks with
// a global one. For example:
//
//	diff {
//	  skip {
//	    drop_schema = true
//	  }
//	}
//
//	env "local" {
//	  ...
//	  diff {
//	    concurrent_index {
//	      create = true
//	      drop = true
//	    }
//	  }
//	}
func (d *Diff) Extend(global *Diff) *Diff {
	if d == nil {
		return global
	}
	if d.SkipChanges == nil {
		d.SkipChanges = global.SkipChanges
	}
	return d
}

// Options converts the diff policy into options.
func (d *Diff) Options() (opts []schema.DiffOption) {
	// Per-driver configuration.
	opts = append(opts, func(opts *schema.DiffOptions) {
		opts.Extra = d.DefaultExtension
	})
	if d.SkipChanges == nil {
		return
	}
	var (
		changes schema.Changes
		rv      = reflect.ValueOf(d.SkipChanges).Elem()
	)
	for _, c := range []schema.Change{
		&schema.AddSchema{}, &schema.DropSchema{}, &schema.ModifySchema{},
		&schema.AddView{}, &schema.DropView{}, &schema.ModifyView{}, &schema.RenameView{},
		&schema.AddFunc{}, &schema.DropFunc{}, &schema.ModifyFunc{}, &schema.RenameFunc{},
		&schema.AddProc{}, &schema.DropProc{}, &schema.ModifyProc{}, &schema.RenameProc{},
		&schema.AddTrigger{}, &schema.DropTrigger{}, &schema.ModifyTrigger{}, &schema.RenameTrigger{},
		&schema.AddTable{}, &schema.DropTable{}, &schema.ModifyTable{}, &schema.RenameTable{},
		&schema.AddColumn{}, &schema.DropColumn{}, &schema.ModifyColumn{}, &schema.AddIndex{},
		&schema.DropIndex{}, &schema.ModifyIndex{}, &schema.AddForeignKey{}, &schema.DropForeignKey{},
		&schema.ModifyForeignKey{},
	} {
		if rt := reflect.TypeOf(c).Elem(); rv.FieldByName(rt.Name()).Bool() {
			changes = append(changes, c)
		}
	}
	if len(changes) > 0 {
		opts = append(opts, schema.DiffSkipChanges(changes...))
	}
	return opts
}

// DiffOptions returns the diff options configured for the environment,
// or nil if no environment or diff policy were set.
func (e *Env) DiffOptions() []schema.DiffOption {
	if e == nil || e.Diff == nil {
		return nil
	}
	return e.Diff.Options()
}

// Sources returns the paths containing the Atlas desired schema.
// The "src" attribute predates the "schema" block. If the "schema"
// is defined, it takes precedence over the "src" attribute.
func (e *Env) Sources() ([]string, error) {
	var (
		ok   bool
		attr *schemahcl.Attr
	)
	if attr, ok = e.Attr("src"); !ok && e.Schema != nil {
		attr, ok = e.Schema.Attr("src")
	}
	if !ok {
		return nil, nil
	}
	switch attr.V.Type() {
	case cty.String:
		s, err := attr.String()
		if err != nil {
			return nil, err
		}
		return []string{s}, nil
	case cty.List(cty.String):
		return attr.Strings()
	default:
		return nil, fmt.Errorf("expected src to be either a string or strings, got: %s", attr.V.Type().FriendlyName())
	}
}

// Vars returns the extra attributes stored in the Env as a map[string]cty.Value.
func (e *Env) Vars() map[string]cty.Value {
	m := make(map[string]cty.Value, len(e.Extra.Attrs))
	for _, attr := range e.Extra.Attrs {
		if attr.K == "src" {
			continue
		}
		m[attr.K] = attr.V
	}
	// For backward compatibility, we append the GlobalFlags as variables.
	for k, v := range GlobalFlags.Vars {
		if _, ok := m[k]; !ok {
			m[k] = v
		}
	}
	return m
}

// Extend allows extending environment blocks with
// a global one. For example:
//
//	test {
//	  schema {
//	    src = [...]
//	  }
//	}
//
//	env "local" {
//	  ...
//	  test {
//	    schema {
//	      src = [...]
//	    }
//	  }
//	}
func (t *Test) Extend(global *Test) *Test {
	if t == nil {
		return global
	}
	return t
}

// EnvByName parses and returns the project configuration with selected environments.
func EnvByName(cmd *cobra.Command, name string, vars map[string]cty.Value) (*Project, []*Env, error) {
	envs := make(map[string][]*Env)
	defer func() {
		setEnvs(cmd.Context(), envs[name])
	}()
	if p, e, ok := envsCache.load(GlobalFlags.ConfigURL, name, vars); ok {
		return p, e, maySetLoginContext(cmd, p)
	}
	u, err := url.Parse(GlobalFlags.ConfigURL)
	if err != nil {
		return nil, nil, err
	}
	switch {
	case u.Scheme == "":
		return nil, nil, fmt.Errorf("missing scheme for config file. Did you mean file://%s?", u)
	case u.Scheme != "file":
		return nil, nil, fmt.Errorf("unsupported config file driver %q", u.Scheme)
	}
	path := filepath.Join(u.Host, u.Path)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			err = fmt.Errorf("config file %q was not found: %w", path, err)
		}
		return nil, nil, err
	}
	project, err := parseConfig(cmd.Context(), path, name, vars)
	if err != nil {
		return nil, nil, err
	}
	// The atlas.hcl token predates 'atlas login' command. If exists,
	// attach it to the context to indicate the user is authenticated.
	if err := maySetLoginContext(cmd, project); err != nil {
		return nil, nil, err
	}
	if err := project.Lint.remainedLog(); err != nil {
		return nil, nil, err
	}
	for _, e := range project.Envs {
		if e.Name == "" {
			return nil, nil, fmt.Errorf("all envs must have names on file %q", path)
		}
		if _, err := e.Sources(); err != nil {
			return nil, nil, err
		}
		if e.Migration == nil {
			e.Migration = &Migration{}
		}
		if err := e.remainedLog(); err != nil {
			return nil, nil, err
		}
		e.Diff = e.Diff.Extend(project.Diff)
		e.Lint = e.Lint.Extend(project.Lint)
		if err := e.Lint.remainedLog(); err != nil {
			return nil, nil, err
		}
		e.Test = e.Test.Extend(project.Test)
		envs[e.Name] = append(envs[e.Name], e)
	}
	envsCache.store(GlobalFlags.ConfigURL, name, vars, project, envs[name])
	switch {
	case name == "":
		// If no env was selected,
		// return only the project.
		return project, nil, nil
	case len(envs[name]) == 0:
		return nil, nil, fmt.Errorf("env %q not defined in config file", name)
	default:
		return project, envs[name], nil
	}
}

type (
	envCacheK struct {
		path, env, vars string
	}
	envCacheV struct {
		p *Project
		e []*Env
	}
	envCache struct {
		sync.RWMutex
		m map[envCacheK]envCacheV
	}
)

var envsCache = &envCache{m: make(map[envCacheK]envCacheV)}

func (c *envCache) load(path, env string, vars Vars) (*Project, []*Env, bool) {
	c.RLock()
	v, ok := c.m[envCacheK{path: path, env: env, vars: vars.String()}]
	c.RUnlock()
	return v.p, v.e, ok
}

func (c *envCache) store(path, env string, vars Vars, p *Project, e []*Env) {
	c.Lock()
	c.m[envCacheK{path: path, env: env, vars: vars.String()}] = envCacheV{p: p, e: e}
	c.Unlock()
}

const (
	blockEnv          = "env"
	refAtlas          = "atlas"
	defaultConfigPath = "file://atlas.hcl"
)

func parseConfig(ctx context.Context, path, env string, vars map[string]cty.Value) (*Project, error) {
	pr, err := partialParse(path, env)
	if err != nil {
		return nil, err
	}
	base, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	cloud := &cmdext.AtlasConfig{
		Project: cloudapi.DefaultProjectName,
	}
	state := schemahcl.New(
		append(
			append(cmdext.SpecOptions, specOptions...),
			cloud.InitBlock(),
			schemahcl.WithContext(ctx),
			schemahcl.WithScopedEnums("env.migration.format", cmdmigrate.Formats...),
			schemahcl.WithScopedEnums("env.migration.exec_order", "LINEAR", "LINEAR_SKIP", "NON_LINEAR"),
			schemahcl.WithScopedEnums("env.lint.review", ReviewModes...),
			schemahcl.WithScopedEnums("lint.review", ReviewModes...),
			schemahcl.WithVariables(map[string]cty.Value{
				refAtlas: cty.ObjectVal(map[string]cty.Value{
					blockEnv: cty.StringVal(env),
				}),
			}),
			schemahcl.WithFunctions(map[string]function.Function{
				"file":    schemahcl.MakeFileFunc(base),
				"glob":    schemahcl.MakeGlobFunc(base),
				"fileset": schemahcl.MakeFileSetFunc(base),
				"getenv":  getEnvFunc,
			}),
		)...,
	)
	p := &Project{Lint: &Lint{}, Diff: &Diff{}, cloud: cloud}
	if err := state.Eval(pr, p, vars); err != nil {
		return nil, err
	}
	for _, e := range p.Envs {
		e.config, e.cloud = p, cloud
	}
	return p, nil
}

func init() {
	cloudapi.SetVersion(version, flavor)
	schemahcl.Register(blockEnv, &Env{})
}

func partialParse(path, env string) (*hclparse.Parser, error) {
	parser := hclparse.NewParser()
	fi, err := parser.ParseHCLFile(path)
	if err != nil {
		return nil, err
	}
	var labeled, nonlabeled, used []*hclsyntax.Block
	for _, b := range fi.Body.(*hclsyntax.Body).Blocks {
		switch b.Type {
		case blockEnv:
			switch n := len(b.Labels); {
			// No env was selected.
			case env == "" && n == 0:
			// Exact env was selected.
			case n == 1 && b.Labels[0] == env:
				labeled = append(labeled, b)
			// Dynamic env selection.
			case n == 0 && b.Body != nil && b.Body.Attributes[schemahcl.AttrName] != nil:
				t, ok := b.Body.Attributes[schemahcl.AttrName].Expr.(*hclsyntax.ScopeTraversalExpr)
				if ok && len(t.Traversal) == 2 && t.Traversal.RootName() == refAtlas && t.Traversal[1].(hcl.TraverseAttr).Name == blockEnv {
					nonlabeled = append(nonlabeled, b)
				}
			}
		default:
			used = append(used, b)
		}
	}
	// Labeled blocks take precedence
	// over non-labeled env blocks.
	switch {
	case len(labeled) > 0:
		used = append(used, labeled...)
	case len(nonlabeled) > 0:
		used = append(used, nonlabeled...)
	}
	fi.Body = &hclsyntax.Body{
		Blocks:     used,
		Attributes: fi.Body.(*hclsyntax.Body).Attributes,
	}
	return parser, nil
}

// Review modes for 'schema apply'.
const (
	ReviewAlways  = "ALWAYS"  // Always review changes. The default mode.
	ReviewWarning = "WARNING" // Review changes only if there are any diagnostics (including warnings).
	ReviewError   = "ERROR"   // Review changes only if there are severe diagnostics (error level).
)

var ReviewModes = []string{ReviewAlways, ReviewWarning, ReviewError}

// getEnvFunc is a custom HCL function that returns
// the value of an environment variable.
var getEnvFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "key",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return cty.StringVal(os.Getenv(args[0].AsString())), nil
	},
})
