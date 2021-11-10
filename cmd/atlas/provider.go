package main

import (
	"database/sql"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
)

func init() {
	defaultMux.RegisterProvider("mysql", mysqlProvider)
	defaultMux.RegisterProvider("postgres", postgresProvider)
}

func mysqlProvider(dsn string) (*Driver, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	drv, err := mysql.Open(db)
	if err != nil {
		return nil, err
	}
	return &Driver{
		Differ:        drv.Diff(),
		Execer:        drv.Migrate(),
		Inspector:     drv,
		MarshalSpec:   mysql.MarshalSpec,
		UnmarshalSpec: mysql.UnmarshalSpec,
	}, nil
}
func postgresProvider(dsn string) (*Driver, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	drv, err := postgres.Open(db)
	if err != nil {
		return nil, err
	}
	return &Driver{
		Differ:        drv.Diff(),
		Execer:        drv.Migrate(),
		Inspector:     drv,
		MarshalSpec:   postgres.MarshalSpec,
		UnmarshalSpec: postgres.UnmarshalSpec,
	}, nil
}
