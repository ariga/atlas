package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"

	"ariga.io/atlas/sql/schema"
)

// Driver represents a MySQL driver for introspecting database schemas
// and apply migrations changes on them.
type Driver struct {
	schema.ExecQuerier
	// System variables that are set on `Open`.
	version string
	collate string
	charset string
}

// Open opens a new MySQL driver.
func Open(db schema.ExecQuerier) (*Driver, error) {
	drv := &Driver{ExecQuerier: db}
	if err := db.QueryRow(variablesQuery).Scan(&drv.version, &drv.collate, &drv.charset); err != nil {
		return nil, fmt.Errorf("mysql: scanning system variables: %w", err)
	}
	return drv, nil
}

// Realm returns schema descriptions of all resources in the given realm.
func (d *Driver) Realm(ctx context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	schemas, err := d.schemas(ctx, opts)
	if err != nil {
		return nil, err
	}
	for _, s := range schemas {
		tables, err := d.Tables(ctx, &schema.InspectTableOptions{
			Schema: s.Name,
		})
		if err != nil {
			return nil, err
		}
		s.Tables = tables
	}
	linkSchemaTables(schemas)
	return &schema.Realm{
		Schemas: schemas,
		Attrs:   []schema.Attr{&schema.Charset{V: d.charset}, &schema.Collation{V: d.collate}},
	}, nil
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
	if d.supportsCheck() {
		if err := d.checks(ctx, t); err != nil {
			return nil, err
		}
	}
	return t, nil
}

// schemas returns the list of the schemas in the database.
func (d *Driver) schemas(ctx context.Context, opts *schema.InspectRealmOption) ([]*schema.Schema, error) {
	var (
		args  []interface{}
		query = schemasQuery
	)
	if opts != nil && len(opts.Schemas) > 0 {
		query += " WHERE `SCHEMA_NAME` IN (" + strings.Repeat("?, ", len(opts.Schemas)-1) + "?)"
		for _, s := range opts.Schemas {
			args = append(args, s)
		}
	} else {
		query += " WHERE `SCHEMA_NAME` NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')"
	}
	rows, err := d.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mysql: querying schemas: %w", err)
	}
	defer rows.Close()
	var schemas []*schema.Schema
	for rows.Next() {
		var name, charset, collation string
		if err := rows.Scan(&name, &charset, &collation); err != nil {
			return nil, err
		}
		schemas = append(schemas, &schema.Schema{
			Name: name,
			Attrs: []schema.Attr{
				&schema.Charset{
					V: charset,
				},
				&schema.Collation{
					V: collation,
				},
			},
		})
	}
	return schemas, nil
}

