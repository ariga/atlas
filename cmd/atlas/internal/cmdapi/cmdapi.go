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
	"net/url"
	"sort"
	"strings"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/cmdext"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/zclconf/go-cty/cty"
	"golang.org/x/exp/maps"
	"golang.org/x/mod/semver"
)

var (
	// Root represents the root command when called without any subcommands.
	Root = &cobra.Command{
		Use:          "atlas",
		Short:        "Manage your database schema as code",
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

	// flavor holds Atlas flavor. Custom flavors (like the community build) should set this by build flag
	// "-X 'ariga.io/atlas/cmd/atlas/internal/cmdapi.flavor=community'"
	flavor string

	// version holds Atlas version. When built with cloud packages should be set by build flag, e.g.
	// "-X 'ariga.io/atlas/cmd/atlas/internal/cmdapi.version=v0.1.2'"
	version string

	// versionCmd represents the subcommand 'atlas version'.
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Prints this Atlas CLI version information.",
		Run: func(cmd *cobra.Command, _ []string) {
			var (
				f    = versionFmt
				args []any
			)
			if flavor != "" {
				f += "%s "
				args = append(args, flavor)
			}
			f += "version %s\n%s\n%s"
			v, u := parseV(version)
			args = append(args, v, u, versionInfo)
			cmd.Printf(f, args...)
		},
	}

	// license holds Atlas license. When built with cloud packages should be set by build flag
	// "-X 'ariga.io/atlas/cmd/atlas/internal/cmdapi.license=${license}'"
	license = `LICENSE
Atlas is licensed under Apache 2.0 as found in https://github.com/ariga/atlas/blob/master/LICENSE.`

	// licenseCmd represents the subcommand 'atlas license'.
	licenseCmd = &cobra.Command{
		Use:   "license",
		Short: "Display license information",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Println(license)
		},
	}
)

type (
	// ErrorFormatter implemented by the errors below to
	// allow them format command output on error.
	ErrorFormatter interface {
		FormatError(*cobra.Command)
	}
	// FormattedError is an error that format the command output when returned.
	FormattedError struct {
		Err    error
		Prefix string // Prefix to use on error.
		Silent bool   // Silent errors are not printed.
	}
	// AbortError returns a command error that is formatted as "Abort: ..." when
	// the execution is aborted by the user.
	AbortError struct {
		Err error
	}
	// Aborter allows errors to signal if the error is an abort error.
	Aborter interface {
		error
		IsAbort()
	}
)

func (e *FormattedError) Error() string { return e.Err.Error() }

func (e *FormattedError) FormatError(cmd *cobra.Command) {
	cmd.SilenceErrors = e.Silent
	if e.Prefix != "" {
		cmd.SetErrPrefix(e.Prefix)
	}
}

// AbortErrorf is like fmt.Errorf for creating AbortError.
func AbortErrorf(format string, a ...any) error {
	return &AbortError{Err: fmt.Errorf(format, a...)}
}

func (e *AbortError) Error() string { return e.Err.Error() }

func (e *AbortError) FormatError(cmd *cobra.Command) {
	cmd.SetErrPrefix("Abort:")
}

