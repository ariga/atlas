package client

import (
	"database/sql"
	"fmt"
	"strings"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
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

	driverName string
)

const (
	MYSQL    driverName = "mysql"
	POSTGRES driverName = "postgres"
	SQLITE   driverName = "sqlite3"
)

var providers = map[driverName]func(string) (*AtlasDriver, func(), error){
	MYSQL:    openMysql,
	POSTGRES: openPostgres,
	SQLITE:   openSqlite,
}

// NewAtlasDriver connects a new Atlas Driver returns AtlasDriver and a closer.
func NewAtlasDriver(dsn string) (*AtlasDriver, func(), error) {
	a := strings.Split(dsn, "://")
	if len(a) != 2 {
		return nil, nil, fmt.Errorf("failed to parse %s", dsn)
	}
	p := providers[driverName(a[0])]
	if p == nil {
		return nil, nil, fmt.Errorf("failed to parse %s", dsn)
	}
	return p(a[1])
}

func openMysql(dsn string) (*AtlasDriver, func(), error) {
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
func openPostgres(dsn string) (*AtlasDriver, func(), error) {
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
func openSqlite(dsn string) (*AtlasDriver, func(), error) {
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
