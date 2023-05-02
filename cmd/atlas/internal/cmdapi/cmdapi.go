// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

// Package cmdapi holds the atlas commands used to build an atlas distribution.
package cmdapi

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/cmdext"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/zclconf/go-cty/cty"
	"golang.org/x/mod/semver"
)

var (
	// Root represents the root command when called without any subcommands.
	Root = &cobra.Command{
		Use:          "atlas",
		Short:        "A database toolkit.",
		SilenceUsage: true,
	}

	// GlobalFlags contains flags common to many Atlas sub-commands.
	GlobalFlags struct {
		// Config defines the path to the Atlas project/config file.
		ConfigURL string
		// SelectedEnv contains the environment selected from the active project via the --env flag.
		SelectedEnv string
		// Vars contains the input variables passed from the CLI to Atlas DDL or project files.
		Vars Vars
	}

	// version holds Atlas version. When built with cloud packages should be set by build flag
	// "-X 'ariga.io/atlas/cmd/atlas/internal/cmdapi.version=${version}'"
	version string

	// schemaCmd represents the subcommand 'atlas version'.
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Prints this Atlas CLI version information.",
		Run: func(cmd *cobra.Command, args []string) {
			v, u := parseV(version)
			cmd.Printf("atlas version %s\n%s\n", v, u)
		},
	}

	// license holds Atlas license. When built with cloud packages should be set by build flag
	// "-X 'ariga.io/atlas/cmd/atlas/internal/cmdapi.license=${license}'"
	license = `LICENSE
Atlas is licensed under Apache 2.0 as found in https://github.com/ariga/atlas/blob/master/LICENSE.`
	licenseCmd = &cobra.Command{
		Use:   "license",
		Short: "Display license information",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Println(license)
		},
	}
)

func init() {
	Root.AddCommand(versionCmd)
	Root.AddCommand(licenseCmd)
	// Register a global function to clean up the global
	// flags regardless if the command passed or failed.
	cobra.OnFinalize(func() {
		GlobalFlags.ConfigURL = ""
		GlobalFlags.Vars = nil
		GlobalFlags.SelectedEnv = ""
	})
}

// inputValuesFromEnv populates GlobalFlags.Vars from the active environment. If we are working
// inside a project, the "var" flag is not propagated to the schema definition. Instead, it
// is used to evaluate the project file which can pass input values via the "values" block
// to the schema.
func inputValuesFromEnv(cmd *cobra.Command, env *Env) error {
	if fl := cmd.Flag(flagVar); fl == nil {
		return nil
	}
	values, err := env.asMap()
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}
	pairs := make([]string, 0, len(values))
	for k, v := range values {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	vars := strings.Join(pairs, ",")
	if err := cmd.Flags().Set(flagVar, vars); err != nil {
		return fmt.Errorf("set flag %q: %w", flagVar, err)
	}
	return nil
}

// parseV returns a user facing version and release notes url
func parseV(version string) (string, string) {
	u := "https://github.com/ariga/atlas/releases/latest"
	if ok := semver.IsValid(version); !ok {
		return "- development", u
	}
	s := strings.Split(version, "-")
	if len(s) != 0 && s[len(s)-1] != "canary" {
		u = fmt.Sprintf("https://github.com/ariga/atlas/releases/tag/%s", version)
	}
	return version, u
}

// Version returns the current Atlas binary version.
func Version() string {
	return version
}

// Vars implements pflag.Value.
type Vars map[string]cty.Value

// String implements pflag.Value.String.
func (v Vars) String() string {
	var b strings.Builder
	for k := range v {
		if b.Len() > 0 {
			b.WriteString(", ")
		}
		b.WriteString(k)
		b.WriteString(":")
		b.WriteString(v[k].GoString())
	}
	return "[" + b.String() + "]"
}

// Copy returns a copy of the current variables.
func (v Vars) Copy() Vars {
	vc := make(Vars)
	for k := range v {
		vc[k] = v[k]
	}
	return vc
}

