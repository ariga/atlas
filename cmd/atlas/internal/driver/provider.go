package driver

import (
	"database/sql"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/sqlite"
)

func init() {
	d := DefaultURLMux()
	d.Providers.Register("mysql", mysqlProvider)
	d.Providers.Register("postgres", postgresProvider)
	d.Providers.Register("sqlite3", sqliteProvider)
}

func mysqlProvider(dsn string) (*Atlas, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	drv, err := mysql.Open(db)
	if err != nil {
		return nil, err
	}
	return &Atlas{
		DB:        db,
		Differ:    drv.Diff(),
		Execer:    drv.Migrate(),
		Inspector: drv,
	}, nil
}
func postgresProvider(dsn string) (*Atlas, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	drv, err := postgres.Open(db)
	if err != nil {
		return nil, err
	}
	return &Atlas{
		DB:        db,
		Differ:    drv.Diff(),
		Execer:    drv.Migrate(),
		Inspector: drv,
	}, nil
}
func sqliteProvider(dsn string) (*Atlas, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	drv, err := sqlite.Open(db)
	if err != nil {
		return nil, err
	}
	return &Atlas{
		DB:        db,
		Differ:    drv.Diff(),
		Execer:    nil,
		Inspector: drv,
	}, nil
}
