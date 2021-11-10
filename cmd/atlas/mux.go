package main

import (
	"database/sql"
	"fmt"
	"strings"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/schema"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgconn"
)

type (
	// Mux is used for routing dsn to correct provider.
	Mux struct {
		providers map[string]func(string) (*Driver, error)
	}

	// Driver implements the Atlas interface.
	Driver struct {
		*sql.DB
		schema.Differ
		schema.Execer
		schema.Inspector
		MarshalSpec func(v interface{}, marshaler schemaspec.Marshaler) ([]byte, error)
	}

	schemaMarshal struct {
		marshalSpec func(v interface{}, marshaler schemaspec.Marshaler) ([]byte, error)
		marshaler   schemaspec.Marshaler
	}

	schemaMarshaler interface {
		marshal(*schema.Schema) ([]byte, error)
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
		return nil, fmt.Errorf("could not find provider, %s", err)
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
	case "mysql":
		cfg, err := mysql.ParseDSN(dsn)
		if err != nil {
			return "", err
		}
		return cfg.DBName, err
	case "postgres":
		cfg, err := pgconn.ParseConfig(dsn)
		if err != nil {
			return "", err
		}
		return cfg.Database, err
	default:
		return "", fmt.Errorf("unknown database type: %q", key)
	}
}

func (p *schemaMarshal) marshal(s *schema.Schema) ([]byte, error) {
	return p.marshalSpec(s, p.marshaler)
}