// Replace overrides the variables.
func (v *Vars) Replace(vc Vars) {
	*v = vc
}

// Set implements pflag.Value.Set.
func (v *Vars) Set(s string) error {
	if *v == nil {
		*v = make(Vars)
	}
	kvs, err := csv.NewReader(strings.NewReader(s)).Read()
	if err != nil {
		return err
	}
	for i := range kvs {
		kv := strings.SplitN(kvs[i], "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("variables must be format as key=value, got: %q", kvs[i])
		}
		v1 := cty.StringVal(kv[1])
		switch v0, ok := (*v)[kv[0]]; {
		case ok && v0.Type().IsListType():
			(*v)[kv[0]] = cty.ListVal(append(v0.AsValueSlice(), v1))
		case ok:
			(*v)[kv[0]] = cty.ListVal([]cty.Value{v0, v1})
		default:
			(*v)[kv[0]] = v1
		}
	}
	return nil
}

// Type implements pflag.Value.Type.
func (v *Vars) Type() string {
	return "<name>=<value>"
}

const (
	flagAllowDirty     = "allow-dirty"
	flagEdit           = "edit"
	flagAutoApprove    = "auto-approve"
	flagBaseline       = "baseline"
	flagConfig         = "config"
	flagDevURL         = "dev-url"
	flagDirURL         = "dir"
	flagDirFormat      = "dir-format"
	flagDryRun         = "dry-run"
	flagEnv            = "env"
	flagExclude        = "exclude"
	flagFile           = "file"
	flagFrom           = "from"
	flagFromShort      = "f"
	flagFormat         = "format"
	flagGitBase        = "git-base"
	flagGitDir         = "git-dir"
	flagLatest         = "latest"
	flagLockTimeout    = "lock-timeout"
	flagLog            = "log"
	flagRevisionSchema = "revisions-schema"
	flagSchema         = "schema"
	flagSchemaShort    = "s"
	flagTo             = "to"
	flagTxMode         = "tx-mode"
	flagURL            = "url"
	flagURLShort       = "u"
	flagVar            = "var"
	flagQualifier      = "qualifier"
)

func addGlobalFlags(set *pflag.FlagSet) {
	set.StringVar(&GlobalFlags.SelectedEnv, flagEnv, "", "set which env from the config file to use")
	set.Var(&GlobalFlags.Vars, flagVar, "input variables")
	set.StringVarP(&GlobalFlags.ConfigURL, flagConfig, "c", defaultConfigPath, "select config (project) file using URL format")
}

func addFlagAutoApprove(set *pflag.FlagSet, target *bool) {
	set.BoolVar(target, flagAutoApprove, false, "apply changes without prompting for approval")
}

func addFlagDirFormat(set *pflag.FlagSet, target *string) {
	set.StringVar(target, flagDirFormat, "atlas", "select migration file format")
}

func addFlagLockTimeout(set *pflag.FlagSet, target *time.Duration) {
	set.DurationVar(target, flagLockTimeout, 10*time.Second, "set how long to wait for the database lock")
}

// addFlagURL adds a URL flag. If given, args[0] override the name, args[1] the shorthand, args[2] the default value.
func addFlagDirURL(set *pflag.FlagSet, target *string, args ...string) {
	name, short, val := flagDirURL, "", "file://migrations"
	switch len(args) {
	case 3:
		val = args[2]
		fallthrough
	case 2:
		short = args[1]
		fallthrough
	case 1:
		name = args[0]
	}
	set.StringVarP(target, name, short, val, "select migration directory using URL format")
}

func addFlagDevURL(set *pflag.FlagSet, target *string) {
	set.StringVar(
		target,
		flagDevURL,
		"",
		"[driver://username:password@address/dbname?param=value] select a dev database using the URL format",
	)
}

func addFlagDryRun(set *pflag.FlagSet, target *bool) {
	set.BoolVar(target, flagDryRun, false, "print SQL without executing it")
}

