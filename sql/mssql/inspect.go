// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// A diff provides a MSSQL implementation for schema.Inspector.
type inspect struct{ conn }

var _ schema.Inspector = (*inspect)(nil)

// InspectRealm returns schema descriptions of all resources in the given realm.
func (i *inspect) InspectRealm(ctx context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	schemas, err := i.schemas(ctx, opts)
	if err != nil {
		return nil, err
	}
	if opts == nil {
		opts = &schema.InspectRealmOption{}
	}
	r := schema.NewRealm(schemas...).SetCollation(i.collate)
	if len(schemas) == 0 || !sqlx.ModeInspectRealm(opts).Is(schema.InspectTables) {
		return r, nil
	}
	if err := i.inspectTables(ctx, r, nil); err != nil {
		return nil, err
	}
	sqlx.LinkSchemaTables(schemas)
	return sqlx.ExcludeRealm(r, opts.Exclude)
}

// InspectSchema returns schema descriptions of the tables in the given schema.
// If the schema name is empty, the result will be the attached schema.
func (i *inspect) InspectSchema(ctx context.Context, name string, opts *schema.InspectOptions) (*schema.Schema, error) {
	schemas, err := i.schemas(ctx, &schema.InspectRealmOption{
		Schemas: []string{name},
	})
	if err != nil {
		return nil, err
	}
	switch n := len(schemas); {
	case n == 0:
		return nil, &schema.NotExistError{Err: fmt.Errorf("mssql: schema %q was not found", name)}
	case n > 1:
		return nil, fmt.Errorf("mssql: %d schemas were found for %q", n, name)
	}
	if opts == nil {
		opts = &schema.InspectOptions{}
	}
	r := schema.NewRealm(schemas...).SetCollation(i.collate)
	if sqlx.ModeInspectSchema(opts).Is(schema.InspectTables) {
		if err := i.inspectTables(ctx, r, opts); err != nil {
			return nil, err
		}
		sqlx.LinkSchemaTables(schemas)
	}
	return sqlx.ExcludeSchema(r.Schemas[0], opts.Exclude)
}

