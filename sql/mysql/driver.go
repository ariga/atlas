package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"

	"golang.org/x/mod/semver"
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

// InspectRealm returns schema descriptions of all resources in the given realm.
func (d *Driver) InspectRealm(ctx context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	schemas, err := d.schemas(ctx, opts)
	if err != nil {
		return nil, err
	}
	realm := &schema.Realm{Schemas: schemas, Attrs: []schema.Attr{&schema.Charset{V: d.charset}, &schema.Collation{V: d.collate}}}
	for _, s := range schemas {
		names, err := d.tableNames(ctx, s.Name, nil)
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			t, err := d.inspectTable(ctx, name, &schema.InspectTableOptions{Schema: s.Name}, s)
			if err != nil {
				return nil, err
			}
			s.Tables = append(s.Tables, t)
		}
		s.Realm = realm
	}
	sqlx.LinkSchemaTables(schemas)
	return realm, nil
}

// InspectSchema returns schema descriptions of all tables in the given schema.
func (d *Driver) InspectSchema(ctx context.Context, name string, opts *schema.InspectOptions) (*schema.Schema, error) {
	schemas, err := d.schemas(ctx, &schema.InspectRealmOption{
		Schemas: []string{name},
	})
	if err != nil {
		return nil, err
	}
	if len(schemas) == 0 {
		return nil, &schema.NotExistError{
			Err: fmt.Errorf("mysql: schema %q was not found", name),
		}
	}
	names, err := d.tableNames(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	s := schemas[0]
	for _, name := range names {
		t, err := d.inspectTable(ctx, name, &schema.InspectTableOptions{Schema: s.Name}, s)
		if err != nil {
			return nil, err
		}
		s.Tables = append(s.Tables, t)
	}
	sqlx.LinkSchemaTables(schemas)
	s.Realm = &schema.Realm{Schemas: schemas, Attrs: []schema.Attr{&schema.Charset{V: d.charset}, &schema.Collation{V: d.collate}}}
	return s, nil
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
	var (
		args  = []interface{}{name}
		query = tableQuery
	)
	if opts != nil && opts.Schema != "" {
		query = tableSchemaQuery
		args = append(args, opts.Schema)
	}
	row := d.QueryRowContext(ctx, query, args...)
	var (
		autoinc                              sql.NullInt64
		tSchema, charset, collation, comment sql.NullString
	)
	if err := row.Scan(&tSchema, &charset, &collation, &autoinc, &comment); err != nil {
		if err == sql.ErrNoRows {
			return nil, &schema.NotExistError{
				Err: fmt.Errorf("mysql: table %q was not found", name),
			}
		}
		return nil, err
	}
	t := &schema.Table{Name: name, Schema: &schema.Schema{Name: tSchema.String}}
	if sqlx.ValidString(charset) {
		t.Attrs = append(t.Attrs, &schema.Charset{
			V: charset.String,
		})
	}
	if sqlx.ValidString(collation) {
		t.Attrs = append(t.Attrs, &schema.Collation{
			V: collation.String,
		})
	}
	if sqlx.ValidString(comment) {
		t.Attrs = append(t.Attrs, &schema.Comment{
			Text: comment.String,
		})
	}
	if autoinc.Valid {
		t.Attrs = append(t.Attrs, &AutoIncrement{
			V: autoinc.Int64,
		})
	}
	return t, nil
}

// columns queries and appends the columns of the given table.
func (d *Driver) columns(ctx context.Context, t *schema.Table) error {
	rows, err := d.QueryContext(ctx, columnsQuery, t.Schema.Name, t.Name)
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
	ct, err := parseRawType(c.Type.Raw)
	if err != nil {
		return err
	}
	c.Type.Type = ct
	if err := extraAttr(c, extra.String); err != nil {
		return err
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
	if key.String == "PRI" {
		if t.PrimaryKey == nil {
			t.PrimaryKey = &schema.Index{Name: key.String}
		}
		t.PrimaryKey.Parts = append(t.PrimaryKey.Parts, &schema.IndexPart{
			C:     c,
			SeqNo: len(t.PrimaryKey.Parts),
		})
	}
	return nil
}

func parseRawType(raw string) (schema.Type, error) {
	parts, size, unsigned, err := parseColumn(raw)
	if err != nil {
		return nil, err
	}
	switch t := parts[0]; t {
	case tBit:
		return &BitType{
			T: t,
		}, nil
	case tTinyInt, tSmallInt, tMediumInt, tInt, tBigInt:
		if size == 1 {
			return &schema.BoolType{
				T: t,
			}, nil
		}
		byteSize, err := intByteSize(t)
		if err != nil {
			return nil, err
		}
		// For integer types, the size represents the display width and does not
		// constrain the range of values that can be stored in the column.
		// The storage byte-size is inferred from the type name (i.e TINYINT takes
		// a single byte).
		ft := &schema.IntegerType{
			T:        t,
			Size:     byteSize,
			Unsigned: unsigned,
		}
		if attr := parts[len(parts)-1]; attr == "zerofill" && size != 0 {
			ft.Attrs = []schema.Attr{
				&DisplayWidth{
					N: int(size),
				},
				&ZeroFill{
					A: attr,
				},
			}
		}
		return ft, nil
	case tNumeric, tDecimal:
		dt := &schema.DecimalType{
			T: t,
		}
		if len(parts) > 1 {
			p, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse precision %q", parts[1])
			}
			dt.Precision = int(p)
		}
		if len(parts) > 2 {
			s, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse scale %q", parts[1])
			}
			dt.Scale = int(s)
		}
		return dt, nil
	case tFloat, tDouble, tReal:
		ft := &schema.FloatType{
			T: t,
		}
		if len(parts) > 1 {
			p, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse precision %q", parts[1])
			}
			ft.Precision = int(p)
		}
		return ft, nil
	case tBinary, tVarBinary:
		return &schema.BinaryType{
			T:    t,
			Size: int(size),
		}, nil
	case tTinyBlob, tMediumBlob, tBlob, tLongBlob:
		return &schema.BinaryType{
			T: t,
		}, nil
	case tChar, tVarchar:
		return &schema.StringType{
			T:    t,
			Size: int(size),
		}, nil
	case tTinyText, tMediumText, tText, tLongText:
		return &schema.StringType{
			T: t,
		}, nil
	case tEnum, tSet:
		values := make([]string, len(parts)-1)
		for i, e := range parts[1:] {
			values[i] = strings.Trim(e, "'")
		}
		if t == tEnum {
			return &schema.EnumType{
				Values: values,
			}, nil
		} else {
			return &SetType{
				Values: values,
			}, nil
		}
	case tDate, tDateTime, tTime, tTimestamp, tYear:
		return &schema.TimeType{
			T: t,
		}, nil
	case tJSON:
		return &schema.JSONType{
			T: t,
		}, nil
	case tPoint, tMultiPoint, tLineString, tMultiLineString, tPolygon, tMultiPolygon, tGeometry, tGeoCollection, tGeometryCollection:
		return &schema.SpatialType{
			T: t,
		}, nil
	default:
		return &schema.UnsupportedType{
			T: t,
		}, nil
	}
}

