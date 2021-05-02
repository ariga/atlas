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

// IsNotExistError reports an error is a NotExistError.
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
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

type (
	// InspectOptions describes options for Inspector.
	InspectOptions struct {
		// Schema defines an optional schema to inspect.
		Schema string
	}

	// Inspector is the interface implemented by the different database drivers for
	// inspecting their schema.
	Inspector interface {
		// Table returns the table description by its name. A NotExistError error
		// is returned if the table does not exists in the database.
		Table(ctx context.Context, name string, opts *InspectOptions) (*Table, error)
	}
)