func addFlagExclude(set *pflag.FlagSet, target *[]string) {
	set.StringSliceVar(
		target,
		flagExclude,
		nil,
		"list of glob patterns used to filter resources from applying",
	)
}

func addFlagLog(set *pflag.FlagSet, target *string) {
	set.StringVar(target, flagLog, "", "Go template to use to format the output")
	// Use MarkHidden instead of MarkDeprecated to avoid
	// spam users' system logs with deprecation warnings.
	cobra.CheckErr(set.MarkHidden(flagLog))
}

func addFlagFormat(set *pflag.FlagSet, target *string) {
	set.StringVar(target, flagFormat, "", "Go template to use to format the output")
}

func addFlagRevisionSchema(set *pflag.FlagSet, target *string) {
	set.StringVar(target, flagRevisionSchema, "", "name of the schema the revisions table resides in")
}

func addFlagSchemas(set *pflag.FlagSet, target *[]string) {
	set.StringSliceVarP(
		target,
		flagSchema, flagSchemaShort,
		nil,
		"set schema names",
	)
}

// addFlagURL adds a URL flag. If given, args[0] override the name, args[1] the shorthand.
func addFlagURL(set *pflag.FlagSet, target *string, args ...string) {
	name, short := flagURL, flagURLShort
	switch len(args) {
	case 2:
		short = args[1]
		fallthrough
	case 1:
		name = args[0]
	}
	set.StringVarP(
		target,
		name, short,
		"",
		"[driver://username:password@address/dbname?param=value] select a resource using the URL format",
	)
}

func addFlagURLs(set *pflag.FlagSet, target *[]string, args ...string) {
	name, short := flagURL, flagURLShort
	switch len(args) {
	case 2:
		short = args[1]
		fallthrough
	case 1:
		name = args[0]
	}
	set.StringSliceVarP(
		target,
		name, short,
		nil,
		"[driver://username:password@address/dbname?param=value] select a resource using the URL format",
	)
}

func addFlagToURLs(set *pflag.FlagSet, target *[]string) {
	set.StringSliceVarP(target, flagTo, "", nil, "[driver://username:password@address/dbname?param=value] select a desired state using the URL format")
}

// maySetFlag sets the flag with the provided name to envVal if such a flag exists
// on the cmd, it was not set by the user via the command line and if envVal is not
// an empty string.
func maySetFlag(cmd *cobra.Command, name, envVal string) error {
	if f := cmd.Flag(name); f == nil || f.Changed || envVal == "" {
		return nil
	}
	return cmd.Flags().Set(name, envVal)
}

// resetFromEnv traverses the command flags, records what flags
// were not set by the user and returns a callback to clear them
// after it was set by the current environment.
func resetFromEnv(cmd *cobra.Command) func() {
	mayReset := make(map[string]func() error)
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			return
		}
		vs := f.Value.String()
		r := func() error { return f.Value.Set(vs) }
		if v, ok := f.Value.(*Vars); ok {
			vs := v.Copy()
			r = func() error {
				v.Replace(vs)
				return nil
			}
		} else if v, ok := f.Value.(pflag.SliceValue); ok {
			vs := v.GetSlice()
			r = func() error {
				return v.Replace(vs)
			}
		}
		mayReset[f.Name] = r
	})
	return func() {
		for name, reset := range mayReset {
			if f := cmd.Flag(name); f != nil && f.Changed {
				f.Changed = false
				// Unexpected error, because this flag was set before.
				cobra.CheckErr(reset())
			}
		}
	}
}

type (
	// stateReadCloser is a migrate.StateReader with an optional io.Closer.
	stateReadCloser struct {
		migrate.StateReader
		io.Closer        // optional close function
		schema    string // in case we work on a single schema
		hcl       bool   // true if state was read from HCL files since in that case we always compare realms
	}
	// stateReaderConfig is given to stateReader.
	stateReaderConfig struct {
		urls        []string          // urls to create a migrate.StateReader from
		client, dev *sqlclient.Client // database connections, while dev is considered a dev database, client is not
		schemas     []string          // schemas to work on
		exclude     []string          // exclude flag values
		vars        Vars
	}
)

