// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

// Package cmdext provides extensions to the Atlas configuration
// file such as schema loaders, data sources and cloud connectors.
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
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"text/template"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/cloudapi"
	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"ariga.io/atlas/sql/sqltool"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"golang.org/x/oauth2/google"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	"gocloud.dev/runtimevar"
	_ "gocloud.dev/runtimevar/awsparamstore"
	_ "gocloud.dev/runtimevar/awssecretsmanager"
	_ "gocloud.dev/runtimevar/constantvar"
	_ "gocloud.dev/runtimevar/filevar"
	_ "gocloud.dev/runtimevar/gcpruntimeconfig"
	_ "gocloud.dev/runtimevar/gcpsecretmanager"
	_ "gocloud.dev/runtimevar/httpvar"
)

// SpecOptions exposes the schema spec options like data-sources provided by this package.
var SpecOptions = append(
	[]schemahcl.Option{
		schemahcl.WithDataSource("sql", Query),
		schemahcl.WithDataSource("external", External),
		schemahcl.WithDataSource("runtimevar", RuntimeVar),
		schemahcl.WithDataSource("template_dir", TemplateDir),
		schemahcl.WithDataSource("remote_dir", RemoteDir),
		schemahcl.WithDataSource("remote_schema", RemoteSchema),
		schemahcl.WithDataSource("hcl_schema", SchemaHCL),
		schemahcl.WithDataSource("external_schema", SchemaExternal),
		schemahcl.WithDataSource("aws_rds_token", AWSRDSToken),
		schemahcl.WithDataSource("gcp_cloudsql_token", GCPCloudSQLToken),
	},
	specOptions...,
)

