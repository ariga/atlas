package action

import (
	"database/sql"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/sqlite"
)

func init() {
	defaultMux.RegisterProvider("mysql", mysqlProvider)
	defaultMux.RegisterProvider("mariadb", mysqlProvider)
	defaultMux.RegisterProvider("postgres", postgresProvider)
	defaultMux.RegisterProvider("sqlite3", sqliteProvider)
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
		Driver:      drv,
		Marshaler:   mysql.MarshalHCL,
		Unmarshaler: mysql.UnmarshalHCL,
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
		Driver:      drv,
		Marshaler:   postgres.MarshalHCL,
		Unmarshaler: postgres.UnmarshalHCL,
	}, nil
}
func sqliteProvider(dsn string) (*Driver, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	drv, err := sqlite.Open(db)
	if err != nil {
		return nil, err
	}
	return &Driver{
		Driver:      drv,
		Marshaler:   sqlite.MarshalHCL,
		Unmarshaler: sqlite.UnmarshalHCL,
	}, nil
}
