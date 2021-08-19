// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"ariga.io/atlas/sql/internal/sqlx"

	"ariga.io/atlas/sql/schema"
)

const Name = "sqlite"

// Driver represents an SQLite driver for introspecting database schemas
// and apply migrations changes on them.
type Driver struct {
	schema.ExecQuerier
	// System variables that are set on `Open`.
	fkEnabled  bool
	version    string
	collations []string
}

// Open opens a new SQLite driver.
func Open(db schema.ExecQuerier) (*Driver, error) {
	drv := &Driver{ExecQuerier: db}
	if err := db.QueryRow("SELECT sqlite_version()").Scan(&drv.version); err != nil {
		return nil, fmt.Errorf("sqlite: scanning database version: %w", err)
	}
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&drv.fkEnabled); err != nil {
		return nil, fmt.Errorf("sqlite: check foreign_keys pragma: %w", err)
	}
	rows, err := db.Query("SELECT name FROM pragma_collation_list()")
	if err != nil {
		return nil, fmt.Errorf("sqlite: check collation_list pragma: %w", err)
	}
	if drv.collations, err = sqlx.ScanStrings(rows); err != nil {
		return nil, fmt.Errorf("sqlite: scanning database collations: %w", err)
	}
	return drv, nil
}

// InspectRealm returns schema descriptions of all resources in the given realm.
func (d *Driver) InspectRealm(ctx context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	schemas, err := d.databases(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(schemas) > 1 {
		return nil, fmt.Errorf("sqlite: multiple database files are not supported by the driver. got: %d", len(schemas))
	}
	realm := &schema.Realm{Schemas: schemas}
	for _, s := range schemas {
		tables, err := d.tables(ctx, nil)
		if err != nil {
			return nil, err
		}
		for _, t := range tables {
			t, err := d.inspectTable(ctx, t)
			if err != nil {
				return nil, err
			}
			t.Schema = s
			s.Tables = append(s.Tables, t)
		}
		s.Realm = realm
	}
	sqlx.LinkSchemaTables(realm.Schemas)
	return realm, nil
}

// InspectSchema returns schema descriptions of all tables in the given schema.
func (d *Driver) InspectSchema(ctx context.Context, name string, _ *schema.InspectOptions) (*schema.Schema, error) {
	schemas, err := d.databases(ctx, &schema.InspectRealmOption{
		Schemas: []string{name},
	})
	if err != nil {
		return nil, err
	}
	if len(schemas) == 0 {
		return nil, &schema.NotExistError{
			Err: fmt.Errorf("sqlite: schema %q was not found", name),
		}
	}
	tables, err := d.tables(ctx, nil)
	if err != nil {
		return nil, err
	}
	s := schemas[0]
	for _, t := range tables {
		t, err := d.inspectTable(ctx, t)
		if err != nil {
			return nil, err
		}
		t.Schema = s
		s.Tables = append(s.Tables, t)
	}
	sqlx.LinkSchemaTables(schemas)
	s.Realm = &schema.Realm{Schemas: schemas}
	return s, nil
}

// InspectTable returns the schema description of the given table.
func (d *Driver) InspectTable(ctx context.Context, name string, opts *schema.InspectTableOptions) (*schema.Table, error) {
	if opts != nil && opts.Schema != "" {
		return nil, fmt.Errorf("sqlite: querying custom schema is not supported. got: %q", opts.Schema)
	}
	tables, err := d.tables(ctx, &schema.InspectOptions{
		Tables: []string{name},
	})
	if err != nil {
		return nil, err
	}
	if len(tables) == 0 {
		return nil, &schema.NotExistError{
			Err: fmt.Errorf("sqlite: table %q was not found", name),
		}
	}
	return d.inspectTable(ctx, tables[0])
}

func (d *Driver) inspectTable(ctx context.Context, t *schema.Table) (*schema.Table, error) {
	if err := d.columns(ctx, t); err != nil {
		return nil, err
	}
	if err := d.indexes(ctx, t); err != nil {
		return nil, err
	}
	if err := d.fks(ctx, t); err != nil {
		return nil, err
	}
	// TODO(a8m): extract CHECK constraints from 'CREATE TABLE' statement.
	return t, nil
}

// columns queries and appends the columns of the given table.
func (d *Driver) columns(ctx context.Context, t *schema.Table) error {
	rows, err := d.QueryContext(ctx, fmt.Sprintf(columnsQuery, t.Name))
	if err != nil {
		return fmt.Errorf("sqlite: querying %q columns: %w", t.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := d.addColumn(t, rows); err != nil {
			return fmt.Errorf("sqlite: %w", err)
		}
	}
	return rows.Err()
}

// addColumn scans the current row and adds a new column from it to the table.
func (d *Driver) addColumn(t *schema.Table, rows *sql.Rows) error {
	var (
		nullable, primary   bool
		name, typ, defaults sql.NullString
		err                 error
	)
	if err = rows.Scan(&name, &typ, &nullable, &defaults, &primary); err != nil {
		return err
	}
	c := &schema.Column{
		Name: name.String,
		Type: &schema.ColumnType{
			Raw:  typ.String,
			Null: nullable,
		},
	}
	c.Type.Type, err = parseRawType(typ.String)
	if err != nil {
		return err
	}
	// TODO(a8m): extract collation from 'CREATE TABLE' statement.
	t.Columns = append(t.Columns, c)
	if primary {
		if t.PrimaryKey == nil {
			t.PrimaryKey = &schema.Index{
				Name:   "PRIMARY",
				Unique: true,
				Table:  t,
			}
		}
		// Columns are ordered by the `pk` field.
		t.PrimaryKey.Parts = append(t.PrimaryKey.Parts, &schema.IndexPart{
			C:     c,
			SeqNo: len(t.PrimaryKey.Parts) + 1,
		})
	}
	return nil
}

func parseRawType(c string) (schema.Type, error) {
	parts := columnParts(c)
	switch t := parts[0]; t {
	case "bool", "boolean":
		return &schema.BoolType{T: t}, nil
	case "blob":
		return &schema.BinaryType{T: t}, nil
	case "int2", "int8", "int", "integer", "tinyint", "smallint", "mediumint", "bigint", "unsigned big int":
		// All integer types have the same "type affinity".
		return &schema.IntegerType{T: t}, nil
	case "real", "double", "double precision", "float":
		return &schema.FloatType{T: t}, nil
	case "numeric", "decimal":
		ct := &schema.DecimalType{T: t}
		if len(parts) > 1 {
			p, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse precision %q", parts[1])
			}
			ct.Precision = int(p)
		}
		if len(parts) > 2 {
			s, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse scale %q", parts[1])
			}
			ct.Scale = int(s)
		}
		return ct, nil
	case "character", "varchar", "varying character", "nchar", "native character", "nvarchar", "text", "clob":
		ct := &schema.StringType{T: t}
		if len(parts) > 1 {
			p, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse size %q", parts[1])
			}
			ct.Size = int(p)
		}
		return ct, nil
	case "json":
		return &schema.JSONType{T: t}, nil
	case "date", "datetime":
		return &schema.TimeType{T: t}, nil
	default:
		return nil, fmt.Errorf("unknown column type %q", t)
	}
}