// table returns the table from the database, or a NotExistError if the table was not found.
func (d *Driver) table(ctx context.Context, name string, opts *schema.InspectTableOptions) (*schema.Table, error) {
	query, args := tableQuery, []interface{}{name}
	if opts != nil && opts.Schema != "" {
		query, args = tableSchemaQuery, []interface{}{opts.Schema, name}
	}
	row := d.QueryRowContext(ctx, query, args...)
	var tSchema, collation sql.NullString
	if err := row.Scan(&tSchema, &collation); err != nil {
		if err == sql.ErrNoRows {
			return nil, &schema.NotExistError{
				Err: fmt.Errorf("mysql: table %q was not found", name),
			}
		}
		return nil, err
	}
	t := &schema.Table{Name: name, Schema: tSchema.String}
	if validString(collation) {
		t.Attrs = append(t.Attrs, &schema.Collation{
			V: collation.String,
		})
	}
	return t, nil
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
	var name, typ, comment, nullable, key, defaults, extra, charset, collation sql.NullString
	if err := rows.Scan(&name, &typ, &comment, &nullable, &key, &defaults, &extra, &charset, &collation); err != nil {
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
	case tBit:
		c.Type.Type = &BitType{
			T:    t,
			Size: int(size),
		}
	case tTinyInt, tSmallInt, tMediumInt, tInt, tBigInt:
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
	case tNumeric, tDecimal:
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
	case tFloat, tDouble, tReal:
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
	case tBinary, tVarBinary:
		c.Type.Type = &schema.BinaryType{
			T:    t,
			Size: int(size),
		}
	case tTinyBlob, tMediumBlob, tBlob, tLongBlob:
		c.Type.Type = &schema.BinaryType{
			T: t,
		}
	case tChar, tVarchar:
		c.Type.Type = &schema.StringType{
			T:    t,
			Size: int(size),
		}
	case tTinyText, tMediumText, tText, tLongText:
		c.Type.Type = &schema.StringType{
			T: t,
		}
	case tEnum, tSet:
		values := make([]string, len(parts)-1)
		for i, e := range parts[1:] {
			values[i] = strings.Trim(e, "'")
		}
		if t == tEnum {
			c.Type.Type = &schema.EnumType{
				Values: values,
			}
		} else {
			c.Type.Type = &SetType{
				Values: values,
			}
		}
	case tDate, tDateTime, tTime, tTimestamp, tYear:
		c.Type.Type = &schema.TimeType{
			T: t,
		}
	case tJSON:
		c.Type.Type = &schema.JSONType{
			T: t,
		}
	case tPoint, tMultiPoint, tLineString, tMultiLineString, tPolygon, tMultiPolygon, tGeometry, tGeoCollection, tGeometryCollection:
		c.Type.Type = &schema.SpatialType{
			T: t,
		}
	default:
		c.Type.Type = &schema.UnsupportedType{
			T: t,
		}
	}
	if err := extraAttr(c, extra.String); err != nil {
		return err
	}
	if validString(defaults) {
		c.Default = &schema.RawExpr{
			X: defaults.String,
		}
	}
	if validString(comment) {
		c.Attrs = append(c.Attrs, &schema.Comment{
			Text: comment.String,
		})
	}
	if validString(charset) {
		c.Attrs = append(c.Attrs, &schema.Charset{
			V: charset.String,
		})
	}
	if validString(collation) {
		c.Attrs = append(c.Attrs, &schema.Collation{
			V: collation.String,
		})
	}
	t.Columns = append(t.Columns, c)
	if key.String == "PRI" {
		t.PrimaryKey = append(t.PrimaryKey, c)
	}
	return nil
}

// indexes queries and appends the indexes of the given table.
func (d *Driver) indexes(ctx context.Context, t *schema.Table) error {
	query := indexesQuery
	if d.supportsIndexExpr() {
		query = indexesExprQuery
	}
	rows, err := d.QueryContext(ctx, query, t.Schema, t.Name)
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
			nonuniq               bool
			seqno                 int
			name                  string
			column, subPart, expr sql.NullString
		)
		if err := rows.Scan(&name, &column, &nonuniq, &seqno, &subPart, &expr); err != nil {
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
		// Rows are ordered by SEQ_IN_INDEX that specifies the
		// position of the column in the index definition.
		part := &schema.IndexPart{SeqNo: seqno}
		switch {
		case validString(expr):
			part.X = &schema.RawExpr{
				X: expr.String,
			}
		case validString(column):
			part.C, ok = t.Column(column.String)
			if !ok {
				return fmt.Errorf("mysql: column %q was not found for index %q", column.String, idx.Name)
			}
			if validString(subPart) {
				n, err := strconv.Atoi(subPart.String)
				if err != nil {
					return fmt.Errorf("mysql: parse index prefix size %q: %w", subPart.String, err)
				}
				part.Attrs = append(part.Attrs, &SubPart{
					Len: n,
				})
			}
			part.C.Indexes = append(part.C.Indexes, idx)
		default:
			return fmt.Errorf("mysql: invalid part for index %q", idx.Name)
		}
		idx.Parts = append(idx.Parts, part)
	}
	return nil
}