// RuntimeVar exposes the gocloud.dev/runtimevar as a schemahcl datasource.
//
//	data "runtimevar" "pass" {
//	  url = "driver://path?query=param"
//	}
//
//	locals {
//	  url = "mysql://root:${data.runtimevar.pass}@:3306/"
//	}
func RuntimeVar(c *hcl.EvalContext, block *hclsyntax.Block) (cty.Value, error) {
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
	u, err := sqlclient.ParseURL(args.URL)
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

// AWSRDSToken exposes an AWS RDS token as a schemahcl datasource.
//
//	data "aws_rds_token" "token" {
//		endpoint = "db.hostname.io:3306"
//		region   = "us-east-1"
//		username = "admin"
//		profile  = "prod-ext"
//	}
func AWSRDSToken(ctx *hcl.EvalContext, block *hclsyntax.Block) (cty.Value, error) {
	var (
		args struct {
			Endpoint string `hcl:"endpoint"`
			Region   string `hcl:"region,optional"`
			Username string `hcl:"username"`
			Profile  string `hcl:"profile,optional"`
		}
		errorf = blockError("data.aws_rds_token", block)
	)
	if diags := gohcl.DecodeBody(block.Body, ctx, &args); diags.HasErrors() {
		return cty.NilVal, errorf("decoding body: %v", diags)
	}
	bgctx := context.Background()
	cfg, err := config.LoadDefaultConfig(
		bgctx,
		config.WithSharedConfigProfile(args.Profile), // Ignored if empty.
	)
	if err != nil {
		return cty.NilVal, errorf("loading aws config: %v", err)
	}
	if args.Region == "" {
		args.Region = cfg.Region
	}
	token, err := auth.BuildAuthToken(bgctx, args.Endpoint, args.Region, args.Username, cfg.Credentials)
	if err != nil {
		return cty.NilVal, errorf("building auth token: %v", err)
	}
	return cty.StringVal(token), nil
}

// GCPCloudSQLToken exposes a CloudSQL token as a schemahcl datasource.
//
//	data "gcp_cloudsql_token" "hello" {}
func GCPCloudSQLToken(_ *hcl.EvalContext, block *hclsyntax.Block) (cty.Value, error) {
	errorf := blockError("data.gcp_cloudsql_token", block)
	bgctx := context.Background()
	ts, err := google.DefaultTokenSource(bgctx, sqladmin.SqlserviceAdminScope)
	if err != nil {
		return cty.NilVal, errorf("finding default credentials: %v", err)
	}
	token, err := ts.Token()
	if err != nil {
		return cty.NilVal, errorf("getting token: %v", err)
	}
	return cty.StringVal(token.AccessToken), nil
}

// Query exposes the database/sql.Query as a schemahcl datasource.
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
func Query(ctx *hcl.EvalContext, block *hclsyntax.Block) (cty.Value, error) {
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

// External allows loading data using external program execution.
//
//	data "external" "env1" {
//	  program = [
//	    "node",
//	    loadenv.js",
//	  ]
//	}
//
//	data "external" "env2" {
//	  program = [
//	    "bash",
//	    "-c",
//	    "env_to_json --file=${var.envfile} | jq '...' ",
//	  ]
//	}
func External(ctx *hcl.EvalContext, block *hclsyntax.Block) (cty.Value, error) {
	var (
		args struct {
			Program []string `hcl:"program"`
			Dir     string   `hcl:"working_dir,optional"`
			Remain  hcl.Body `hcl:",remain"`
		}
		errorf = blockError("data.external", block)
	)
	if diags := gohcl.DecodeBody(block.Body, ctx, &args); diags.HasErrors() {
		return cty.NilVal, errorf("decoding body: %v", diags)
	}
	attrs, diags := args.Remain.JustAttributes()
	if diags.HasErrors() {
		return cty.NilVal, errorf("getting attributes: %v", diags)
	}
	if len(attrs) > 0 {
		return cty.NilVal, errorf("unexpected attributes: %v", attrs)
	}
	if len(args.Program) == 0 {
		return cty.NilVal, errorf("program cannot be empty")
	}
	cmd := exec.Command(args.Program[0], args.Program[1:]...)
	if args.Dir != "" {
		cmd.Dir = args.Dir
	}
	out, err := cmd.Output()
	if err != nil {
		msg := err.Error()
		if err1 := (*exec.ExitError)(nil); errors.As(err, &err1) && len(err1.Stderr) > 0 {
			msg = string(err1.Stderr)
		}
		return cty.NilVal, errorf("running program %v: %v", cmd.Path, msg)
	}
	return cty.StringVal(string(out)), nil
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
				case cty.List(cty.String):
					var s []string
					if err := gocty.FromCtyValue(v, &s); err != nil {
						return cty.NilVal, errorf("convert strings: %w", err)
					}
					vars[k.AsString()] = s
				case cty.List(cty.Number):
					var s []float64
					if err := gocty.FromCtyValue(v, &s); err != nil {
						return cty.NilVal, errorf("convert floats: %w", err)
					}
					vars[k.AsString()] = s
				case cty.List(cty.Bool):
					var s []bool
					if err := gocty.FromCtyValue(v, &s); err != nil {
						return cty.NilVal, errorf("convert bools: %w", err)
					}
					vars[k.AsString()] = s
				default:
					return cty.NilVal, errorf(`attribute "vars" must be a map of strings, numbers or booleans, got: %s`, v.Type().FriendlyName())
				}
			}
		}
		delete(attrs, "vars")
	}
	if len(attrs) > 0 {
		return cty.NilVal, errorf("unexpected attributes: %v", attrs)
	}
	if d, err := os.Stat(args.Path); err != nil || !d.IsDir() {
		return cty.NilVal, errorf("path %s is not a directory", args.Path)
	}
	dirname := path.Join(args.Path, block.Labels[1])
	dir := migrate.OpenMemDir(dirname)
	// Clear existing directories in case the config was called
	// multiple times with different variables.
	if files, err := dir.Files(); err != nil || len(files) > 0 {
		dir.Reset()
	}
	t := template.New("template_dir").Option("missingkey=error")
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
	// Only top-level (template) files are treated as migrations.
	matches, err := fs.Glob(os.DirFS(args.Path), "*.sql")
	if err != nil {
		return cty.NilVal, errorf("globbing templates: %w", err)
	}
	for _, m := range matches {
		var b bytes.Buffer
		if err := t.ExecuteTemplate(&b, m, vars); err != nil {
			return cty.NilVal, errorf("executing template: %w", err)
		}
		if err := dir.WriteFile(m, b.Bytes()); err != nil {
			return cty.NilVal, errorf("writing file %q: %w", m, err)
		}
	}
	sum, err := dir.Checksum()
	if err != nil {
		return cty.NilVal, err
	}
	if err := migrate.WriteSumFile(dir, sum); err != nil {
		return cty.NilVal, err
	}
	// Sync template dir to local filesystem.
	dir.SyncWrites(func(name string, data []byte) error {
		if name == migrate.HashFileName {
			return nil
		}
		l, err := migrate.NewLocalDir(args.Path)
		if err != nil {
			return err
		}
		if err := l.WriteFile(name, data); err != nil {
			return err
		}
		sum, err := l.Checksum()
		if err != nil {
			return err
		}
		return migrate.WriteSumFile(l, sum)
	})
	u := fmt.Sprintf("mem://%s", dirname)
	// Allow using reading the computed dir as a state source.
	memLoader.states[u] = StateLoaderFunc(func(ctx context.Context, config *StateReaderConfig) (*StateReadCloser, error) {
		return stateReaderSQL(ctx, config, dir, nil, nil)
	})
	return cty.ObjectVal(map[string]cty.Value{
		"url": cty.StringVal(u),
	}), nil
}

