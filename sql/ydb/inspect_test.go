// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package ydb

import (
	"context"
	"errors"
	"testing"

	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
	"github.com/ydb-platform/ydb-go-sdk/v3/scheme"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/options"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/result"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
)

// mockSchemeClient implements YDB scheme.Client for testing
type mockSchemeClient struct {
	database    string
	directories map[string]scheme.Directory
	listDirErr  error
}

func (m *mockSchemeClient) Database() string {
	return m.database
}

func (m *mockSchemeClient) ListDirectory(_ context.Context, path string) (scheme.Directory, error) {
	if m.listDirErr != nil {
		return scheme.Directory{}, m.listDirErr
	}
	if dir, ok := m.directories[path]; ok {
		return dir, nil
	}
	return scheme.Directory{}, errors.New("path not found: " + path)
}

func (m *mockSchemeClient) DescribePath(_ context.Context, _ string) (scheme.Entry, error) {
	return scheme.Entry{}, nil
}

func (m *mockSchemeClient) MakeDirectory(_ context.Context, _ string) error {
	return nil
}

func (m *mockSchemeClient) RemoveDirectory(_ context.Context, _ string) error {
	return nil
}

func (m *mockSchemeClient) ModifyPermissions(_ context.Context, _ string, _ ...scheme.PermissionsOption) error {
	return nil
}

// mockTableClient implements YDB table.Client for testing
type mockTableClient struct {
	tables      map[string]*options.Description
	describeErr error
}

func (m *mockTableClient) CreateSession(_ context.Context, _ ...table.Option) (table.ClosableSession, error) {
	return nil, nil
}

func (m *mockTableClient) Do(_ context.Context, _ table.Operation, _ ...table.Option) error {
	return nil
}

func (m *mockTableClient) DoTx(_ context.Context, _ table.TxOperation, _ ...table.Option) error {
	return nil
}

func (m *mockTableClient) BulkUpsert(_ context.Context, _ string, _ table.BulkUpsertData, _ ...table.Option) error {
	return nil
}

func (m *mockTableClient) DescribeTable(_ context.Context, path string, _ ...options.DescribeTableOption) (*options.Description, error) {
	if m.describeErr != nil {
		return nil, m.describeErr
	}
	if desc, ok := m.tables[path]; ok {
		return desc, nil
	}
	return nil, errors.New("table not found: " + path)
}

func (m *mockTableClient) ReadRows(
	_ context.Context,
	_ string,
	_ types.Value,
	_ []options.ReadRowsOption,
	_ ...table.Option,
) (result.Result, error) {
	return nil, nil
}

func newTestInspect(
	database string,
	schemeClient scheme.Client,
	tableClient table.Client,
) *inspect {
	return &inspect{
		conn:         &conn{database: database},
		schemeClient: schemeClient,
		tableClient:  tableClient,
	}
}

