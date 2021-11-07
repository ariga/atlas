package driver

import (
	"database/sql"
	"fmt"
	"strings"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"
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

func atlasDriverMysql(dsn string) (*Atlas, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	drv, err := mysql.Open(db)
	if err != nil {
		return nil, err
	}
	return &Atlas{
		db:        db,
		Differ:    drv.Diff(),
		Execer:    drv.Migrate(),
		Inspector: drv,
	}, nil
}
func atlasDriverPostgres(dsn string) (*Atlas, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	drv, err := postgres.Open(db)
	if err != nil {
		return nil, err
	}
	return &Atlas{
		db:        db,
		Differ:    drv.Diff(),
		Execer:    drv.Migrate(),
		Inspector: drv,
	}, nil
}
func atlasDriverSqlite(dsn string) (*Atlas, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	drv, err := sqlite.Open(db)
	if err != nil {
		return nil, err
	}
	return &Atlas{
		db:        db,
		Differ:    drv.Diff(),
		Execer:    nil,
		Inspector: drv,
	}, nil
}
