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
	"strings"

	cmdmigrate "ariga.io/atlas/cmd/atlas/internal/migrate"
	"ariga.io/atlas/schemahcl"
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
		WithPos     bool              // Indicate if schema.Pos should be loaded.
		Vars        map[string]cty.Value
	}
)

// Close redirects calls to Close to the enclosed io.Closer.
func (r *StateReadCloser) Close() error {
	if r.Closer != nil {
		return r.Closer.Close()
	}
	return nil
}

// isSchemaDir returns true if the given path is a schema directory (not a migration directory).
func isSchemaDir(u *url.URL, path string) bool {
	if q := u.Query(); q.Has("version") || q.Has("format") || filepath.Base(path) == cmdmigrate.DefaultDirName {
		return false
	}
	_, err := os.Stat(filepath.Join(path, migrate.HashFileName))
	return errors.Is(err, os.ErrNotExist)
}

// errNoDevURL is returned when trying to read an SQL schema file/directory or replay a migration directory,
// the dev-url was not set.
var errNoDevURL = errors.New("--dev-url cannot be empty. See: https://atlasgo.io/atlas-schema/sql#dev-database")

// stateSchemaSQL wraps stateReaderSQL for SQL schema files or directories to control errors when replay/read fails.
func stateSchemaSQL(ctx context.Context, cfg *StateReaderConfig, dir migrate.Dir) (*StateReadCloser, error) {
	if cfg.Dev == nil {
		return nil, errNoDevURL
	}
	log := &errorRecorder{}
	r, err := stateReaderSQL(ctx, cfg, dir, []migrate.ExecutorOption{migrate.WithLogger(log)}, nil)
	if n := len(log.applied); err != nil && n > 0 && log.stmt != "" && log.text != "" {
		err = fmt.Errorf("read state from %q: executing statement: %q: %s", log.applied[n-1], log.stmt, log.text)
	}
	return r, err
}

type errorRecorder struct {
	applied    []string // applied files.
	stmt, text string   // error statement and text.
}

// Log implements migrate.Logger.
func (r *errorRecorder) Log(e migrate.LogEntry) {
	switch e := e.(type) {
	case migrate.LogFile:
		r.applied = append(r.applied, e.File.Name())
	case migrate.LogError:
		r.stmt = e.SQL
		r.text = e.Error.Error()
	}
}

