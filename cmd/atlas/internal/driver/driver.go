package driver

import (
	"database/sql"
	"fmt"
	"strings"

	"ariga.io/atlas/sql/schema"
)

type (
	// Atlas implements the Driver interface using Atlas.
	Atlas struct {
		DB        *sql.DB
		Differ    schema.Differ
		Execer    schema.Execer
		Inspector schema.Inspector
	}
)

func (a *Atlas) Close() error {
	return a.DB.Close()
}

type providerMap struct {
	m map[string]func(string) (*Atlas, error)
}

type URLMux struct {
	providers *providerMap
}

func NewURLMux() *URLMux {
	return &URLMux{
		providers: &providerMap{
			m: make(map[string]func(string) (*Atlas, error)),
		},
	}
}

var defaultURLMux *URLMux

func DefaultURLMux() *URLMux {
	if defaultURLMux == nil {
		defaultURLMux = NewURLMux()
	}
	return defaultURLMux
}

func (m *providerMap) register(key string, p func(string) (*Atlas, error)) {
	if _, ok := m.m[key]; ok {
		panic("provider is already initialized")
	}
	m.m[key] = p
}

func (u *URLMux) RegisterProvider(key string, p func(string) (*Atlas, error)) {
	u.providers.register(key, p)
}

// NewAtlas connects a new Atlas Driver returns Atlas and a closer.
func NewAtlas(dsn string) (*Atlas, error) {
	key, dsn, err := parseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to init atlas driver, %s", err)
	}
	p, ok := DefaultURLMux().providers.m[key]
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