// AtlasConfig exposes non-sensitive information returned by the "atlas" init-block.
// By invoking AtlasInitBlock() a new config is returned that is set by the init block
// defined and executed on schemahcl Eval functions.
type AtlasConfig struct {
	Client  *cloudapi.Client // Client attached to Atlas Cloud.
	Token   string           // User token.
	Project string           // Optional project.
}

// InitBlock returns the handler for the "atlas" init block.
//
//	atlas {
//	  cloud {
//	    token   = data.runtimevar.token  // User token.
//	    url     = var.cloud_url          // Optional URL.
//	    project = var.project            // Optional project. Defaults to DefaultProjectName.
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
			args.Cloud.Project = cloudapi.DefaultProjectName
		}
		c.Token = args.Cloud.Token
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

// SchemaHCL is a data source that reads an Atlas HCL schema file(s), evaluates it
// with the given variables and exposes its resulting schema as in-memory HCL file.
func SchemaHCL(ctx *hcl.EvalContext, block *hclsyntax.Block) (cty.Value, error) {
	var (
		args struct {
			Path   string               `hcl:"path"`
			Vars   map[string]cty.Value `hcl:"vars,optional"`
			Remain hcl.Body             `hcl:",remain"`
		}
		errorf = blockError("data.hcl_schema", block)
	)
	if diags := gohcl.DecodeBody(block.Body, ctx, &args); diags.HasErrors() {
		return cty.NilVal, errorf("decoding body: %v", diags)
	}
	attrs, diags := args.Remain.JustAttributes()
	if diags.HasErrors() {
		return cty.NilVal, errorf("getting attributes: %v", diags)
	}
	if len(attrs) > 0 {
		return cty.NilVal, errorf("unexpected attributes: %v", attrs)
	}
	if args.Path == "" {
		return cty.NilVal, errorf("path cannot be empty")
	}
	u, err := url.JoinPath("mem://hcl_schema", block.Labels[1])
	if err != nil {
		return cty.NilVal, errorf("build url: %v", err)
	}
	memLoader.states[u] = StateLoaderFunc(func(ctx context.Context, config *StateReaderConfig) (*StateReadCloser, error) {
		cfg := *config
		cfg.Vars = args.Vars
		return stateReaderHCL(ctx, &cfg, []string{args.Path})
	})
	return cty.ObjectVal(map[string]cty.Value{
		"url": cty.StringVal(u),
	}), nil
}

