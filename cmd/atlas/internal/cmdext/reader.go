// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdext

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	cmdmigrate "ariga.io/atlas/cmd/atlas/internal/migrate"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

type (
	// StateReadCloser is a migrate.StateReader with an optional io.Closer.
	StateReadCloser struct {
		migrate.StateReader
		io.Closer        // optional close function
		Schema    string // in case we work on a single schema
		HCL       bool   // true if state was read from HCL files since in that case we always compare realms
	}
	// StateReaderConfig is given to stateReader.
	StateReaderConfig struct {
		URLs        []*url.URL        // urls to create a migrate.StateReader from
		Client, Dev *sqlclient.Client // database connections, while dev is considered a dev database, client is not
		Schemas     []string          // schemas to work on
		Exclude     []string          // exclude flag values
		Vars        map[string]cty.Value
	}
)

// Close redirects calls to Close to the enclosed io.Closer.
func (r *StateReadCloser) Close() {
	if r.Closer != nil {
		r.Closer.Close()
	}
}

// StateReaderSQL returns a migrate.StateReader from an SQL file or a directory of migrations.
func StateReaderSQL(ctx context.Context, config *StateReaderConfig) (*StateReadCloser, error) {
	if len(config.URLs) != 1 {
		return nil, fmt.Errorf("the provided SQL state must be either a single schema file or a migration directory, but %d paths were found", len(config.URLs))
	}
	// Replaying a migration directory requires a dev connection.
	if config.Dev == nil {
		return nil, errors.New("--dev-url cannot be empty")
	}
	var (
		dir  migrate.Dir
		opts []migrate.ReplayOption
		path = filepath.Join(config.URLs[0].Host, config.URLs[0].Path)
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
		if dir, err = fileAsDir(fi.Name(), b); err != nil {
			return nil, err
		}
	// A migration directory.
	default:
		if dir, err = cmdmigrate.DirURL(config.URLs[0], false); err != nil {
			return nil, err
		}
		if v := config.URLs[0].Query().Get("version"); v != "" {
			opts = append(opts, migrate.ReplayToVersion(v))
		}
	}
	return stateReaderSQL(ctx, config, dir, opts...)
}

// stateReaderSQL returns a migrate.StateReader from an SQL file or a directory of migrations.
func stateReaderSQL(ctx context.Context, config *StateReaderConfig, dir migrate.Dir, opts ...migrate.ReplayOption) (*StateReadCloser, error) {
	ex, err := migrate.NewExecutor(config.Dev.Driver, dir, migrate.NopRevisionReadWriter{})
	if err != nil {
		return nil, err
	}
	sr, err := ex.Replay(ctx, func() migrate.StateReader {
		if config.Dev.URL.Schema != "" {
			return migrate.SchemaConn(config.Dev, "", nil)
		}
		return migrate.RealmConn(config.Dev, &schema.InspectRealmOption{
			Schemas: config.Schemas,
			Exclude: config.Exclude,
		})
	}(), opts...)
	if err != nil && !errors.Is(err, migrate.ErrNoPendingFiles) {
		return nil, err
	}
	return &StateReadCloser{
		StateReader: migrate.Realm(sr),
		Schema:      config.Dev.URL.Schema,
	}, nil
}

// StateReaderHCL returns a StateReader that reads the state from the given HCL paths urls.
func StateReaderHCL(ctx context.Context, c *StateReaderConfig) (*StateReadCloser, error) {
	paths := make([]string, len(c.URLs))
	for i, u := range c.URLs {
		paths[i] = filepath.Join(u.Host, u.Path)
	}
	return stateReaderHCL(ctx, c, paths)
}

// stateReaderHCL is shared between StateReaderHCL and "hcl_schema" datasource.
func stateReaderHCL(ctx context.Context, config *StateReaderConfig, paths []string) (*StateReadCloser, error) {
	var client *sqlclient.Client
	switch {
	case config.Dev != nil:
		client = config.Dev
	case config.Client != nil:
		client = config.Client
	default:
		return nil, errors.New("--dev-url cannot be empty")
	}
	parser, err := parseHCLPaths(paths...)
	if err != nil {
		return nil, err
	}
	realm := &schema.Realm{}
	if err := client.Eval(parser, realm, config.Vars); err != nil {
		return nil, err
	}
	if len(config.Schemas) > 0 {
		// Validate all schemas in file were selected by user.
		sm := make(map[string]bool, len(config.Schemas))
		for _, s := range config.Schemas {
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
			config.Dev.URL.Schema,
		)
	}
	if nr, ok := client.Driver.(schema.Normalizer); ok && config.Dev != nil { // only normalize on a dev database
		if config.Dev.URL.Schema != "" {
			realm.Schemas[0], err = nr.NormalizeSchema(ctx, realm.Schemas[0])
		} else {
			realm, err = nr.NormalizeRealm(ctx, realm)
		}
		if err != nil {
			return nil, err
		}
	}
	return &StateReadCloser{StateReader: migrate.Realm(realm), HCL: true}, nil
}

// parseHCLPaths parses the HCL files in the given paths. If a path represents a directory,
// its direct descendants will be considered, skipping any subdirectories. If a project file
// is present in the input paths, an error is returned.
func parseHCLPaths(paths ...string) (*hclparse.Parser, error) {
	p := hclparse.NewParser()
	for _, path := range paths {
		switch stat, err := os.Stat(path); {
		case err != nil:
			return nil, err
		case stat.IsDir():
			dir, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}
			for _, f := range dir {
				// Skip nested dirs.
				if f.IsDir() {
					continue
				}
				if err := mayParse(p, filepath.Join(path, f.Name())); err != nil {
					return nil, err
				}
			}
		default:
			if err := mayParse(p, path); err != nil {
				return nil, err
			}
		}
	}
	if len(p.Files()) == 0 {
		return nil, fmt.Errorf("no schema files found in: %s", paths)
	}
	return p, nil
}

const (
	extHCL = ".hcl"
	extSQL = ".sql"
)

// mayParse will parse the file in path if it is an HCL file. If the file is an Atlas
// project file an error is returned.
func mayParse(p *hclparse.Parser, path string) error {
	if n := filepath.Base(path); filepath.Ext(n) != extHCL {
		return nil
	}
	switch f, diag := p.ParseHCLFile(path); {
	case diag.HasErrors():
		return diag
	case isProjectFile(f):
		return fmt.Errorf("cannot parse project file %q as a schema file", path)
	default:
		return nil
	}
}

func isProjectFile(f *hcl.File) bool {
	for _, b := range f.Body.(*hclsyntax.Body).Blocks {
		if b.Type == "env" {
			return true
		}
	}
	return false
}

func fileAsDir(name string, b []byte) (migrate.Dir, error) {
	dir := &migrate.MemDir{}
	if err := dir.WriteFile(name, b); err != nil {
		return nil, err
	}
	// Create a checksum file to bypass the checksum check.
	sum, err := dir.Checksum()
	if err != nil {
		return nil, err
	}
	if err = migrate.WriteSumFile(dir, sum); err != nil {
		return nil, err
	}
	return dir, nil
}