// indexes queries and appends the indexes of the given table.
func (d *Driver) indexes(ctx context.Context, t *schema.Table) error {
	rows, err := d.QueryContext(ctx, fmt.Sprintf(indexesQuery, t.Name))
	if err != nil {
		return fmt.Errorf("sqlite: querying %q indexes: %w", t.Name, err)
	}
	if err := d.addIndexes(t, rows); err != nil {
		return fmt.Errorf("sqlite: scan %q indexes: %w", t.Name, err)
	}
	for _, idx := range t.Indexes {
		if err := d.indexColumns(ctx, t, idx); err != nil {
			return err
		}
	}
	return nil
}

// addIndexes scans the rows and adds the indexes to the table.
func (d *Driver) addIndexes(t *schema.Table, rows *sql.Rows) error {
	defer rows.Close()
	for rows.Next() {
		var (
			uniq, partial      bool
			name, origin, stmt sql.NullString
		)
		if err := rows.Scan(&name, &uniq, &origin, &partial, &stmt); err != nil {
			return err
		}
		if origin.String == "pk" {
			continue
		}
		idx := &schema.Index{
			Name:   name.String,
			Unique: uniq,
			Table:  t,
			Attrs:  []schema.Attr{&CreateStmt{S: stmt.String}},
		}
		if partial {
			i := strings.Index(stmt.String, "WHERE")
			if i == -1 {
				return fmt.Errorf("missing partial WHERE clause in: %s", stmt.String)
			}
			idx.Attrs = append(idx.Attrs, &IndexPredicate{
				P: strings.TrimSpace(stmt.String[i+5:]),
			})
		}
		t.Indexes = append(t.Indexes, idx)
	}
	return nil
}

