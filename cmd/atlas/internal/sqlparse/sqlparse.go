// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlparse

import (
	"sync"

	"ariga.io/atlas/cmd/atlas/internal/sqlparse/myparse"
	"ariga.io/atlas/cmd/atlas/internal/sqlparse/pgparse"
	"ariga.io/atlas/cmd/atlas/internal/sqlparse/sqliteparse"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"
)

// A ChangesFixer wraps the FixChange method.
type ChangesFixer interface {
	// FixChange fixes the given changes according to the given statement.
	FixChange(d migrate.Driver, stmt string, changes schema.Changes) (schema.Changes, error)
}

// FixerFunc allows using ordinary functions as change fixers.
type FixerFunc func(migrate.Driver, string, schema.Changes) (schema.Changes, error)

// FixChange calls f.
func (f FixerFunc) FixChange(d migrate.Driver, stmt string, changes schema.Changes) (schema.Changes, error) {
	return f(d, stmt, changes)
}

// drivers specific fixers.
var drivers sync.Map

// Register a fixer with the given name.
func Register(name string, f ChangesFixer) {
	drivers.Store(name, f)
}

// FixerFor returns a ChangesFixer for the given driver.
func FixerFor(name string) ChangesFixer {
	f, ok := drivers.Load(name)
	if ok {
		return f.(ChangesFixer)
	}
	// A nop analyzer.
	return FixerFunc(func(_ migrate.Driver, _ string, changes schema.Changes) (schema.Changes, error) {
		return changes, nil
	})
}

func init() {
	Register(mysql.DriverName, FixerFunc(myparse.FixChange))
	Register(postgres.DriverName, FixerFunc(pgparse.FixChange))
	Register(sqlite.DriverName, FixerFunc(sqliteparse.FixChange))
}