// SchemaExternal is a data source that reads a SQL schema state from external program.
func SchemaExternal(ctx *hcl.EvalContext, block *hclsyntax.Block) (cty.Value, error) {
	var (
		args struct {
			Program []string `hcl:"program"`
			Dir     string   `hcl:"working_dir,optional"`
			Remain  hcl.Body `hcl:",remain"`
		}
		errorf = blockError("data.external_schema", block)
	)
	if diags := gohcl.DecodeBody(block.Body, ctx, &args); diags.HasErrors() {
		return cty.NilVal, errorf("decoding body: %v", diags)
	}
	attrs, diags := args.Remain.JustAttributes()
	if diags.HasErrors() {
		return cty.NilVal, errorf("getting attributes: %v", diags)
	}
	if len(attrs) > 0 {
		return cty.NilVal, errorf("unexpected attributes: %v", attrs)
	}
	if len(args.Program) == 0 {
		return cty.NilVal, errorf("program cannot be empty")
	}
	cmd := exec.Command(args.Program[0], args.Program[1:]...)
	if args.Dir != "" {
		cmd.Dir = args.Dir
	}
	out, err := cmd.Output()
	if err != nil {
		msg := err.Error()
		if err1 := (*exec.ExitError)(nil); errors.As(err, &err1) && len(err1.Stderr) > 0 {
			msg = string(err1.Stderr)
		}
		return cty.NilVal, errorf("running program %v: %v", cmd.Path, msg)
	}
	// Directory files must have an .sql extension to be read by the executor.
	// The "schema" word is added to indicate that unlike data-source errors, load error
	// comes from the output of the data-source (SQL representation of the state/schema).
	dir, err := filesAsDir(migrate.NewLocalFile(fmt.Sprintf("%s/schema.sql", block.Labels[1]), out))
	if err != nil {
		return cty.NilVal, errorf("converting output to migration: %v", err)
	}
	u, err := url.JoinPath("mem://external_schema", block.Labels[1])
	if err != nil {
		return cty.NilVal, errorf("build url: %v", err)
	}
	memLoader.states[u] = StateLoaderFunc(func(ctx context.Context, config *StateReaderConfig) (*StateReadCloser, error) {
		return stateSchemaSQL(ctx, config, dir)
	})
	return cty.ObjectVal(map[string]cty.Value{
		"url": cty.StringVal(u),
	}), nil
}

// memdir converts the given (cloud) migrate.Dir to a memory migrate.Dir.
func memdir(client *cloudapi.Client, name, tag string) (string, error) {
	dir, err := client.Dir(context.Background(), cloudapi.DirInput{
		Name: name,
		Tag:  tag,
	})
	if err != nil {
		return "", err
	}
	md := migrate.OpenMemDir(name)
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
	return "mem://" + name, nil
}

func blockError(name string, b *hclsyntax.Block) func(string, ...any) error {
	return func(format string, args ...any) error {
		return fmt.Errorf("%s.%s: %w", name, b.Labels[1], fmt.Errorf(format, args...))
	}
}

type (
	// StateLoader allows loading StateReader's from external sources.
	StateLoader interface {
		LoadState(context.Context, *StateReaderConfig) (*StateReadCloser, error)
	}
	// The StateLoaderFunc type is an adapter to allow the use of ordinary
	// function as StateLoader.
	StateLoaderFunc func(context.Context, *StateReaderConfig) (*StateReadCloser, error)

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

// LoadState calls f(ctx, opts).
func (f StateLoaderFunc) LoadState(ctx context.Context, opts *StateReaderConfig) (*StateReadCloser, error) {
	return f(ctx, opts)
}

var (
	// States is a global registry for external state loaders.
	States = registry{
		"ent": EntLoader{},
		"mem": memLoader,
	}
	memLoader       = MemLoader{states: make(map[string]StateLoader)}
	errNotSchemaURL = errors.New("missing schema in --dev-url. See: https://atlasgo.io/url")
)

type registry map[string]StateLoader

// HasLoader returns true if the given scheme is registered.
func (r registry) HasLoader(scheme string) bool {
	_, ok := r[scheme]
	return ok
}

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

// MemLoader is a StateLoader for loading data-sources
// that were loaded into program memory.
type MemLoader struct {
	states map[string]StateLoader
}

// LoadState loads the state loaded from data-sources into memory.
func (l MemLoader) LoadState(ctx context.Context, config *StateReaderConfig) (*StateReadCloser, error) {
	if len(config.URLs) != 1 {
		return nil, errors.New(`"mem://" requires exactly one data-source URL`)
	}
	u := config.URLs[0].String()
	if l.states[u] == nil {
		return nil, fmt.Errorf("data-source state %q not found in memory", u)
	}
	return l.states[u].LoadState(ctx, config)
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
