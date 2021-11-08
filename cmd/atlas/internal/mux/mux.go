package mux

import (
	"database/sql"
	"fmt"
	"strings"

	"ariga.io/atlas/sql/schema"
)

type (
	// Mux is used for routing dsn to correct provider.
	Mux struct {
		providers map[string]func(string) (*Driver, error)
	}

	// Driver implements the Atlas interface.
	Driver struct {
		*sql.DB
		Differ    schema.Differ
		Execer    schema.Execer
		Inspector schema.Inspector
	}
)

// Close release all resources taken by Driver.
func (a *Driver) Close() error {
	return a.DB.Close()
}

// NewMux returns a new Mux.
func NewMux() *Mux {
	return &Mux{
		providers: make(map[string]func(string) (*Driver, error)),
	}
}

var defaultURLMux = NewMux()

// DefaultMux returns the default system "Mux".
func DefaultMux() *Mux {
	return defaultURLMux
}

// RegisterProvider is used to register an Driver provider by key..
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
		return "", "nil", fmt.Errorf("failed to parse dsn")
	}
	return a[0], a[1], nil
}
