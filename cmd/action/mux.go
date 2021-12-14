package action

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/schema"
	"github.com/go-sql-driver/mysql"
)

type (
	// Mux is used for routing dsn to correct provider.
	Mux struct {
		providers map[string]func(string) (*Driver, error)
	}

	// Driver implements the Atlas interface.
	Driver struct {
		driver
		MarshalSpec   func(v interface{}, marshaler schemaspec.Marshaler) ([]byte, error)
		UnmarshalSpec func(data []byte, unmarshaler schemaspec.Unmarshaler, v interface{}) error
		interceptor   *interceptor
	}

	// A schema driver.
	driver interface {
		schema.Differ
		schema.Execer
		schema.Inspector
	}

	schemaUnmarshal struct {
		unmarshalSpec func(data []byte, unmarshaler schemaspec.Unmarshaler, v interface{}) error
		unmarshaler   schemaspec.Unmarshaler
	}

	schemaUnmarshaler interface {
		unmarshal([]byte, interface{}) error
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
	case "mysql":
		cfg, err := mysql.ParseDSN(dsn)
		if err != nil {
			return "", err
		}
		return cfg.DBName, err
	case "postgres":
		return "public", nil
	default:
		return "", fmt.Errorf("unknown database type: %q", key)
	}
}

func (p *schemaMarshal) marshal(s *schema.Schema) ([]byte, error) {
	return p.marshalSpec(s, p.marshaler)
}

func (p *schemaUnmarshal) unmarshal(b []byte, v interface{}) error {
	return p.unmarshalSpec(b, p.unmarshaler, v)
}

// interceptor is an ExecQuerier that when activated can intercept and store queries initiated by Exec
// invocations. interceptor is used for capturing the planned SQL statement that a migration is going
// to perform.
type interceptor struct {
	schema.ExecQuerier
	intercept bool
	history   []string
}

func (i *interceptor) on() {
	i.intercept = true
}

func (i *interceptor) off() {
	i.intercept = false
}

func (i *interceptor) clear() {
	i.history = nil
}

func (i *interceptor) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if i.intercept {
		i.history = append(i.history, query)
		return nil, nil
	}
	return i.ExecQuerier.ExecContext(ctx, query, args...)
}