func (i *inspect) inspectTables(ctx context.Context, r *schema.Realm, opts *schema.InspectOptions) error {
	if err := i.tables(ctx, r, opts); err != nil {
		return err
	}
	for _, s := range r.Schemas {
		if len(s.Tables) == 0 {
			continue
		}
		if err := i.columns(ctx, s, queryScope{
			exec: i.querySchema,
			append: func(s *schema.Schema, table string, column *schema.Column) error {
				t, ok := s.Table(table)
				if !ok {
					return fmt.Errorf("mssql: table %q was not found in schema", table)
				}
				t.AddColumns(column)
				return nil
			},
		}); err != nil {
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

type queryScope struct {
	exec   func(context.Context, string, *schema.Schema) (*sql.Rows, error)
	append func(*schema.Schema, string, *schema.Column) error
}

// checks queries and appends the check constraints of the given table.
func (i *inspect) checks(ctx context.Context, s *schema.Schema) error {
	rows, err := i.querySchema(ctx, checksQuery, s)
	if err != nil {
		return fmt.Errorf("mssql: querying schema %q check constraints: %w", s.Name, err)
	}
	defer rows.Close()
	if err := i.addChecks(s, rows); err != nil {
		return err
	}
	return rows.Err()
}

// addChecks scans the rows and adds the checks to the table.
func (i *inspect) addChecks(s *schema.Schema, rows *sql.Rows) error {
	names := make(map[string]*schema.Check)
	for rows.Next() {
		var (
			table, name, clause string
			column              sql.NullString
			disabled            int64
		)
		if err := rows.Scan(&table, &name, &clause, &column, &disabled); err != nil {
			return fmt.Errorf("mssql: scanning check: %w", err)
		}
		t, ok := s.Table(table)
		if !ok {
			return fmt.Errorf("mssql: table %q was not found in schema", table)
		}
		if column.Valid {
			if _, ok := t.Column(column.String); !ok {
				return fmt.Errorf("mssql: column %q was not found for check %q", column.String, name)
			}
		}
		check, ok := names[name]
		if !ok {
			check = &schema.Check{
				Name:  name,
				Expr:  clause,
			}
			if column.Valid {
				check.Attrs = append(check.Attrs, &CheckColumns{
					Columns: []string{column.String},
				})
			}
			if disabled == 1 {
				check.Attrs = append(check.Attrs, &CheckDisabled{})
			}
			names[name] = check
			t.Attrs = append(t.Attrs, check)
		}
	}
	return nil
}

// fks queries and appends the foreign keys of the given table.
func (i *inspect) fks(ctx context.Context, s *schema.Schema) error {
	rows, err := i.querySchema(ctx, fksQuery, s)
	if err != nil {
		return fmt.Errorf("mssql: querying %q foreign keys: %w", s.Name, err)
	}
	defer rows.Close()
	if err := sqlx.SchemaFKs(s, rows); err != nil {
		return fmt.Errorf("mssql: %w", err)
	}
	return rows.Err()
}

// indexes queries and appends the indexes of the given table.
func (i *inspect) indexes(ctx context.Context, s *schema.Schema) error {
	rows, err := i.querySchema(ctx, indexesQuery, s)
	if err != nil {
		return fmt.Errorf("mssql: querying schema %q indexes: %w", s.Name, err)
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
			table, name, typ                          string
			primary, uniq, included, desc, seqInIndex int64
			column, comment, pred                     sql.NullString
		)
		if err := rows.Scan(
			&table, &name, &typ, &column, &comment, &pred,
			&primary, &uniq, &included, &desc, &seqInIndex,
		); err != nil {
			return fmt.Errorf("mssql: scanning indexes for schema %q: %w", s.Name, err)
		}
		t, ok := s.Table(table)
		if !ok {
			return fmt.Errorf("mssql: table %q was not found in schema", table)
		}
		idx, ok := names[name]
		if !ok {
			idx = &schema.Index{
				Name:   name,
				Unique: uniq == 1,
				Table:  t,
				Attrs: []schema.Attr{
					&IndexType{T: typ},
				},
			}
			if sqlx.ValidString(comment) {
				idx.Attrs = append(idx.Attrs, &schema.Comment{Text: comment.String})
			}
			if sqlx.ValidString(pred) {
				idx.Attrs = append(idx.Attrs, &IndexPredicate{P: pred.String})
			}
			names[name] = idx
			if primary == 1 {
				t.PrimaryKey = idx
			} else {
				t.Indexes = append(t.Indexes, idx)
			}
		}
		part := &schema.IndexPart{
			SeqNo: int(seqInIndex),
			Desc:  desc == 1,
		}
		switch {
		case included == 1:
			c, ok := t.Column(column.String)
			if !ok {
				return fmt.Errorf("mssql: INCLUDE column %q was not found for index %q", column.String, idx.Name)
			}
			include := &IndexInclude{}
			sqlx.Has(idx.Attrs, include)
			include.Columns = append(include.Columns, c)
			schema.ReplaceOrAppend(&idx.Attrs, include)
		case sqlx.ValidString(column):
			part.C, ok = t.Column(column.String)
			if !ok {
				return fmt.Errorf("mssql: column %q was not found for index %q", column.String, idx.Name)
			}
			part.C.Indexes = append(part.C.Indexes, idx)
			idx.Parts = append(idx.Parts, part)
		default:
			return fmt.Errorf("mssql: invalid part for index %q", idx.Name)
		}
	}
	return nil
}

// columns queries and appends the columns of the given table.
func (i *inspect) columns(ctx context.Context, s *schema.Schema, scope queryScope) error {
	rows, err := scope.exec(ctx, columnsQuery, s)
	if err != nil {
		return fmt.Errorf("mssql: querying schema %q columns: %w", s.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := i.addColumn(s, rows, scope); err != nil {
			return fmt.Errorf("mssql: %w", err)
		}
	}
	return rows.Close()
}

// addColumn scans the current row and adds a new column from it to the scope (table or view).
func (i *inspect) addColumn(s *schema.Schema, rows *sql.Rows, scope queryScope) (err error) {
	var (
		table, name, typeName, comment, collation sql.NullString
		nullable, userDefined                     sql.NullInt64
		identity, identitySeek, identityIncrement sql.NullInt64
		size, precision, scale, isPersisted       sql.NullInt64
		genexpr, defaults                         sql.NullString
		isComputed                                int64
	)
	if err = rows.Scan(
		&table, &name, &typeName, &comment,
		&nullable, &userDefined,
		&identity, &identitySeek, &identityIncrement,
		&collation, &size, &precision, &scale, &isComputed,
		&genexpr, &isPersisted, &defaults,
	); err != nil {
		return err
	}
	c := &schema.Column{
		Name: name.String,
		Type: &schema.ColumnType{
			Raw:  typeName.String,
			Null: nullable.Int64 == 1,
		},
	}
	c.Type.Type, err = columnType(&columnDesc{
		typ:         typeName.String,
		size:        size.Int64,
		scale:       scale.Int64,
		precision:   precision.Int64,
		userDefined: userDefined.Int64 == 1,
	})
	if identity.Valid && identity.Int64 == 1 {
		c.Attrs = append(c.Attrs, &Identity{
			Seek:      identitySeek.Int64,
			Increment: identityIncrement.Int64,
		})
	}
	if isComputed == 1 {
		if !sqlx.ValidString(genexpr) {
			return fmt.Errorf("mssql: computed column %q is missing its definition", name.String)
		}
		x := &schema.GeneratedExpr{
			Expr: genexpr.String,
		}
		if isPersisted.Valid && isPersisted.Int64 == 1 {
			x.Type = computedPersisted
		}
		c.SetGeneratedExpr(x)
	}
	if defaults.Valid {
		c.Default = i.defaultExpr(c, defaults.String)
	}
	if sqlx.ValidString(comment) {
		c.SetComment(comment.String)
	}
	if sqlx.ValidString(collation) {
		c.SetCollation(collation.String)
	}
	return scope.append(s, table.String, c)
}

// schemas returns the list of the schemas in the database.
func (i *inspect) schemas(ctx context.Context, opts *schema.InspectRealmOption) ([]*schema.Schema, error) {
	var (
		args  = []any{}
		query = schemasQuery
	)
	if opts != nil {
		switch n := len(opts.Schemas); {
		case n == 1 && opts.Schemas[0] == "":
			query = fmt.Sprintf(schemasQueryArgs, "= SCHEMA_NAME()")
		case n == 1 && opts.Schemas[0] != "":
			query = fmt.Sprintf(schemasQueryArgs, "= @1")
			args = append(args, opts.Schemas[0])
		case n > 0:
			query = fmt.Sprintf(schemasQueryArgs, "IN ("+nArgs(0, len(opts.Schemas))+")")
			for _, s := range opts.Schemas {
				args = append(args, s)
			}
		}
	}
	rows, err := i.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mssql: querying schemas: %w", err)
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

func (i *inspect) tables(ctx context.Context, realm *schema.Realm, opts *schema.InspectOptions) error {
	var (
		args  = []any{}
		query = fmt.Sprintf(tablesQuery, nArgs(0, len(realm.Schemas)))
	)
	for _, s := range realm.Schemas {
		args = append(args, s.Name)
	}
	if opts != nil && len(opts.Tables) > 0 {
		for _, t := range opts.Tables {
			args = append(args, t)
		}
		query = fmt.Sprintf(tablesQueryArgs,
			nArgs(0, len(realm.Schemas)),
			nArgs(len(realm.Schemas), len(opts.Tables)),
		)
	}
	rows, err := i.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			tSchema, name, comment sql.NullString
			memoryOptimized        sql.NullInt64
		)
		if err := rows.Scan(&tSchema, &name, &comment, &memoryOptimized); err != nil {
			return fmt.Errorf("scan table information: %w", err)
		}
		if !sqlx.ValidString(tSchema) || !sqlx.ValidString(name) {
			return fmt.Errorf("invalid schema or table name: %q.%q", tSchema.String, name.String)
		}
		s, ok := realm.Schema(tSchema.String)
		if !ok {
			return fmt.Errorf("schema %q was not found in realm", tSchema.String)
		}
		t := &schema.Table{Name: name.String}
		s.AddTables(t)
		if sqlx.ValidString(comment) {
			t.SetComment(comment.String)
		}
		if memoryOptimized.Valid && memoryOptimized.Int64 == 1 {
			t.Attrs = append(t.Attrs, &MemoryOptimized{})
		}
	}
	return rows.Close()
}

func (i *inspect) querySchema(ctx context.Context, query string, s *schema.Schema) (*sql.Rows, error) {
	args := []any{s.Name}
	for _, t := range s.Tables {
		args = append(args, t.Name)
	}
	return i.QueryContext(ctx, fmt.Sprintf(query, nArgs(1, len(s.Tables))), args...)
}

func nArgs(start, n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		if i > 1 {
			b.WriteString(", ")
		}
		b.WriteByte('@')
		b.WriteString(strconv.Itoa(start + i))
	}
	return b.String()
}

// defaultExpr returns the default expression of the given column.
//
// https://learn.microsoft.com/en-us/sql/relational-databases/tables/specify-default-values-for-columns
func (i *inspect) defaultExpr(_ *schema.Column, x string) schema.Expr {
	// Remove the parenthesis from the expression.
	x = mayUnwrap(x)
	// Literal expression is quoted or wrapped with parenthesis.
	if sqlx.IsQuoted(x, '\'') || mayUnwrap(x) != x {
		return &schema.Literal{V: x}
	}
	// Raw expression does not have a parenthesis.
	return &schema.RawExpr{X: x}
}

// mayUnwrap removes the wrapping parentheses from the given string.
func mayUnwrap(s string) string {
	n := len(s) - 1
	if len(s) < 2 || s[0] != '(' || s[n] != ')' {
		return s
	}
	return s[1:n]
}

const (
	// Query to list server properties.
	propertiesQuery = "SELECT SERVERPROPERTY('ProductVersion'), SERVERPROPERTY('Collation'), SERVERPROPERTY('SqlCharSetName')"

	// Query to list database schemas.
	schemasQuery = `
SELECT
	[schema_name] = [name]
FROM
	[sys].[schemas]
WHERE
	[name] NOT IN (
		'db_accessadmin', 'db_backupoperator', 'db_datareader',
		'db_datawriter', 'db_ddladmin', 'db_denydatareader',
		'db_denydatawriter', 'db_owner', 'db_securityadmin',
		'guest', 'information_schema', 'sys'
	)
ORDER BY
	[schema_name]`
	// Query to list specific database schemas.
	schemasQueryArgs = `
SELECT
	[schema_name] = [name]
FROM
	[sys].[schemas]
WHERE
	[name] NOT IN (
		'db_accessadmin', 'db_backupoperator', 'db_datareader',
		'db_datawriter', 'db_ddladmin', 'db_denydatareader',
		'db_denydatawriter', 'db_owner', 'db_securityadmin',
		'guest', 'information_schema', 'sys'
	) AND [name] %s
ORDER BY
	[schema_name]`

	tablesQuery = `
SELECT
	[table_schema] = SCHEMA_NAME([t1].[schema_id]),
	[table_name] = [t1].[name],
	[comment] = [td].[value],
	[is_memory_optimized] = [t1].[is_memory_optimized]
FROM
	[sys].[tables] as [t1]
	LEFT JOIN [sys].[extended_properties] [td]
	ON [td].[major_id] = [t1].[object_id]
	AND [td].[minor_id] = 0
	AND [td].[class_desc] = N'OBJECT_OR_COLUMN'
	AND [td].[name] = N'MS_Description'
WHERE
	SCHEMA_NAME([t1].[schema_id]) IN (%s)
	AND [t1].[type] = 'U'
	AND [t1].[is_ms_shipped] = 0
ORDER BY
	[table_schema], [table_name]`

	tablesQueryArgs = `
SELECT
	[table_schema] = SCHEMA_NAME([t1].[schema_id]),
	[table_name] = [t1].[name],
	[comment] = [td].[value],
	[is_memory_optimized] = [t1].[is_memory_optimized]
FROM
	[sys].[tables] as [t1]
	LEFT JOIN [sys].[extended_properties] [td]
	ON [td].[major_id] = [t1].[object_id]
	AND [td].[minor_id] = 0
	AND [td].[class_desc] = N'OBJECT_OR_COLUMN'
	AND [td].[name] = N'MS_Description'
WHERE
	SCHEMA_NAME([t1].[schema_id]) IN (%s)
	AND [t1].[name] IN (%s)
	AND [t1].[type] = 'U'
	AND [t1].[is_ms_shipped] = 0
ORDER BY
	[table_schema], [table_name]`

	columnsQuery = `
SELECT
	[table_name] = [t1].[name],
	[column_name] = [c1].[name],
	[type_name] = [tp].[name],
	[comment] = [cd].[value],
	[is_nullable] = [c1].[is_nullable],
	[is_user_defined] = [tp].[is_user_defined],
	[is_identity] = [ti].[is_identity],
	[identity_seek] = [ti].[seed_value],
	[identity_increment] = [ti].[increment_value],
	[collation_name] = [c1].[collation_name],
	[max_length] = [c1].[max_length],
	[precision] = [c1].[precision],
	[scale] = [c1].[scale],
	[is_computed] = [c1].[is_computed],
	[computed_definition] = [cc].[definition],
	[computed_persisted] = [cc].[is_persisted],
	[default_definition] = [dc].[definition]
FROM
	[sys].[tables] [t1]
	INNER JOIN [sys].[columns] [c1]
	ON [t1].[object_id] = [c1].[object_id]
	INNER JOIN [sys].[types] [tp]
	ON [c1].[user_type_id] = [tp].[user_type_id]
	LEFT JOIN [sys].[computed_columns] [cc]
	ON [cc].[object_id] = [c1].[object_id]
	AND [cc].[column_id] = [c1].[column_id]
	LEFT JOIN [sys].[default_constraints] [dc]
	ON [dc].[object_id] = [c1].[default_object_id]
	AND [dc].[parent_object_id] = [c1].[object_id]
	AND [dc].[parent_column_id] = [c1].[column_id]
	LEFT JOIN [sys].[identity_columns] [ti]
	ON [ti].[object_id] = [c1].[object_id]
	AND [ti].[column_id] = [c1].[column_id]
	LEFT JOIN [sys].[extended_properties] [cd]
	ON [cd].[major_id] = [c1].[object_id]
	AND [cd].[minor_id] = [c1].[column_id]
	AND [cd].[class_desc] = N'OBJECT_OR_COLUMN'
	AND [cd].[name] = N'MS_Description'
WHERE
	SCHEMA_NAME([t1].[schema_id]) = @1
	AND [t1].[name] IN (%s)
	AND [t1].[type] = 'U'
	AND [t1].[is_ms_shipped] = 0
ORDER BY
	[c1].[column_id]`

	indexesQuery = `
SELECT
	[table_name] = OBJECT_NAME([i1].[object_id]),
	[index_name] = [i1].[name],
	[index_type] = [i1].[type_desc],
	[column_name] = [c1].[name],
	[comment] = [pd].[value],
	[where_pred] = [i1].[filter_definition],
	[primary] = [i1].[is_primary_key],
	[is_unique] = [i1].[is_unique],
	[included] = [ic].[is_included_column],
	[is_desc] = [ic].[is_descending_key],
	[seq_in_index] = [ic].[key_ordinal]
FROM
	[sys].[indexes] [i1]
	INNER JOIN [sys].[index_columns] [ic]
	ON [ic].[object_id] = [i1].[object_id]
	AND [ic].[index_id] = [i1].[index_id]
	LEFT JOIN [sys].[columns] [c1]
	ON [c1].[object_id] = [i1].[object_id]
	AND [c1].[column_id] = [ic].[column_id]
	LEFT JOIN [sys].[extended_properties] [pd]
	ON [pd].[major_id] = [i1].[object_id]
	AND [pd].[minor_id] = [i1].[index_id]
	AND [pd].[class_desc] = N'INDEX'
	AND [pd].[name] = N'MS_Description'
WHERE
	[i1].[object_id] IN (
		SELECT
			[t1].[object_id]
		FROM
			[sys].[tables] [t1]
		WHERE
			SCHEMA_NAME([t1].[schema_id]) = @1
			AND [t1].[name] IN (%s)
	)
ORDER BY
	[i1].[index_id], [ic].[key_ordinal]`

	fksQuery = `
SELECT
	[constraint_name] = [fk].[name],
	[table_name] = OBJECT_NAME([fk].[parent_object_id]),
	[column_name] = [cp].[name],
	[table_schema] = SCHEMA_NAME([fk].[schema_id]),
	[referenced_table_name] = OBJECT_NAME([fk].[referenced_object_id]),
	[referenced_column_name] = [cr].[name],
	[referenced_table_schema] = SCHEMA_NAME([tr].[schema_id]),
	[update_rule] = [fk].[update_referential_action_desc],
	[delete_rule] = [fk].[delete_referential_action_desc]
FROM
	[sys].[foreign_keys] [fk]
	INNER JOIN [sys].[foreign_key_columns] [fkc]
	ON [fkc].[constraint_object_id] = [fk].[object_id]
	AND [fkc].[constraint_column_id] = [fk].[key_index_id]
	INNER JOIN [sys].[tables] [tr]
	ON [tr].[object_id] = [fk].[referenced_object_id]
	LEFT JOIN [sys].[columns] [cp]
	ON [cp].[object_id] = [fkc].[parent_object_id]
	AND [cp].[column_id] = [fkc].[parent_column_id]
	LEFT JOIN [sys].[columns] [cr]
	ON [cr].[object_id] = [fkc].[referenced_object_id]
	AND [cr].[column_id] = [fkc].[referenced_column_id]
WHERE
	[fk].[is_ms_shipped] = 0
	AND SCHEMA_NAME([fk].[schema_id]) = @1
	AND OBJECT_NAME([fk].[parent_object_id]) IN (%s)
ORDER BY
	[table_schema], [constraint_name], [fk].[key_index_id]`
	checksQuery = `
SELECT
	[table_name] = OBJECT_NAME([cc].[parent_object_id]),
	[constraint_name] = [cc].[name],
	[expression] = [cc].[definition],
	[column_name] = [c1].[name],
	[disabled] = [cc].[is_disabled]
FROM
	[sys].[check_constraints] [cc]
	LEFT JOIN [sys].[columns] [c1]
	ON [c1].[object_id] = [cc].[parent_object_id]
	AND [c1].column_id = [cc].[parent_column_id]
WHERE
	SCHEMA_NAME([cc].[schema_id]) = @1
	AND OBJECT_NAME([cc].[parent_object_id]) IN (%s)
ORDER BY
	[cc].[name]`
)

type (
	// MemoryOptimized attribute describes if the table is memory-optimized or disk-based.
	MemoryOptimized struct {
		schema.Attr
	}
	// Identity defines an identity column.
	Identity struct {
		schema.Attr
		Seek      int64
		Increment int64
	}
	// IndexType represents an index type.
	// https://learn.microsoft.com/en-us/sql/relational-databases/indexes/indexes#available-index-types
	IndexType struct {
		schema.Attr
		T string // HASH, CLUSTERED, NONCLUSTERED, COLUMNSTORE, XML, Spatial, Filtered, FullText
	}
	// IndexInclude describes the INCLUDE clause allows specifying
	// a list of column which added to the index as non-key columns.
	// https://www.postgresql.org/docs/current/sql-createindex.html
	IndexInclude struct {
		schema.Attr
		Columns []*schema.Column
	}
	// IndexPredicate describes a where index predicate.
	// https://learn.microsoft.com/en-us/sql/relational-databases/indexes/create-filtered-indexes
	IndexPredicate struct {
		schema.Attr
		P string
	}
	// CheckColumns attribute hold the column named used by the CHECK constraints.
	CheckColumns struct {
		schema.Attr
		Columns []string
	}
	// CheckDisabled attribute describes a disabled CHECK constraint.
	CheckDisabled struct {
		schema.Attr
	}
	// BitType defines a bit type.
	// https://learn.microsoft.com/en-us/sql/t-sql/data-types/bit-transact-sql
	BitType struct {
		schema.Type
		T string
	}
	// HierarchyIDType defines a hierarchyid type.
	// https://learn.microsoft.com/en-us/sql/t-sql/data-types/hierarchyid-data-type-method-reference
	HierarchyIDType struct {
		schema.Type
		T string
	}
	// MoneyType defines a money type.
	// https://learn.microsoft.com/en-us/sql/t-sql/data-types/money-and-smallmoney-transact-sql
	MoneyType struct {
		schema.Type
		T string
	}
	// UniqueIdentifierType defines a uniqueidentifier type.
	// https://learn.microsoft.com/en-us/sql/t-sql/data-types/uniqueidentifier-transact-sql
	UniqueIdentifierType struct {
		schema.Type
		T string
	}
	// RowVersionType defines a rowversion type.
	RowVersionType struct {
		schema.Type
		T string
	}
	// UserDefinedType defines a user-defined type attribute.
	UserDefinedType struct {
		schema.Type
		T string
	}
	// XMLType defines an XML type.
	// https://learn.microsoft.com/en-us/sql/t-sql/xml/xml-transact-sql
	XMLType struct {
		schema.Type
		T string
	}
)
