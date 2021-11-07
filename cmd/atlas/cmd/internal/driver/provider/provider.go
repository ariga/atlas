package provider

import (
	"database/sql"

	"ariga.io/atlas/cmd/atlas/cmd/internal/driver"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/sqlite"
)

func init() {
	driver.Register("mysql", mysqlP)
	driver.Register("postgres", postgresP)
	driver.Register("sqlite3", sqliteP)
}

func mysqlP(dsn string) (*driver.Atlas, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	drv, err := mysql.Open(db)
	if err != nil {
		return nil, err
	}
	return &driver.Atlas{
		DB:        db,
		Differ:    drv.Diff(),
		Execer:    drv.Migrate(),
		Inspector: drv,
	}, nil
}
func postgresP(dsn string) (*driver.Atlas, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	drv, err := postgres.Open(db)
	if err != nil {
		return nil, err
	}
	return &driver.Atlas{
		DB:        db,
		Differ:    drv.Diff(),
		Execer:    drv.Migrate(),
		Inspector: drv,
	}, nil
}
func sqliteP(dsn string) (*driver.Atlas, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	drv, err := sqlite.Open(db)
	if err != nil {
		return nil, err
	}
	return &driver.Atlas{
		DB:        db,
		Differ:    drv.Diff(),
		Execer:    nil,
		Inspector: drv,
	}, nil
}
