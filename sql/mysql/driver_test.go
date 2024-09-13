// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"context"
	"database/sql"
	"net/url"
	"testing"
	"time"

	"ariga.io/atlas/sql/internal/sqltest"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestParser_ParseURL(t *testing.T) {
	t.Run("ParseTime", func(t *testing.T) {
		u, err := url.Parse("mysql://user:pass@localhost:3306/my_db?foo=bar")
		require.NoError(t, err)
		ac := parser{}.ParseURL(u)
		require.Equal(t, "true", ac.Query().Get("parseTime"))
	})
	t.Run("UnixDSN", func(t *testing.T) {
		for u, d := range map[string]string{
			"mysql+unix:///path/to/socket":                                    "unix(/path/to/socket)/?parseTime=true",
			"mysql+unix://user:pass@/path/to/socket":                          "user:pass@unix(/path/to/socket)/?parseTime=true",
			"mysql+unix://user@/path/to/socket?database=dbname":               "user@unix(/path/to/socket)/dbname?parseTime=true",
			"maria+unix:///path/to/file.socket?database=test&tls=skip-verify": "unix(/path/to/file.socket)/test?parseTime=true&tls=skip-verify",
		} {
			u1, err := url.Parse(u)
			require.NoError(t, err)
			p := parser{}.ParseURL(u1)
			require.Equal(t, d, p.DSN)
		}
	})
}

func TestDriver_LockAcquired(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	lock := func(l schema.Locker) {
		name, sec := "name", 1
		m.ExpectQuery(sqltest.Escape("SELECT GET_LOCK(?, ?)")).
			WithArgs(name, sec).
			WillReturnRows(sqlmock.NewRows([]string{"acquired"}).AddRow(1)).
			RowsWillBeClosed()
		m.ExpectQuery(sqltest.Escape("SELECT RELEASE_LOCK(?)")).
			WithArgs(name).
			WillReturnRows(sqlmock.NewRows([]string{"released"}).AddRow(1)).
			RowsWillBeClosed()
		unlock, err := l.Lock(context.Background(), name, time.Second)
		require.NoError(t, err)
		require.NoError(t, unlock())
	}

	t.Run("OnPool", func(t *testing.T) {
		d := &Driver{conn: &conn{}}
		d.ExecQuerier = &mockOpener{DB: db}
		lock(d)
		require.EqualValues(t, 1, d.ExecQuerier.(*mockOpener).opened)
	})

	t.Run("OnConn", func(t *testing.T) {
		d := &Driver{conn: &conn{}}
		conn, err := db.Conn(context.Background())
		require.NoError(t, err)
		d.ExecQuerier = conn
		lock(d)
	})

	t.Run("OnTx", func(t *testing.T) {
		m.ExpectBegin()
		tx, err := db.Begin()
		require.NoError(t, err)
		d := &Driver{conn: &conn{}}
		d.ExecQuerier = tx
		lock(d)
	})
	require.NoError(t, m.ExpectationsWereMet())
}

func TestDriver_LockError(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	d := &Driver{conn: &conn{}}
	d.ExecQuerier = db

	t.Run("Timeout", func(t *testing.T) {
		name, sec := "name", 60
		m.ExpectQuery(sqltest.Escape("SELECT GET_LOCK(?, ?)")).
			WithArgs(name, sec).
			WillReturnRows(sqlmock.NewRows([]string{"acquired"}).AddRow(0)).
			RowsWillBeClosed()
		unlock, err := d.Lock(context.Background(), name, time.Minute)
		require.Equal(t, schema.ErrLocked, err)
		require.Nil(t, unlock)
	})

	t.Run("Internal", func(t *testing.T) {
		name, sec := "migrate", -1
		m.ExpectQuery(sqltest.Escape("SELECT GET_LOCK(?, ?)")).
			WithArgs(name, sec).
			WillReturnRows(sqlmock.NewRows([]string{"acquired"}).AddRow(nil)).
			RowsWillBeClosed()
		unlock, err := d.Lock(context.Background(), name, -time.Second)
		require.EqualError(t, err, `sql/mysql: unexpected internal error on Lock("migrate", -1s)`)
		require.Nil(t, unlock)
	})
}

