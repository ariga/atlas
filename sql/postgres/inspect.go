// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// A diff provides a PostgreSQL implementation for schema.Inspector.
type inspect struct{ conn }

var _ schema.Inspector = (*inspect)(nil)

// InspectRealm returns schema descriptions of all resources in the given realm.
func (i *inspect) InspectRealm(ctx context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	schemas, err := i.schemas(ctx, opts)
	if err != nil {
		return nil, err
	}
	realm := &schema.Realm{Schemas: schemas, Attrs: []schema.Attr{&schema.Collation{V: i.collate}, &CType{V: i.ctype}}}
	for _, s := range schemas {
		names, err := i.tableNames(ctx, s.Name, nil)
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			t, err := i.inspectTable(ctx, name, &schema.InspectTableOptions{Schema: s.Name}, s)
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
func (i *inspect) InspectSchema(ctx context.Context, name string, opts *schema.InspectOptions) (*schema.Schema, error) {
	schemas, err := i.schemas(ctx, &schema.InspectRealmOption{
		Schemas: []string{name},
	})
	if err != nil {
		return nil, err
	}
	if len(schemas) == 0 {
		return nil, &schema.NotExistError{
			Err: fmt.Errorf("postgres: schema %q was not found", name),
		}
	}
	names, err := i.tableNames(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	s := schemas[0]
	for _, name := range names {
		t, err := i.inspectTable(ctx, name, &schema.InspectTableOptions{Schema: s.Name}, s)
		if err != nil {
			return nil, err
		}
		s.Tables = append(s.Tables, t)
	}
	sqlx.LinkSchemaTables(schemas)
	s.Realm = &schema.Realm{Schemas: schemas, Attrs: []schema.Attr{&schema.Collation{V: i.collate}, &CType{V: i.ctype}}}
	return s, nil
}

// InspectTable returns the schema description of the given table.
func (i *inspect) InspectTable(ctx context.Context, name string, opts *schema.InspectTableOptions) (*schema.Table, error) {
	return i.inspectTable(ctx, name, opts, nil)
}

func (i *inspect) inspectTable(ctx context.Context, name string, opts *schema.InspectTableOptions, top *schema.Schema) (*schema.Table, error) {
	t, err := i.table(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	if top != nil {
		// Link the table to its top element if provided.
		t.Schema = top
	}
	if err := i.columns(ctx, t); err != nil {
		return nil, err
	}
	if err := i.indexes(ctx, t); err != nil {
		return nil, err
	}
	if err := i.fks(ctx, t); err != nil {
		return nil, err
	}
	if err := i.checks(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// table returns the table from the database, or a NotExistError if the table was not found.
func (i *inspect) table(ctx context.Context, name string, opts *schema.InspectTableOptions) (*schema.Table, error) {
	var (
		args  = []interface{}{name}
		query = tableQuery
	)
	if opts != nil && opts.Schema != "" {
		query = tableSchemaQuery
		args = append(args, opts.Schema)
	}
	row := i.QueryRowContext(ctx, query, args...)
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
func (i *inspect) columns(ctx context.Context, t *schema.Table) error {
	rows, err := i.QueryContext(ctx, columnsQuery, t.Schema.Name, t.Name)
	if err != nil {
		return fmt.Errorf("postgres: querying %q columns: %w", t.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := i.addColumn(t, rows); err != nil {
			return fmt.Errorf("postgres: %w", err)
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := i.enumValues(ctx, t.Columns); err != nil {
		return err
	}
	return nil
}

// addColumn scans the current row and adds a new column from it to the table.
func (i *inspect) addColumn(t *schema.Table, rows *sql.Rows) error {
	var (
		typid, maxlen, precision, scale                                                                sql.NullInt64
		name, typ, nullable, defaults, udt, identity, generation, charset, collation, comment, typtype sql.NullString
	)
	if err := rows.Scan(&name, &typ, &nullable, &defaults, &maxlen, &precision, &scale, &charset, &collation, &udt, &identity, &generation, &comment, &typtype, &typid); err != nil {
		return err
	}
	c := &schema.Column{
		Name: name.String,
		Type: &schema.ColumnType{
			Raw:  typ.String,
			Null: nullable.String == "YES",
		},
	}
	c.Type.Type = columnType(&columnDesc{
		typ:       typ.String,
		size:      maxlen.Int64,
		udt:       udt.String,
		precision: precision.Int64,
		scale:     scale.Int64,
		typtype:   typtype.String,
		typid:     typid.Int64,
	})
	if sqlx.ValidString(defaults) {
		c.Default = defaultExpr(c, defaults.String)
	}
	if identity.String == "YES" {
		c.Attrs = append(c.Attrs, &Identity{
			Generation: generation.String,
		})
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

func columnType(c *columnDesc) schema.Type {
	var typ schema.Type
	switch t := c.typ; strings.ToLower(t) {
	case tBigInt, tInt8, tInt, tInteger, tInt4, tSmallInt, tInt2:
		typ = &schema.IntegerType{T: t}
	case tBit, tBitVar:
		typ = &BitType{T: t, Len: c.size}
	case tBool, tBoolean:
		typ = &schema.BoolType{T: t}
	case tBytea:
		typ = &schema.BinaryType{T: t}
	case tCharacter, tChar, tCharVar, tVarChar, tText:
		// A `character` column without length specifier is equivalent to `character(1)`,
		// but `varchar` without length accepts strings of any size (same as `text`).
		typ = &schema.StringType{T: t, Size: int(c.size)}
	case tCIDR, tInet, tMACAddr, tMACAddr8:
		typ = &NetworkType{T: t}
	case tCircle, tLine, tLseg, tBox, tPath, tPolygon, tPoint:
		typ = &schema.SpatialType{T: t}
	case tDate, tTime, tTimeWTZ, tTimeWOTZ,
		tTimestamp, tTimestampTZ, tTimestampWTZ, tTimestampWOTZ:
		typ = &schema.TimeType{T: t}
	case tInterval:
		// TODO: get 'interval_type' from query above before implementing.
		typ = &schema.UnsupportedType{T: t}
	case tReal, tDouble, tFloat4, tFloat8:
		typ = &schema.FloatType{T: t, Precision: int(c.precision)}
	case tJSON, tJSONB:
		typ = &schema.JSONType{T: t}
	case tMoney:
		typ = &CurrencyType{T: t}
	case tDecimal, tNumeric:
		typ = &schema.DecimalType{T: t, Precision: int(c.precision), Scale: int(c.scale)}
	case tSmallSerial, tSerial, tBigSerial, tSerial2, tSerial4, tSerial8:
		typ = &SerialType{T: t, Precision: int(c.precision)}
	case tUUID:
		typ = &UUIDType{T: t}
	case tXML:
		typ = &XMLType{T: t}
	case tArray:
		// Note that for ARRAY types, the 'udt_name' column holds the array type
		// prefixed with '_'. For example, for 'integer[]' the result is '_int',
		// and for 'text[N][M]' the result is also '_text'. That's because, the
		// database ignores any size or multi-dimensions constraints.
		typ = &ArrayType{T: strings.TrimPrefix(c.udt, "_") + "[]"}
	case tUserDefined:
		typ = &UserDefinedType{T: c.udt}
		// The `typtype` column is set to 'e' for enum types, and the
		// values are filled in batch after the rows above is closed.
		// https://www.postgresql.org/docs/current/catalog-pg-type.html
		if c.typtype == "e" {
			typ = &EnumType{T: c.udt, ID: c.typid}
		}
	default:
		typ = &schema.UnsupportedType{T: t}
	}
	return typ
}

// enumValues fills enum columns with their values from the database.
func (i *inspect) enumValues(ctx context.Context, columns []*schema.Column) error {
	var (
		args  []interface{}
		ids   = make(map[int64][]*schema.EnumType)
		query = "SELECT enumtypid, enumlabel FROM pg_enum WHERE enumtypid IN ("
	)
	for _, c := range columns {
		if enum, ok := c.Type.Type.(*EnumType); ok {
			if _, ok := ids[enum.ID]; !ok {
				if len(args) > 0 {
					query += ", "
				}
				args = append(args, enum.ID)
				query += fmt.Sprintf("$%d", len(args))
			}
			// Convert the intermediate type to the standard schema.EnumType.
			e := &schema.EnumType{T: enum.T}
			c.Type.Type = e
			c.Type.Raw = enum.T
			ids[enum.ID] = append(ids[enum.ID], e)
		}
	}
	query += ")"
	if len(ids) == 0 {
		return nil
	}
	rows, err := i.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("postgres: querying enum values: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id int64
			v  string
		)
		if err := rows.Scan(&id, &v); err != nil {
			return fmt.Errorf("postgres: scanning enum label: %w", err)
		}
		for _, enum := range ids[id] {
			enum.Values = append(enum.Values, v)
		}
	}
	return nil
}

// indexes queries and appends the indexes of the given table.
func (i *inspect) indexes(ctx context.Context, t *schema.Table) error {
	rows, err := i.QueryContext(ctx, indexesQuery, t.Schema.Name, t.Name)
	if err != nil {
		return fmt.Errorf("postgres: querying %q indexes: %w", t.Name, err)
	}
	defer rows.Close()
	if err := i.addIndexes(t, rows); err != nil {
		return err
	}
	return rows.Err()
}

// addIndexes scans the rows and adds the indexes to the table.
func (i *inspect) addIndexes(t *schema.Table, rows *sql.Rows) error {
	names := make(map[string]*schema.Index)
	for rows.Next() {
		var (
			name, typ                            string
			uniq, primary                        bool
			asc, desc, nullsfirst, nullslast     sql.NullBool
			column, contype, pred, expr, comment sql.NullString
		)
		if err := rows.Scan(&name, &typ, &column, &primary, &uniq, &contype, &pred, &expr, &asc, &desc, &nullsfirst, &nullslast, &comment); err != nil {
			return fmt.Errorf("postgres: scanning index: %w", err)
		}
		idx, ok := names[name]
		if !ok {
			idx = &schema.Index{
				Name:   name,
				Unique: uniq,
				Table:  t,
				Attrs: []schema.Attr{
					&IndexType{T: typ},
				},
			}
			if sqlx.ValidString(comment) {
				idx.Attrs = append(idx.Attrs, &schema.Comment{Text: comment.String})
			}
			if sqlx.ValidString(contype) {
				idx.Attrs = append(idx.Attrs, &ConType{T: contype.String})
			}
			if sqlx.ValidString(pred) {
				idx.Attrs = append(idx.Attrs, &IndexPredicate{P: pred.String})
			}
			names[name] = idx
			if primary {
				t.PrimaryKey = idx
			} else {
				t.Indexes = append(t.Indexes, idx)
			}
		}
		part := &schema.IndexPart{
			SeqNo: len(idx.Parts) + 1,
			Attrs: []schema.Attr{
				&IndexColumnProperty{
					Asc:        asc.Bool,
					Desc:       desc.Bool,
					NullsFirst: nullsfirst.Bool,
					NullsLast:  nullslast.Bool,
				},
			},
		}
		switch {
		case sqlx.ValidString(expr):
			part.X = &schema.RawExpr{
				X: expr.String,
			}
		case sqlx.ValidString(column):
			part.C, ok = t.Column(column.String)
			if !ok {
				return fmt.Errorf("postgres: column %q was not found for index %q", column.String, idx.Name)
			}
			part.C.Indexes = append(part.C.Indexes, idx)
		default:
			return fmt.Errorf("postgres: invalid part for index %q", idx.Name)
		}
		idx.Parts = append(idx.Parts, part)
	}
	return nil
}

// fks queries and appends the foreign keys of the given table.
func (i *inspect) fks(ctx context.Context, t *schema.Table) error {
	rows, err := i.QueryContext(ctx, fksQuery, t.Schema.Name, t.Name)
	if err != nil {
		return fmt.Errorf("postgres: querying %q foreign keys: %w", t.Name, err)
	}
	defer rows.Close()
	if err := sqlx.ScanFKs(t, rows); err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	return rows.Err()
}

// checks queries and appends the check constraints of the given table.
func (i *inspect) checks(ctx context.Context, t *schema.Table) error {
	rows, err := i.QueryContext(ctx, checksQuery, t.Schema.Name, t.Name)
	if err != nil {
		return fmt.Errorf("mysql: querying %q check constraints: %w", t.Name, err)
	}
	defer rows.Close()
	if err := i.addChecks(t, rows); err != nil {
		return err
	}
	return rows.Err()
}

// addChecks scans the rows and adds the checks to the table.
func (i *inspect) addChecks(t *schema.Table, rows *sql.Rows) error {
	names := make(map[string]*Check)
	for rows.Next() {
		var (
			noInherit                     bool
			name, column, clause, indexes string
		)
		if err := rows.Scan(&name, &clause, &column, &indexes, &noInherit); err != nil {
			return fmt.Errorf("postgres: scanning check: %w", err)
		}
		if _, ok := t.Column(column); !ok {
			return fmt.Errorf("postgres: column %q was not found for check %q", column, name)
		}
		check, ok := names[name]
		if !ok {
			check = &Check{Name: name, Clause: clause, NoInherit: noInherit}
			names[name] = check
			t.Attrs = append(t.Attrs, check)
		}
		check.Columns = append(check.Columns, column)
	}
	return nil
}

// schemas returns the list of the schemas in the database.
func (i *inspect) schemas(ctx context.Context, opts *schema.InspectRealmOption) ([]*schema.Schema, error) {
	var (
		args  []interface{}
		query = schemasQuery + " WHERE schema_name "
	)
	if opts != nil && len(opts.Schemas) > 0 {
		query, args = inStrings(opts.Schemas, query, args)
	} else {
		query += "NOT IN ('information_schema', 'pg_catalog', 'pg_toast')"
	}
	rows, err := i.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: querying schemas: %w", err)
	}
	defer rows.Close()
	var schemas []*schema.Schema
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		schemas = append(schemas, &schema.Schema{
			Name: name,
		})
	}
	return schemas, nil
}

// tableNames returns a list of all tables exist in the schema.
func (i *inspect) tableNames(ctx context.Context, schema string, opts *schema.InspectOptions) ([]string, error) {
	query, args := tablesQuery, []interface{}{schema}
	if opts != nil && len(opts.Tables) > 0 {
		query += " AND table_name "
		query, args = inStrings(opts.Tables, query, args)
	}
	rows, err := i.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: querying schema tables: %w", err)
	}
	names, err := sqlx.ScanStrings(rows)
	if err != nil {
		return nil, fmt.Errorf("postgres: scanning table names: %w", err)
	}
	return names, nil
}

func inStrings(s []string, query string, args []interface{}) (string, []interface{}) {
	query += "IN ("
	for i := range s {
		args = append(args, s[i])
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("$%d", len(args))
	}
	query += ")"
	return query, args
}

func defaultExpr(c *schema.Column, x string) schema.Expr {
	switch {
	case sqlx.IsLiteralBool(x), sqlx.IsLiteralNumber(x), sqlx.IsQuoted(x, '\''):
		return &schema.Literal{V: x}
	default:
		// Try casting or fallback to raw expressions (e.g. column text[] has the default of '{}':text[]).
		if v, ok := canConvert(c.Type, x); ok {
			return &schema.Literal{V: v}
		}
		return &schema.RawExpr{X: x}
	}
}

func canConvert(t *schema.ColumnType, x string) (string, bool) {
	r := t.Raw
	if t, ok := t.Type.(*ArrayType); ok {
		r = t.T
	}
	i := strings.Index(x, "::"+r)
	if i == -1 || !sqlx.IsQuoted(x[:i], '\'') {
		return "", false
	}
	q := x[0:i]
	x = x[1 : i-1]
	switch t.Type.(type) {
	case *schema.BoolType:
		if sqlx.IsLiteralBool(x) {
			return x, true
		}
	case *schema.DecimalType, *schema.IntegerType, *schema.FloatType:
		if sqlx.IsLiteralNumber(x) {
			return x, true
		}
	case *ArrayType, *schema.BinaryType, *schema.JSONType, *NetworkType, *schema.SpatialType, *schema.StringType, *schema.TimeType, *UUIDType, *XMLType:
		return q, true
	}
	return "", false
}

type (
	// CType describes the character classification setting (LC_CTYPE).
	CType struct {
		schema.Attr
		V string
	}

	// UserDefinedType defines a user-defined type attribute.
	UserDefinedType struct {
		schema.Type
		T string
	}

	// EnumType represents an enum type.
	EnumType struct {
		schema.Type
		T      string // Type name.
		ID     int64  // Type id.
		Values []string
	}

	// ArrayType defines an array type.
	// https://www.postgresql.org/docs/current/arrays.html
	ArrayType struct {
		schema.Type
		T string
	}

	// BitType defines a bit type.
	// https://www.postgresql.org/docs/current/datatype-bit.html
	BitType struct {
		schema.Type
		T   string
		Len int64
	}

	// A NetworkType defines a network type.
	// https://www.postgresql.org/docs/current/datatype-net-types.html
	NetworkType struct {
		schema.Type
		T   string
		Len int64
	}

	// A CurrencyType defines a currency type.
	CurrencyType struct {
		schema.Type
		T string
	}

	// A SerialType defines a serial type.
	SerialType struct {
		schema.Type
		T         string
		Precision int
	}

	// A UUIDType defines a UUID type.
	UUIDType struct {
		schema.Type
		T string
	}

	// A XMLType defines an XML type.
	XMLType struct {
		schema.Type
		T string
	}

	// ConType describes constraint type.
	// https://www.postgresql.org/docs/current/catalog-pg-constraint.html
	ConType struct {
		schema.Attr
		T string // c, f, p, u, t, x.
	}

	// Identity defines an identity column.
	Identity struct {
		schema.Attr
		Generation string // ALWAYS, BY DEFAULT.
	}

	// IndexType represents an index type.
	// https://www.postgresql.org/docs/current/indexes-types.html
	IndexType struct {
		schema.Attr
		T string // BTREE, BRIN, HASH, GiST, SP-GiST, GIN.
	}

	// IndexPredicate describes a partial index predicate.
	// https://www.postgresql.org/docs/current/catalog-pg-index.html
	IndexPredicate struct {
		schema.Attr
		P string
	}

	// IndexColumnProperty describes an index column property.
	// https://www.postgresql.org/docs/current/functions-info.html#FUNCTIONS-INFO-INDEX-COLUMN-PROPS
	IndexColumnProperty struct {
		schema.Attr
		Asc        bool
		Desc       bool
		NullsFirst bool
		NullsLast  bool
	}

	// Check attributes defines a CHECK constraint.
	// https://www.postgresql.org/docs/current/catalog-pg-constraint.html
	Check struct {
		schema.Attr
		Name      string
		Clause    string
		NoInherit bool
		Columns   []string
	}
)

const (
	// Query to list runtime parameters.
	paramsQuery = `SELECT setting FROM pg_settings WHERE name IN ('lc_collate', 'lc_ctype', 'server_version_num') ORDER BY name`

	// Query to list database schemas.
	schemasQuery = "SELECT schema_name FROM information_schema.schemata"

	// Query to list schema tables.
	tablesQuery = "SELECT table_name FROM information_schema.tables WHERE table_type = 'BASE TABLE' AND table_schema = $1 ORDER BY table_name"

	// Query to list table information.
	tableQuery = `
SELECT
	t1.table_schema,
	pg_catalog.obj_description(t2.oid, 'pg_class') AS COMMENT
FROM
	information_schema.tables AS t1
	INNER JOIN pg_catalog.pg_class AS t2
	ON t1.table_name = t2.relname
WHERE
	t1.table_type = 'BASE TABLE'
	AND t1.table_name = $1
	AND t1.table_schema = (CURRENT_SCHEMA())
`
	tableSchemaQuery = `
SELECT
	t1.TABLE_SCHEMA,
	pg_catalog.obj_description(t2.oid, 'pg_class') AS COMMENT
FROM
	INFORMATION_SCHEMA.TABLES AS t1
	JOIN pg_catalog.pg_class AS t2
	ON t1.table_name = t2.relname
WHERE
	t1.TABLE_TYPE = 'BASE TABLE'
	AND t1.TABLE_NAME = $1
	AND t1.TABLE_SCHEMA = $2
`
	// Query to list table columns.
	columnsQuery = `
SELECT
	t1.column_name,
	t1.data_type,
	t1.is_nullable,
	t1.column_default,
	t1.character_maximum_length,
	t1.numeric_precision,
	t1.numeric_scale,
	t1.character_set_name,
	t1.collation_name,
	t1.udt_name,
	t1.is_identity,
	t1.identity_generation,
	col_description(to_regclass("table_schema" || '.' || "table_name")::oid, "ordinal_position") AS comment,
	t2.typtype,
	t2.oid
FROM
	"information_schema"."columns" AS t1
	LEFT JOIN pg_catalog.pg_type AS t2
	ON t1.udt_name = t2.typname
WHERE
	TABLE_SCHEMA = $1 AND TABLE_NAME = $2
`

	// Query to list table indexes.
	indexesQuery = `
SELECT
	i.relname AS index_name,
	am.amname AS index_type,
	a.attname AS column_name,
	idx.indisprimary AS primary,
	idx.indisunique AS unique,
	c.contype AS constraint_type,
	pg_get_expr(idx.indpred, idx.indrelid) AS predicate,
	pg_get_expr(idx.indexprs, idx.indrelid) AS expression,
	pg_index_column_has_property(idx.indexrelid, a.attnum, 'asc') AS asc,
	pg_index_column_has_property(idx.indexrelid, a.attnum, 'desc') AS desc,
	pg_index_column_has_property(idx.indexrelid, a.attnum, 'nulls_first') AS nulls_first,
	pg_index_column_has_property(idx.indexrelid, a.attnum, 'nulls_last') AS nulls_last,
	obj_description(to_regclass($1 || i.relname)::oid) AS comment
FROM
	pg_index idx
	JOIN pg_class i
	ON i.oid = idx.indexrelid
	LEFT JOIN pg_constraint c
	ON idx.indexrelid = c.conindid
	LEFT JOIN pg_attribute a
	ON a.attrelid = idx.indexrelid
	JOIN pg_am am
	ON am.oid = i.relam
WHERE
	idx.indrelid = to_regclass($1 || '.' || $2)::oid
	AND COALESCE(c.contype, '') <> 'f'
ORDER BY
	index_name, a.attnum
`
	fksQuery = `
SELECT
    t1.constraint_name,
    t1.table_name,
    t2.column_name,
    t1.table_schema,
    t3.table_name AS referenced_table_name,
    t3.column_name AS referenced_column_name,
    t3.table_schema AS referenced_schema_name,
    t4.update_rule,
    t4.delete_rule
FROM
    information_schema.table_constraints t1
    JOIN information_schema.key_column_usage t2
    ON t1.constraint_name = t2.constraint_name
    AND t1.table_schema = t2.constraint_schema
    JOIN information_schema.constraint_column_usage t3
    ON t1.constraint_name = t3.constraint_name
    AND t1.table_schema = t3.constraint_schema
    JOIN information_schema.referential_constraints t4
    ON t1.constraint_name = t4.constraint_name
    AND t1.table_schema = t4.constraint_schema
WHERE
    t1.constraint_type = 'FOREIGN KEY'
    AND t1.table_schema = $1
    AND t1.table_name = $2
ORDER BY
    t1.constraint_name,
    t2.ordinal_position
`

	// Query to list table check constraints.
	checksQuery = `
SELECT
	t1.conname AS constraint_name,
	pg_get_expr(t1.conbin, to_regclass($1 || '.' || $2)::oid) as expression,
	t2.attname as column_name,
	t1.conkey as column_indexes,
	t1.connoinherit as no_inherit
FROM
	pg_catalog.pg_constraint t1
	JOIN pg_attribute t2
	ON t2.attrelid = t1.conrelid
	AND t2.attnum = ANY (t1.conkey)
WHERE
	t1.contype = 'c'
	AND t1.conrelid = to_regclass($1 || '.' || $2)::oid
ORDER BY
	t1.conname, array_position(t1.conkey, t2.attnum)
`
)
