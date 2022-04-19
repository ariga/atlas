// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"time"

	"ariga.io/atlas/internal/docker"

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
	DefaultMux.RegisterProvider("docker", dockerProvider)
}

func mysqlProvider(_ context.Context, dsn string, _ ...ProviderOption) (*Driver, error) {
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
		Closer:      db,
	}, nil
}

func postgresProvider(_ context.Context, dsn string, _ ...ProviderOption) (*Driver, error) {
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
		Closer:      db,
	}, nil
}

// TODO: temp, remove
func SqliteProvider(ctx context.Context, dsn string, _ ...ProviderOption) (*Driver, error) {
	return sqliteProvider(ctx, dsn)
}

func sqliteProvider(_ context.Context, dsn string, _ ...ProviderOption) (*Driver, error) {
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
		Closer:      db,
	}, nil
}

var reDockerConfig = regexp.MustCompile("(mysql|mariadb|tidb|postgres)(?::([0-9a-zA-Z.-]+)?(\\?.*)?)?")

type dockerCloser struct {
	c   *docker.Container
	drv *Driver
}

func (dc *dockerCloser) Close() (err error) {
	derr, cerr := dc.drv.Close(), dc.c.Down()
	if derr != nil {
		err = derr
	}
	if cerr != nil {
		err = fmt.Errorf("%w: %v", derr, cerr)
	}
	return
}

func dockerProvider(ctx context.Context, dsn string, opts ...ProviderOption) (*Driver, error) {
	// The DSN has the driver part (docker:// removed already.
	// Get rid of the query arguments, and we have the image name.
	m := reDockerConfig.FindStringSubmatch(dsn)
	if len(m) != 4 {
		return nil, fmt.Errorf("invalid docker url %q", dsn)
	}
	img, v := m[1], m[2]
	var (
		cfg *docker.Config
		err error
	)
	switch img {
	case "mysql":
		cfg, err = docker.MySQL(v)
	case "mariadb":
		cfg, err = docker.MariaDB(v)
	case "postgres":
		cfg, err = docker.PostgreSQL(v)
	default:
		return nil, fmt.Errorf("unsupported docker image %q", img)
	}
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		switch opt.(type) {
		case VerboseLogging, *VerboseLogging:
			err = docker.Out(os.Stdout)(cfg)
		}
		if err != nil {
			return nil, err
		}
	}
	c, err := cfg.Run(ctx)
	if err != nil {
		return nil, err
	}
	if err := c.Wait(ctx, time.Minute); err != nil {
		_ = c.Down()
		return nil, err
	}
	u, err := c.URL()
	if err != nil {
		_ = c.Down()
		return nil, err
	}
	drv, err := DefaultMux.OpenAtlas(ctx, u)
	if err != nil {
		_ = c.Down()
		return nil, err
	}
	return &Driver{
		Driver:      drv.Driver,
		Marshaler:   drv.Marshaler,
		Unmarshaler: drv.Unmarshaler,
		Closer:      &dockerCloser{c, drv},
	}, nil
}