func (d *Driver) indexColumns(ctx context.Context, t *schema.Table, idx *schema.Index) error {
	rows, err := d.QueryContext(ctx, fmt.Sprintf(indexColumnsQuery, idx.Name))
	if err != nil {
		return fmt.Errorf("sqlite: querying %q indexes: %w", t.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		var name sql.NullString
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("sqlite: scanning index names: %w", err)
		}
		switch c, ok := t.Column(name.String); {
		case ok:
			idx.Parts = append(idx.Parts, &schema.IndexPart{
				SeqNo: len(idx.Parts) + 1,
				C:     c,
			})
		// NULL name indicates that the index-part is an expression and we
		// should extract it from the `CREATE INDEX` statement (not supported atm).
		case !sqlx.ValidString(name):
			idx.Parts = append(idx.Parts, &schema.IndexPart{
				SeqNo: len(idx.Parts) + 1,
				X:     &schema.RawExpr{X: "<unsupported>"},
			})
		default:
			return fmt.Errorf("sqlite: column %q was not found for index %q", name.String, idx.Name)
		}
	}
	return nil
}

// fks queries and appends the foreign-keys of the given table.
func (d *Driver) fks(ctx context.Context, t *schema.Table) error {
	rows, err := d.QueryContext(ctx, fmt.Sprintf(fksQuery, t.Name))
	if err != nil {
		return fmt.Errorf("sqlite: querying %q foreign-keys: %w", t.Name, err)
	}
	if err := d.addFKs(t, rows); err != nil {
		return fmt.Errorf("sqlite: scan %q foreign-keys: %w", t.Name, err)
	}
	return nil
}

func (d *Driver) addFKs(t *schema.Table, rows *sql.Rows) error {
	ids := make(map[int]*schema.ForeignKey)
	for rows.Next() {
		var (
			id                                                  int
			column, refColumn, refTable, updateRule, deleteRule string
		)
		if err := rows.Scan(&id, &column, &refColumn, &refTable, &updateRule, &deleteRule); err != nil {
			return err
		}
		fk, ok := ids[id]
		if !ok {
			fk = &schema.ForeignKey{
				Symbol:   strconv.Itoa(id),
				Table:    t,
				RefTable: t,
				OnDelete: schema.ReferenceOption(deleteRule),
				OnUpdate: schema.ReferenceOption(updateRule),
			}
			if refTable != t.Name {
				fk.RefTable = &schema.Table{Name: refTable}
			}
			ids[id] = fk
			t.ForeignKeys = append(t.ForeignKeys, fk)
		}
		c, ok := t.Column(column)
		if !ok {
			return fmt.Errorf("column %q was not found for fk %q", column, fk.Symbol)
		}
		// Rows are ordered by SEQ that specifies the
		// position of the column in the FK definition.
		if _, ok := fk.Column(c.Name); !ok {
			fk.Columns = append(fk.Columns, c)
			c.ForeignKeys = append(c.ForeignKeys, fk)
		}

		// Stub referenced columns or link if it's a self-reference.
		var rc *schema.Column
		if fk.Table != fk.RefTable {
			rc = &schema.Column{Name: refColumn}
		} else if c, ok := t.Column(refColumn); ok {
			rc = c
		} else {
			return fmt.Errorf("referenced column %q was not found for fk %q", refColumn, fk.Symbol)
		}
		if _, ok := fk.RefColumn(rc.Name); !ok {
			fk.RefColumns = append(fk.RefColumns, rc)
		}
	}
	// TODO(a8m): extract the foreign-key name from the `CREATE TABLE` statement.
	return nil
}

