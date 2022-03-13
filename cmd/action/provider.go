// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"database/sql"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/sqlite"
)

func init() {
	DefaultMux.RegisterProvider("mysql", mysqlProvider)
	DefaultMux.RegisterProvider("maria", mysqlProvider)
	DefaultMux.RegisterProvider("mariadb", mysqlProvider)
	DefaultMux.RegisterProvider("postgres", postgresProvider)
	DefaultMux.RegisterProvider("sqlite", sqliteProvider)
}

func mysqlProvider(dsn string) (*Driver, error) {
	d, err := mysqlDSN(dsn)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", d)
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
	u := "postgres://" + dsn
	db, err := sql.Open("postgres", u)
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
