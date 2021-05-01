package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/schema"
)

// Driver represents a MySQL driver for introspecting database schemas
// and apply migrations changes on them.
type Driver struct {
	schema.ExecQuerier
	version string
}

// NewDriver returns a new MySQL driver.
func NewDriver(db schema.ExecQuerier) (*Driver, error) {
	var version [2]string
	if err := db.QueryRow("SHOW VARIABLES LIKE 'version'").Scan(&version[0], &version[1]); err != nil {
		return nil, fmt.Errorf("mysql: scanning version: %w", err)
	}
	return &Driver{ExecQuerier: db, version: version[1]}, nil
}

// Table returns the schema description of the given table.
func (d *Driver) Table(ctx context.Context, name string, opts *schema.InspectOptions) (*schema.Table, error) {
	exists, err := d.tableExists(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &schema.NotExistError{
			Err: fmt.Errorf("mysql: table %q was not found", name),
		}
	}
	t := &schema.Table{Name: name}
	if err := d.columns(ctx, t, opts); err != nil {
		return nil, err
	}
	return t, nil
}

// tableExists checks if the given table exists in the database.
func (d *Driver) tableExists(ctx context.Context, name string, opts *schema.InspectOptions) (bool, error) {
	query, args := existsQuery, []interface{}{name}
	if opts != nil && opts.Schema != "" {
		query, args = existsSchemaQuery, []interface{}{opts.Schema, name}
	}
	row := d.QueryRowContext(ctx, query, args...)
	var n int
	if err := row.Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (d *Driver) columns(ctx context.Context, t *schema.Table, opts *schema.InspectOptions) error {
	query, args := columnsQuery, []interface{}{t.Name}
	if opts != nil && opts.Schema != "" {
		query, args = columnsSchemaQuery, []interface{}{opts.Schema, t.Name}
	}
	rows, err := d.QueryContext(ctx, query, args)
	if err != nil {
		return fmt.Errorf("mysql: querying %q columns: %w", t.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		c := &schema.Column{}
		if err := d.scanColumn(c, rows); err != nil {
			return fmt.Errorf("mysql: %w", err)
		}
		// TODO(a8m): set primary-key (including composite) and add columns to table.
	}
	return rows.Err()
}

func (d *Driver) scanColumn(c *schema.Column, rows *sql.Rows) error {
	var vs [7]sql.NullString // type, nullable, key, default, extra, charset, collation.
	if err := rows.Scan(&c.Name, &vs[0], &vs[1], &vs[2], &vs[3], &vs[4], &vs[5], &vs[6]); err != nil {
		return err
	}
	cType := &schema.ColumnType{
		Raw:  vs[0].String,
		Null: vs[1].String == "YES",
	}
	if vs[3].Valid {
		cType.Default = &schema.RawExpr{
			X: vs[3].String,
		}
	}
	parts, size, unsigned, err := parseColumn(cType.Raw)
	if err != nil {
		return err
	}
	switch t := parts[0]; t {
	case "tinyint":
		if size == 1 {
			cType.Type = &schema.BoolType{
				T: t,
			}
			break
		}
		fallthrough
	case "smallint", "mediumint", "int", "bigint":
		cType.Type = &schema.IntegerType{
			T:      t,
			Size:   int(size),
			Signed: !unsigned,
		}
	case "numeric", "decimal":
		dt := &schema.DecimalType{
			T: t,
		}
		if len(parts) > 1 {
			p, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return fmt.Errorf("parse precision %q", parts[1])
			}
			dt.Precision = int(p)
		}
		if len(parts) > 2 {
			s, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return fmt.Errorf("parse scale %q", parts[1])
			}
			dt.Scale = int(s)
		}
		cType.Type = dt
	case "float", "double":
		ft := &schema.FloatType{
			T: t,
		}
		if len(parts) > 1 {
			p, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return fmt.Errorf("parse precision %q", parts[1])
			}
			ft.Precision = int(p)
		}
		cType.Type = ft
	case "binary", "varbinary":
		cType.Type = &schema.BinaryType{
			T:    t,
			Size: int(size),
		}
	case "tinyblob", "mediumblob", "blob", "longblob":
		cType.Type = &schema.BinaryType{
			T: t,
		}
	// TODO(a8m): strings, time, JSON and spatial types in next PRs.
	default:
		cType.Type = &schema.UnsupportedType{
			T: t,
		}
	}
	c.Type = cType
	return nil
}

// parseColumn returns column parts, size and signed-info from a MySQL type.
func parseColumn(typ string) (parts []string, size int64, unsigned bool, err error) {
	switch parts = strings.FieldsFunc(typ, func(r rune) bool {
		return r == '(' || r == ')' || r == ' ' || r == ','
	}); parts[0] {
	case "tinyint", "smallint", "mediumint", "int", "bigint":
		switch {
		case len(parts) == 2 && parts[1] == "unsigned": // int unsigned
			unsigned = true
		case len(parts) == 3: // int(10) unsigned
			unsigned = true
			fallthrough
		case len(parts) == 2: // int(10)
			size, err = strconv.ParseInt(parts[1], 10, 0)
		}
	case "varbinary", "varchar", "char", "binary":
		size, err = strconv.ParseInt(parts[1], 10, 64)
	}
	if err != nil {
		return parts, size, unsigned, fmt.Errorf("converting %s size to int: %w", parts[0], err)
	}
	return parts, size, unsigned, nil
}

const (
	// Queries to check table existence in the database.
	existsSchemaQuery = "SELECT COUNT(*) FROM `INFORMATION_SCHEMA`.`TABLES` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?"
	existsQuery       = "SELECT COUNT(*) FROM `INFORMATION_SCHEMA`.`TABLES` WHERE `TABLE_SCHEMA` = (SELECT DATABASE()) AND `TABLE_NAME` = ?"

	// Queries to check table existence in the database.
	columnsSchemaQuery = "SELECT `column_name`, `column_type`, `is_nullable`, `column_key`, `column_default`, `extra`, `character_set_name`, `collation_name` FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?"
	columnsQuery       = "SELECT `column_name`, `column_type`, `is_nullable`, `column_key`, `column_default`, `extra`, `character_set_name`, `collation_name` FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = (SELECT DATABASE()) AND `TABLE_NAME` = ?"
)

var _ schema.Inspector = (*Driver)(nil)
