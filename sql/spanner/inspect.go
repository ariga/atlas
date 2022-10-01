// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package spanner

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// A diff provides an SQLite implementation for schema.Inspector.
type inspect struct{ conn }

var _ schema.Inspector = (*inspect)(nil)

const defaultSchemaNameAlias = "default"

// columnDesc represents a column descriptor.
type columnDesc struct {
	typ       string
	size      int
	sizeIsMax bool
}

// InspectRealm returns schema descriptions of all resources in the given realm.
func (i *inspect) InspectRealm(ctx context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	schemas, err := i.schemas(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("issue in schemas(): %w", err)
	}
	r := schema.NewRealm(schemas...)
	if len(schemas) == 0 || !sqlx.ModeInspectRealm(opts).Is(schema.InspectTables) {
		return r, nil
	}
	if err := i.inspectTables(ctx, r, nil); err != nil {
		return nil, err
	}
	sqlx.LinkSchemaTables(schemas)
	return r, nil
}

// InspectSchema returns schema descriptions of the tables in the given schema.
// If the schema name is empty, the result will be the attached schema.
func (i *inspect) InspectSchema(ctx context.Context, name string, opts *schema.InspectOptions) (s *schema.Schema, err error) {
	schemas, err := i.schemas(ctx, &schema.InspectRealmOption{Schemas: []string{name}})
	if err != nil {
		return nil, fmt.Errorf("issue lookings up schema: %w", err)
	}
	switch n := len(schemas); {
	case n == 0:
		return nil, &schema.NotExistError{Err: fmt.Errorf("spanner: schema %q was not found", name)}
	case n > 1:
		return nil, fmt.Errorf("spanner: %d schemas were found for %q", n, name)
	}
	r := schema.NewRealm(schemas...)
	if sqlx.ModeInspectSchema(opts).Is(schema.InspectTables) {

		if err := i.inspectTables(ctx, r, opts); err != nil {
			return nil, err
		}
		sqlx.LinkSchemaTables(schemas)
	}
	return r.Schemas[0], nil
}

