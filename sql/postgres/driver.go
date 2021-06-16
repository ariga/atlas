package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"

	"golang.org/x/mod/semver"
)

// Driver represents a PostgreSQL driver for introspecting database schemas
// and apply migrations changes on them.
type Driver struct {
	schema.ExecQuerier
	// System variables that are set on `Open`.
	collate string
	ctype   string
	version string
}

// Open opens a new PostgreSQL driver.
func Open(db schema.ExecQuerier) (*Driver, error) {
	drv := &Driver{ExecQuerier: db}
	rows, err := db.Query(paramsQuery)
	if err != nil {
		return nil, fmt.Errorf("postgres: scanning system variables: %w", err)
	}
	defer rows.Close()
	params := make([]string, 0, 3)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("postgres: failed scanning row value: %w", err)
		}
		params = append(params, v)
	}
	if len(params) != 3 {
		return nil, fmt.Errorf("postgres: unexpected number of rows: %d", len(params))
	}
	drv.collate, drv.ctype, drv.version = params[0], params[1], params[2]
	if len(drv.version) < 6 {
		return nil, fmt.Errorf("postgres: malformed version: %s", drv.version)
	}
	drv.version = fmt.Sprintf("%s.%s.%s", drv.version[:2], drv.version[2:4], drv.version[4:])
	if semver.Compare("v"+drv.version, "v10.0.0") != -1 {
		return nil, fmt.Errorf("postgres: unsupported postgres version: %s", drv.version)
	}
	return drv, nil
}

// InspectTable returns the schema description of the given table.
func (d *Driver) InspectTable(ctx context.Context, name string, opts *schema.InspectTableOptions) (*schema.Table, error) {
	return d.inspectTable(ctx, name, opts, nil)
}

func (d *Driver) inspectTable(ctx context.Context, name string, opts *schema.InspectTableOptions, top *schema.Schema) (*schema.Table, error) {
	t, err := d.table(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	if top != nil {
		// Link the table to its top element if provided.
		t.Schema = top
	}
	if err := d.columns(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// table returns the table from the database, or a NotExistError if the table was not found.
func (d *Driver) table(ctx context.Context, name string, opts *schema.InspectTableOptions) (*schema.Table, error) {
	var (
		args  = []interface{}{name}
		query = tableQuery
	)
	if opts != nil && opts.Schema != "" {
		query = tableSchemaQuery
		args = append(args, opts.Schema)
	}
	row := d.QueryRowContext(ctx, query, args...)
	var tSchema, comment sql.NullString
	if err := row.Scan(&tSchema, &comment); err != nil {
		if err == sql.ErrNoRows {
			return nil, &schema.NotExistError{
				Err: fmt.Errorf("postgres: table %q was not found", name),
			}
		}
		return nil, err
	}
	t := &schema.Table{Name: name, Schema: &schema.Schema{Name: tSchema.String}}
	if sqlx.ValidString(comment) {
		t.Attrs = append(t.Attrs, &schema.Comment{
			Text: comment.String,
		})
	}
	return t, nil
}

// columns queries and appends the columns of the given table.
func (d *Driver) columns(ctx context.Context, t *schema.Table) error {
	rows, err := d.QueryContext(ctx, columnsQuery, t.Schema.Name, t.Name)
	if err != nil {
		return fmt.Errorf("postgres: querying %q columns: %w", t.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := d.addColumn(t, rows); err != nil {
			return fmt.Errorf("postgres: %w", err)
		}
	}
	return rows.Err()
}

// addColumn scans the current row and adds a new column from it to the table.
func (d *Driver) addColumn(t *schema.Table, rows *sql.Rows) error {
	var (
		maxlen, precision, scale                                                  sql.NullInt64
		name, typ, nullable, defaults, udt, identity, charset, collation, comment sql.NullString
	)
	if err := rows.Scan(&name, &typ, &nullable, &defaults, &maxlen, &precision, &scale, &charset, &collation, &udt, &identity, &comment); err != nil {
		return err
	}
	c := &schema.Column{
		Name: name.String,
		Type: &schema.ColumnType{
			Raw:  typ.String,
			Null: nullable.String == "YES",
		},
	}
	if sqlx.ValidString(defaults) {
		c.Default = &schema.RawExpr{
			X: defaults.String,
		}
	}
	if sqlx.ValidString(comment) {
		c.Attrs = append(c.Attrs, &schema.Comment{
			Text: comment.String,
		})
	}
	if sqlx.ValidString(charset) {
		c.Attrs = append(c.Attrs, &schema.Charset{
			V: charset.String,
		})
	}
	if sqlx.ValidString(collation) {
		c.Attrs = append(c.Attrs, &schema.Collation{
			V: collation.String,
		})
	}
	t.Columns = append(t.Columns, c)
	return nil
}

const (
	// Query to list runtime parameters.
	paramsQuery = `SELECT setting FROM pg_settings WHERE name IN ('lc_collate', 'lc_ctype', 'server_version_num') ORDER BY name`
	// Query to list table information.
	tableQuery = `
SELECT
	t1.TABLE_SCHEMA,
	pg_catalog.obj_description(t2.oid, 'pg_class') AS COMMENT
FROM
	INFORMATION_SCHEMA.TABLES AS t1
	INNER JOIN pg_catalog.pg_class AS t2
	ON t1.table_name = t2.relname
WHERE
	t1.TABLE_TYPE = 'BASE TABLE'
	AND t1.TABLE_NAME = ?
	AND t1.TABLE_SCHEMA = (CURRENT_SCHEMA())
`
	tableSchemaQuery = `
SELECT
	t1.TABLE_SCHEMA,
	pg_catalog.obj_description(t2.oid, 'pg_class') AS COMMENT
FROM
	INFORMATION_SCHEMA.TABLES AS t1
	INNER JOIN pg_catalog.pg_class AS t2
	ON t1.table_name = t2.relname
WHERE
	t1.TABLE_TYPE = 'BASE TABLE'
	AND t1.TABLE_NAME = ?
	AND t1.TABLE_SCHEMA = ?
`
	// Query to list table columns.
	columnsQuery = `
SELECT
	"column_name",
	"data_type",
	"is_nullable",
	"column_default",
	"character_maximum_length",
	"numeric_precision",
	"numeric_scale",
	"character_set_name",
	"collation_name",
	"udt_name",
	"is_identity",
	col_description(to_regclass("table_schema" || '.' || "table_name")::oid, "ordinal_position") AS "comment"
FROM
	"information_schema"."columns"
WHERE
	TABLE_SCHEMA = ? AND TABLE_NAME = ?
`
)
