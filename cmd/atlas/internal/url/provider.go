package url

import (
	"database/sql"
	"errors"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
)

func init() {
	d := DefaultURLMux()
	d.RegisterProvider("mysql", mysqlProvider)
	d.RegisterProvider("postgres", postgresProvider)
	d.RegisterProvider("sqlite3", sqliteProvider)
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
	return nil, errors.New("sqlite3 not supported")
}
