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

// Realm returns schema descriptions of all resources in the given realm.
func (d *Driver) Realm(ctx context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	if opts == nil || len(opts.Schemas) == 0 {
		return nil, fmt.Errorf("mysql: at least 1 schema is required")
	}
	realm := &schema.Realm{}
	for _, s := range opts.Schemas {
		tables, err := d.Tables(ctx, &schema.InspectTableOptions{
			Schema: s,
		})
		if err != nil {
			return nil, err
		}
		realm.Schemas = append(realm.Schemas, &schema.Schema{Name: s, Tables: tables})
	}
	linkSchemaTables(realm.Schemas)
	return realm, nil
}

// Tables returns schema descriptions of all tables in the given schema.
func (d *Driver) Tables(ctx context.Context, opts *schema.InspectTableOptions) ([]*schema.Table, error) {
	names, err := d.tableNames(ctx, opts)
	if err != nil {
		return nil, err
	}
	tables := make([]*schema.Table, 0, len(names))
	for _, name := range names {
		t, err := d.Table(ctx, name, opts)
		if err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	if len(tables) > 0 {
		// Link all tables reside on the same schema.
		linkSchemaTables([]*schema.Schema{{Name: tables[0].Schema, Tables: tables}})
	}
	return tables, nil
}

// Table returns the schema description of the given table.
func (d *Driver) Table(ctx context.Context, name string, opts *schema.InspectTableOptions) (*schema.Table, error) {
	t, err := d.table(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	if err := d.columns(ctx, t); err != nil {
		return nil, err
	}
	if err := d.indexes(ctx, t); err != nil {
		return nil, err
	}
	if err := d.fks(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// table returns the table from the database, or a NotExistError if the table was not found.
func (d *Driver) table(ctx context.Context, name string, opts *schema.InspectTableOptions) (*schema.Table, error) {
	query, args := tableQuery, []interface{}{name}
	if opts != nil && opts.Schema != "" {
		query, args = tableSchemaQuery, []interface{}{opts.Schema, name}
	}
	row := d.QueryRowContext(ctx, query, args...)
	var s string
	if err := row.Scan(&s); err != nil {
		if err == sql.ErrNoRows {
			return nil, &schema.NotExistError{
				Err: fmt.Errorf("mysql: table %q was not found", name),
			}
		}
		return nil, err
	}
	return &schema.Table{Name: name, Schema: s}, nil
}

// columns queries and appends the columns of the given table.
func (d *Driver) columns(ctx context.Context, t *schema.Table) error {
	rows, err := d.QueryContext(ctx, columnsQuery, t.Schema, t.Name)
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
	case "point", "multipoint", "linestring", "multilinestring", "polygon", "multipolygon", "geometry", "geomcollection", "geometrycollection":
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

// indexes queries and appends the indexes of the given table.
func (d *Driver) indexes(ctx context.Context, t *schema.Table) error {
	rows, err := d.QueryContext(ctx, indexesQuery, t.Schema, t.Name)
	if err != nil {
		return fmt.Errorf("mysql: querying %q indexes: %w", t.Name, err)
	}
	defer rows.Close()
	if err := d.addIndexes(t, rows); err != nil {
		return err
	}
	return rows.Err()
}

// addIndexes scans the rows and adds the indexes to the table.
func (d *Driver) addIndexes(t *schema.Table, rows *sql.Rows) error {
	names := make(map[string]*schema.Index)
	for rows.Next() {
		var (
			name    string
			column  string
			nonuniq bool
		)
		if err := rows.Scan(&name, &column, &nonuniq); err != nil {
			return fmt.Errorf("mysql: scanning index: %w", err)
		}
		// Ignore primary keys.
		if name == "PRIMARY" {
			continue
		}
		idx, ok := names[name]
		if !ok {
			idx = &schema.Index{Name: name, Unique: !nonuniq, Table: t}
			names[name] = idx
			t.Indexes = append(t.Indexes, idx)
		}
		c, ok := t.Column(column)
		if !ok {
			return fmt.Errorf("mysql: column %q was not found for index %q", column, idx.Name)
		}
		// Rows are ordered by SEQ_IN_INDEX that specifies the
		// position of the column in the index definition.
		idx.Columns = append(idx.Columns, c)
		c.Indexes = append(c.Indexes, idx)
	}
	return nil
}

// fks queries and appends the foreign keys of the given table.
func (d *Driver) fks(ctx context.Context, t *schema.Table) error {
	rows, err := d.QueryContext(ctx, fksQuery, t.Schema, t.Name)
	if err != nil {
		return fmt.Errorf("mysql: querying %q indexes: %w", t.Name, err)
	}
	defer rows.Close()
	if err := d.addFKs(t, rows); err != nil {
		return fmt.Errorf("mysql: %w", err)
	}
	return rows.Err()
}

// addFK scans the rows and adds the foreign-key to the table.
// Reference elements are added as stubs and should be linked
// manually by the caller.
func (d *Driver) addFKs(t *schema.Table, rows *sql.Rows) error {
	names := make(map[string]*schema.ForeignKey)
	for rows.Next() {
		var name, table, column, tSchema, refTable, refColumn, refSchema, updateRule, deleteRule string
		if err := rows.Scan(&name, &table, &column, &tSchema, &refTable, &refColumn, &refSchema, &updateRule, &deleteRule); err != nil {
			return err
		}
		fk, ok := names[name]
		if !ok {
			fk = &schema.ForeignKey{
				Symbol:   name,
				Table:    t,
				RefTable: t,
				OnDelete: schema.ReferenceOption(deleteRule),
				OnUpdate: schema.ReferenceOption(updateRule),
			}
			if refTable != t.Name || tSchema != refSchema {
				fk.RefTable = &schema.Table{Name: refTable, Schema: refSchema}
			}
			names[name] = fk
			t.ForeignKeys = append(t.ForeignKeys, fk)
		}
		c, ok := t.Column(column)
		if !ok {
			return fmt.Errorf("column %q was not found for fk %q", column, fk.Symbol)
		}
		// Rows are ordered by ORDINAL_POSITION that specifies
		// the position of the column in the FK definition.
		fk.Columns = append(fk.Columns, c)
		c.ForeignKeys = append(c.ForeignKeys, fk)

		// Stub referenced columns or link if it's a self-reference.
		if fk.Table != fk.RefTable {
			fk.RefColumns = append(fk.RefColumns, &schema.Column{Name: refColumn})
		} else if c, ok := t.Column(refColumn); ok {
			fk.RefColumns = append(fk.RefColumns, c)
		} else {
			return fmt.Errorf("referenced column %q was not found for fk %q", refColumn, fk.Symbol)
		}
	}
	return nil
}

// tableNames returns a list of all tables exist in the schema.
func (d *Driver) tableNames(ctx context.Context, opts *schema.InspectTableOptions) ([]string, error) {
	query, args := tablesQuery, []interface{}(nil)
	if opts != nil && opts.Schema != "" {
		query, args = tablesSchemaQuery, []interface{}{opts.Schema}
	}
	rows, err := d.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mysql: querying schema tables: %w", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("mysql: scanning table name: %w", err)
		}
		names = append(names, name)
	}
	return names, nil
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
	case "on update current_timestamp", "default_generated":
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

// linkSchemaTables links foreign-key stub tables/columns to actual elements.
func linkSchemaTables(schemas []*schema.Schema) {
	byName := make(map[string]map[string]*schema.Table)
	for _, s := range schemas {
		byName[s.Name] = make(map[string]*schema.Table)
		for _, t := range s.Tables {
			byName[s.Name][t.Name] = t
		}
	}
	for _, s := range schemas {
		for _, t := range s.Tables {
			for _, fk := range t.ForeignKeys {
				rs, ok := byName[fk.RefTable.Schema]
				if !ok {
					continue
				}
				ref, ok := rs[fk.RefTable.Name]
				if !ok {
					continue
				}
				fk.RefTable = ref
				for i, c := range fk.RefColumns {
					rc, ok := ref.Column(c.Name)
					if ok {
						fk.RefColumns[i] = rc
					}
				}
			}
		}
	}
}

const (
	// Queries to list schema tables.
	tablesSchemaQuery = "SELECT `TABLE_NAME` FROM `INFORMATION_SCHEMA`.`TABLES` WHERE `TABLE_SCHEMA` = ?"
	tablesQuery       = "SELECT `TABLE_NAME` FROM `INFORMATION_SCHEMA`.`TABLES` WHERE `TABLE_SCHEMA` = (SELECT DATABASE())"

	// Queries to fetch table schema.
	tableQuery       = "SELECT `TABLE_SCHEMA` FROM `INFORMATION_SCHEMA`.`TABLES` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?"
	tableSchemaQuery = "SELECT `TABLE_SCHEMA` FROM `INFORMATION_SCHEMA`.`TABLES` WHERE `TABLE_SCHEMA` = (SELECT DATABASE()) AND `TABLE_NAME` = ?"

	// Query to list table columns.
	columnsQuery = "SELECT `column_name`, `column_type`, `is_nullable`, `column_key`, `column_default`, `extra`, `character_set_name`, `collation_name` FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?"

	// Query to list table indexes.
	indexesQuery = "SELECT `index_name`, `column_name`, `non_unique` FROM `INFORMATION_SCHEMA`.`STATISTICS` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ? ORDER BY `index_name`, `seq_in_index`"

	// Query to list table foreign keys.
	fksQuery = `
SELECT
  t1.CONSTRAINT_NAME,
  t1.TABLE_NAME,
  t1.COLUMN_NAME,
  t1.TABLE_SCHEMA,
  t1.REFERENCED_TABLE_NAME,
  t1.REFERENCED_COLUMN_NAME,
  t1.REFERENCED_TABLE_SCHEMA,
  t3.UPDATE_RULE,
  t3.DELETE_RULE
FROM
  INFORMATION_SCHEMA.KEY_COLUMN_USAGE AS t1
  JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS AS t2
  JOIN INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS AS t3
  ON t1.CONSTRAINT_NAME = t2.CONSTRAINT_NAME
  AND t1.CONSTRAINT_NAME = t3.CONSTRAINT_NAME
  AND t1.TABLE_SCHEMA = t2.TABLE_SCHEMA
  AND t1.TABLE_SCHEMA = t3.CONSTRAINT_SCHEMA
WHERE
  t2.CONSTRAINT_TYPE = 'FOREIGN KEY'
  AND t1.TABLE_SCHEMA = ?
  AND t1.TABLE_NAME = ?
ORDER BY
  t1.CONSTRAINT_NAME,
  t1.ORDINAL_POSITION`
)

var _ schema.TableInspector = (*Driver)(nil)

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
