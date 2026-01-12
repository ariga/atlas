// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package ydb

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"github.com/ydb-platform/ydb-go-sdk/v3/scheme"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/options"
)

// inspect provides a YDB implementation for schema.Inspector.
type inspect struct {
	database     string
	schemeClient scheme.Client
	tableClient  table.Client
}

// newInspect creates a new inspect from conn.
func newInspect(c *conn) *inspect {
	return &inspect{
		database:     c.database,
		schemeClient: c.nativeDriver.Scheme(),
		tableClient:  c.nativeDriver.Table(),
	}
}

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

	realm := schema.NewRealm(schemas...)
	mode := sqlx.ModeInspectRealm(opts)

	if len(schemas) > 0 && mode.Is(schema.InspectTables) {
		if err := i.inspectTables(ctx, realm, nil); err != nil {
			return nil, err
		}
	}
	return schema.ExcludeRealm(realm, opts.Exclude)
}

// InspectSchema returns schema descriptions of the tables in the given schema.
// If the schema name is empty, the result will be the connected database.
func (i *inspect) InspectSchema(
	ctx context.Context,
	name string,
	opts *schema.InspectOptions,
) (*schema.Schema, error) {
	if name == "" && i.database != "" {
		name = i.database
	}

	schemas, err := i.schemas(ctx, &schema.InspectRealmOption{Schemas: []string{name}})
	if err != nil {
		return nil, err
	}

	switch n := len(schemas); {
	case n == 0:
		if name == "" {
			return nil, &schema.NotExistError{Err: fmt.Errorf("ydb: no schema found")}
		}
		return nil, &schema.NotExistError{Err: fmt.Errorf("ydb: schema %q was not found", name)}
	case n > 1:
		return nil, fmt.Errorf("ydb: %d schemas were found for %q", n, name)
	}

	if opts == nil {
		opts = &schema.InspectOptions{}
	}

	realm := schema.NewRealm(schemas...)
	mode := sqlx.ModeInspectSchema(opts)

	if mode.Is(schema.InspectTables) {
		if err := i.inspectTables(ctx, realm, opts); err != nil {
			return nil, err
		}
	}

	return schema.ExcludeSchema(realm.Schemas[0], opts.Exclude)
}

// schemas returns the list of schemas in the database.
func (i *inspect) schemas(ctx context.Context, opts *schema.InspectRealmOption) ([]*schema.Schema, error) {
	var names []string
	if opts != nil && len(opts.Schemas) > 0 && opts.Schemas[0] != "" {
		names = opts.Schemas
	} else if i.database != "" {
		names = []string{i.database}
	} else {
		return nil, errors.New("ydb: database path is not configured")
	}

	var schemas []*schema.Schema
	for _, name := range names {
		_, err := i.schemeClient.ListDirectory(ctx, name)
		if err != nil {
			return nil, &schema.NotExistError{
				Err: fmt.Errorf("ydb: path %q does not exist or is not accessible: %w", name, err),
			}
		}
		schemas = append(schemas, schema.New(name))
	}
	return schemas, nil
}

// inspectTables inspects all tables in the realm.
func (i *inspect) inspectTables(
	ctx context.Context,
	realm *schema.Realm,
	opts *schema.InspectOptions,
) error {

	for _, schema := range realm.Schemas {
		if err := i.tables(ctx, schema, opts); err != nil {
			return err
		}
		for _, table := range schema.Tables {
			tableDesc, err := i.tableClient.DescribeTable(ctx, table.Name)
			if err != nil {
				return fmt.Errorf("ydb: failed describe table: %v", err)
			}

			if err := i.columns(table, tableDesc); err != nil {
				return err
			}
			if err := i.indexes(table, tableDesc); err != nil {
				return err
			}
		}
	}
	return nil
}

type entryWithPath struct {
	*scheme.Entry
	fullPath string
}