func TestInspect_InspectSchema_Simple(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry: scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{
					{Name: "users", Type: scheme.EntryTable},
				},
			},
		},
	}

	tableClient := &mockTableClient{
		tables: map[string]*options.Description{
			"/local/users": {
				Columns: []options.Column{
					{Name: "id", Type: types.TypeInt64},
					{Name: "name", Type: types.TypeUTF8},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	insp := newTestInspect("/local", schemeClient, tableClient)
	ctx := context.Background()

	s, err := insp.InspectSchema(ctx, "", nil)
	require.NoError(t, err)
	require.Equal(t, "/local", s.Name)
	require.Len(t, s.Tables, 1)
	require.Equal(t, "users", s.Tables[0].Name)
	require.Len(t, s.Tables[0].Columns, 2)
	require.NotNil(t, s.Tables[0].PrimaryKey)
	require.Equal(t, "PRIMARY", s.Tables[0].PrimaryKey.Name)
	require.True(t, s.Tables[0].PrimaryKey.Unique)
}

func TestInspect_InspectSchema_WithColumns(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry: scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{
					{Name: "users", Type: scheme.EntryTable},
				},
			},
		},
	}

	tableClient := &mockTableClient{
		tables: map[string]*options.Description{
			"/local/users": {
				Columns: []options.Column{
					{Name: "id", Type: types.TypeInt64},
					{Name: "name", Type: types.Optional(types.TypeUTF8)},
					{Name: "email", Type: types.TypeUTF8},
					{Name: "age", Type: types.Optional(types.TypeInt32)},
					{Name: "active", Type: types.TypeBool},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	insp := newTestInspect("/local", schemeClient, tableClient)
	ctx := context.Background()

	s, err := insp.InspectSchema(ctx, "/local", nil)
	require.NoError(t, err)

	tbl := s.Tables[0]
	require.Len(t, tbl.Columns, 5)

	// Check nullable columns (Optional types)
	nameCol, ok := tbl.Column("name")
	require.True(t, ok)
	require.True(t, nameCol.Type.Null)

	ageCol, ok := tbl.Column("age")
	require.True(t, ok)
	require.True(t, ageCol.Type.Null)

	// Check non-nullable columns
	idCol, ok := tbl.Column("id")
	require.True(t, ok)
	require.False(t, idCol.Type.Null)

	emailCol, ok := tbl.Column("email")
	require.True(t, ok)
	require.False(t, emailCol.Type.Null)

	activeCol, ok := tbl.Column("active")
	require.True(t, ok)
	require.False(t, activeCol.Type.Null)
}

func TestInspect_InspectSchema_WithIndexes(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry:    scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{{Name: "users", Type: scheme.EntryTable}},
			},
		},
	}

	tableClient := &mockTableClient{
		tables: map[string]*options.Description{
			"/local/users": {
				Columns: []options.Column{
					{Name: "id", Type: types.TypeInt64},
					{Name: "email", Type: types.TypeUTF8},
					{Name: "created_at", Type: types.TypeTimestamp},
				},
				PrimaryKey: []string{"id"},
				Indexes: []options.IndexDescription{
					{Name: "idx_email", IndexColumns: []string{"email"}},
					{Name: "idx_created", IndexColumns: []string{"created_at"}},
				},
			},
		},
	}

	insp := newTestInspect("/local", schemeClient, tableClient)
	ctx := context.Background()

	s, err := insp.InspectSchema(ctx, "/local", nil)
	require.NoError(t, err)

	tbl := s.Tables[0]
	require.NotNil(t, tbl.PrimaryKey)
	require.Equal(t, "PRIMARY", tbl.PrimaryKey.Name)
	require.True(t, tbl.PrimaryKey.Unique)
	require.Len(t, tbl.PrimaryKey.Parts, 1)
	require.Equal(t, "id", tbl.PrimaryKey.Parts[0].C.Name)

	require.Len(t, tbl.Indexes, 2)
	require.Equal(t, "idx_email", tbl.Indexes[0].Name)
	require.Equal(t, "idx_created", tbl.Indexes[1].Name)
}

func TestInspect_InspectSchema_CompositePrimaryKey(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry:    scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{{Name: "order_items", Type: scheme.EntryTable}},
			},
		},
	}

	tableClient := &mockTableClient{
		tables: map[string]*options.Description{
			"/local/order_items": {
				Columns: []options.Column{
					{Name: "order_id", Type: types.TypeInt64},
					{Name: "item_id", Type: types.TypeInt64},
					{Name: "quantity", Type: types.TypeInt32},
				},
				PrimaryKey: []string{"order_id", "item_id"},
			},
		},
	}

	insp := newTestInspect("/local", schemeClient, tableClient)
	ctx := context.Background()

	s, err := insp.InspectSchema(ctx, "/local", nil)
	require.NoError(t, err)

	tbl := s.Tables[0]
	require.NotNil(t, tbl.PrimaryKey)
	require.Len(t, tbl.PrimaryKey.Parts, 2)
	require.Equal(t, "order_id", tbl.PrimaryKey.Parts[0].C.Name)
	require.Equal(t, 1, tbl.PrimaryKey.Parts[0].SeqNo)
	require.Equal(t, "item_id", tbl.PrimaryKey.Parts[1].C.Name)
	require.Equal(t, 2, tbl.PrimaryKey.Parts[1].SeqNo)
}