func TestDriver_UnlockError(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	d := &Driver{conn: &conn{ExecQuerier: db}}
	acquired := func(name string, sec int) {
		m.ExpectQuery(sqltest.Escape("SELECT GET_LOCK(?, ?)")).
			WithArgs(name, sec).
			WillReturnRows(sqlmock.NewRows([]string{"acquired"}).AddRow(1)).
			RowsWillBeClosed()
	}

	t.Run("NotHeld", func(t *testing.T) {
		name, sec := "unknown_lock", 0
		acquired(name, sec)
		unlock, err := d.Lock(context.Background(), name, time.Millisecond)
		require.NoError(t, err)
		m.ExpectQuery(sqltest.Escape("SELECT RELEASE_LOCK(?)")).
			WithArgs(name).
			WillReturnRows(sqlmock.NewRows([]string{"released"}).AddRow(0)).
			RowsWillBeClosed()
		require.Error(t, unlock())
	})

	t.Run("Internal", func(t *testing.T) {
		name, sec := "unknown_error", 1
		acquired(name, sec)
		unlock, err := d.Lock(context.Background(), name, time.Second+time.Millisecond)
		require.NoError(t, err)
		m.ExpectQuery(sqltest.Escape("SELECT RELEASE_LOCK(?)")).
			WithArgs(name).
			WillReturnRows(sqlmock.NewRows([]string{"released"}).AddRow(nil)).
			RowsWillBeClosed()
		require.Error(t, unlock())
	})
}

func TestDriver_CheckClean(t *testing.T) {
	s := schema.New("test")
	drv := &Driver{Inspector: &mockInspector{schema: s}}
	// Empty schema.
	err := drv.CheckClean(context.Background(), nil)
	require.NoError(t, err)
	// Revisions table found.
	s.AddTables(schema.NewTable("revisions"))
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Name: "revisions", Schema: "test"})
	require.NoError(t, err)
	// Multiple tables.
	s.Tables = []*schema.Table{schema.NewTable("a"), schema.NewTable("revisions")}
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Name: "revisions", Schema: "test"})
	require.EqualError(t, err, `sql/migrate: connected database is not clean: found table "a" in schema "test"`)

	r := schema.NewRealm()
	drv.Inspector = &mockInspector{realm: r}
	// Empty realm.
	err = drv.CheckClean(context.Background(), nil)
	require.NoError(t, err)
	// Revisions table found.
	s.Tables = []*schema.Table{schema.NewTable("revisions")}
	r.AddSchemas(s)
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Name: "revisions", Schema: "test"})
	require.NoError(t, err)
	// Unknown table.
	s.Tables[0].Name = "unknown"
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Schema: "test", Name: "revisions"})
	require.EqualError(t, err, `sql/migrate: connected database is not clean: found table "unknown"`)
	// Multiple tables.
	s.Tables = []*schema.Table{schema.NewTable("a"), schema.NewTable("revisions")}
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Schema: "test", Name: "revisions"})
	require.EqualError(t, err, `sql/migrate: connected database is not clean: found multiple tables: 2`)
}

func TestDriver_Version(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mock{m}.version("8.0.13")
	drv, err := Open(db)
	require.NoError(t, err)

	type vr interface{ Version() string }
	require.Implements(t, (*vr)(nil), drv)
	require.Equal(t, "8.0.13", drv.(vr).Version())
}

type mockInspector struct {
	schema.Inspector
	realm  *schema.Realm
	schema *schema.Schema
}

func (m *mockInspector) InspectSchema(context.Context, string, *schema.InspectOptions) (*schema.Schema, error) {
	if m.schema == nil {
		return nil, &schema.NotExistError{}
	}
	return m.schema, nil
}

func (m *mockInspector) InspectRealm(context.Context, *schema.InspectRealmOption) (*schema.Realm, error) {
	return m.realm, nil
}

type mockOpener struct {
	*sql.DB
	opened uint
}

func (m *mockOpener) Conn(ctx context.Context) (*sql.Conn, error) {
	m.opened++
	return m.DB.Conn(ctx)
}
