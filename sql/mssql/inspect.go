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
		// if err := i.indexes(ctx, s); err != nil {
		// 	return err
		// }
		// if err := i.fks(ctx, s); err != nil {
		// 	return err
		// }
		// if err := i.checks(ctx, s); err != nil {
		// 	return err
		// }
		// if err := i.showCreate(ctx, s); err != nil {
		// 	return err
		// }
	}
	return nil
}

type queryScope struct {
	exec   func(context.Context, string, *schema.Schema) (*sql.Rows, error)
	append func(*schema.Schema, string, *schema.Column) error
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
		size, precision, scale                    sql.NullInt64
	)
	if err = rows.Scan(
		&table, &name, &typeName, &comment,
		&nullable, &userDefined,
		&identity, &identitySeek, &identityIncrement,
		&collation, &size, &precision, &scale,
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
		if memoryOptimized.Valid {
			t.Attrs = append(t.Attrs, &MemoryOptimized{
				V: memoryOptimized.Int64 == 1,
			})
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
	[scale] = [c1].[scale]
FROM
	[sys].[tables] [t1]
	INNER JOIN [sys].[columns] [c1]
	ON [t1].[object_id] = [c1].[object_id]
	INNER JOIN [sys].[types] [tp]
	ON [c1].[user_type_id] = [tp].[user_type_id]
	LEFT JOIN [sys].[identity_columns] [ti]
	ON [ti].[object_id] = [c1].[object_id]
	AND [ti].[column_id] = [c1].[column_id]
	LEFT JOIN [sys].[extended_properties] [cd]
	ON [cd].[major_id] = [c1].[object_id]
	AND [cd].[minor_id] = [c1].[column_id]
	AND [cd].[name] = N'MS_Description'
WHERE
	SCHEMA_NAME([t1].[schema_id]) = @1
	AND [t1].[name] IN (%s)
	AND [t1].[type] = 'U'
	AND [t1].[is_ms_shipped] = 0
ORDER BY
	[c1].[column_id]`
)

type (
	// MemoryOptimized attribute describes if the table is memory-optimized or disk-based.
	MemoryOptimized struct {
		schema.Attr
		V bool
	}
	// Identity defines an identity column.
	Identity struct {
		schema.Attr
		Seek      int64
		Increment int64
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
