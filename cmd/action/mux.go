// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlite"

	"github.com/go-sql-driver/mysql"
)

type (
	// Mux is used for routing URLs to their correct provider.
	Mux struct {
		providers map[string]func(string) (*Driver, error)
	}

	// Driver implements the Atlas interface.
	Driver struct {
		migrate.Driver
		schemaspec.Marshaler
		schemaspec.Unmarshaler
	}
)

// NewMux returns a new Mux.
func NewMux() *Mux {
	return &Mux{
		providers: make(map[string]func(string) (*Driver, error)),
	}
}

var (
	// DefaultMux is the default Mux that is used by the different commands.
	DefaultMux = NewMux()
	reMemMode  = regexp.MustCompile(":memory:|^file:.*mode=memory.*")
)

// RegisterProvider is used to register a Driver provider by key.
func (u *Mux) RegisterProvider(key string, p func(string) (*Driver, error)) {
	if _, ok := u.providers[key]; ok {
		panic("provider is already initialized")
	}
	u.providers[key] = p
}

// OpenAtlas is used for opening an atlas driver on a specific data source.
func (u *Mux) OpenAtlas(url string) (*Driver, error) {
	key, dsn, err := urlParts(url)
	if err != nil {
		return nil, fmt.Errorf("failed to init atlas driver, %s", err)
	}
	p, ok := u.providers[key]
	if !ok {
		return nil, fmt.Errorf("could not find provider: %s", key)
	}
	if key == "sqlite" {
		if err := sqliteFileExists(dsn); err != nil {
			return nil, err
		}
	}
	return p(dsn)
}

func urlParts(url string) (string, string, error) {
	a := strings.SplitN(url, "://", 2)
	if len(a) != 2 {
		return "", "", fmt.Errorf(`failed to parse url: "%s"`, url)
	}
	return a[0], a[1], nil
}

// SchemaNameFromURL parses the url the returns schema name
func SchemaNameFromURL(url string) (string, error) {
	key, dsn, err := urlParts(url)
	if err != nil {
		return "", err
	}
	switch key {
	case "mysql", "maria", "mariadb":
		cfg, err := mysql.ParseDSN(dsn)
		if err != nil {
			return "", err
		}
		return cfg.DBName, err
	case "postgres":
		return postgresSchema(dsn)
	case "sqlite":
		return schemaName(dsn)
	default:
		return "", fmt.Errorf("unknown database type: %q", key)
	}
}

func postgresSchema(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	// lib/pq supports setting default schemas via the `search_path` parameter
	// in a url.
	//
	// See: https://github.com/lib/pq/blob/8446d16b8935fdf2b5c0fe333538ac395e3e1e4b/conn.go#L1155-L1165
	if schema := u.Query().Get("search_path"); schema != "" {
		return schema, nil
	}
	return "", nil
}

func schemaName(dsn string) (string, error) {
	err := sqliteFileExists(dsn)
	if err != nil {
		return "", err
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return "", err
	}
	drv, err := sqlite.Open(db)
	if err != nil {
		return "", err
	}
	r, err := drv.InspectRealm(context.Background(), nil)
	if err != nil {
		return "", err
	}
	if len(r.Schemas) != 1 {
		return "", fmt.Errorf("must have exactly 1 schema, got: %d", len(r.Schemas))
	}
	return r.Schemas[0].Name, nil
}

func sqliteFileExists(dsn string) error {
	if !reMemMode.MatchString(dsn) {
		return fileExists(dsn)
	}
	return nil
}

func fileExists(dsn string) error {
	s := strings.Split(dsn, "?")
	f := dsn
	if len(s) == 2 {
		f = s[0]
	}
	if strings.Contains(f, "file:") {
		f = strings.SplitAfter(f, "file:")[1]
	}
	f = filepath.Clean(f)
	if _, err := os.Stat(f); err != nil {
		return fmt.Errorf("failed opening %q: %w", f, err)
	}
	return nil
}

func mysqlDSN(d string) (string, error) {
	cfg, err := mysql.ParseDSN(d)
	// A standard MySQL DSN.
	if err == nil {
		return d, nil
	}
	u, err := url.Parse("mysql://" + d)
	if err != nil {
		return "", nil
	}
	schema := strings.TrimPrefix(u.Path, "/")
	// In case of a URL (non-standard DSN),
	// parse the options from query string.
	if u.RawQuery != "" {
		cfg, err = mysql.ParseDSN(fmt.Sprintf("/%s?%s", schema, u.RawQuery))
		if err != nil {
			return "", err
		}
	} else {
		cfg = mysql.NewConfig()
	}
	cfg.Net = "tcp"
	cfg.Addr = u.Host
	cfg.User = u.User.Username()
	cfg.Passwd, _ = u.User.Password()
	cfg.DBName = schema
	return cfg.FormatDSN(), nil
}
