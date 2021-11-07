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
		db        *sql.DB
		Differ    schema.Differ
		Execer    schema.Execer
		Inspector schema.Inspector
	}
	dbName string
)

func (a *Atlas) Close() error {
	return a.db.Close()
}

var providers = map[string]func(string) (*Atlas, error){}

func Register(key string, p func(string) (*Atlas, error)) {
	if _, ok := providers[key]; ok {
		panic("provider is already initialized")
	}
	providers[key] = p
}

// NewAtlas connects a new Atlas Driver returns Atlas and a closer.
func NewAtlas(dsn string) (*Atlas, error) {
	key, dsn, err := parseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to init atlas driver, %s", err)
	}
	p, ok := providers[key]
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