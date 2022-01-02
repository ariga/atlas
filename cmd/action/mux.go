package action

import (
	"context"
	"database/sql"
	"fmt"
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

var defaultMux = NewMux()

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
	return p(dsn)
}

func parseDSN(url string) (string, string, error) {
	a := strings.Split(url, "://")
	if len(a) != 2 {
		return "", "", fmt.Errorf("failed to parse dsn")
	}
	return a[0], a[1], nil
}

func schemaNameFromDSN(url string) (string, error) {
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
		return "public", nil
	case "sqlite3":
		return schemaName(dsn)
	default:
		return "", fmt.Errorf("unknown database type: %q", key)
	}
}

func schemaName(dsn string) (string, error) {
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
	if len(r.Schemas) > 1 {
		return "", fmt.Errorf("number of schemas > 1, n=%d", len(r.Schemas))
	}
	return r.Schemas[0].Name, nil
}