// tableNames returns a list of all tables exist in the schema.
func (d *Driver) tables(ctx context.Context, opts *schema.InspectOptions) ([]*schema.Table, error) {
	var (
		args  []interface{}
		query = tablesQuery
	)
	if opts != nil && len(opts.Tables) > 0 {
		query += " AND name IN (" + strings.Repeat("?, ", len(opts.Tables)-1) + "?)"
		for _, s := range opts.Tables {
			args = append(args, s)
		}
	}
	rows, err := d.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: querying schema tables: %w", err)
	}
	defer rows.Close()
	var tables []*schema.Table
	for rows.Next() {
		var name, stmt string
		if err := rows.Scan(&name, &stmt); err != nil {
			return nil, fmt.Errorf("sqlite: scanning table: %w", err)
		}
		stmt = strings.TrimSpace(stmt)
		t := &schema.Table{
			Name: name,
			Attrs: []schema.Attr{
				&CreateStmt{S: strings.TrimSpace(stmt)},
			},
		}
		if strings.HasSuffix(stmt, "WITHOUT ROWID") || strings.HasSuffix(stmt, "without rowid") {
			t.Attrs = append(t.Attrs, &WithoutRowID{})
		}
		tables = append(tables, t)
	}
	return tables, nil
}

// schemas returns the list of the schemas in the database.
func (d *Driver) databases(ctx context.Context, opts *schema.InspectRealmOption) ([]*schema.Schema, error) {
	var (
		args  []interface{}
		query = databasesQuery
	)
	if opts != nil && len(opts.Schemas) > 0 {
		query += " name IN (" + strings.Repeat("?, ", len(opts.Schemas)-1) + "?)"
		for _, s := range opts.Schemas {
			args = append(args, s)
		}
	}
	rows, err := d.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: querying schemas: %w", err)
	}
	defer rows.Close()
	var schemas []*schema.Schema
	for rows.Next() {
		var name, file string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		schemas = append(schemas, &schema.Schema{
			Name:  name,
			Attrs: []schema.Attr{&File{Name: file}},
		})
	}
	return schemas, nil
}

type (
	// File describes a database file.
	File struct {
		schema.Attr
		Name string
	}

	// CreateStmt describes the SQL statement used to create a resource.
	CreateStmt struct {
		schema.Attr
		S string
	}

	// WithoutRowID describes the `WITHOUT ROWID` configuration.
	// See: https://sqlite.org/withoutrowid.html
	WithoutRowID struct {
		schema.Attr
	}

	// IndexPredicate describes a partial index predicate.
	// See: https://www.sqlite.org/partialindex.html
	IndexPredicate struct {
		schema.Attr
		P string
	}
)

func columnParts(t string) []string {
	t = strings.TrimSpace(strings.ToLower(t))
	parts := strings.FieldsFunc(t, func(r rune) bool {
		return r == '(' || r == ')' || r == ' ' || r == ','
	})
	// Join the type back if it was separated with space (e.g. 'varying character').
	if len(parts) > 1 && !isNumber(parts[0]) && !isNumber(parts[1]) {
		parts[1] = parts[0] + " " + parts[1]
		parts = parts[1:]
	}
	return parts
}

// isNumber reports whether the string is a number (category N).
func isNumber(s string) bool {
	for _, r := range s {
		if !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}

const (
	// Query to list attached database files.
	databasesQuery = "SELECT `name`, `file` FROM pragma_database_list()"
	// Query to list database tables.
	tablesQuery = "SELECT `name`, `sql` FROM sqlite_master WHERE type='table'"
	// Query to list table information.
	columnsQuery = "SELECT `name`, `type`, (not `notnull`) AS `nullable`, `dflt_value`, (`pk` <> 0) AS `pk`  FROM pragma_table_info('%s') ORDER BY `pk`, `cid`"
	// Query to list table indexes.
	indexesQuery = "SELECT `il`.`name`, `il`.`unique`, `il`.`origin`, `il`.`partial`, `m`.`sql` FROM pragma_index_list('%s') AS il JOIN sqlite_master AS m ON il.name = m.name"
	// Query to list index columns.
	indexColumnsQuery = "SELECT name FROM pragma_index_info('%s') ORDER BY seqno"
	// Query to list table foreign-keys.
	fksQuery = "SELECT `id`, `from`, `to`, `table`, `on_update`, `on_delete` FROM pragma_foreign_key_list('%s') ORDER BY id, seq"
)