// RunE wraps the command cobra.Command.RunE function with additional postrun logic.
func RunE(f func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) (err error) {
		if err = f(cmd, args); err != nil {
			if err1 := (Aborter)(nil); errors.As(err, &err1) {
				err = &AbortError{Err: err}
			}
			if ef, ok := err.(ErrorFormatter); ok {
				ef.FormatError(cmd)
			}
		}
		return err
	}
}

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
	var (
		b  strings.Builder
		ks = maps.Keys(v)
	)
	sort.Strings(ks)
	for _, k := range ks {
		if b.Len() > 0 {
			b.WriteString(", ")
		}
		b.WriteString(k)
		b.WriteString(":")
		switch v1 := v[k]; v1.Type() {
		case cty.String:
			b.WriteString(v1.AsString())
		case cty.List(cty.String):
			b.WriteString("[")
			for i, v2 := range v1.AsValueSlice() {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(v2.AsString())
			}
			b.WriteString("]")
		default:
			b.WriteString(v1.GoString())
		}
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
	flagContext        = "context"
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
	flagPlan           = "plan"
	flagRevisionSchema = "revisions-schema"
	flagSchema         = "schema"
	flagSchemaShort    = "s"
	flagTo             = "to"
	flagTxMode         = "tx-mode"
	flagExecOrder      = "exec-order"
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
				v.Replace(vs.Copy())
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

// stateReaderConfig is given to stateReader.
type stateReaderConfig struct {
	urls        []string          // urls to create a migrate.StateReader from
	client, dev *sqlclient.Client // database connections, while dev is considered a dev database, client is not
	schemas     []string          // schemas to work on
	exclude     []string          // exclude flag values
	withPos     bool              // indicate if schema.Pos should be loaded.
	vars        Vars
}

// Exported is a temporary method to convert the stateReaderConfig to cmdext.StateReaderConfig.
func (c *stateReaderConfig) Exported() (*cmdext.StateReaderConfig, error) {
	var (
		err    error
		parsed = make([]*url.URL, len(c.urls))
	)
	for i, u := range c.urls {
		if parsed[i], err = sqlclient.ParseURL(u); err != nil {
			return nil, err
		}
	}
	return &cmdext.StateReaderConfig{
		URLs:    parsed,
		Client:  c.client,
		Dev:     c.dev,
		Schemas: c.schemas,
		Exclude: c.exclude,
		WithPos: c.withPos,
		Vars:    c.vars,
	}, nil
}

// readerUseDev reports if any of the URL uses the dev-database.
func readerUseDev(env *Env, urls ...string) (bool, error) {
	s, err := selectScheme(urls)
	if err != nil {
		return false, err
	}
	switch {
	case s == envAttrScheme && env != nil && len(urls) == 1:
		u, err := env.VarFromURL(urls[0])
		if err != nil {
			return false, err
		}
		// No circular reference possible with env:// variable.
		return readerUseDev(env, u)
	case s == cmdext.SchemaTypeFile, s == cmdext.SchemaTypeAtlas:
		return true, nil
	default:
		return cmdext.States.HasLoader(s), nil
	}
}

// stateReader returns a migrate.StateReader that reads the state from the given urls.
func stateReader(ctx context.Context, env *Env, config *stateReaderConfig) (*cmdext.StateReadCloser, error) {
	excfg, err := config.Exported()
	if err != nil {
		return nil, err
	}
	scheme, err := selectScheme(config.urls)
	if err != nil {
		return nil, err
	}
	switch scheme {
	// "file" scheme is valid for both migration directory and HCL paths.
	case cmdext.SchemaTypeFile:
		switch ext, err := cmdext.FilesExt(excfg.URLs); {
		case err != nil:
			return nil, err
		case ext == cmdext.FileTypeHCL:
			return cmdext.StateReaderHCL(ctx, excfg)
		case ext == cmdext.FileTypeSQL:
			return cmdext.StateReaderSQL(ctx, excfg)
		default:
			panic("unreachable") // checked by filesExt.
		}
	// "atlas" scheme represents an Atlas Cloud schema.
	case cmdext.SchemaTypeAtlas:
		return cmdext.StateReaderAtlas(ctx, excfg)
	// "env" scheme represents an attribute defined
	// on the selected environment.
	case envAttrScheme:
		switch {
		case GlobalFlags.SelectedEnv == "":
			return nil, errors.New("cannot use env:// variables without selecting an environment")
		case len(config.urls) != 1:
			return nil, errors.New("cannot use multiple env:// variables in a single flag")
		default:
			u, err := env.VarFromURL(config.urls[0])
			if err != nil {
				return nil, err
			}
			cfg := *config
			cfg.urls = []string{u}
			return stateReader(ctx, env, &cfg)
		}
	default:
		// In case there is an external state-loader registered with this scheme.
		if l, ok := cmdext.States.Loader(scheme); ok {
			rc, err := l.LoadState(ctx, excfg)
			if err != nil {
				return nil, err
			}
			return rc, nil
		}
		// All other schemes are database (or docker) connections.
		c, err := env.openClient(ctx, config.urls[0]) // call to selectScheme already checks for len > 0
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
		return &cmdext.StateReadCloser{
			StateReader: sr,
			Closer:      c,
			Schema:      c.URL.Schema,
		}, nil
	}
}

const localStateFile = "local-community.json"

// LocalState keeps track of local state to enhance developer experience.
type LocalState struct {
	UpgradeSuggested time.Time `json:"v1.us"`
}
