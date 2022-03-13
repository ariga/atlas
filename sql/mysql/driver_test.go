// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"ariga.io/atlas/sql/internal/sqltest"
	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

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
		d := &Driver{}
		d.ExecQuerier = &mockOpener{DB: db}
		lock(d)
		require.EqualValues(t, 1, d.ExecQuerier.(*mockOpener).opened)
	})

	t.Run("OnConn", func(t *testing.T) {
		conn, err := db.Conn(context.Background())
		require.NoError(t, err)
		d := &Driver{}
		d.ExecQuerier = conn
		lock(d)
	})

	t.Run("OnTx", func(t *testing.T) {
		m.ExpectBegin()
		tx, err := db.Begin()
		require.NoError(t, err)
		d := &Driver{}
		d.ExecQuerier = tx
		lock(d)
	})
	require.NoError(t, m.ExpectationsWereMet())
}

func TestDriver_LockError(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	d := &Driver{}
	d.ExecQuerier = db

	t.Run("Timeout", func(t *testing.T) {
		name, sec := "name", 1
		m.ExpectQuery(sqltest.Escape("SELECT GET_LOCK(?, ?)")).
			WithArgs(name, sec).
			WillReturnRows(sqlmock.NewRows([]string{"acquired"}).AddRow(0)).
			RowsWillBeClosed()
		unlock, err := d.Lock(context.Background(), name, time.Second)
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
	d := &Driver{}
	d.ExecQuerier = db
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

type mockOpener struct {
	*sql.DB
	opened uint
}

func (m *mockOpener) Conn(ctx context.Context) (*sql.Conn, error) {
	m.opened++
	return m.DB.Conn(ctx)
}
