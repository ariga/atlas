package action

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
		Driver:        drv,
		MarshalSpec:   mysql.MarshalSpec,
		UnmarshalSpec: mysql.UnmarshalSpec,
		Types:         mysql.TypeRegistry.Specs(),
	}, nil
}
func postgresProvider(dsn string) (*Driver, error) {
	url := "postgres://" + dsn
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	drv, err := postgres.Open(db)
	if err != nil {
		return nil, err
	}
	return &Driver{
		Driver:        drv,
		MarshalSpec:   postgres.MarshalSpec,
		UnmarshalSpec: postgres.UnmarshalSpec,
		Types:         postgres.TypeRegistry.Specs(),
	}, nil
}
