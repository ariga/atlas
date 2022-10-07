// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"context"
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
)

type diffCmdOpts struct {
	fromURL string
	toURL   string
	devURL  string
}

// newDiffCmd returns a new *cobra.Command that runs cmdDiffRun with the given flags.
func newDiffCmd() *cobra.Command {
	var opts diffCmdOpts
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Calculate and print the diff between two schemas.",
		Long: `'atlas schema diff' reads the state of two given schema definitions, 
calculates the difference in their schemas, and prints a plan of
SQL statements to migrate the "from" database to the schema of the "to" database.
The database states can be read from a connected database, an HCL project or a migration directory.`,
		Example: `  atlas schema diff --from mysql://user:pass@localhost:3306/test --to file://schema.hcl
  atlas schema diff --from mysql://user:pass@localhost:3306 --to file://schema_1.hcl --to file://schema_2.hcl
  atlas schema diff --from mysql://user:pass@localhost:3306 --to file://migrations`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdDiffRun(cmd, &opts)
		},
	}
	cmd.Flags().StringVarP(&opts.fromURL, "from", "", "", "[driver://username:password@protocol(address)/dbname?param=value] select a database using the URL format")
	cmd.Flags().StringVarP(&opts.toURL, "to", "", "", "[driver://username:password@protocol(address)/dbname?param=value] select a database using the URL format")
	cmd.Flags().StringVarP(&opts.devURL, "dev-url", "", "", "[driver://username:password@protocol(address)/dbname?param=value] select a database using the URL format")
	cmd.Flags().SortFlags = false
	cobra.CheckErr(cmd.MarkFlagRequired("from"))
	cobra.CheckErr(cmd.MarkFlagRequired("to"))
	return cmd
}

// cmdDiffRun connects to the given databases, and prints an SQL plan to get from
// the "from" schema to the "to" schema.
func cmdDiffRun(cmd *cobra.Command, flags *diffCmdOpts) error {
	var (
		ctx = cmd.Context()
		c   *sqlclient.Client
	)
	// We need a driver for diffing and planning. If given, dev database has precedence.
	if flags.devURL != "" {
		var err error
		c, err = sqlclient.Open(ctx, flags.devURL)
		if err != nil {
			return err
		}
		defer c.Close()
	}
	from, err := stateReader(ctx, &stateReaderConfig{urls: []string{flags.fromURL}, dev: c})
	if err != nil {
		return err
	}
	defer from.Close()
	to, err := stateReader(ctx, &stateReaderConfig{urls: []string{flags.toURL}, dev: c})
	if err != nil {
		return err
	}
	defer to.Close()
	if c == nil {
		// If not both states are provided by a database connection, the call to state-reader would have returned
		// an error already. If we land in this case, we can assume both states are database connections.
		c = to.Closer.(*sqlclient.Client)
	}
	current, err := from.ReadState(ctx)
	if err != nil {
		return err
	}
	desired, err := to.ReadState(ctx)
	if err != nil {
		return err
	}
	var diff []schema.Change
	switch {
	// compare realm
	case from.Schema == "" && to.Schema == "":
		diff, err = c.RealmDiff(current, desired)
		if err != nil {
			return err
		}
	case from.Schema == "":
		return fmt.Errorf("cannot diff schema %q with a database connection", from.Schema)
	case to.Schema == "":
		return fmt.Errorf("cannot diff database connection with a schema %q", to.Schema)
	default:
		// SchemaDiff checks for name equality which is irrelevant in the case
		// the user wants to compare their contents, reset them to allow the comparison.
		current.Schemas[0].Name, desired.Schemas[0].Name = "", ""
		diff, err = c.SchemaDiff(current.Schemas[0], desired.Schemas[0])
		if err != nil {
			return err
		}
	}
	p, err := c.PlanChanges(ctx, "plan", diff)
	if err != nil {
		return err
	}
	if len(p.Changes) == 0 {
		cmd.Println("Schemas are synced, no changes to be made.")
	}
	for _, c := range p.Changes {
		if c.Comment != "" {
			cmd.Println("--", strings.ToUpper(c.Comment[:1])+c.Comment[1:])
		}
		cmd.Println(c.Cmd)
	}
	return nil
}

type (
	// stateReadCloser is a migrate.StateReader with an optional io.Closer.
	stateReadCloser struct {
		migrate.StateReader
		io.Closer        // optional close function
		Schema    string // in case we work on a single schema
	}
	// stateReaderConfig is given to stateReader.
	stateReaderConfig struct {
		urls    []string          // urls to create a migrate.StateReader from
		dev     *sqlclient.Client // dev database connection
		schemas []string          // schemas to work on
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
		// Replaying a migration directory or evaluating an HCL file requires a dev connection.
		if config.dev == nil {
			return nil, errors.New("--dev-url cannot be empty")
		}
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
			dir, err := dirURL(parsed[0], false)
			if err != nil {
				return nil, err
			}
			ex, err := migrate.NewExecutor(config.dev.Driver, dir, migrate.NopRevisionReadWriter{})
			if err != nil {
				return nil, err
			}
			sr, err := ex.Replay(ctx, func() migrate.StateReader {
				if config.dev.URL.Schema != "" {
					return migrate.SchemaConn(config.dev, "", nil)
				}
				return migrate.RealmConn(config.dev, nil)
			}())
			if err != nil && !errors.Is(err, migrate.ErrNoPendingFiles) {
				return nil, fmt.Errorf("replaying the migration directory: %w", err)
			}
			return &stateReadCloser{
				StateReader: migrate.Realm(sr),
				Schema:      config.dev.URL.Schema,
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
			sr = migrate.RealmConn(c.Driver, nil)
		default:
			sr = migrate.SchemaConn(c.Driver, c.URL.Schema, nil)
		}
		return &stateReadCloser{
			StateReader: sr,
			Closer:      c,
			Schema:      c.URL.Schema,
		}, nil
	}
}

// hclStateReadr returns a migrate.StateReader that reads the state from the given HCL paths urls.
func hclStateReader(ctx context.Context, config *stateReaderConfig, urls []*url.URL) (*stateReadCloser, error) {
	paths := make([]string, len(urls))
	for i, u := range urls {
		paths[i] = u.Path
	}
	parser, err := parseHCLPaths(paths...)
	if err != nil {
		return nil, err
	}
	realm := &schema.Realm{}
	if err := config.dev.Eval(parser, realm, nil); err != nil {
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
	if config.dev.URL.Schema != "" && len(realm.Schemas) > 1 {
		return nil, fmt.Errorf(
			"cannot use HCL with more than 1 schema when dev-url is limited to schema %q",
			config.dev.URL.Schema,
		)
	}
	if norm, ok := config.dev.Driver.(schema.Normalizer); ok && len(realm.Schemas) > 0 {
		realm, err = norm.NormalizeRealm(ctx, realm)
		if err != nil {
			return nil, err
		}
	}
	t := &stateReadCloser{StateReader: migrate.Realm(realm)}
	if len(realm.Schemas) == 1 {
		t.Schema = realm.Schemas[0].Name
	}
	return t, nil
}

// Close redirects calls to Close to the enclosed io.Closer.
func (sr *stateReadCloser) Close() {
	if sr.Closer != nil {
		sr.Closer.Close()
	}
}