// tables queries and populates the tables in the schema.
func (i *inspect) tables(ctx context.Context, s *schema.Schema, opts *schema.InspectOptions) error {
	rootPath := s.Name
	rootDir, err := i.schemeClient.ListDirectory(ctx, rootPath)
	if err != nil {
		return fmt.Errorf("ydb: failed list directory: %v", err)
	}

	queue := make([]entryWithPath, 0, len(rootDir.Children))
	for _, child := range rootDir.Children {
		queue = append(queue, entryWithPath{
			Entry:    &child,
			fullPath: fmt.Sprintf("%s/%s", rootPath, child.Name),
		})
	}

	// using BFS to traverse all directories inside database
	for len(queue) != 0 {
		currEntry := queue[0]
		queue = queue[1:]

		switch currEntry.Type {
		case scheme.EntryTable:
			shouldAdd := opts == nil ||
				len(opts.Tables) == 0 ||
				slices.Contains(opts.Tables, currEntry.fullPath)

			if shouldAdd {
				t := schema.NewTable(currEntry.fullPath)
				s.AddTables(t)
			}

		case scheme.EntryDirectory:
			dir, err := i.schemeClient.ListDirectory(ctx, currEntry.fullPath)
			if err != nil {
				return fmt.Errorf("ydb: failed list directory: %v", err)
			}

			for _, child := range dir.Children {
				queue = append(queue, entryWithPath{
					Entry:    &child,
					fullPath: fmt.Sprintf("%s/%s", currEntry.fullPath, child.Name),
				})
			}
		}
	}

	return nil
}

// columns queries and populates the columns for the given table.
func (i *inspect) columns(
	table *schema.Table,
	tableDesc *options.Description,
) error {
	for _, column := range tableDesc.Columns {
		dataType := column.Type.String()
		columnType, err := ParseType(dataType)
		if err != nil {
			columnType = &schema.UnsupportedType{T: dataType}
		}

		_, nullable := columnType.(*OptionalType)

		atlasColumn := &schema.Column{
			Name: column.Name,
			Type: &schema.ColumnType{
				Type: columnType,
				Raw:  dataType,
				Null: nullable,
			},
		}

		if column.DefaultValue != nil {
			if defaultLiteral := column.DefaultValue.Literal(); defaultLiteral != nil {
				atlasColumn.Default = &schema.Literal{
					V: defaultLiteral.Yql(),
				}
			}
		}

		table.AddColumns(atlasColumn)
	}

	return nil
}

// indexes queries and populates the indexes for the given table.
func (i *inspect) indexes(
	table *schema.Table,
	tableDesc *options.Description,
) error {
	// primary key index
	var primaryKeyParts []*schema.IndexPart
	for _, keyColumn := range tableDesc.PrimaryKey {
		column, ok := table.Column(keyColumn)
		if !ok {
			return fmt.Errorf("ydb: primary key column %q not found in table %q", keyColumn, table.Name)
		}

		primaryKeyParts = append(primaryKeyParts, &schema.IndexPart{
			SeqNo: len(primaryKeyParts) + 1,
			C:     column,
		})
	}
	if len(primaryKeyParts) > 0 {
		pk := &schema.Index{
			Name:   "PRIMARY",
			Unique: true,
			Table:  table,
			Parts:  primaryKeyParts,
		}
		table.SetPrimaryKey(pk)
	}

	// secondary indexes
	for _, idx := range tableDesc.Indexes {
		atlasIdx := &schema.Index{
			Name:  idx.Name,
			Table: table,
		}

		for _, columnName := range idx.IndexColumns {
			column, ok := table.Column(columnName)
			if !ok {
				return fmt.Errorf("ydb: index column %q not found in table %q", columnName, table.Name)
			}
			atlasIdx.Parts = append(atlasIdx.Parts, &schema.IndexPart{
				SeqNo: len(atlasIdx.Parts) + 1,
				C:     column,
			})
		}

		table.AddIndexes(atlasIdx)
	}

	return nil
}