func (i *inspect) inspectTables(ctx context.Context, r *schema.Realm, opts *schema.InspectOptions) error {
	if err := i.tables(ctx, r, opts); err != nil {
		return fmt.Errorf("issue in tables(): %w", err)
	}
	for _, s := range r.Schemas {
		if len(s.Tables) == 0 {
			continue
		}
		if err := i.columns(ctx, s); err != nil {
			return err
		}
		if err := i.indexes(ctx, s); err != nil {
			return err
		}
		if err := i.fks(ctx, s); err != nil {
			return err
		}
		if err := i.checks(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

// table returns the table from the database, or a NotExistError if the table was not found.
func (i *inspect) tables(ctx context.Context, realm *schema.Realm, opts *schema.InspectOptions) error {
	var schemas []string
	for _, s := range realm.Schemas {
		sName := s.Name
		if s.Name == defaultSchemaNameAlias {
			sName = ""
		}
		schemas = append(schemas, sName)
	}
	rows, err := i.QueryContext(ctx, tablesQuery, schemas)
	if err != nil {
		return fmt.Errorf("QueryContext issue: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tSchema, name, parentTable, onDeleteAction, spannerState sql.NullString
		if err := rows.Scan(&tSchema, &name, &parentTable, &onDeleteAction, &spannerState); err != nil {
			return fmt.Errorf("scan table information: %w", err)
		}
		if !sqlx.ValidString(name) {
			return fmt.Errorf("invalid table name: %q", name.String)
		}
		sName := tSchema.String
		if sName == "" {
			sName = defaultSchemaNameAlias
		}
		s, ok := realm.Schema(sName)
		if !ok {
			return fmt.Errorf("schema %q was not found in realm", tSchema.String)
		}
		t := &schema.Table{
			Name: name.String,
		}
		s.AddTables(t)
		// TODO(tmc): handle parentTable, onDeleteAction, spannerState as attrs
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	return rows.Close()
}

// columns queries and appends the columns of the given table.
func (i *inspect) columns(ctx context.Context, s *schema.Schema) error {
	query := columnsQuery
	rows, err := i.querySchema(ctx, query, s)
	if err != nil {
		return fmt.Errorf("spanner: querying schema %q columns: %w", s.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := i.addColumn(s, rows); err != nil {
			return fmt.Errorf("spanner: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	/*
		if err := i.enumValues(ctx, s); err != nil {
			return err
		}
	*/
	return nil
}

// addColumn scans the current row and adds a new column from it to the table.
func (i *inspect) addColumn(s *schema.Schema, rows *sql.Rows) error {
	var (
		tableName, columnName                                     sql.NullString
		ordinalPosition                                           sql.NullInt64
		columnDefault, dataType, isNullable, spannerType          sql.NullString
		isGenerated, generationExpression, isStored, spannerState sql.NullString
	)
	if err := rows.Scan(
		&tableName, &columnName,
		&ordinalPosition,
		&columnDefault, &dataType, &isNullable, &spannerType,
		&isGenerated, &generationExpression, &isStored, &spannerState,
	); err != nil {
		return err
	}
	t, ok := s.Table(tableName.String)
	if !ok {
		return fmt.Errorf("table %q was not found in schema", tableName.String)
	}
	c := &schema.Column{
		Name: columnName.String,
		Type: &schema.ColumnType{
			Raw:  dataType.String,
			Null: isNullable.String == "YES",
		},
	}
	c.Type.Type = columnType(spannerType.String)

	if columnDefault.Valid {
		c.Default = defaultExpr(c, columnDefault.String)
	}
	if isGenerated.String == "ALWAYS" {
		c.Attrs = []schema.Attr{
			&schema.GeneratedExpr{
				Expr: generationExpression.String,
				Type: "STORED",
			},
		}
	}
	t.Columns = append(t.Columns, c)
	return nil
}

func columnParts(t string) []string {
	t = strings.TrimSpace(strings.ToUpper(t))
	parts := strings.FieldsFunc(t, func(r rune) bool {
		return r == '(' || r == ')' || r == ' ' || r == ','
	})
	return parts
}

// columnType converts a string representation of a column type to a schema.Type.
func columnType(c string) schema.Type {
	var (
		typ      schema.Type
		typeSize int
		err      error
	)

	// Get Type and Size from the column description.
	parts := columnParts(c)

	if len(parts) > 1 {
		if parts[1] == "MAX" {
			typeSize = -1
		} else {
			typeSize, err = strconv.Atoi(parts[1])
			if err != nil {
				// Note: What should we do if an "unsupported string" arises?
				typeSize = 0
			}
		}
	}
	t := parts[0]

	switch {
	case t == TypeInt64:
		typ = &schema.IntegerType{T: t}
	case t == TypeBool:
		typ = &schema.BoolType{T: t}
	case strings.HasPrefix(t, TypeBytes) && len(parts) > 1:
		typ = &BytesType{
			T:         parts[0],
			Size:      typeSize,
			SizeIsMax: typeSize == -1,
		}
	case strings.HasPrefix(t, TypeString) && len(parts) > 1:
		typ = &StringType{
			T:         parts[0],
			Size:      typeSize,
			SizeIsMax: typeSize == -1,
		}
	case t == TypeTimestamp:
		typ = &schema.TimeType{T: t}
	case t == TypeDate:
		typ = &schema.TimeType{T: t}
	case t == TypeJSON:
		typ = &schema.JSONType{T: t}
	case t == TypeNumeric:
		// typ = &schema.DecimalType{T: t, Precision: int(c.precision), Scale: int(c.scale)}
		typ = &schema.DecimalType{T: t}
	// case TypeBoolArray:
	// 	// Note that for ARRAY types, the 'udt_name' column holds the array type
	// 	// prefixed with '_'. For example, for 'integer[]' the result is '_int',
	// 	// and for 'text[N][M]' the result is also '_text'. That's because, the
	// 	// database ignores any size or multi-dimensions constraints.
	// 	typ = &ArrayType{T: strings.TrimPrefix(c.udt, "_") + "[]"}
	default:
		// typ = &schema.StringType{T: t}
		// // TODO(tmc): clean this up
		typ = &schema.UnsupportedType{T: t}
	}
	return typ
}

// enumValues fills enum columns with their values from the database.
func (i *inspect) enumValues(ctx context.Context, s *schema.Schema) error {
	return fmt.Errorf("enumValues: not implemented")
}

// indexes queries and appends the indexes of the given table.
func (i *inspect) indexes(ctx context.Context, s *schema.Schema) error {
	rows, err := i.querySchema(ctx, indexesQuery, s)
	if err != nil {
		return fmt.Errorf("spanner: querying schema %q indexes: %w", s.Name, err)
	}
	defer rows.Close()
	if err := i.addIndexes(s, rows); err != nil {
		return err
	}
	return rows.Err()
}

// addIndexes scans the rows and adds the indexes to the table.
func (i *inspect) addIndexes(s *schema.Schema, rows *sql.Rows) error {
	names := make(map[string]*schema.Index)
	for rows.Next() {
		var (
			tableSchema                     sql.NullString
			tableName, indexName, indexType string
			parentTableName                 sql.NullString
			isUnique, isNullFiltered        bool
			indexState                      sql.NullString
			columnName                      sql.NullString
			ordinalPosition                 int
			columnOrdering                  sql.NullString
			isNullable                      sql.NullString
		)
		if err := rows.Scan(
			&tableSchema, &tableName, &indexName, &indexType, &parentTableName, &isUnique, &isNullFiltered, &indexState,
			&columnName, &ordinalPosition, &columnOrdering, &isNullable); err != nil {
			return fmt.Errorf("spanner: scanning indexes for schema %q: %w", s.Name, err)
		}
		if tableName == "" {
			continue
		}

		t, ok := s.Table(tableName)
		if !ok {
			return fmt.Errorf("table %q was not found in schema", tableName)
		}
		name := tableName + indexName
		idx, ok := names[name]
		if !ok {
			idx = &schema.Index{
				Name:   indexName,
				Unique: isUnique,
				Table:  t,
				Attrs: []schema.Attr{
					&IndexType{T: indexType},
				},
			}
			// TODO(tmc): Add additional attrs.
			names[name] = idx
			if indexType == "PRIMARY_KEY" {
				if t.PrimaryKey == nil {
					t.PrimaryKey = idx
				}
			} else {
				t.Indexes = append(t.Indexes, idx)
			}
		}
		// TODO(tmc): Handle this data better.
		part := &schema.IndexPart{}
		part.C, ok = t.Column(columnName.String)
		idx.Parts = append(idx.Parts, part)
	}
	return nil
}

// fks queries and appends the foreign keys of the given table.
func (i *inspect) fks(ctx context.Context, s *schema.Schema) error {
	rows, err := i.querySchema(ctx, fksQuery, s)
	if err != nil {
		return fmt.Errorf("spanner: querying schema %q foreign keys: %w", s.Name, err)
	}
	defer rows.Close()
	if err := sqlx.SchemaFKs(s, rows); err != nil {
		return fmt.Errorf("spanner: %w", err)
	}
	return rows.Err()
}

// pks queries and appends the foreign keys of the given table.
func (i *inspect) pks(ctx context.Context, s *schema.Schema) error {
	rows, err := i.querySchema(ctx, primaryKeysQuery, s)
	if err != nil {
		return fmt.Errorf("spanner: querying schema %q foreign keys: %w", s.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		var name, table, column, tSchema, refTable, refColumn, refSchema string
		if err := rows.Scan(&name, &table, &column, &tSchema, &refTable, &refColumn, &refSchema); err != nil {
			return err
		}
		// spew.Dump(name, table, column, tSchema, refTable, refColumn, refSchema)
		// t, ok := s.Table(table)
		// if !ok {
		// 	return fmt.Errorf("table %q was not found in schema", table)
		// }
		// t.PrimaryKey = &schema.Index{
		// 	Name:   name,
		// 	Unique: isUnique,
		// 	Table:  t,
		// 	Attrs: []schema.Attr{
		// 		&IndexType{T: indexType},
		// 	},
		// }
	}
	return rows.Err()
}

// checks queries and appends the check constraints of the given table.
func (i *inspect) checks(ctx context.Context, s *schema.Schema) error {
	rows, err := i.querySchema(ctx, checksQuery, s)
	if err != nil {
		return fmt.Errorf("spanner: querying schema '%q' check constraints: %w", s.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		var tableName, checkName, clause, spannerState string
		if err := rows.Scan(&tableName, &checkName, &clause, &spannerState); err != nil {
			return fmt.Errorf("spanner: scanning check: %w", err)
		}
		t, ok := s.Table(tableName)
		if !ok {
			return fmt.Errorf("table %q was not found in schema", tableName)
		}
		t.AddChecks(&schema.Check{
			Name: checkName,
			Expr: clause,
		})
	}
	return nil
}

// schemas returns the list of the schemas in the database.
func (i *inspect) schemas(ctx context.Context, opts *schema.InspectRealmOption) ([]*schema.Schema, error) {
	var (
		args  []any
		sArgs []string
		query = schemasQuery
	)
	if opts != nil {
		switch n := len(opts.Schemas); {
		case n == 0:
			query = schemasQuery
		case n > 0:
			query = schemasQueryArgs
			for _, s := range opts.Schemas {
				if s == defaultSchemaNameAlias {
					s = ""
				}
				sArgs = append(sArgs, s)
			}
			args = append(args, sArgs)
		}
	}
	rows, err := i.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("spanner: querying schemas: %w", err)
	}
	defer rows.Close()
	var schemas []*schema.Schema
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if name == "" {
			name = defaultSchemaNameAlias
		}
		schemas = append(schemas, &schema.Schema{
			Name: name,
		})
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return schemas, nil
}

func (i *inspect) querySchema(ctx context.Context, query string, s *schema.Schema) (*sql.Rows, error) {
	tables := []string{}
	for _, t := range s.Tables {
		tables = append(tables, t.Name)
	}
	sName := s.Name
	if sName == defaultSchemaNameAlias {
		sName = ""
	}
	return i.QueryContext(ctx, query, sName, tables)
}

func nArgs(start, n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		if i > 1 {
			b.WriteString(", ")
		}
		b.WriteByte('$')
		b.WriteString(strconv.Itoa(start + i))
	}
	return b.String()
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
	case *schema.BinaryType, *schema.JSONType, *schema.SpatialType, *schema.StringType, *schema.TimeType, *StringType, *BytesType, *TimestampType:
		return q, true
	}
	return "", false
}

type (
	// ArrayType defines an array type.
	// https://www.spannerql.org/docs/current/arrays.html
	ArrayType struct {
		schema.Type
		T string
	}

	// A BytesType represents a BYTES type.
	BytesType struct {
		schema.Type
		T         string
		Size      int
		SizeIsMax bool
	}

	// CheckColumns attribute hold the column named used by the CHECK constraints.
	// This attribute is added on inspection for internal usage and has no meaning
	// on migration.
	CheckColumns struct {
		schema.Attr
		Columns []string
	}

	// IndexType represents an index type.
	// https://www.spannerql.org/docs/current/indexes-types.html
	IndexType struct {
		schema.Attr
		T string // BTREE, BRIN, HASH, GiST, SP-GiST, GIN.
	}

	// IndexPredicate describes a partial index predicate.
	// https://www.spannerql.org/docs/current/catalog-pg-index.html
	IndexPredicate struct {
		schema.Attr
		P string
	}

	// IndexColumnProperty describes an index column property.
	// https://www.spannerql.org/docs/current/functions-info.html#FUNCTIONS-INFO-INDEX-COLUMN-PROPS
	IndexColumnProperty struct {
		schema.Attr
		// NullsFirst defaults to true for DESC indexes.
		NullsFirst bool
		// NullsLast defaults to true for ASC indexes.
		NullsLast bool
	}

	// IndexStorageParams describes index storage parameters add with the WITH clause.
	// https://www.spannerql.org/docs/current/sql-createindex.html#SQL-CREATEINDEX-STORAGE-PARAMETERS
	IndexStorageParams struct {
		schema.Attr
		// AutoSummarize defines the authsummarize storage parameter.
		AutoSummarize bool
		// PagesPerRange defines pages_per_range storage
		// parameter for BRIN indexes. Defaults to 128.
		PagesPerRange int64
	}

	// A StringType represents a STRING type.
	StringType struct {
		schema.Type
		T         string
		Size      int
		SizeIsMax bool
	}

	// A TimestampType represents a TIMESTAMP type.
	TimestampType struct {
		schema.Type
		T string
	}
)

const (
	// Query to list runtime parameters.
	paramsQuery = `SELECT option_value FROM information_schema.database_options where option_name IN ('database_dialect')`

	// Query to list database schemas.
	schemasQuery = "SELECT schema_name FROM information_schema.schemata WHERE schema_name NOT IN ('INFORMATION_SCHEMA', 'SPANNER_SYS') ORDER BY schema_name"

	// Query to list specific database schemas.
	schemasQueryArgs = "SELECT schema_name FROM information_schema.schemata WHERE schema_name IN UNNEST (@schemas) ORDER BY schema_name"

	// Query to list table information.
	tablesQuery = `
SELECT
	t1.table_schema,
	t1.table_name,
	t1.parent_table_name,
	t1.on_delete_action,
	t1.spanner_state
FROM
	information_schema.tables AS t1
WHERE
	t1.table_type = 'BASE TABLE'
    AND t1.table_schema IN UNNEST (@schemas)
ORDER BY
	t1.table_schema, t1.table_name
`
	tablesQueryArgs = `
SELECT
	t1.table_schema,
	t1.table_name,
FROM
	information_schema.tables AS t1
WHERE
	t1.table_type = 'BASE TABLE'
	AND t1.table_schema IN (@schema)
	AND t1.table_name IN (@table)
ORDER BY
	t1.table_schema, t1.table_name
`
	// Query to list table columns.
	columnsQuery = `
SELECT
	table_name,
	column_name,
	ordinal_position,
	column_default,
	data_type,
	is_nullable,
	spanner_type,
	is_generated,
	generation_expression,
	is_stored,
	spanner_state
FROM
	information_schema.columns AS t1
WHERE
	table_schema = @schema
	AND table_name IN UNNEST (@table)
ORDER BY
	t1.table_name
`

	// Query to list table indexes.
	indexesQuery = `
SELECT
	t1.table_schema,
	t1.table_name,
	t1.index_name,
	t1.index_type,
	t1.parent_table_name,
	t1.is_unique,
	t1.is_null_filtered,
	t1.index_state,
	t2.column_name,
	t2.ordinal_position,
	t2.column_ordering,
	t2.is_nullable
FROM
	information_schema.indexes as t1
    JOIN information_schema.index_columns t2
    ON (
		t1.table_schema = t2.table_schema
		AND t1.table_name = t2.table_name
		AND t1.index_name = t2.index_name
	)
WHERE
	t1.table_schema = @schema
	AND t2.table_name IN UNNEST (@table)
ORDER BY
	t1.table_name, t1.index_name, t2.ordinal_position
`
	// Query to list foreign keys.
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
	AND t1.table_schema = @schema
	AND t1.table_name IN UNNEST (@table)
ORDER BY
    t1.constraint_name,
    t2.ordinal_position
`

	// Query to list primary keys.
	primaryKeysQuery = `
SELECT
	t1.constraint_name,
	t1.table_name,
	t2.column_name,
	t1.table_schema,
	t3.table_name AS referenced_table_name,
	t3.column_name AS referenced_column_name,
	t3.table_schema AS referenced_schema_name
FROM
	information_schema.table_constraints t1
	JOIN information_schema.key_column_usage t2
	ON t1.constraint_name = t2.constraint_name
	AND t1.table_schema = t2.constraint_schema
	JOIN information_schema.constraint_column_usage t3
	ON t1.constraint_name = t3.constraint_name
	AND t1.table_schema = t3.constraint_schema
WHERE
	t1.constraint_type = 'PRIMARY KEY'
	AND t1.table_schema = @schema
	AND t1.table_name IN UNNEST (@table)
ORDER BY
    t1.constraint_name,
    t2.ordinal_position
`

	// Query to list table check constraints.
	checksQuery = `
select
	tbl.table_name as table_name,
	chk.constraint_name as check_name,
	chk.check_clause as clause,
	chk.spanner_state as spanner_state,
from information_schema.table_constraints as tbl
inner join information_schema.check_constraints as chk
	on tbl.constraint_catalog = chk.constraint_catalog
	and tbl.constraint_schema = chk.constraint_schema
	and tbl.constraint_name = chk.constraint_name
where
	tbl.constraint_type = 'CHECK'
	and not STARTS_WITH(chk.constraint_name, 'CK_IS_NOT_NULL_')
	and tbl.table_schema = @schema
	and tbl.table_name IN UNNEST (@table)
order by
	check_name
`
)