// fks queries and appends the foreign keys of the given table.
func (d *Driver) fks(ctx context.Context, t *schema.Table) error {
	rows, err := d.QueryContext(ctx, fksQuery, t.Schema, t.Name)
	if err != nil {
		return fmt.Errorf("mysql: querying %q foreign keys: %w", t.Name, err)
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

// checks queries and appends the check constraints of the given table.
func (d *Driver) checks(ctx context.Context, t *schema.Table) error {
	rows, err := d.QueryContext(ctx, checksQuery, t.Schema, t.Name)
	if err != nil {
		return fmt.Errorf("mysql: querying %q check constraints: %w", t.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		var name, enforced, clause string
		if err := rows.Scan(&name, &enforced, &clause); err != nil {
			return fmt.Errorf("mysql: %w", err)
		}
		t.Attrs = append(t.Attrs, &Check{
			Name:     name,
			Clause:   unescape(clause),
			Enforced: clause != "NO",
		})

	}
	return rows.Err()
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

// Exec executes the changes on the database. An error is returned
// if one of the operations fail, or a change is not supported.
func (d *Driver) Exec(ctx context.Context, changes []schema.Change) (err error) {
	for _, c := range changes {
		switch c := c.(type) {
		case *schema.AddTable:
			err = d.addTable(ctx, c)
		case *schema.DropTable:
			err = d.dropTable(ctx, c)
		default:
			err = fmt.Errorf("mysql: unsupported change %T", c)
		}
		if err != nil {
			return err
		}
	}
	return
}

func (d *Driver) addTable(ctx context.Context, add *schema.AddTable) error {
	var b strings.Builder
	b.WriteString("CREATE TABLE ")
	ident(&b, add.T.Name)
	b.WriteString("(")
	for i, c := range add.T.Columns {
		if i > 0 {
			b.WriteString(", ")
		}
		ident(&b, c.Name)
		b.WriteByte(' ')
		b.WriteString(c.Type.Raw)
		b.WriteByte(' ')
		if !c.Type.Null {
			b.WriteString("NOT ")
		}
		b.WriteString("NULL")
	}
	if len(add.T.PrimaryKey) > 0 {
		b.WriteString(", PRIMARY KEY(")
		for i, c := range add.T.PrimaryKey {
			if i > 0 {
				b.WriteString(", ")
			}
			ident(&b, c.Name)
		}
		b.WriteByte(')')
	}
	b.WriteString(")")
	if _, err := d.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("mysql: create table: %w", err)
	}
	return nil
}

func (d *Driver) dropTable(ctx context.Context, drop *schema.DropTable) error {
	var b strings.Builder
	b.WriteString("DROP TABLE ")
	if drop.T.Schema != "" {
		ident(&b, drop.T.Schema)
		b.WriteByte('.')
	}
	ident(&b, drop.T.Name)
	if _, err := d.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("mysql: drop table: %w", err)
	}
	return nil
}

// supportsCheck reports if connected MySQL database supports the CHECK clause.
func (d *Driver) supportsCheck() bool { return semver.Compare("v"+d.version, "v8.0.16") != -1 }

// supportsIndexExpr reports if connected MySQL database supports index expressions (functional key part).
func (d *Driver) supportsIndexExpr() bool { return semver.Compare("v"+d.version, "v8.0.13") != -1 }

// ident writes the given identifier in MySQL format.
func ident(b *strings.Builder, ident string) {
	b.WriteByte('`')
	b.WriteString(ident)
	b.WriteByte('`')
}

// parseColumn returns column parts, size and signed-info from a MySQL type.
func parseColumn(typ string) (parts []string, size int64, unsigned bool, err error) {
	switch parts = strings.FieldsFunc(typ, func(r rune) bool {
		return r == '(' || r == ')' || r == ' ' || r == ','
	}); parts[0] {
	case tTinyInt, tSmallInt, tMediumInt, tInt, tBigInt:
		switch {
		case len(parts) == 2 && parts[1] == "unsigned": // int unsigned
			unsigned = true
		case len(parts) == 3: // int(10) unsigned
			unsigned = true
			fallthrough
		case len(parts) == 2: // int(10)
			size, err = strconv.ParseInt(parts[1], 10, 0)
		}
	case tBinary, tVarBinary, tChar, tVarchar:
		size, err = strconv.ParseInt(parts[1], 10, 64)
	}
	if err != nil {
		return nil, 0, false, fmt.Errorf("parse %q to int: %w", parts[1], err)
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

// validString reports if the given string is valid and not nullable.
func validString(s sql.NullString) bool {
	return s.Valid && s.String != "" && strings.ToLower(s.String) != "null"
}

const (
	// Query to list system variables.
	variablesQuery = "SELECT @@version, @@collation_server, @@character_set_server"

	// Query to list database schemas.
	schemasQuery = "SELECT `SCHEMA_NAME`, `DEFAULT_CHARACTER_SET_NAME`, `DEFAULT_COLLATION_NAME` from `INFORMATION_SCHEMA`.`SCHEMATA`"

	// Queries to list schema tables.
	tablesSchemaQuery = "SELECT `TABLE_NAME` FROM `INFORMATION_SCHEMA`.`TABLES` WHERE `TABLE_SCHEMA` = ?"
	tablesQuery       = "SELECT `TABLE_NAME` FROM `INFORMATION_SCHEMA`.`TABLES` WHERE `TABLE_SCHEMA` = (SELECT DATABASE())"

	// Queries to fetch table schema.
	tableQuery       = "SELECT `TABLE_SCHEMA`, `TABLE_COLLATION` FROM `INFORMATION_SCHEMA`.`TABLES` WHERE `TABLE_SCHEMA` = (SELECT DATABASE()) AND `TABLE_NAME` = ?"
	tableSchemaQuery = "SELECT `TABLE_SCHEMA`, `TABLE_COLLATION` FROM `INFORMATION_SCHEMA`.`TABLES` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?"

	// Query to list table columns.
	columnsQuery = "SELECT `COLUMN_NAME`, `COLUMN_TYPE`, `COLUMN_COMMENT`, `IS_NULLABLE`, `COLUMN_KEY`, `COLUMN_DEFAULT`, `EXTRA`, `CHARACTER_SET_NAME`, `COLLATION_NAME` FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?"

	// Query to list table indexes.
	indexesQuery     = "SELECT `INDEX_NAME`, `COLUMN_NAME`, `NON_UNIQUE`, `SEQ_IN_INDEX`, `SUB_PART`, NULL AS `EXPRESSION` FROM `INFORMATION_SCHEMA`.`STATISTICS` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ? ORDER BY `index_name`, `seq_in_index`"
	indexesExprQuery = "SELECT `INDEX_NAME`, `COLUMN_NAME`, `NON_UNIQUE`, `SEQ_IN_INDEX`, `SUB_PART`, `EXPRESSION` FROM `INFORMATION_SCHEMA`.`STATISTICS` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ? ORDER BY `index_name`, `seq_in_index`"

	// Query to list table check constraints.
	checksQuery = `
SELECT
  t1.CONSTRAINT_NAME,
  t1.ENFORCED,
  t2.CHECK_CLAUSE
FROM
  INFORMATION_SCHEMA.TABLE_CONSTRAINTS AS t1
  JOIN INFORMATION_SCHEMA.CHECK_CONSTRAINTS AS t2
  ON t1.CONSTRAINT_NAME = t2.CONSTRAINT_NAME
WHERE
  t1.CONSTRAINT_TYPE = 'CHECK'
  AND t1.TABLE_SCHEMA = ?
  AND t1.TABLE_NAME = ?
ORDER BY
  t1.CONSTRAINT_NAME
`

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

	// SubPart attribute defines an option index prefix length for columns.
	SubPart struct {
		schema.Attr
		Len int
	}

	// Check attributes defines a CHECK constraint.
	Check struct {
		schema.Attr
		Name     string
		Clause   string
		Enforced bool
	}

	// BitType represents a bit type.
	BitType struct {
		schema.Type
		T    string
		Size int
	}

	// SetType represents a set type.
	SetType struct {
		schema.Type
		Values []string
	}
)

// MySQL standard unescape field function from its codebase:
// https://github.com/mysql/mysql-server/blob/8.0/sql/dd/impl/utils.cc
func unescape(s string) string {
	var b strings.Builder
	for i, c := range s {
		if c != '\\' || i+1 < len(s) && s[i+1] != '\\' && s[i+1] != '=' && s[i+1] != ';' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// MySQL standard column types as defined in its codebase. Name and order
// is organized differently than MySQL.
//
// https://github.com/mysql/mysql-server/blob/8.0/include/field_types.h
// https://github.com/mysql/mysql-server/blob/8.0/sql/dd/types/column.h
// https://github.com/mysql/mysql-server/blob/8.0/sql/sql_show.cc
// https://github.com/mysql/mysql-server/blob/8.0/sql/gis/geometries.cc
const (
	tBit       = "bit"       // MYSQL_TYPE_BIT
	tInt       = "int"       // MYSQL_TYPE_LONG
	tTinyInt   = "tinyint"   // MYSQL_TYPE_TINY
	tSmallInt  = "smallint"  // MYSQL_TYPE_SHORT
	tMediumInt = "mediumint" // MYSQL_TYPE_INT24
	tBigInt    = "bigint"    // MYSQL_TYPE_LONGLONG

	tDecimal = "decimal" // MYSQL_TYPE_DECIMAL
	tNumeric = "numeric" // MYSQL_TYPE_DECIMAL (numeric_type rule in sql_yacc.yy)
	tFloat   = "float"   // MYSQL_TYPE_FLOAT
	tDouble  = "double"  // MYSQL_TYPE_DOUBLE
	tReal    = "real"    // MYSQL_TYPE_FLOAT or MYSQL_TYPE_DOUBLE (real_type in sql_yacc.yy)

	tTimestamp = "timestamp" // MYSQL_TYPE_TIMESTAMP
	tDate      = "date"      // MYSQL_TYPE_DATE
	tTime      = "time"      // MYSQL_TYPE_TIME
	tDateTime  = "datetime"  // MYSQL_TYPE_DATETIME
	tYear      = "year"      // MYSQL_TYPE_YEAR

	tVarchar    = "varchar"    // MYSQL_TYPE_VAR_STRING, MYSQL_TYPE_VARCHAR
	tChar       = "char"       // MYSQL_TYPE_STRING
	tVarBinary  = "varbinary"  // MYSQL_TYPE_VAR_STRING + NULL CHARACTER_SET.
	tBinary     = "binary"     // MYSQL_TYPE_STRING + NULL CHARACTER_SET.
	tBlob       = "blob"       // MYSQL_TYPE_BLOB
	tTinyBlob   = "tinyblob"   // MYSQL_TYPE_TINYBLOB
	tMediumBlob = "mediumblob" // MYSQL_TYPE_MEDIUM_BLOB
	tLongBlob   = "longblob"   // MYSQL_TYPE_LONG_BLOB
	tText       = "text"       // MYSQL_TYPE_BLOB + CHARACTER_SET utf8mb4
	tTinyText   = "tinytext"   // MYSQL_TYPE_TINYBLOB + CHARACTER_SET utf8mb4
	tMediumText = "mediumtext" // MYSQL_TYPE_MEDIUM_BLOB + CHARACTER_SET utf8mb4
	tLongText   = "longtext"   // MYSQL_TYPE_LONG_BLOB with + CHARACTER_SET utf8mb4

	tEnum = "enum" // MYSQL_TYPE_ENUM
	tSet  = "set"  // MYSQL_TYPE_SET
	tJSON = "json" // MYSQL_TYPE_JSON

	tGeometry           = "geometry"           // MYSQL_TYPE_GEOMETRY
	tPoint              = "point"              // Geometry_type::kPoint
	tMultiPoint         = "multipoint"         // Geometry_type::kMultipoint
	tLineString         = "linestring"         // Geometry_type::kLinestring
	tMultiLineString    = "multilinestring"    // Geometry_type::kMultilinestring
	tPolygon            = "polygon"            // Geometry_type::kPolygon
	tMultiPolygon       = "multipolygon"       // Geometry_type::kMultipolygon
	tGeoCollection      = "geomcollection"     // Geometry_type::kGeometrycollection
	tGeometryCollection = "geometrycollection" // Geometry_type::kGeometrycollection
)
