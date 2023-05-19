// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

// Package cmdext provides extensions to the Atlas CLI that
// may be moved to a separate repository in the future.
package cmdext

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"text/template"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/cloudapi"
	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"ariga.io/atlas/sql/sqltool"

	"entgo.io/ent/dialect/sql"
	entschema "entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"gocloud.dev/runtimevar"
	_ "gocloud.dev/runtimevar/awssecretsmanager"
	_ "gocloud.dev/runtimevar/constantvar"
	_ "gocloud.dev/runtimevar/filevar"
	_ "gocloud.dev/runtimevar/gcpsecretmanager"
	_ "gocloud.dev/runtimevar/httpvar"
)

// DataSources exposes the data sources provided by this package.
var DataSources = []schemahcl.Option{
	schemahcl.WithDataSource("sql", QuerySrc),
	schemahcl.WithDataSource("runtimevar", RuntimeVarSrc),
	schemahcl.WithDataSource("template_dir", TemplateDir),
	schemahcl.WithDataSource("remote_dir", RemoteDir),
}

// RuntimeVarSrc exposes the gocloud.dev/runtimevar as a schemahcl datasource.
//
//	data "runtimevar" "pass" {
//	  url = "driver://path?query=param"
//	}
//
//	locals {
//	  url = "mysql://root:${data.runtimevar.pass}@:3306/"
//	}
func RuntimeVarSrc(c *hcl.EvalContext, block *hclsyntax.Block) (cty.Value, error) {
	var (
		args struct {
			URL string `hcl:"url"`
		}
		ctx    = context.Background()
		errorf = blockError("data.runtimevar", block)
	)
	if diags := gohcl.DecodeBody(block.Body, c, &args); diags.HasErrors() {
		return cty.NilVal, errorf("decoding body: %v", diags)
	}
	u, err := url.Parse(args.URL)
	if err != nil {
		return cty.NilVal, errorf("parsing url: %v", err)
	}
	if d := u.Query().Get("decoder"); d != "" && d != "string" {
		return cty.NilVal, errorf("unsupported decoder: %q", d)
	}
	q := u.Query()
	q.Set("decoder", "string")
	// Default timeout is 10s unless specified otherwise.
	timeout := 10 * time.Second
	if t := q.Get("timeout"); t != "" {
		if timeout, err = time.ParseDuration(t); err != nil {
			return cty.NilVal, errorf("parsing timeout: %v", err)
		}
		q.Del("timeout")
	}
	u.RawQuery = q.Encode()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	vr, err := runtimevar.OpenVariable(ctx, u.String())
	if err != nil {
		return cty.Value{}, errorf("opening variable: %v", err)
	}
	defer vr.Close()
	snap, err := vr.Latest(ctx)
	if err != nil {
		return cty.Value{}, errorf("getting latest snapshot: %v", err)
	}
	sv, ok := snap.Value.(string)
	if !ok {
		return cty.Value{}, errorf("unexpected snapshot value type: %T", snap.Value)
	}
	return cty.StringVal(sv), nil
}

