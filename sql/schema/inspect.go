// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schema

import (
	"context"
	"database/sql"
	"errors"
)

// A NotExistError wraps another error to retain its original text
// but makes it possible to the migrator to catch it.
type NotExistError struct {
	Err error
}

func (e NotExistError) Error() string { return e.Err.Error() }

// IsNotExistError reports if an error is a NotExistError.
func IsNotExistError(err error) bool {
	if err == nil {
		return false
	}
	var e *NotExistError
	return errors.As(err, &e)
}

// ExecQuerier wraps the standard sql.DB methods.
type ExecQuerier interface {
	QueryRow(query string, args ...interface{}) *sql.Row
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

type (
	// InspectOptions describes options for Inspector.
	InspectOptions struct {
		// Tables to inspect. Empty means all tables in the schema.
		Tables []string
	}

	// Inspector is the interface implemented by the different database
	// drivers for inspecting multiple tables.
	Inspector interface {
		// InspectSchema returns the schema description by its name. A NotExistError
		// error is returned if the schema does not exists in the database.
		InspectSchema(ctx context.Context, name string, opts *InspectOptions) (*Schema, error)
	}

	// InspectTableOptions describes options for TableInspector.
	InspectTableOptions struct {
		// Schema defines an optional schema to inspect.
		Schema string
	}

	// TableInspector is the interface implemented by the different database
	// drivers for inspecting their schema tables.
	TableInspector interface {
		// InspectTable returns the table description by its name. A NotExistError
		// error is returned if the table does not exists in the database.
		InspectTable(ctx context.Context, name string, opts *InspectTableOptions) (*Table, error)
	}

	// InspectRealmOption describes options for RealmInspector.
	InspectRealmOption struct {
		// Schemas to inspect. Empty means all tables in the schema.
		Schemas []string
	}

	// RealmInspector is the interface implemented by the different database
	// drivers for inspecting multiple schemas (realm).
	RealmInspector interface {
		InspectRealm(ctx context.Context, opts *InspectRealmOption) (*Realm, error)
	}
)
