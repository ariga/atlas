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
			v, u := parse(version)
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

// inputValsFromEnv populates GlobalFlags.Vars from the active environment. If we are working
// inside a project, the "var" flag is not propagated to the schema definition. Instead, it
// is used to evaluate the project file which can pass input values via the "values" block
// to the schema.
func inputValsFromEnv(cmd *cobra.Command, env *Env) error {
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
		return err
	}
	return nil
}

// parse returns a user facing version and release notes url
func parse(version string) (string, string) {
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
	flagAutoApprove    = "auto-approve"
	flagBaseline       = "baseline"
	flagConfig         = "config"
	flagDevURL         = "dev-url"
	flagDirURL         = "dir"
	flagDirFormat      = "dir-format"
	flagDryRun         = "dry-run"
	flagDSN            = "dsn" // deprecated in favor of flagURL
	flagEnv            = "env"
	flagExclude        = "exclude"
	flagFile           = "file"
	flagFrom           = "from"
	flagFromShort      = "f"
	flagGitBase        = "git-base"
	flagGitDir         = "git-dir"
	flagLatest         = "latest"
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
	set.StringVarP(&GlobalFlags.ConfigURL, flagConfig, "c", projectFileName, "select config (project) file using URL format")
}

func addFlagAutoApprove(set *pflag.FlagSet, target *bool) {
	set.BoolVar(target, flagAutoApprove, false, "apply changes without prompting for approval")
}

func addFlagDirFormat(set *pflag.FlagSet, target *string) {
	set.StringVar(target, flagDirFormat, "atlas", "select migration file format")
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

func addFlagDSN(set *pflag.FlagSet, target *string) {
	set.StringVarP(target, flagDSN, "d", "", "")
	cobra.CheckErr(set.MarkHidden(flagDSN))
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
	set.StringVar(target, flagLog, "", "go template to use to format logs")
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

func dsn2url(cmd *cobra.Command) error {
	dsnF, urlF := cmd.Flag(flagDSN), cmd.Flag(flagURL)
	switch {
	case dsnF == nil:
	case dsnF.Changed && urlF.Changed:
		return errors.New(`both flags "url" and "dsn" were set`)
	case dsnF.Changed && !urlF.Changed:
		return cmd.Flags().Set(flagURL, dsnF.Value.String())
	}
	return nil
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
	mayReset := make(map[string]string)
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed {
			mayReset[f.Name] = f.Value.String()
		}
	})
	return func() {
		for n, v := range mayReset {
			if f := cmd.Flag(n); f != nil && f.Changed {
				f.Changed = false
				// Unexpected error, because this flag was set before.
				cobra.CheckErr(f.Value.Set(v))
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
		urls      []string          // urls to create a migrate.StateReader from
		norm, dev *sqlclient.Client // database connections, while dev is considered a dev database, norm is not
		schemas   []string          // schemas to work on
		exclude   []string          // exclude flag values
		vars      Vars
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
		if len(config.urls) > 1 {
			// Consider urls being HCL paths.
			return hclStateReader(ctx, config, parsed)
		}
		path := filepath.Join(parsed[0].Host, parsed[0].Path)
		fi, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !fi.IsDir() {
			// If there is only one url given, and it is a file, consider it an HCL path.
			return hclStateReader(ctx, config, parsed)
		}
		files, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		// Check all files, if we find both HCL and SQL files, abort. Otherwise, proceed accordingly.
		var hcl, sql bool
		for _, f := range files {
			ext := filepath.Ext(f.Name())
			switch {
			case hcl && ext == ".sql", sql && ext == ".hcl":
				return nil, fmt.Errorf("ambiguos files: %q contains both SQL and HCL files", path)
			case ext == ".hcl":
				hcl = true
			case ext == ".sql":
				sql = true
			default:
				// unknown extension, we don't care
			}
		}
		switch {
		case hcl:
			return hclStateReader(ctx, config, parsed)
		case sql:
			// Replaying a migration directory requires a dev connection.
			if config.dev == nil {
				return nil, errors.New("--dev-url cannot be empty")
			}
			dir, err := dirURL(parsed[0], false)
			if err != nil {
				return nil, err
			}
			ex, err := migrate.NewExecutor(config.dev.Driver, dir, migrate.NopRevisionReadWriter{})
			if err != nil {
				return nil, err
			}
			var opts []migrate.ReplayOption
			if v := parsed[0].Query().Get("version"); v != "" {
				opts = append(opts, migrate.ReplayToVersion(v))
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
				return nil, fmt.Errorf("replaying the migration directory: %w", err)
			}
			return &stateReadCloser{
				StateReader: migrate.Realm(sr),
				schema:      config.dev.URL.Schema,
			}, nil
		default:
			return nil, fmt.Errorf("%q contains neither SQL nor HCL files", path)
		}
	default:
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

// hclStateReadr returns a migrate.StateReader that reads the state from the given HCL paths urls.
func hclStateReader(ctx context.Context, config *stateReaderConfig, urls []*url.URL) (*stateReadCloser, error) {
	var client *sqlclient.Client
	switch {
	case config.dev != nil:
		client = config.dev
	case config.norm != nil:
		client = config.norm
	default:
		return nil, errors.New("no database connection available")
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
	if norm, ok := client.Driver.(schema.Normalizer); ok && len(realm.Schemas) > 0 {
		realm, err = norm.NormalizeRealm(ctx, realm)
		if err != nil {
			return nil, err
		}
	}
	t := &stateReadCloser{StateReader: migrate.Realm(realm), hcl: true}
	return t, nil
}

// Close redirects calls to Close to the enclosed io.Closer.
func (sr *stateReadCloser) Close() {
	if sr.Closer != nil {
		sr.Closer.Close()
	}
}
