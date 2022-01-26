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
	// Mux is used for routing dsn to correct provider.
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
	defaultMux = NewMux()
	inMemory   = regexp.MustCompile("^file:.*:memory:$|:memory:|^file:.*mode=memory.*")
)

// RegisterProvider is used to register a Driver provider by key.
func (u *Mux) RegisterProvider(key string, p func(string) (*Driver, error)) {
	if _, ok := u.providers[key]; ok {
		panic("provider is already initialized")
	}
	u.providers[key] = p
}

// OpenAtlas is used for opening an atlas driver on a specific data source.
func (u *Mux) OpenAtlas(dsn string) (*Driver, error) {
	key, dsn, err := parseDSN(dsn)
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

func parseDSN(url string) (string, string, error) {
	a := strings.SplitN(url, "://", 2)
	if len(a) != 2 {
		return "", "", fmt.Errorf(`failed to parse dsn: "%s"`, url)
	}
	return a[0], a[1], nil
}

// SchemaNameFromDSN parses the dsn the returns schema name
func SchemaNameFromDSN(url string) (string, error) {
	key, dsn, err := parseDSN(url)
	if err != nil {
		return "", err
	}
	switch key {
	case "mysql", "mariadb":
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
	url, err := url.Parse(dsn)
	if err != nil {
		// For backwards compatibility, we default to "public" when failing to
		// parse.
		return "public", nil
	}
	// lib/pq supports setting default schemas via the `search_path` parameter
	// in a dsn.
	//
	// See: https://github.com/lib/pq/blob/8446d16b8935fdf2b5c0fe333538ac395e3e1e4b/conn.go#L1155-L1165
	if schema := url.Query().Get("search_path"); schema != "" {
		return schema, nil
	}

	return "public", nil
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
	if !inMemory.MatchString(dsn) {
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
