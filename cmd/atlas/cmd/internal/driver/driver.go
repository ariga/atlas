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

const (
	mysqlDB    dbName = "mysql"
	postgresDB dbName = "postgres"
	sqliteDB   dbName = "sqlite3"
)

var providers = map[dbName]func(string) (*Atlas, error){
	mysqlDB:    atlasDriverMysql,
	postgresDB: atlasDriverPostgres,
	sqliteDB:   atlasDriverSqlite,
}

// NewAtlasDriver connects a new Atlas Driver returns Atlas and a closer.
func NewAtlasDriver(dsn string) (*Atlas, error) {
	a := strings.Split(dsn, "://")
	if len(a) != 2 {
		return nil, fmt.Errorf("failed to parse %s", dsn)
	}
	p := providers[dbName(a[0])]
	if p == nil {
		return nil, fmt.Errorf("failed to parse %s", dsn)
	}
	return p(a[1])
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