func TestInspect_InspectSchema_NestedDirectories(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry: scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{
					{Name: "users", Type: scheme.EntryTable},
					{Name: "app", Type: scheme.EntryDirectory},
				},
			},
			"/local/app": {
				Entry: scheme.Entry{Name: "app", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{
					{Name: "settings", Type: scheme.EntryTable},
				},
			},
		},
	}

	tableClient := &mockTableClient{
		tables: map[string]*options.Description{
			"/local/users": {
				Columns:    []options.Column{{Name: "id", Type: types.TypeInt64}},
				PrimaryKey: []string{"id"},
			},
			"/local/app/settings": {
				Columns:    []options.Column{{Name: "key", Type: types.TypeUTF8}},
				PrimaryKey: []string{"key"},
			},
		},
	}

	insp := newTestInspect("/local", schemeClient, tableClient)
	ctx := context.Background()

	s, err := insp.InspectSchema(ctx, "/local", nil)
	require.NoError(t, err)
	require.Len(t, s.Tables, 2)

	tableNames := []string{s.Tables[0].Name, s.Tables[1].Name}
	require.Contains(t, tableNames, "users")
	require.Contains(t, tableNames, "app/settings")
}

func TestInspect_InspectRealm(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry:    scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{{Name: "users", Type: scheme.EntryTable}},
			},
		},
	}

	tableClient := &mockTableClient{
		tables: map[string]*options.Description{
			"/local/users": {
				Columns:    []options.Column{{Name: "id", Type: types.TypeInt64}},
				PrimaryKey: []string{"id"},
			},
		},
	}

	insp := newTestInspect("/local", schemeClient, tableClient)
	ctx := context.Background()

	r, err := insp.InspectRealm(ctx, nil)
	require.NoError(t, err)
	require.Len(t, r.Schemas, 1)
	require.Equal(t, "/local", r.Schemas[0].Name)
	require.Len(t, r.Schemas[0].Tables, 1)
}

func TestInspect_SchemaNotExist(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database:    "/local",
		directories: map[string]scheme.Directory{},
	}

	insp := newTestInspect("/nonexistent", schemeClient, nil)
	ctx := context.Background()

	_, err := insp.InspectSchema(ctx, "", nil)
	require.Error(t, err)
	var notExist *schema.NotExistError
	require.ErrorAs(t, err, &notExist)
}

func TestInspect_NoDatabaseConfigured(t *testing.T) {
	insp := newTestInspect("", nil, nil)
	ctx := context.Background()

	_, err := insp.InspectSchema(ctx, "", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "database path is not configured")
}

func TestInspect_DescribeTableError(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry:    scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{{Name: "users", Type: scheme.EntryTable}},
			},
		},
	}

	tableClient := &mockTableClient{
		describeErr: errors.New("connection failed"),
	}

	insp := newTestInspect("/local", schemeClient, tableClient)
	ctx := context.Background()

	_, err := insp.InspectSchema(ctx, "/local", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed describe table")
}

func TestInspect_PrimaryKeyColumnNotFound(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry:    scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{{Name: "broken", Type: scheme.EntryTable}},
			},
		},
	}

	tableClient := &mockTableClient{
		tables: map[string]*options.Description{
			"/local/broken": {
				Columns:    []options.Column{{Name: "id", Type: types.TypeInt64}},
				PrimaryKey: []string{"nonexistent"},
			},
		},
	}

	insp := newTestInspect("/local", schemeClient, tableClient)
	ctx := context.Background()

	_, err := insp.InspectSchema(ctx, "/local", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "primary key column")
	require.Contains(t, err.Error(), "not found")
}

func TestInspect_IndexColumnNotFound(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry:    scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{{Name: "broken", Type: scheme.EntryTable}},
			},
		},
	}

	tableClient := &mockTableClient{
		tables: map[string]*options.Description{
			"/local/broken": {
				Columns:    []options.Column{{Name: "id", Type: types.TypeInt64}},
				PrimaryKey: []string{"id"},
				Indexes: []options.IndexDescription{
					{Name: "idx_bad", IndexColumns: []string{"nonexistent"}},
				},
			},
		},
	}

	insp := newTestInspect("/local", schemeClient, tableClient)
	ctx := context.Background()

	_, err := insp.InspectSchema(ctx, "/local", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "index column")
	require.Contains(t, err.Error(), "not found")
}

