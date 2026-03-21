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
	"os"
	"sort"
	"strings"
	"testing"
	"text/template"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/cmdext"
	"ariga.io/atlas/cmd/atlas/internal/cmdlog"
	cmdmigrate "ariga.io/atlas/cmd/atlas/internal/migrate"
	"ariga.io/atlas/cmd/atlas/internal/cmdstate"
	"ariga.io/atlas/cmd/atlas/internal/migratelint"
	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
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

func (e *AbortError) Unwrap() error {
	return e.Err
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
	flagInclude        = "include"
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

func addFlagInclude(set *pflag.FlagSet, target *[]string) {
	set.StringSliceVar(
		target,
		flagInclude,
		nil,
		"list of glob patterns used to select which resources to keep in inspection",
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
	include     []string          // include flag values
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
		Include: c.include,
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
				Include: config.include,
			})
		default:
			sr = migrate.SchemaConn(c.Driver, c.URL.Schema, &schema.InspectOptions{
				Exclude: config.exclude,
				Include: config.include,
			})
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

func init() {
	schemaCmd := schemaCmd()
	schemaCmd.AddCommand(
		schemaApplyCmd(),
		schemaCleanCmd(),
		schemaDiffCmd(),
		schemaFmtCmd(),
		schemaInspectCmd(),
		unsupportedCommand("schema", "test"),
		unsupportedCommand("schema", "plan"),
		unsupportedCommand("schema", "push"),
	)
	Root.AddCommand(schemaCmd)
	migrateCmd := migrateCmd()
	migrateCmd.AddCommand(
		migrateApplyCmd(),
		migrateDiffCmd(),
		migrateHashCmd(),
		migrateImportCmd(),
		migrateLintCmd(),
		migrateNewCmd(),
		migrateSetCmd(),
		migrateStatusCmd(),
		migrateValidateCmd(),
		unsupportedCommand("migrate", "checkpoint"),
		unsupportedCommand("migrate", "down"),
		unsupportedCommand("migrate", "rebase"),
		unsupportedCommand("migrate", "rm"),
		unsupportedCommand("migrate", "edit"),
		unsupportedCommand("migrate", "push"),
		unsupportedCommand("migrate", "test"),
	)
	Root.AddCommand(migrateCmd)
}

// unsupportedCommand create a stub command that reports
// the command is not supported by this build.
func unsupportedCommand(cmd, sub string) *cobra.Command {
	s := unsupportedMessage(cmd, sub)
	c := &cobra.Command{
		Hidden: true,
		Use:    fmt.Sprintf("%s is not supported by this build", sub),
		Short:  s,
		Long:   s,
		RunE: RunE(func(*cobra.Command, []string) error {
			return AbortErrorf("%s", s)
		}),
	}
	c.SetHelpTemplate(s + "\n")
	return c
}

// unsupportedMessage returns a message informing the user that the command
// or one of its options are not supported. For example:
//
// unsupportedMessage("migrate", "checkpoint")
// unsupportedMessage("schema", "apply --plan")
func unsupportedMessage(cmd, sub string) string {
	return fmt.Sprintf(
		`'atlas %s %s' is not supported by the community version.

To install the non-community version of Atlas, use the following command:

	curl -sSf https://atlasgo.sh | sh

Or, visit the website to see all installation options:

	https://atlasgo.io/docs#installation
`,
		cmd, sub,
	)
}

type (
	// Project represents an atlas.hcl project config file.
	Project struct {
		Envs  []*Env `spec:"env"`  // List of environments
		Lint  *Lint  `spec:"lint"` // Optional global lint policy
		Diff  *Diff  `spec:"diff"` // Optional global diff policy
		Test  *Test  `spec:"test"` // Optional test configuration
		cloud *cmdext.AtlasConfig
	}
)

const (
	envSkipUpgradeSuggestions = "ATLAS_NO_UPGRADE_SUGGESTIONS"
	oneWeek                   = 7 * 24 * time.Hour
)

// maySuggestUpgrade informs the user about the limitations of the community edition to stderr
// at most once a week. The user can disable this message by setting the ATLAS_NO_UPGRADE_SUGGESTIONS
// environment variable.
func maySuggestUpgrade(cmd *cobra.Command) {
	if os.Getenv(envSkipUpgradeSuggestions) != "" || testing.Testing() {
		return
	}
	state := cmdstate.File[LocalState]{Name: localStateFile}
	prev, err := state.Read()
	if err != nil {
		return
	}
	if time.Since(prev.UpgradeSuggested) < oneWeek {
		return
	}
	s := `Notice: This Atlas edition lacks support for features such as checkpoints,
testing, down migrations, and more. Additionally, advanced database objects such as views,
triggers, and stored procedures are not supported. To read more: https://atlasgo.io/community-edition

To install the non-community version of Atlas, use the following command:

	curl -sSf https://atlasgo.sh | sh

Or, visit the website to see all installation options:

	https://atlasgo.io/docs#installation

`
	_ = cmdlog.WarnOnce(cmd.ErrOrStderr(), cmdlog.ColorCyan(s))
	prev.UpgradeSuggested = time.Now()
	_ = state.Write(prev)
}

// migrateLintSetFlags allows setting extra flags for the 'migrate lint' command.
func migrateLintSetFlags(*cobra.Command, *migrateLintFlags) {}

// migrateLintRun is the run command for 'migrate lint'.
func migrateLintRun(cmd *cobra.Command, _ []string, flags migrateLintFlags, env *Env) error {
	dev, err := sqlclient.Open(cmd.Context(), flags.devURL)
	if err != nil {
		return err
	}
	defer dev.Close()
	dir, err := cmdmigrate.Dir(cmd.Context(), flags.dirURL, false)
	if err != nil {
		return err
	}
	var detect migratelint.ChangeDetector
	switch {
	case flags.latest == 0 && flags.gitBase == "":
		return fmt.Errorf("--%s or --%s is required", flagLatest, flagGitBase)
	case flags.latest > 0 && flags.gitBase != "":
		return fmt.Errorf("--%s and --%s are mutually exclusive", flagLatest, flagGitBase)
	case flags.latest > 0:
		detect = migratelint.LatestChanges(dir, int(flags.latest))
	case flags.gitBase != "":
		detect, err = migratelint.NewGitChangeDetector(
			dir,
			migratelint.WithWorkDir(flags.gitDir),
			migratelint.WithBase(flags.gitBase),
			migratelint.WithMigrationsPath(dir.(interface{ Path() string }).Path()),
		)
		if err != nil {
			return err
		}
	}
	format := migratelint.DefaultTemplate
	if f := flags.logFormat; f != "" {
		format, err = template.New("format").Funcs(migratelint.TemplateFuncs).Parse(f)
		if err != nil {
			return fmt.Errorf("parse format: %w", err)
		}
	}
	az, err := sqlcheck.AnalyzerFor(dev.Name, env.Lint.Remain())
	if err != nil {
		return err
	}
	r := &migratelint.Runner{
		Dev:            dev,
		Dir:            dir,
		ChangeDetector: detect,
		ReportWriter: &migratelint.TemplateWriter{
			T: format,
			W: cmd.OutOrStdout(),
		},
		Analyzers: az,
	}
	err = r.Run(cmd.Context())
	// Print the error in case it was not printed before.
	cmd.SilenceErrors = errors.As(err, &migratelint.SilentError{})
	cmd.SilenceUsage = cmd.SilenceErrors
	return err
}

func migrateDiffRun(cmd *cobra.Command, args []string, flags migrateDiffFlags, env *Env) error {
	if flags.dryRun {
		return errors.New("'--dry-run' is not supported in the community version")
	}
	ctx := cmd.Context()
	dev, err := sqlclient.Open(ctx, flags.devURL)
	if err != nil {
		return err
	}
	defer dev.Close()
	// Acquire a lock.
	unlock, err := dev.Lock(ctx, "atlas_migrate_diff", flags.lockTimeout)
	if err != nil {
		return fmt.Errorf("acquiring database lock: %w", err)
	}
	// If unlocking fails notify the user about it.
	defer func() { cobra.CheckErr(unlock()) }()
	// Open the migration directory.
	u, err := url.Parse(flags.dirURL)
	if err != nil {
		return err
	}
	dir, err := cmdmigrate.DirURL(ctx, u, false)
	if err != nil {
		return err
	}
	if flags.edit {
		l, ok := dir.(*migrate.LocalDir)
		if !ok {
			return fmt.Errorf("--edit flag supports only atlas directories, but got: %T", dir)
		}
		dir = &editDir{l}
	}
	var name, indent string
	if len(args) > 0 {
		name = args[0]
	}
	f, err := cmdmigrate.Formatter(u)
	if err != nil {
		return err
	}
	if f, indent, err = mayIndent(u, f, flags.format); err != nil {
		return err
	}
	diffOpts := diffOptions(cmd, env)
	// If there is a state-loader that requires a custom
	// 'migrate diff' handling, offload it the work.
	if d, ok := cmdext.States.Differ(flags.desiredURLs); ok {
		err := d.MigrateDiff(ctx, &cmdext.MigrateDiffOptions{
			To:      flags.desiredURLs,
			Name:    name,
			Indent:  indent,
			Dir:     dir,
			Dev:     dev,
			Options: diffOpts,
		})
		return maskNoPlan(cmd, err)
	}
	// Get a state reader for the desired state.
	desired, err := stateReader(ctx, env, &stateReaderConfig{
		urls:    flags.desiredURLs,
		dev:     dev,
		client:  dev,
		schemas: flags.schemas,
		vars:    env.Vars(),
	})
	if err != nil {
		return err
	}
	defer desired.Close()
	opts := []migrate.PlannerOption{
		migrate.PlanFormat(f),
		migrate.PlanWithIndent(indent),
		migrate.PlanWithDiffOptions(diffOpts...),
	}
	if dev.URL.Schema != "" {
		// Disable tables qualifier in schema-mode.
		opts = append(opts, migrate.PlanWithSchemaQualifier(flags.qualifier))
	}
	// Plan the changes and create a new migration file.
	pl := migrate.NewPlanner(dev.Driver, dir, opts...)
	plan, err := func() (*migrate.Plan, error) {
		if dev.URL.Schema != "" {
			return pl.PlanSchema(ctx, name, desired.StateReader)
		}
		return pl.Plan(ctx, name, desired.StateReader)
	}()
	var cerr *migrate.NotCleanError
	switch {
	case errors.As(err, &cerr) && dev.URL.Schema == "" && desired.Schema != "":
		return fmt.Errorf("dev database is not clean (%s). Add a schema to the URL to limit the scope of the connection", cerr.Reason)
	case err != nil:
		return maskNoPlan(cmd, err)
	default:
		return pl.WritePlan(plan)
	}
}

// schemaApplyRunE is the community version of the 'atlas schema apply' command.
func schemaApplyRunE(cmd *cobra.Command, _ []string, flags *schemaApplyFlags) error {
	switch {
	case flags.edit:
		return AbortErrorf("%s", unsupportedMessage("schema", "apply --edit"))
	case flags.planURL != "":
		return AbortErrorf("%s", unsupportedMessage("schema", "apply --plan"))
	case len(flags.include) > 0:
		return AbortErrorf("%s", unsupportedMessage("schema", "apply --include"))
	case GlobalFlags.SelectedEnv == "":
		env, err := selectEnv(cmd)
		if err != nil {
			return err
		}
		return schemaApplyRun(cmd, *flags, env)
	default:
		_, envs, err := EnvByName(cmd, GlobalFlags.SelectedEnv, GlobalFlags.Vars)
		if err != nil {
			return err
		}
		if len(envs) != 1 {
			return fmt.Errorf("multi-environment %q is not supported", GlobalFlags.SelectedEnv)
		}
		if err := setSchemaEnvFlags(cmd, envs[0]); err != nil {
			return err
		}
		return schemaApplyRun(cmd, *flags, envs[0])
	}
}

func schemaApplyRun(cmd *cobra.Command, flags schemaApplyFlags, env *Env) error {
	var (
		err    error
		ctx    = cmd.Context()
		dev    *sqlclient.Client
		format = cmdlog.SchemaPlanTemplate
	)
	if err = flags.check(env); err != nil {
		return err
	}
	if v := flags.logFormat; v != "" {
		if !flags.dryRun && !flags.autoApprove {
			return errors.New(`--log and --format can only be used with --dry-run or --auto-approve`)
		}
		if format, err = template.New("format").Funcs(cmdlog.ApplyTemplateFuncs).Parse(v); err != nil {
			return fmt.Errorf("parse log format: %w", err)
		}
	}
	if flags.devURL != "" {
		if dev, err = sqlclient.Open(ctx, flags.devURL); err != nil {
			return err
		}
		defer dev.Close()
	}
	from, err := stateReader(ctx, env, &stateReaderConfig{
		urls:    []string{flags.url},
		schemas: flags.schemas,
		exclude: flags.exclude,
	})
	if err != nil {
		return err
	}
	defer from.Close()
	client, ok := from.Closer.(*sqlclient.Client)
	if !ok {
		return errors.New("--url must be a database connection")
	}
	to, err := stateReader(ctx, env, &stateReaderConfig{
		urls:    flags.toURLs,
		dev:     dev,
		client:  client,
		schemas: flags.schemas,
		exclude: flags.exclude,
		vars:    env.Vars(),
	})
	if err != nil {
		return err
	}
	defer to.Close()
	diff, err := computeDiff(ctx, client, from, to, diffOptions(cmd, env)...)
	if err != nil {
		return err
	}
	maySuggestUpgrade(cmd)
	// Returning at this stage should
	// not trigger the help message.
	cmd.SilenceUsage = true
	switch changes := diff.changes; {
	case len(changes) == 0:
		return format.Execute(cmd.OutOrStdout(), &cmdlog.SchemaApply{})
	case flags.logFormat != "" && flags.autoApprove:
		var (
			applied int
			plan    *migrate.Plan
			cause   *cmdlog.StmtError
			out     = cmd.OutOrStdout()
		)
		if plan, err = client.PlanChanges(ctx, "", changes, planOptions(client)...); err != nil {
			return err
		}
		if err = applyChanges(ctx, client, changes, flags.txMode); err == nil {
			applied = len(plan.Changes)
		} else if i, ok := err.(interface{ Applied() int }); ok && i.Applied() < len(plan.Changes) {
			applied, cause = i.Applied(), &cmdlog.StmtError{Stmt: plan.Changes[i.Applied()].Cmd, Text: err.Error()}
		} else {
			cause = &cmdlog.StmtError{Text: err.Error()}
		}
		err1 := format.Execute(out, cmdlog.NewSchemaApply(ctx, cmdlog.NewEnv(client, nil), plan.Changes[:applied], plan.Changes[applied:], cause))
		return errors.Join(err, err1)
	default:
		switch err := summary(cmd, client, changes, format); {
		case err != nil:
			return err
		case flags.dryRun:
			return nil
		case flags.autoApprove:
			return applyChanges(ctx, client, changes, flags.txMode)
		default:
			return promptApply(cmd, flags, diff, client, dev)
		}
	}
}

// applySchemaClean is the community-version of the 'atlas schema clean' handler.
func applySchemaClean(cmd *cobra.Command, client *sqlclient.Client, drop []schema.Change, flags schemaCleanFlags) error {
	if flags.dryRun {
		return AbortErrorf("%s", unsupportedMessage("schema", "clean --dry-run"))
	}
	if flags.logFormat != "" {
		return AbortErrorf("%s", unsupportedMessage("schema", "clean --format"))
	}
	if len(drop) == 0 {
		cmd.Println("Nothing to drop")
		return nil
	}
	if err := summary(cmd, client, drop, cmdlog.SchemaPlanTemplate); err != nil {
		return err
	}
	if flags.autoApprove || promptUser(cmd) {
		if err := client.ApplyChanges(cmd.Context(), drop); err != nil {
			return err
		}
	}
	return nil
}

func schemaDiffRun(cmd *cobra.Command, _ []string, flags schemaDiffFlags, env *Env) error {
	var (
		ctx = cmd.Context()
		c   *sqlclient.Client
	)
	if len(flags.include) > 0 {
		return AbortErrorf("%s", unsupportedMessage("schema", "diff --include"))
	}
	// We need a driver for diffing and planning. If given, dev database has precedence.
	if flags.devURL != "" {
		var err error
		c, err = sqlclient.Open(ctx, flags.devURL)
		if err != nil {
			return err
		}
		defer c.Close()
	}
	from, err := stateReader(ctx, env, &stateReaderConfig{
		urls:    flags.fromURL,
		dev:     c,
		vars:    env.Vars(),
		schemas: flags.schemas,
		exclude: flags.exclude,
	})
	if err != nil {
		return err
	}
	defer from.Close()
	to, err := stateReader(ctx, env, &stateReaderConfig{
		urls:    flags.toURL,
		dev:     c,
		vars:    env.Vars(),
		schemas: flags.schemas,
		exclude: flags.exclude,
	})
	if err != nil {
		return err
	}
	defer to.Close()
	if c == nil {
		// If not both states are provided by a database connection, the call to state-reader would have returned
		// an error already. If we land in this case, we can assume both states are database connections.
		c = to.Closer.(*sqlclient.Client)
	}
	format := cmdlog.SchemaDiffTemplate
	if v := flags.format; v != "" {
		if format, err = template.New("format").Funcs(cmdlog.SchemaDiffFuncs).Parse(v); err != nil {
			return fmt.Errorf("parse log format: %w", err)
		}
	}
	diff, err := computeDiff(ctx, c, from, to, diffOptions(cmd, env)...)
	if err != nil {
		return err
	}
	maySuggestUpgrade(cmd)
	return format.Execute(cmd.OutOrStdout(),
		cmdlog.NewSchemaDiff(ctx, c, diff.from, diff.to, diff.changes),
	)
}

func summary(cmd *cobra.Command, c *sqlclient.Client, changes []schema.Change, t *template.Template) error {
	p, err := c.PlanChanges(cmd.Context(), "", changes, planOptions(c)...)
	if err != nil {
		return err
	}
	return t.Execute(
		cmd.OutOrStdout(),
		cmdlog.NewSchemaPlan(cmd.Context(), cmdlog.NewEnv(c, nil), p.Changes, nil),
	)
}

func promptApply(cmd *cobra.Command, flags schemaApplyFlags, diff *diff, client, _ *sqlclient.Client) error {
	if !flags.dryRun && (flags.autoApprove || promptUser(cmd)) {
		return applyChanges(cmd.Context(), client, diff.changes, flags.txMode)
	}
	return nil
}

func maySetLoginContext(*cobra.Command, *Project) error {
	return nil
}

func setEnvs(context.Context, []*Env) {}

// specOptions are the options for the schema spec.
var specOptions []schemahcl.Option

// diffOptions returns environment-aware diff options.
func diffOptions(_ *cobra.Command, env *Env) []schema.DiffOption {
	return append(env.DiffOptions(), schema.DiffNormalized())
}

// openClient allows opening environment-aware clients.
func (*Env) openClient(ctx context.Context, u string) (*sqlclient.Client, error) {
	return sqlclient.Open(ctx, u)
}

type schemaInspectFlags struct {
	url       string   // URL of resource to inspect.
	devURL    string   // URL of the dev database.
	logFormat string   // Format of the log output.
	schemas   []string // Schemas to take into account when diffing.
	exclude   []string // List of glob patterns used to filter resources from applying (see schema.InspectOptions).
}

// schemaInspectCmd represents the 'atlas schema inspect' subcommand.
func schemaInspectCmd() *cobra.Command {
	cmd, _ := schemaInspectCmdWithFlags()
	return cmd
}

func schemaInspectCmdWithFlags() (*cobra.Command, *schemaInspectFlags) {
	var (
		env   *Env
		flags schemaInspectFlags
		cmd   = &cobra.Command{
			Use:   "inspect",
			Short: "Inspect a database and print its schema in Atlas DDL syntax.",
			Long: `'atlas schema inspect' connects to the given database and inspects its schema.
It then prints to the screen the schema of that database in Atlas DDL syntax. This output can be
saved to a file, commonly by redirecting the output to a file named with a ".hcl" suffix:

  atlas schema inspect -u "mysql://user:pass@localhost:3306/dbname" > schema.hcl

This file can then be edited and used with the` + " `atlas schema apply` " + `command to plan
and execute schema migrations against the given database. In cases where users wish to inspect
all multiple schemas in a given database (for instance a MySQL server may contain multiple named
databases), omit the relevant part from the url, e.g. "mysql://user:pass@localhost:3306/".
To select specific schemas from the databases, users may use the "--schema" (or "-s" shorthand)
flag.
	`,
			Example: `  atlas schema inspect -u "mysql://user:pass@localhost:3306/dbname"
  atlas schema inspect -u "mariadb://user:pass@localhost:3306/" --schema=schemaA,schemaB -s schemaC
  atlas schema inspect --url "postgres://user:pass@host:port/dbname?sslmode=disable"
  atlas schema inspect -u "sqlite://file:ex1.db?_fk=1"`,
			PreRunE: RunE(func(cmd *cobra.Command, args []string) (err error) {
				if env, err = selectEnv(cmd); err != nil {
					return err
				}
				return setSchemaEnvFlags(cmd, env)
			}),
			RunE: RunE(func(cmd *cobra.Command, args []string) error {
				return schemaInspectRun(cmd, args, flags, env)
			}),
		}
	)
	cmd.Flags().SortFlags = false
	addFlagURL(cmd.Flags(), &flags.url)
	addFlagDevURL(cmd.Flags(), &flags.devURL)
	addFlagSchemas(cmd.Flags(), &flags.schemas)
	addFlagExclude(cmd.Flags(), &flags.exclude)
	addFlagLog(cmd.Flags(), &flags.logFormat)
	addFlagFormat(cmd.Flags(), &flags.logFormat)
	cobra.CheckErr(cmd.MarkFlagRequired(flagURL))
	cmd.MarkFlagsMutuallyExclusive(flagLog, flagFormat)
	return cmd, &flags
}

func schemaInspectRun(cmd *cobra.Command, _ []string, flags schemaInspectFlags, env *Env) error {
	var (
		ctx = cmd.Context()
		dev *sqlclient.Client
	)
	useDev, err := readerUseDev(env, flags.url)
	if err != nil {
		return err
	}
	if flags.devURL != "" && useDev {
		if dev, err = sqlclient.Open(ctx, flags.devURL); err != nil {
			return err
		}
		defer dev.Close()
	}
	r, err := stateReader(ctx, env, &stateReaderConfig{
		urls:    []string{flags.url},
		dev:     dev,
		vars:    env.Vars(),
		schemas: flags.schemas,
		exclude: flags.exclude,
	})
	if err != nil {
		return err
	}
	defer r.Close()
	client, ok := r.Closer.(*sqlclient.Client)
	if !ok && dev != nil {
		client = dev
	}
	format := cmdlog.SchemaInspectTemplate
	if v := flags.logFormat; v != "" {
		if format, err = template.New("format").Funcs(cmdlog.InspectTemplateFuncs).Parse(v); err != nil {
			return fmt.Errorf("parse log format: %w", err)
		}
	}
	s, err := r.ReadState(ctx)
	if err != nil {
		return err
	}
	maySuggestUpgrade(cmd)
	i := cmdlog.NewSchemaInspect(ctx, client, s)
	i.URL = flags.url
	return format.Execute(cmd.OutOrStdout(), i)
}