// QuerySrc exposes the database/sql.Query as a schemahcl datasource.
//
//	data "sql" "tenants" {
//	  url = var.url
//	  query = <query>
//	  args = [<arg1>, <arg2>, ...]
//	}
//
//	env "prod" {
//	  for_each = toset(data.sql.tenants.values)
//	  url      = urlsetpath(var.url, each.value)
//	}
func QuerySrc(ctx *hcl.EvalContext, block *hclsyntax.Block) (cty.Value, error) {
	var (
		args struct {
			URL    string   `hcl:"url"`
			Query  string   `hcl:"query"`
			Remain hcl.Body `hcl:",remain"`
			Args   []any
		}
		values []cty.Value
		errorf = blockError("data.sql", block)
	)
	if diags := gohcl.DecodeBody(block.Body, ctx, &args); diags.HasErrors() {
		return cty.NilVal, errorf("decoding body: %v", diags)
	}
	attrs, diags := args.Remain.JustAttributes()
	if diags.HasErrors() {
		return cty.NilVal, errorf("getting attributes: %v", diags)
	}
	if at, ok := attrs["args"]; ok {
		switch v, diags := at.Expr.Value(ctx); {
		case diags.HasErrors():
			return cty.NilVal, errorf(`evaluating "args": %w`, diags)
		case !v.CanIterateElements():
			return cty.NilVal, errorf(`attribute "args" must be a list, got: %s`, v.Type())
		default:
			for it := v.ElementIterator(); it.Next(); {
				switch _, v := it.Element(); v.Type() {
				case cty.String:
					args.Args = append(args.Args, v.AsString())
				case cty.Number:
					f, _ := v.AsBigFloat().Float64()
					args.Args = append(args.Args, f)
				case cty.Bool:
					args.Args = append(args.Args, v.True())
				default:
					return cty.NilVal, errorf(`attribute "args" must be a list of strings, numbers or booleans, got: %s`, v.Type())
				}
			}
		}
		delete(attrs, "args")
	}
	if len(attrs) > 0 {
		return cty.NilVal, errorf("unexpected attributes: %v", attrs)
	}
	c, err := sqlclient.Open(context.Background(), args.URL)
	if err != nil {
		return cty.NilVal, errorf("opening connection: %w", err)
	}
	defer c.Close()
	rows, err := c.QueryContext(context.Background(), args.Query, args.Args...)
	if err != nil {
		return cty.NilVal, errorf("executing query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v any
		if err := rows.Scan(&v); err != nil {
			return cty.NilVal, errorf("scanning row: %w", err)
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
			return cty.NilVal, errorf("unsupported row type: %T", v)
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

// TemplateDir implements migrate.Dir interface for template directories.
//
//	data "template_dir" "name" {
//	  path = "path/to/directory"
//	  vars = {
//	    Env  = atlas.env
//	    Seed = var.seed
//	  }
//	}
//
//	env "dev" {
//	  url = "driver://path?query=param"
//	  migration {
//	    dir = data.template_dir.name.url
//	  }
//	}
func TemplateDir(ctx *hcl.EvalContext, block *hclsyntax.Block) (cty.Value, error) {
	var (
		args struct {
			Path   string   `hcl:"path"`
			Remain hcl.Body `hcl:",remain"`
		}
		vars   = make(map[string]any)
		errorf = blockError("data.template_dir", block)
	)
	if diags := gohcl.DecodeBody(block.Body, ctx, &args); diags.HasErrors() {
		return cty.NilVal, errorf("decoding body: %v", diags)
	}
	attrs, diags := args.Remain.JustAttributes()
	if diags.HasErrors() {
		return cty.NilVal, errorf("getting attributes: %v", diags)
	}
	if vs, ok := attrs["vars"]; ok {
		switch vs, diags := vs.Expr.Value(ctx); {
		case diags.HasErrors():
			return cty.NilVal, errorf(`evaluating "vars": %w`, diags)
		case !vs.CanIterateElements():
			return cty.NilVal, errorf(`attribute "vars" must be a map, got: %s`, vs.Type())
		default:
			for it := vs.ElementIterator(); it.Next(); {
				k, v := it.Element()
				switch v.Type() {
				case cty.String:
					vars[k.AsString()] = v.AsString()
				case cty.Number:
					f, _ := v.AsBigFloat().Float64()
					vars[k.AsString()] = f
				case cty.Bool:
					vars[k.AsString()] = v.True()
				default:
					return cty.NilVal, errorf(`attribute "vars" must be a map of strings, numbers or booleans, got: %s`, v.Type())
				}
			}
		}
		delete(attrs, "vars")
	}
	if len(attrs) > 0 {
		return cty.NilVal, errorf("unexpected attributes: %v", attrs)
	}
	t := template.New("template_dir")
	err := filepath.Walk(args.Path, func(path string, d os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk path %s: %w", path, err)
		}
		if !d.IsDir() {
			_, err = t.ParseFiles(path)
		}
		return err
	})
	if err != nil {
		return cty.NilVal, errorf(err.Error())
	}
	dir := migrate.OpenMemDir(args.Path)
	// Only top-level (template) files are treated as migrations.
	matches, err := fs.Glob(os.DirFS(args.Path), "*.sql")
	if err != nil {
		return cty.NilVal, errorf("globbing templates: %w", err)
	}
	for _, m := range matches {
		var b bytes.Buffer
		if err := t.ExecuteTemplate(&b, m, vars); err != nil {
			return cty.NilVal, errorf("executing template %q: %w", m, err)
		}
		if err := dir.WriteFile(m, b.Bytes()); err != nil {
			return cty.NilVal, errorf("writing file %q: %w", m, err)
		}
	}
	sum, err := dir.Checksum()
	if err != nil {
		return cty.NilVal, err
	}
	b, err := sum.MarshalText()
	if err != nil {
		return cty.NilVal, err
	}
	if err := dir.WriteFile(migrate.HashFileName, b); err != nil {
		return cty.NilVal, err
	}
	return cty.ObjectVal(map[string]cty.Value{
		"url": cty.StringVal(fmt.Sprintf("mem://%s", args.Path)),
	}), nil
}

// AtlasConfig exposes non-sensitive information returned by the "atlas" init-block.
// By invoking AtlasInitBlock() a new config is returned that is set by the init block
// defined and executed on schemahcl Eval functions.
type AtlasConfig struct {
	Client  *cloudapi.Client // Client attached to Atlas Cloud.
	Project string           // Optional project.
}

// DefaultProjectName is the default name for projects.
const DefaultProjectName = "default"

// InitBlock returns the handler for the "atlas" init block.
//
//	atlas {
//	  cloud {
//	    token   = data.runtimevar.token  // User token.
//	    url     = var.cloud_url          // Optional URL.
//	    project = var.project            // Optional project. If set, cloud reporting is enabled.
//	  }
//	}
func (c *AtlasConfig) InitBlock() schemahcl.Option {
	return schemahcl.WithInitBlock("atlas", func(ctx *hcl.EvalContext, block *hclsyntax.Block) (cty.Value, error) {
		var args struct {
			Cloud struct {
				Token   string `hcl:"token"`
				URL     string `hcl:"url,optional"`
				Project string `hcl:"project,optional"`
			} `hcl:"cloud,block"`
		}
		if diags := gohcl.DecodeBody(block.Body, ctx, &args); diags.HasErrors() {
			return cty.NilVal, fmt.Errorf("atlas.cloud: decoding body: %v", diags)
		}
		if args.Cloud.Project == "" {
			args.Cloud.Project = DefaultProjectName
		}
		c.Project = args.Cloud.Project
		c.Client = cloudapi.New(args.Cloud.URL, args.Cloud.Token)
		cloud := cty.ObjectVal(map[string]cty.Value{
			"client":  cty.CapsuleVal(clientType, c.Client),
			"project": cty.StringVal(args.Cloud.Project),
		})
		av, diags := (&hclsyntax.ScopeTraversalExpr{
			Traversal: hcl.Traversal{hcl.TraverseRoot{Name: "atlas", SrcRange: block.Range()}},
		}).Value(ctx)
		switch {
		case !diags.HasErrors():
			m := av.AsValueMap()
			m["cloud"] = cloud
			return cty.ObjectVal(m), nil
		case len(diags) == 1 && diags[0].Summary == "Unknown variable":
			return cty.ObjectVal(map[string]cty.Value{"cloud": cloud}), nil
		default:
			return cty.NilVal, fmt.Errorf("atlas.cloud: getting config: %v", diags)
		}
	})
}

var clientType = cty.Capsule("client", reflect.TypeOf(cloudapi.Client{}))

// RemoteDir is a data source that reads a remote migration directory.
func RemoteDir(ctx *hcl.EvalContext, block *hclsyntax.Block) (cty.Value, error) {
	var (
		args struct {
			Name string `hcl:"name"`
			Tag  string `hcl:"tag,optional"`
		}
		errorf = blockError("data.remote_dir", block)
	)
	if diags := gohcl.DecodeBody(block.Body, ctx, &args); diags.HasErrors() {
		return cty.NilVal, errorf("decoding body: %v", diags)
	}
	cv, diags := (&hclsyntax.ScopeTraversalExpr{
		Traversal: hcl.Traversal{
			hcl.TraverseRoot{Name: "atlas", SrcRange: block.Range()},
			hcl.TraverseAttr{Name: "cloud", SrcRange: block.Range()},
			hcl.TraverseAttr{Name: "client", SrcRange: block.Range()},
		},
	}).Value(ctx)
	if len(diags) == 1 && diags[0].Summary == "Unknown variable" {
		return cty.NilVal, errorf("missing atlas cloud config")
	} else if diags.HasErrors() {
		return cty.NilVal, errorf("getting atlas client: %v", diags)
	}
	client := cv.EncapsulatedValue().(*cloudapi.Client)
	u, err := memdir(client, args.Name, args.Tag)
	if err != nil {
		return cty.NilVal, errorf("reading remote dir: %v", err)
	}
	return cty.ObjectVal(map[string]cty.Value{
		"url": cty.StringVal(u),
	}), nil
}

func memdir(client *cloudapi.Client, dirName string, tag string) (string, error) {
	input := cloudapi.DirInput{
		Name: dirName,
		Tag:  tag,
	}
	dir, err := client.Dir(context.Background(), input)
	if err != nil {
		return "", err
	}
	md := migrate.OpenMemDir(dirName)
	files, err := dir.Files()
	if err != nil {
		return "", err
	}
	for _, f := range files {
		if err := md.WriteFile(f.Name(), f.Bytes()); err != nil {
			return "", err
		}
	}
	if hf, err := dir.Open(migrate.HashFileName); err == nil {
		b, err := io.ReadAll(hf)
		if err != nil {
			return "", err
		}
		if err := md.WriteFile(migrate.HashFileName, b); err != nil {
			return "", err
		}
	}
	return "mem://" + dirName, nil
}

func blockError(name string, b *hclsyntax.Block) func(string, ...any) error {
	return func(format string, args ...any) error {
		return fmt.Errorf("%s.%s: %w", name, b.Labels[1], fmt.Errorf(format, args...))
	}
}

type (
	// LoadStateOptions for external state loaders.
	LoadStateOptions struct {
		URLs []*url.URL
		Dev  *sqlclient.Client // Client for the dev database.
	}
	// StateLoader allows loading StateReader's from external sources.
	StateLoader interface {
		LoadState(context.Context, *LoadStateOptions) (migrate.StateReader, error)
	}

	// MigrateDiffOptions for external migration differ.
	MigrateDiffOptions struct {
		Name    string
		Indent  string
		To      []string
		Dir     migrate.Dir
		Dev     *sqlclient.Client
		Options []schema.DiffOption
	}
	// MigrateDiffer allows external sources to implement custom migration differs.
	MigrateDiffer interface {
		MigrateDiff(context.Context, *MigrateDiffOptions) error
		needDiff([]string) bool
	}
)

var (
	// States is a global registry for external state loaders.
	States = registry{
		"ent": EntLoader{},
	}
	errNotSchemaURL = errors.New("missing schema in --dev-url. See: https://atlasgo.io/url")
)

type registry map[string]StateLoader

// Loader returns the state loader for the given scheme.
func (r registry) Loader(scheme string) (StateLoader, bool) {
	l, ok := r[scheme]
	return l, ok
}

// Differ returns the raw states differ for the given URLs, if registered.
func (r registry) Differ(to []string) (MigrateDiffer, bool) {
	for _, l := range r {
		if d, ok := l.(MigrateDiffer); ok && d.needDiff(to) {
			return d, true
		}
	}
	return nil, false
}

// EntLoader is a StateLoader for loading ent.Schema's as StateReader's.
type EntLoader struct{}

// LoadState returns a migrate.StateReader that reads the schema from an ent.Schema.
func (l EntLoader) LoadState(ctx context.Context, opts *LoadStateOptions) (migrate.StateReader, error) {
	switch {
	case len(opts.URLs) != 1:
		return nil, errors.New(`"ent://" requires exactly one schema URL`)
	case opts.Dev == nil:
		return nil, errors.New(`required flag "--dev-url" not set`)
	case opts.Dev.URL.Schema == "":
		return nil, errNotSchemaURL
	case opts.URLs[0].Query().Has("globalid"):
		return nil, errors.New("globalid is not supported by this command. Use 'migrate diff' instead")
	}
	tables, err := l.tables(opts.URLs[0])
	if err != nil {
		return nil, err
	}
	m, err := entschema.NewMigrate(sql.OpenDB(opts.Dev.Name, opts.Dev.DB))
	if err != nil {
		return nil, fmt.Errorf("creating migrate reader: %w", err)
	}
	realm, err := m.StateReader(tables...).ReadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading schema state: %w", err)
	}
	if nr, ok := opts.Dev.Driver.(schema.Normalizer); ok {
		if realm, err = nr.NormalizeRealm(ctx, realm); err != nil {
			return nil, err
		}
	}
	if len(realm.Schemas) != 1 {
		return nil, fmt.Errorf("expect exactly one schema, got %d", len(realm.Schemas))
	}
	// Use the dev-database schema name if the schema name is empty.
	if realm.Schemas[0].Name == "" && opts.Dev.URL.Schema != "" {
		realm.Schemas[0].Name = opts.Dev.URL.Schema
	}
	for _, t := range realm.Schemas[0].Tables {
		t.Schema = realm.Schemas[0]
	}
	return migrate.Realm(realm), nil
}

// MigrateDiff returns the diff between ent.Schema and a directory.
func (l EntLoader) MigrateDiff(ctx context.Context, opts *MigrateDiffOptions) error {
	if !l.needDiff(opts.To) {
		return errors.New("invalid diff call")
	}
	if opts.Dev.URL.Schema == "" {
		return errNotSchemaURL
	}
	u, err := url.Parse(opts.To[0])
	if err != nil {
		return nil
	}
	tables, err := l.tables(u)
	if err != nil {
		return err
	}
	m, err := entschema.NewMigrate(
		sql.OpenDB(opts.Dev.Name, opts.Dev.DB),
		entschema.WithDir(opts.Dir),
		entschema.WithDropColumn(true),
		entschema.WithDropIndex(true),
		entschema.WithErrNoPlan(true),
		entschema.WithFormatter(dirFormatter(opts.Dir)),
		entschema.WithGlobalUniqueID(true),
		entschema.WithIndent(opts.Indent),
		entschema.WithMigrationMode(entschema.ModeReplay),
		entschema.WithDiffOptions(opts.Options...),
	)
	if err != nil {
		return fmt.Errorf("creating migrate reader: %w", err)
	}
	return m.NamedDiff(ctx, opts.Name, tables...)
}

// needDiff indicates if we need to offload the diffing to Ent in
// case global unique id is enabled in versioned migration mode.
func (EntLoader) needDiff(to []string) bool {
	if len(to) != 1 {
		return false
	}
	u1, err := url.Parse(to[0])
	if err != nil || u1.Scheme != "ent" {
		return false
	}
	gid, _ := strconv.ParseBool(u1.Query().Get("globalid"))
	return gid
}

func (EntLoader) tables(u *url.URL) ([]*entschema.Table, error) {
	abs, err := filepath.Abs(filepath.Join(u.Host, u.Path))
	if err != nil {
		return nil, err
	}
	graph, err := entc.LoadGraph(abs, &gen.Config{})
	if err != nil {
		return nil, fmt.Errorf("loading schema: %w", err)
	}
	return graph.Tables()
}

func dirFormatter(dir migrate.Dir) migrate.Formatter {
	switch dir.(type) {
	case *sqltool.DBMateDir:
		return sqltool.DBMateFormatter
	case *sqltool.GolangMigrateDir:
		return sqltool.GolangMigrateFormatter
	case *sqltool.GooseDir:
		return sqltool.GooseFormatter
	case *sqltool.FlywayDir:
		return sqltool.FlywayFormatter
	case *sqltool.LiquibaseDir:
		return sqltool.LiquibaseFormatter
	default:
		return migrate.DefaultFormatter
	}
}