func TestInspect_AllColumnTypes(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry:    scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{{Name: "types_test", Type: scheme.EntryTable}},
			},
		},
	}

	tableClient := &mockTableClient{
		tables: map[string]*options.Description{
			"/local/types_test": {
				Columns: []options.Column{
					{Name: "c_bool", Type: types.TypeBool},
					{Name: "c_int8", Type: types.TypeInt8},
					{Name: "c_int16", Type: types.TypeInt16},
					{Name: "c_int32", Type: types.TypeInt32},
					{Name: "c_int64", Type: types.TypeInt64},
					{Name: "c_uint8", Type: types.TypeUint8},
					{Name: "c_uint16", Type: types.TypeUint16},
					{Name: "c_uint32", Type: types.TypeUint32},
					{Name: "c_uint64", Type: types.TypeUint64},
					{Name: "c_float", Type: types.TypeFloat},
					{Name: "c_double", Type: types.TypeDouble},
					{Name: "c_utf8", Type: types.TypeUTF8},
					{Name: "c_json", Type: types.TypeJSON},
					{Name: "c_uuid", Type: types.TypeUUID},
					{Name: "c_date", Type: types.TypeDate},
					{Name: "c_datetime", Type: types.TypeDatetime},
					{Name: "c_timestamp", Type: types.TypeTimestamp},
				},
				PrimaryKey: []string{"c_int64"},
			},
		},
	}

	insp := newTestInspect("/local", schemeClient, tableClient)
	ctx := context.Background()

	s, err := insp.InspectSchema(ctx, "/local", nil)
	require.NoError(t, err)

	tbl := s.Tables[0]
	require.Len(t, tbl.Columns, 17)

	// All columns should be non-nullable (non-Optional types)
	for _, col := range tbl.Columns {
		require.False(t, col.Type.Null, "column %s should not be nullable", col.Name)
	}
}

func TestInspect_TableFilter(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry: scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{
					{Name: "users", Type: scheme.EntryTable},
					{Name: "orders", Type: scheme.EntryTable},
					{Name: "products", Type: scheme.EntryTable},
				},
			},
		},
	}

	tableClient := &mockTableClient{
		tables: map[string]*options.Description{
			"/local/users": {
				Columns:    []options.Column{{Name: "id", Type: types.TypeInt64}},
				PrimaryKey: []string{"id"},
			},
			"/local/orders": {
				Columns:    []options.Column{{Name: "id", Type: types.TypeInt64}},
				PrimaryKey: []string{"id"},
			},
		},
	}

	insp := newTestInspect("/local", schemeClient, tableClient)
	ctx := context.Background()

	s, err := insp.InspectSchema(ctx, "/local", &schema.InspectOptions{
		Tables: []string{"users", "orders"},
	})
	require.NoError(t, err)
	require.Len(t, s.Tables, 2)

	tableNames := []string{s.Tables[0].Name, s.Tables[1].Name}
	require.Contains(t, tableNames, "users")
	require.Contains(t, tableNames, "orders")
	require.NotContains(t, tableNames, "products")
}

func TestInspect_EmptySchema(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry:    scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{},
			},
		},
	}

	insp := newTestInspect("/local", schemeClient, nil)
	ctx := context.Background()

	s, err := insp.InspectSchema(ctx, "/local", nil)
	require.NoError(t, err)
	require.Equal(t, "/local", s.Name)
	require.Len(t, s.Tables, 0)
}

func TestInspect_ListDirectoryError(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database:   "/local",
		listDirErr: errors.New("network error"),
	}

	insp := newTestInspect("/local", schemeClient, nil)
	ctx := context.Background()

	_, err := insp.InspectSchema(ctx, "/local", nil)
	require.Error(t, err)
	var notExist *schema.NotExistError
	require.ErrorAs(t, err, &notExist)
}

func TestInspect_CompositeIndex(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry:    scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{{Name: "users", Type: scheme.EntryTable}},
			},
		},
	}

	tableClient := &mockTableClient{
		tables: map[string]*options.Description{
			"/local/users": {
				Columns: []options.Column{
					{Name: "id", Type: types.TypeInt64},
					{Name: "tenant_id", Type: types.TypeInt64},
					{Name: "email", Type: types.TypeUTF8},
				},
				PrimaryKey: []string{"id"},
				Indexes: []options.IndexDescription{
					{Name: "idx_tenant_email", IndexColumns: []string{"tenant_id", "email"}},
				},
			},
		},
	}

	insp := newTestInspect("/local", schemeClient, tableClient)
	ctx := context.Background()

	s, err := insp.InspectSchema(ctx, "/local", nil)
	require.NoError(t, err)

	tbl := s.Tables[0]
	require.Len(t, tbl.Indexes, 1)

	idx := tbl.Indexes[0]
	require.Equal(t, "idx_tenant_email", idx.Name)
	require.Len(t, idx.Parts, 2)
	require.Equal(t, "tenant_id", idx.Parts[0].C.Name)
	require.Equal(t, 1, idx.Parts[0].SeqNo)
	require.Equal(t, "email", idx.Parts[1].C.Name)
	require.Equal(t, 2, idx.Parts[1].SeqNo)
}

