// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"io"
	"testing"
	"time"

	"ariga.io/atlas/sql/internal/sqltest"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestDriver_LockAcquired(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	name, hash := "name", 797654004
	m.ExpectQuery(sqltest.Escape("SELECT pg_try_advisory_lock($1)")).
		WithArgs(hash).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_lock"}).AddRow(1)).
		RowsWillBeClosed()
	m.ExpectQuery(sqltest.Escape("SELECT pg_advisory_unlock($1)")).
		WithArgs(hash).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(1)).
		RowsWillBeClosed()

	d := &Driver{}
	d.ExecQuerier = db
	unlock, err := d.Lock(context.Background(), name, 0)
	require.NoError(t, err)
	require.NoError(t, unlock())
	require.NoError(t, m.ExpectationsWereMet())
}

func TestDriver_LockError(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	d := &Driver{}
	d.ExecQuerier = db
	name, hash := "migrate", 979249972

	t.Run("Timeout", func(t *testing.T) {
		m.ExpectQuery(sqltest.Escape("SELECT pg_advisory_lock($1)")).
			WithArgs(hash).
			WillReturnError(context.DeadlineExceeded).
			RowsWillBeClosed()
		unlock, err := d.Lock(context.Background(), name, time.Minute)
		require.Equal(t, schema.ErrLocked, err)
		require.Nil(t, unlock)
	})

	t.Run("Internal", func(t *testing.T) {
		m.ExpectQuery(sqltest.Escape("SELECT pg_advisory_lock($1)")).
			WithArgs(hash).
			WillReturnError(io.EOF).
			RowsWillBeClosed()
		unlock, err := d.Lock(context.Background(), name, time.Minute)
		require.Equal(t, io.EOF, err)
		require.Nil(t, unlock)
	})
}

func TestDriver_UnlockError(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	d := &Driver{}
	d.ExecQuerier = db
	name, hash := "up", 1551306158
	acquired := func() {
		m.ExpectQuery(sqltest.Escape("SELECT pg_try_advisory_lock($1)")).
			WithArgs(hash).
			WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(1)).
			RowsWillBeClosed()
	}

	t.Run("NotHeld", func(t *testing.T) {
		acquired()
		unlock, err := d.Lock(context.Background(), name, 0)
		require.NoError(t, err)
		m.ExpectQuery(sqltest.Escape("SELECT pg_advisory_unlock($1)")).
			WithArgs(hash).
			WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(0)).
			RowsWillBeClosed()
		require.Error(t, unlock())
	})

	t.Run("Internal", func(t *testing.T) {
		acquired()
		unlock, err := d.Lock(context.Background(), name, 0)
		require.NoError(t, err)
		m.ExpectQuery(sqltest.Escape("SELECT pg_advisory_unlock($1)")).
			WithArgs(hash).
			WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(nil)).
			RowsWillBeClosed()
		require.Error(t, unlock())
	})
}

func TestDriver_CheckClean(t *testing.T) {
	s := schema.New("test")
	drv := &Driver{Inspector: &mockInspector{schema: s}, conn: conn{schema: "test"}}
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
	drv.schema = ""
	drv.Inspector = &mockInspector{realm: r}
	// Empty realm.
	err = drv.CheckClean(context.Background(), nil)
	require.NoError(t, err)
	// Revisions table found.
	s.Tables = []*schema.Table{schema.NewTable("revisions").SetSchema(s)}
	r.AddSchemas(s)
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Name: "revisions", Schema: "test"})
	require.NoError(t, err)
	// Unknown table.
	s.Tables[0].Name = "unknown"
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Schema: "test", Name: "revisions"})
	require.EqualError(t, err, `sql/migrate: connected database is not clean: found table "unknown" in schema "test"`)
	// Multiple tables.
	s.Tables = []*schema.Table{schema.NewTable("a"), schema.NewTable("revisions")}
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Schema: "test", Name: "revisions"})
	require.EqualError(t, err, `sql/migrate: connected database is not clean: found 2 tables in schema "test"`)
	// With auto created public schema.
	s.Tables = []*schema.Table{schema.NewTable("revisions")}
	r.AddSchemas(schema.New("public"))
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Schema: "test", Name: "revisions"})
	require.NoError(t, err)
}

func TestDriver_Version(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mock{m}.version("130000")
	drv, err := Open(db)
	require.NoError(t, err)

	type vr interface{ Version() string }
	require.Implements(t, (*vr)(nil), drv)
	require.Equal(t, "130000", drv.(vr).Version())
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