// stateReader returns a migrate.StateReader that reads the state from the given urls.
func stateReader(ctx context.Context, config *stateReaderConfig) (*stateReadCloser, error) {
	scheme, err := selectScheme(config.urls)
	if err != nil {
		return nil, err
	}
	parsed := make([]*url.URL, len(config.urls))
	for i, u := range config.urls {
		parsed[i], err = url.Parse(u)
		if err != nil {
			return nil, err
		}
	}
	switch scheme {
	// "file" scheme is valid for both migration directory and HCL paths.
	case "file":
		switch ext, err := filesExt(parsed); {
		case err != nil:
			return nil, err
		case ext == extHCL:
			return hclStateReader(ctx, config, parsed)
		case ext == extSQL:
			return sqlStateReader(ctx, config, parsed)
		default:
			panic("unreachable") // checked by filesExt.
		}
	default:
		// In case there is an external state-loader registered with this scheme.
		if l, ok := cmdext.States.Loader(scheme); ok {
			sr, err := l.LoadState(ctx, &cmdext.LoadStateOptions{URLs: parsed, Dev: config.dev})
			if err != nil {
				return nil, err
			}
			return &stateReadCloser{StateReader: sr}, nil
		}
		// All other schemes are database (or docker) connections.
		c, err := sqlclient.Open(ctx, config.urls[0]) // call to selectScheme already checks for len > 0
		if err != nil {
			return nil, err
		}
		var sr migrate.StateReader
		switch c.URL.Schema {
		case "":
			sr = migrate.RealmConn(c.Driver, &schema.InspectRealmOption{
				Schemas: config.schemas,
				Exclude: config.exclude,
			})
		default:
			sr = migrate.SchemaConn(c.Driver, c.URL.Schema, &schema.InspectOptions{Exclude: config.exclude})
		}
		return &stateReadCloser{
			StateReader: sr,
			Closer:      c,
			schema:      c.URL.Schema,
		}, nil
	}
}

// hclStateReadr returns a StateReader that reads the state from the given HCL paths urls.
func hclStateReader(ctx context.Context, config *stateReaderConfig, urls []*url.URL) (*stateReadCloser, error) {
	var client *sqlclient.Client
	switch {
	case config.dev != nil:
		client = config.dev
	case config.client != nil:
		client = config.client
	default:
		return nil, errors.New("--dev-url cannot be empty")
	}
	paths := make([]string, len(urls))
	for i, u := range urls {
		paths[i] = filepath.Join(u.Host, u.Path)
	}
	parser, err := parseHCLPaths(paths...)
	if err != nil {
		return nil, err
	}
	realm := &schema.Realm{}
	if err := client.Eval(parser, realm, config.vars); err != nil {
		return nil, err
	}
	if len(config.schemas) > 0 {
		// Validate all schemas in file were selected by user.
		sm := make(map[string]bool, len(config.schemas))
		for _, s := range config.schemas {
			sm[s] = true
		}
		for _, s := range realm.Schemas {
			if !sm[s.Name] {
				return nil, fmt.Errorf("schema %q from paths %q is not requested (all schemas in HCL must be requested)", s.Name, paths)
			}
		}
	}
	// In case the dev connection is bound to a specific schema, we require the
	// desired schema to contain only one schema. Thus, executing diff will be
	// done on the content of these two schema and not the whole realm.
	if client.URL.Schema != "" && len(realm.Schemas) > 1 {
		return nil, fmt.Errorf(
			"cannot use HCL with more than 1 schema when dev-url is limited to schema %q",
			config.dev.URL.Schema,
		)
	}
	if norm, ok := client.Driver.(schema.Normalizer); ok && config.dev != nil { // only normalize on a dev database
		realm, err = norm.NormalizeRealm(ctx, realm)
		if err != nil {
			return nil, err
		}
	}
	t := &stateReadCloser{StateReader: migrate.Realm(realm), hcl: true}
	return t, nil
}