func TestInspect_DeeplyNestedDirectories(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry: scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{
					{Name: "level1", Type: scheme.EntryDirectory},
				},
			},
			"/local/level1": {
				Entry: scheme.Entry{Name: "level1", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{
					{Name: "level2", Type: scheme.EntryDirectory},
				},
			},
			"/local/level1/level2": {
				Entry: scheme.Entry{Name: "level2", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{
					{Name: "deep_table", Type: scheme.EntryTable},
				},
			},
		},
	}

	tableClient := &mockTableClient{
		tables: map[string]*options.Description{
			"/local/level1/level2/deep_table": {
				Columns:    []options.Column{{Name: "id", Type: types.TypeInt64}},
				PrimaryKey: []string{"id"},
			},
		},
	}

	insp := newTestInspect("/local", schemeClient, tableClient)
	ctx := context.Background()

	s, err := insp.InspectSchema(ctx, "/local", nil)
	require.NoError(t, err)
	require.Len(t, s.Tables, 1)
	require.Equal(t, "level1/level2/deep_table", s.Tables[0].Name)
}

func TestInspect_MixedEntryTypes(t *testing.T) {
	// Test that non-table and non-directory entries are ignored
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry: scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{
					{Name: "users", Type: scheme.EntryTable},
					{Name: "topic1", Type: scheme.EntryTopic},       // Should be ignored
					{Name: "store1", Type: scheme.EntryColumnStore}, // Should be ignored
					{Name: "subdir", Type: scheme.EntryDirectory},
				},
			},
			"/local/subdir": {
				Entry: scheme.Entry{Name: "subdir", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{
					{Name: "orders", Type: scheme.EntryTable},
				},
			},
		},
	}

	tableClient := &mockTableClient{
		tables: map[string]*options.Description{
			"/local/users": {
				Columns:    []options.Column{{Name: "id", Type: types.TypeInt64}},
				PrimaryKey: []string{"id"},
			},
			"/local/subdir/orders": {
				Columns:    []options.Column{{Name: "id", Type: types.TypeInt64}},
				PrimaryKey: []string{"id"},
			},
		},
	}

	insp := newTestInspect("/local", schemeClient, tableClient)
	ctx := context.Background()

	s, err := insp.InspectSchema(ctx, "/local", nil)
	require.NoError(t, err)
	require.Len(t, s.Tables, 2)

	tableNames := []string{s.Tables[0].Name, s.Tables[1].Name}
	require.Contains(t, tableNames, "users")
	require.Contains(t, tableNames, "subdir/orders")
}

func TestInspect_ColumnTypeRawValue(t *testing.T) {
	schemeClient := &mockSchemeClient{
		database: "/local",
		directories: map[string]scheme.Directory{
			"/local": {
				Entry:    scheme.Entry{Name: "local", Type: scheme.EntryDirectory},
				Children: []scheme.Entry{{Name: "test", Type: scheme.EntryTable}},
			},
		},
	}

	tableClient := &mockTableClient{
		tables: map[string]*options.Description{
			"/local/test": {
				Columns: []options.Column{
					{Name: "id", Type: types.TypeInt64},
					{Name: "optional_name", Type: types.Optional(types.TypeUTF8)},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	insp := newTestInspect("/local", schemeClient, tableClient)
	ctx := context.Background()

	s, err := insp.InspectSchema(ctx, "/local", nil)
	require.NoError(t, err)

	tbl := s.Tables[0]

	idCol, ok := tbl.Column("id")
	require.True(t, ok)
	require.Equal(t, "Int64", idCol.Type.Raw)

	optCol, ok := tbl.Column("optional_name")
	require.True(t, ok)
	require.Contains(t, optCol.Type.Raw, "Optional")
}
