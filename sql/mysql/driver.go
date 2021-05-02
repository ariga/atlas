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

// Open opens a new MySQL driver.
func Open(db schema.ExecQuerier) (*Driver, error) {
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

// columns queries and appends the columns of the given table.
func (d *Driver) columns(ctx context.Context, t *schema.Table, opts *schema.InspectOptions) error {
	query, args := columnsQuery, []interface{}{t.Name}
	if opts != nil && opts.Schema != "" {
		query, args = columnsSchemaQuery, []interface{}{opts.Schema, t.Name}
	}
	rows, err := d.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("mysql: querying %q columns: %w", t.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := d.addColumn(t, rows); err != nil {
			return fmt.Errorf("mysql: %w", err)
		}
	}
	return rows.Err()
}

// addColumn scans the current row and adds a new column from it to the table.
func (d *Driver) addColumn(t *schema.Table, rows *sql.Rows) error {
	var name, typ, nullable, key, defaults, extra, charset, collation sql.NullString
	if err := rows.Scan(&name, &typ, &nullable, &key, &defaults, &extra, &charset, &collation); err != nil {
		return err
	}
	c := &schema.Column{
		Name: name.String,
		Type: &schema.ColumnType{
			Raw:  typ.String,
			Null: nullable.String == "YES",
		},
	}
	parts, size, unsigned, err := parseColumn(c.Type.Raw)
	if err != nil {
		return err
	}
	switch t := parts[0]; t {
	case "tinyint", "smallint", "mediumint", "int", "bigint":
		if size == 1 {
			c.Type.Type = &schema.BoolType{
				T: t,
			}
			break
		}
		c.Type.Type = &schema.IntegerType{
			T:        t,
			Size:     int(size),
			Unsigned: unsigned,
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
		c.Type.Type = dt
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
		c.Type.Type = ft
	case "binary", "varbinary":
		c.Type.Type = &schema.BinaryType{
			T:    t,
			Size: int(size),
		}
	case "tinyblob", "mediumblob", "blob", "longblob":
		c.Type.Type = &schema.BinaryType{
			T: t,
		}
	case "char", "varchar":
		c.Type.Type = &schema.StringType{
			T:    t,
			Size: int(size),
		}
	case "tinytext", "mediumtext", "text", "longtext":
		c.Type.Type = &schema.StringType{
			T: t,
		}
	case "enum":
		values := make([]string, len(parts)-1)
		for i, e := range parts[1:] {
			values[i] = strings.Trim(e, "'")
		}
		c.Type.Type = &schema.EnumType{
			Values: values,
		}
	case "date", "datetime", "time", "timestamp", "year":
		c.Type.Type = &schema.TimeType{
			T: t,
		}
	case "json":
		c.Type.Type = &schema.JSONType{
			T: t,
		}
	case "point", "multipoint", "linestring", "multilinestring", "polygon", "multipolygon", "geometry", "geometrycollection":
		c.Type.Type = &schema.SpatialType{
			T: t,
		}
	default:
		c.Type.Type = &schema.UnsupportedType{
			T: t,
		}
	}
	defaultAttr(c, defaults.String)
	if err := extraAttr(c, extra.String); err != nil {
		return err
	}
	t.Columns = append(t.Columns, c)
	if key.String == "PRI" {
		t.PrimaryKey = append(t.PrimaryKey, c)
	}
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

// extraAttr parses the EXTRA column from the INFORMATION_SCHEMA.COLUMNS table
// and appends its parsed representation to the column.
func extraAttr(c *schema.Column, extra string) error {
	switch extra := strings.ToLower(extra); extra {
	case "", "null": // ignore.
	case "auto_increment":
		c.Attrs = append(c.Attrs, &AutoIncrement{A: extra})
	case "on update current_timestamp":
		c.Attrs = append(c.Attrs, &OnUpdate{A: extra})
	default:
		return fmt.Errorf("unknown attribute %q", extra)
	}
	return nil
}

// defaultAttr parses the COLUMN_DEFAULT column from the INFORMATION_SCHEMA.COLUMNS table
// and appends its parsed representation to the column-type.
func defaultAttr(c *schema.Column, defaults string) {
	if defaults != "" && strings.ToLower(defaults) != "null" {
		c.Type.Default = &schema.RawExpr{
			X: defaults,
		}
	}
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

type (
	// AutoIncrement attribute for columns with "AUTO_INCREMENT" as a default.
	AutoIncrement struct {
		schema.Attr
		A string
	}

	// OnUpdate attribute for columns with "ON UPDATE CURRENT_TIMESTAMP" as a default.
	OnUpdate struct {
		schema.Attr
		A string
	}
)