func sqlStateReader(ctx context.Context, config *stateReaderConfig, urls []*url.URL) (*stateReadCloser, error) {
	if len(urls) != 1 {
		return nil, fmt.Errorf("the provided SQL state must be either a single schema file or a migration directory, but %d paths were found", len(urls))
	}
	// Replaying a migration directory requires a dev connection.
	if config.dev == nil {
		return nil, errors.New("--dev-url cannot be empty")
	}
	var (
		dir  migrate.Dir
		opts []migrate.ReplayOption
		path = filepath.Join(urls[0].Host, urls[0].Path)
	)
	switch fi, err := os.Stat(path); {
	case err != nil:
		return nil, err
	// A single schema file.
	case !fi.IsDir():
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		dir = &validMemDir{}
		if err := dir.WriteFile(fi.Name(), b); err != nil {
			return nil, err
		}
	// A migration directory.
	default:
		if dir, err = dirURL(urls[0], false); err != nil {
			return nil, err
		}
		if v := urls[0].Query().Get("version"); v != "" {
			opts = append(opts, migrate.ReplayToVersion(v))
		}
	}
	ex, err := migrate.NewExecutor(config.dev.Driver, dir, migrate.NopRevisionReadWriter{})
	if err != nil {
		return nil, err
	}
	sr, err := ex.Replay(ctx, func() migrate.StateReader {
		if config.dev.URL.Schema != "" {
			return migrate.SchemaConn(config.dev, "", nil)
		}
		return migrate.RealmConn(config.dev, &schema.InspectRealmOption{
			Schemas: config.schemas,
			Exclude: config.exclude,
		})
	}(), opts...)
	if err != nil && !errors.Is(err, migrate.ErrNoPendingFiles) {
		return nil, err
	}
	return &stateReadCloser{
		StateReader: migrate.Realm(sr),
		schema:      config.dev.URL.Schema,
	}, nil
}

// Close redirects calls to Close to the enclosed io.Closer.
func (sr *stateReadCloser) Close() {
	if sr.Closer != nil {
		sr.Closer.Close()
	}
}

// validMemDir will not throw an error when put into migrate.Validate.
type validMemDir struct{ migrate.MemDir }

func (d *validMemDir) Validate() error { return nil }

const (
	extHCL = ".hcl"
	extSQL = ".sql"
)

func filesExt(urls []*url.URL) (string, error) {
	var path, ext string
	set := func(curr string) error {
		switch e := filepath.Ext(curr); {
		case e != extHCL && e != extSQL:
			return fmt.Errorf("unknown schema file: %q", curr)
		case ext != "" && ext != e:
			return fmt.Errorf("ambiguous schema: both SQL and HCL files found: %q, %q", path, curr)
		default:
			path, ext = curr, e
			return nil
		}
	}
	for _, u := range urls {
		path := filepath.Join(u.Host, u.Path)
		switch fi, err := os.Stat(path); {
		case err != nil:
			return "", err
		case fi.IsDir():
			files, err := os.ReadDir(path)
			if err != nil {
				return "", err
			}
			for _, f := range files {
				switch filepath.Ext(f.Name()) {
				// Ignore unknown extensions in case we read directories.
				case extHCL, extSQL:
					if err := set(f.Name()); err != nil {
						return "", err
					}
				}
			}
		default:
			if err := set(fi.Name()); err != nil {
				return "", err
			}
		}
	}
	switch {
	case ext != "":
	case len(urls) == 1 && (urls[0].Host != "" || urls[0].Path != ""):
		return "", fmt.Errorf(
			"%q contains neither SQL nor HCL files",
			filepath.Base(filepath.Join(urls[0].Host, urls[0].Path)),
		)
	default:
		return "", errors.New("schema contains neither SQL nor HCL files")
	}
	return ext, nil
}
