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
	i := &interceptor{ExecQuerier: db}
	drv, err := mysql.Open(i)
	if err != nil {
		return nil, err
	}
	return &Driver{
		Driver:        drv,
		interceptor:   i,
		MarshalSpec:   mysql.MarshalSpec,
		UnmarshalSpec: mysql.UnmarshalSpec,
	}, nil
}
func postgresProvider(dsn string) (*Driver, error) {
	url := "postgres://" + dsn
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	i := &interceptor{ExecQuerier: db}
	drv, err := postgres.Open(i)
	if err != nil {
		return nil, err
	}
	return &Driver{
		Driver:        drv,
		interceptor:   i,
		MarshalSpec:   postgres.MarshalSpec,
		UnmarshalSpec: postgres.UnmarshalSpec,
	}, nil
}
