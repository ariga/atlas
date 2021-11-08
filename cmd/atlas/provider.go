package main

import (
	"database/sql"

	"ariga.io/atlas/cmd/atlas/internal/mux"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
)

func init() {
	d := mux.DefaultMux()
	d.RegisterProvider("mysql", mysqlProvider)
	d.RegisterProvider("postgres", postgresProvider)
}

func mysqlProvider(dsn string) (*mux.Driver, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	drv, err := mysql.Open(db)
	if err != nil {
		return nil, err
	}
	return &mux.Driver{
		DB:        db,
		Differ:    drv.Diff(),
		Execer:    drv.Migrate(),
		Inspector: drv,
	}, nil
}
func postgresProvider(dsn string) (*mux.Driver, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	drv, err := postgres.Open(db)
	if err != nil {
		return nil, err
	}
	return &mux.Driver{
		DB:        db,
		Differ:    drv.Diff(),
		Execer:    drv.Migrate(),
		Inspector: drv,
	}, nil
}
