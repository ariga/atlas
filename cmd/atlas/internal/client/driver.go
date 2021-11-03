package client

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
	// Driver provides to diff, execute and inspect against a Data Source.
	Driver interface {
		schema.Differ
		schema.Execer
		schema.Inspector
	}
	// AtlasDriver implements the Driver interface using Atlas.
	AtlasDriver struct {
		schema.Differ
		schema.Execer
		schema.Inspector
	}

	dbName string
)

const (
	mysqlDB    dbName = "mysql"
	postgresDB dbName = "postgres"
	sqliteDB   dbName = "sqlite3"
)

var providers = map[dbName]func(string) (*AtlasDriver, func(), error){
	mysqlDB:    atlasDriverMysql,
	postgresDB: atlasDriverPostgres,
	sqliteDB:   atlasDriverSqlite,
}

// NewAtlasDriver connects a new Atlas Driver returns AtlasDriver and a closer.
func NewAtlasDriver(dsn string) (*AtlasDriver, func(), error) {
	a := strings.Split(dsn, "://")
	if len(a) != 2 {
		return nil, nil, fmt.Errorf("failed to parse %s", dsn)
	}
	p := providers[dbName(a[0])]
	if p == nil {
		return nil, nil, fmt.Errorf("failed to parse %s", dsn)
	}
	return p(a[1])
}

func atlasDriverMysql(dsn string) (*AtlasDriver, func(), error) {
	db, err := sql.Open("mysql", dsn)
	closer := func() {
		_ = db.Close()
	}
	if err != nil {
		return nil, nil, err
	}
	drv, err := mysql.Open(db)
	if err != nil {
		return nil, nil, err
	}
	return &AtlasDriver{
		drv.Diff(),
		drv.Migrate(),
		drv,
	}, closer, nil
}
func atlasDriverPostgres(dsn string) (*AtlasDriver, func(), error) {
	db, err := sql.Open("postgres", dsn)
	closer := func() {
		_ = db.Close()
	}
	if err != nil {
		return nil, nil, err
	}
	drv, err := postgres.Open(db)
	if err != nil {
		return nil, nil, err
	}
	return &AtlasDriver{
		drv.Diff(),
		drv.Migrate(),
		drv,
	}, closer, nil
}
func atlasDriverSqlite(dsn string) (*AtlasDriver, func(), error) {
	db, err := sql.Open("sqlite3", dsn)
	closer := func() {
		_ = db.Close()
	}
	if err != nil {
		return nil, nil, err
	}
	drv, err := sqlite.Open(db)
	if err != nil {
		return nil, nil, err
	}
	return &AtlasDriver{
		drv.Diff(),
		nil,
		drv,
	}, closer, nil
}