// indexes queries and appends the indexes of the given table.
func (d *Driver) indexes(ctx context.Context, t *schema.Table) error {
	query := indexesQuery
	if d.supportsIndexExpr() {
		query = indexesExprQuery
	}
	rows, err := d.QueryContext(ctx, query, t.Schema.Name, t.Name)
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
			nonuniq                                 bool
			seqno                                   int
			name, indexType                         string
			column, subPart, expr, comment, collate sql.NullString
		)
		if err := rows.Scan(&name, &column, &nonuniq, &seqno, &indexType, &collate, &comment, &subPart, &expr); err != nil {
			return fmt.Errorf("mysql: scanning index: %w", err)
		}
		// Ignore primary keys.
		if name == "PRIMARY" {
			continue
		}
		idx, ok := names[name]
		if !ok {
			idx = &schema.Index{
				Name:   name,
				Unique: !nonuniq,
				Table:  t,
				Attrs: []schema.Attr{
					&IndexType{T: indexType},
				},
			}
			if sqlx.ValidString(comment) {
				idx.Attrs = append(t.Attrs, &schema.Comment{
					Text: comment.String,
				})
			}
			names[name] = idx
			t.Indexes = append(t.Indexes, idx)
		}
		// Rows are ordered by SEQ_IN_INDEX that specifies the
		// position of the column in the index definition.
		part := &schema.IndexPart{
			SeqNo: seqno,
			Attrs: []schema.Attr{&schema.Collation{V: collate.String}},
		}
		switch {
		case sqlx.ValidString(expr):
			part.X = &schema.RawExpr{
				X: expr.String,
			}
		case sqlx.ValidString(column):
			part.C, ok = t.Column(column.String)
			if !ok {
				return fmt.Errorf("mysql: column %q was not found for index %q", column.String, idx.Name)
			}
			if sqlx.ValidString(subPart) {
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
	rows, err := d.QueryContext(ctx, fksQuery, t.Schema.Name, t.Name)
	if err != nil {
		return fmt.Errorf("mysql: querying %q foreign keys: %w", t.Name, err)
	}
	defer rows.Close()
	if err := sqlx.ScanFKs(t, rows); err != nil {
		return fmt.Errorf("mysql: %w", err)
	}
	return rows.Err()
}

// checks queries and appends the check constraints of the given table.
func (d *Driver) checks(ctx context.Context, t *schema.Table) error {
	rows, err := d.QueryContext(ctx, checksQuery, t.Schema.Name, t.Name)
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
func (d *Driver) tableNames(ctx context.Context, schema string, opts *schema.InspectOptions) ([]string, error) {
	query, args := tablesQuery, []interface{}{schema}
	if opts != nil && len(opts.Tables) > 0 {
		query += " AND `TABLE_NAME` IN (" + strings.Repeat("?, ", len(opts.Tables)-1) + "?)"
		for _, s := range opts.Tables {
			args = append(args, s)
		}
	}
	rows, err := d.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mysql: querying schema tables: %w", err)
	}
	names, err := sqlx.ScanStrings(rows)
	if err != nil {
		return nil, fmt.Errorf("mysql: scanning table names: %w", err)
	}
	return names, nil
}

// supportsCheck reports if connected MySQL database supports the CHECK clause.
func (d *Driver) supportsCheck() bool { return semver.Compare("v"+d.version, "v8.0.16") != -1 }

// supportsIndexExpr reports if connected MySQL database supports index expressions (functional key part).
func (d *Driver) supportsIndexExpr() bool { return semver.Compare("v"+d.version, "v8.0.13") != -1 }

// parseColumn returns column parts, size and signed-info from a MySQL type.
func parseColumn(typ string) (parts []string, size int64, unsigned bool, err error) {
	switch parts = strings.FieldsFunc(typ, func(r rune) bool {
		return r == '(' || r == ')' || r == ' ' || r == ','
	}); parts[0] {
	case tTinyInt, tSmallInt, tMediumInt, tInt, tBigInt:
		if attr := parts[len(parts)-1]; attr == "unsigned" || attr == "zerofill" {
			unsigned = true
		}
		if len(parts) > 2 || len(parts) == 2 && !unsigned {
			size, err = strconv.ParseInt(parts[1], 10, 64)
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
	case "default_generated":
		// The column has an expression default value
		// and it's handled in Driver.addColumn.
	case "auto_increment":
		c.Attrs = append(c.Attrs, &AutoIncrement{A: extra})
	case "default_generated on update current_timestamp", "on update current_timestamp":
		c.Attrs = append(c.Attrs, &OnUpdate{A: extra})
	default:
		return fmt.Errorf("unknown attribute %q", extra)
	}
	return nil
}

const (
	// Query to list system variables.
	variablesQuery = "SELECT @@version, @@collation_server, @@character_set_server"

	// Query to list database schemas.
	schemasQuery = "SELECT `SCHEMA_NAME`, `DEFAULT_CHARACTER_SET_NAME`, `DEFAULT_COLLATION_NAME` from `INFORMATION_SCHEMA`.`SCHEMATA`"

	// Query to list schema tables.
	tablesQuery = "SELECT `TABLE_NAME` FROM `INFORMATION_SCHEMA`.`TABLES` WHERE `TABLE_SCHEMA` = ?"

	// Query to list table columns.
	columnsQuery = "SELECT `COLUMN_NAME`, `COLUMN_TYPE`, `COLUMN_COMMENT`, `IS_NULLABLE`, `COLUMN_KEY`, `COLUMN_DEFAULT`, `EXTRA`, `CHARACTER_SET_NAME`, `COLLATION_NAME` FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?"

	// Query to list table indexes.
	indexesQuery     = "SELECT `INDEX_NAME`, `COLUMN_NAME`, `NON_UNIQUE`, `SEQ_IN_INDEX`, `INDEX_TYPE`, `COLLATION`, `INDEX_COMMENT`, `SUB_PART`, NULL AS `EXPRESSION` FROM `INFORMATION_SCHEMA`.`STATISTICS` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ? ORDER BY `index_name`, `seq_in_index`"
	indexesExprQuery = "SELECT `INDEX_NAME`, `COLUMN_NAME`, `NON_UNIQUE`, `SEQ_IN_INDEX`, `INDEX_TYPE`, `COLLATION`, `INDEX_COMMENT`, `SUB_PART`, `EXPRESSION` FROM `INFORMATION_SCHEMA`.`STATISTICS` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ? ORDER BY `index_name`, `seq_in_index`"

	// Query to list table information.
	tableQuery = `
SELECT
	t1.TABLE_SCHEMA,
	t2.CHARACTER_SET_NAME,
	t1.TABLE_COLLATION,
	t1.AUTO_INCREMENT,
	t1.TABLE_COMMENT
FROM
	INFORMATION_SCHEMA.TABLES AS t1
	JOIN INFORMATION_SCHEMA.COLLATIONS AS t2
	ON t1.TABLE_COLLATION = t2.COLLATION_NAME
WHERE
	TABLE_NAME = ?
	AND TABLE_SCHEMA = (SELECT DATABASE())
`
	tableSchemaQuery = `
SELECT
	t1.TABLE_SCHEMA,
	t2.CHARACTER_SET_NAME,
	t1.TABLE_COLLATION,
	t1.AUTO_INCREMENT,
	t1.TABLE_COMMENT
FROM
	INFORMATION_SCHEMA.TABLES AS t1
	JOIN INFORMATION_SCHEMA.COLLATIONS AS t2
	ON t1.TABLE_COLLATION = t2.COLLATION_NAME
WHERE
	TABLE_NAME = ?
	AND TABLE_SCHEMA = ?
`

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
		V int64
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

	// The DisplayWidth represents a display width of an integer type.
	DisplayWidth struct {
		schema.Attr
		N int
	}

	// The ZeroFill represents the ZEROFILL attribute which is
	// deprecated for MySQL version >= 8.0.17.
	ZeroFill struct {
		schema.Attr
		A string
	}

	// IndexType represents an index type.
	IndexType struct {
		schema.Attr
		T string // BTREE, FULLTEXT, HASH, RTREE
	}

	// BitType represents a bit type.
	BitType struct {
		schema.Type
		T string
	}

	// SetType represents a set type.
	SetType struct {
		schema.Type
		Values []string
	}
)

// convert int type name to storage size in bytes, sizes taken from:
// https://dev.mysql.com/doc/refman/8.0/en/integer-types.html
func intByteSize(t string) (uint8, error) {
	switch t {
	case tTinyInt:
		return 1, nil
	case tSmallInt:
		return 2, nil
	case tMediumInt:
		return 3, nil
	case tInt:
		return 4, nil
	case tBigInt:
		return 8, nil
	}
	return 0, fmt.Errorf("mysql: unsupported integer type %q", t)
}

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