// stateReaderSQL returns a migrate.StateReader from an SQL file or a directory of migrations.
func stateReaderSQL(ctx context.Context, cfg *StateReaderConfig, dir migrate.Dir, optsExec []migrate.ExecutorOption, optsReplay []migrate.ReplayOption) (*StateReadCloser, error) {
	if cfg.Dev == nil {
		return nil, errNoDevURL
	}
	ex, err := migrate.NewExecutor(cfg.Dev.Driver, dir, migrate.NopRevisionReadWriter{}, optsExec...)
	if err != nil {
		return nil, err
	}
	sr, err := ex.Replay(ctx, func() migrate.StateReader {
		if cfg.Dev.URL.Schema != "" {
			return migrate.SchemaConn(cfg.Dev, "", &schema.InspectOptions{
				Exclude: cfg.Exclude,
			})
		}
		return migrate.RealmConn(cfg.Dev, &schema.InspectRealmOption{
			Schemas: cfg.Schemas,
			Exclude: cfg.Exclude,
		})
	}(), optsReplay...)
	if err != nil && !errors.Is(err, migrate.ErrNoPendingFiles) {
		return nil, err
	}
	return &StateReadCloser{
		StateReader: migrate.Realm(sr),
		Schema:      cfg.Dev.URL.Schema,
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
func stateReaderHCL(_ context.Context, config *StateReaderConfig, paths []string) (*StateReadCloser, error) {
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
	var (
		eval  = client.Eval
		realm = &schema.Realm{}
	)
	if e, ok := client.Evaluator.(interface {
		EvalOptions(*hclparse.Parser, any, *schemahcl.EvalOptions) error
	}); ok && config.WithPos {
		eval = func(pr *hclparse.Parser, v any, vars map[string]cty.Value) error {
			return e.EvalOptions(pr, v, &schemahcl.EvalOptions{Variables: vars, RecordPos: true})
		}
	}
	if err := eval(parser, realm, config.Vars); err != nil {
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
	// In case the dev or client connection is bound to a specific schema, we require
	// the desired schema to contain only one schema. Thus, executing diff will be
	// done on the content of these two schema and not the whole realm.
	switch {
	case config.Dev != nil && config.Dev.URL.Schema != "" && len(realm.Schemas) > 1:
		return nil, fmt.Errorf(
			"cannot use HCL with more than 1 schema when dev-url is limited to schema %q",
			config.Dev.URL.Schema,
		)
	case config.Client != nil && config.Client.URL.Schema != "" && len(realm.Schemas) > 1:
		return nil, fmt.Errorf(
			"cannot use HCL with more than 1 schema when url is limited to schema %q",
			config.Client.URL.Schema,
		)
	}
	var (
		normalized  bool
		schemaScope string
	)
	// The "Schema" below indicates the HCL represents a single
	// database schema, and the work is scoped to this schema.
	if len(realm.Schemas) == 1 && (config.Dev != nil && config.Dev.URL.Schema != "" || config.Client != nil && config.Client.URL.Schema != "") {
		schemaScope = realm.Schemas[0].Name
	}
	return &StateReadCloser{
		HCL:    true,
		Schema: schemaScope,
		// Defer normalization until the first call to ReadState. This is required because the same
		// dev-database is used for both migration replaying and schema normalization. As a result,
		// objects created by the migrations, which are not yet supported by Atlas, such as functions,
		// won't be cleaned and can be referenced by the HCL schema.
		StateReader: migrate.StateReaderFunc(func(ctx context.Context) (*schema.Realm, error) {
			// Normalize once, only on dev database connection.
			if nr, ok := client.Driver.(schema.Normalizer); ok && !normalized && config.Dev != nil {
				switch {
				// Empty schema file.
				case len(realm.Schemas) == 0:
				case config.Dev.URL.Schema != "":
					realm.Schemas[0], err = nr.NormalizeSchema(ctx, realm.Schemas[0])
				default:
					realm, err = nr.NormalizeRealm(ctx, realm)
				}
				if err != nil {
					return nil, err
				}
			}
			switch {
			case len(config.Exclude) == 0:
			case schemaScope != "" && len(realm.Schemas) == 1:
				realm.Schemas[0], err = schema.ExcludeSchema(realm.Schemas[0], config.Exclude)
			default:
				realm, err = schema.ExcludeRealm(realm, config.Exclude)
			}
			if err != nil {
				return nil, err
			}
			return realm, nil
		}),
	}, nil
}

// FilesExt returns the file extension of the given URLs.
// Note, all URL must have the same extension.
func FilesExt(urls []*url.URL) (string, error) {
	var path, ext string
	set := func(curr string) error {
		switch e := filepath.Ext(curr); {
		case e != FileTypeHCL && e != FileTypeSQL:
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
				case FileTypeHCL, FileTypeSQL:
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

// Schema reader types (URL schemes).
const (
	SchemaTypeFile  = "file"
	SchemaTypeAtlas = "atlas"
)

// File extensions supported by the file driver.
const (
	FileTypeHCL  = ".hcl"
	FileTypeSQL  = ".sql"
	FileTypeTest = ".test.hcl"
)

// mayParse will parse the file in path if it is an HCL file. If the file is an Atlas
// project file an error is returned.
func mayParse(p *hclparse.Parser, path string) error {
	if n := filepath.Base(path); filepath.Ext(n) != FileTypeHCL && !strings.HasSuffix(path, FileTypeTest) {
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

// FilesAsDir wraps the given files as MemDir.
func FilesAsDir(files ...migrate.File) (migrate.Dir, error) {
	dir := &migrate.MemDir{}
	for _, f := range files {
		if err := dir.WriteFile(f.Name(), f.Bytes()); err != nil {
			return nil, err
		}
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
