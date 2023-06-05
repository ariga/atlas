// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

type mockDriver struct {
	driver.Driver
	opened []string
}

func (m *mockDriver) Open(name string) (driver.Conn, error) {
	db, _, err := sqlmock.New()
	if err != nil {
		return nil, err
	}
	m.opened = append(m.opened, name)
	return db.Driver().Open(name)
}

func TestParser_ParseURL(t *testing.T) {
	drv := &mockDriver{}
	sql.Register("libsql", drv)
	_, err := sqlclient.Open(context.Background(), "libsql+wss://example.com/db.sqlite3?_fk=1")
	require.Error(t, err, "did not mock queries")
	require.Equal(t, []string{"wss://example.com/db.sqlite3?_fk=1"}, drv.opened)
}

func TestDriver_LockAcquired(t *testing.T) {
	drv := &Driver{}

	// Acquiring a lock does work.
	unlock, err := drv.Lock(context.Background(), "lock", time.Second)
	require.NoError(t, err)
	require.NotNil(t, unlock)

	// Acquiring a lock on the same value will fail.
	_, err = drv.Lock(context.Background(), "lock", time.Second)
	require.Error(t, err)

	// After unlock it will succeed again.
	require.NoError(t, unlock())
	_, err = drv.Lock(context.Background(), "lock", time.Second)
	require.NoError(t, err)
	require.NotNil(t, unlock)

	// Acquiring a lock on a value that has been expired works.
	dir, err := os.UserCacheDir()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "lock.lock"),
		[]byte(strconv.FormatInt(time.Now().Add(-time.Second).UnixNano(), 10)),
		0666,
	))
	_, err = drv.Lock(context.Background(), "lock", time.Second)

	// Acquiring a lock on another value works as well.
	_, err = drv.Lock(context.Background(), "another", time.Second)
}

func TestDriver_CheckClean(t *testing.T) {
	var (
		r   = schema.NewRealm()
		drv = &Driver{Inspector: &mockInspector{realm: r}}
	)
	// Empty realm.
	err := drv.CheckClean(context.Background(), nil)
	require.NoError(t, err)
	// Empty schema.
	r.AddSchemas(schema.New("main"))
	err = drv.CheckClean(context.Background(), nil)
	require.NoError(t, err)
	// Schema with revisions table only.
	r.Schemas[0].AddTables(schema.NewTable("revisions"))
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Name: "revisions"})
	require.NoError(t, err)
	// Unknown table.
	r.Schemas[0].Tables[0].Name = "unknown"
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Name: "revisions"})
	require.EqualError(t, err, `sql/migrate: connected database is not clean: found table "unknown"`)
	// Multiple tables.
	r.Schemas[0].Tables = []*schema.Table{schema.NewTable("a"), schema.NewTable("revisions")}
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Schema: "test", Name: "revisions"})
	require.EqualError(t, err, `sql/migrate: connected database is not clean: found multiple tables: 2`)
}

func TestDriver_Version(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mock{m}.systemVars("3.36.0")
	drv, err := Open(db)
	require.NoError(t, err)

	type vr interface{ Version() string }
	require.Implements(t, (*vr)(nil), drv)
	require.Equal(t, "3.36.0", drv.(vr).Version())
}

type mockInspector struct {
	schema.Inspector
	realm *schema.Realm
}

func (m *mockInspector) InspectRealm(context.Context, *schema.InspectRealmOption) (*schema.Realm, error) {
	return m.realm, nil
}
