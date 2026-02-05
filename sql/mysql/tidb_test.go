// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"strconv"
	"testing"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

const testTiDBVersion = "5.7.25-TiDB-v6.1.0"

// newTiDBDiffer returns a tdiff configured for testing.
func newTiDBDiffer() *sqlx.Diff {
	conn := &conn{ExecQuerier: sqlx.NoRows, V: testTiDBVersion}
	return &sqlx.Diff{DiffDriver: &tdiff{diff{conn: conn}}}
}

func TestAutoRandom_ParseCreateTable(t *testing.T) {
	tests := []struct {
		name      string
		create    string
		wantCol   string
		wantShard int
		wantRange int
	}{
		{
			name:      "shard bits only",
			create:    "CREATE TABLE `t` (`id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(5) */, PRIMARY KEY (`id`)) CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
			wantCol:   "id",
			wantShard: 5,
			wantRange: 0,
		},
		{
			name:      "shard and range bits",
			create:    "CREATE TABLE `t` (`id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(5, 64) */, PRIMARY KEY (`id`)) CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
			wantCol:   "id",
			wantShard: 5,
			wantRange: 64,
		},
		{
			name:      "min shard bits",
			create:    "CREATE TABLE `t` (`id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(1) */, PRIMARY KEY (`id`)) CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
			wantCol:   "id",
			wantShard: 1,
			wantRange: 0,
		},
		{
			name:      "max shard bits",
			create:    "CREATE TABLE `t` (`id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(15) */, PRIMARY KEY (`id`)) CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
			wantCol:   "id",
			wantShard: 15,
			wantRange: 0,
		},
		{
			name:      "min range bits",
			create:    "CREATE TABLE `t` (`id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(3, 32) */, PRIMARY KEY (`id`)) CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
			wantCol:   "id",
			wantShard: 3,
			wantRange: 32,
		},
		{
			name:      "custom shard bits",
			create:    "CREATE TABLE `t` (`id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(10) */, PRIMARY KEY (`id`)) CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
			wantCol:   "id",
			wantShard: 10,
			wantRange: 0,
		},
		{
			name:      "multi-column table",
			create:    "CREATE TABLE `t` (`name` varchar(100) NOT NULL, `id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(5) */, PRIMARY KEY (`id`)) CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
			wantCol:   "id",
			wantShard: 5,
			wantRange: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := reAutoRandom.FindStringSubmatch(tt.create)
			require.GreaterOrEqual(t, len(matches), 3)
			require.Equal(t, tt.wantCol, matches[1])
			require.Equal(t, tt.wantShard, mustAtoi(t, matches[2]))
			if tt.wantRange > 0 {
				require.Equal(t, tt.wantRange, mustAtoi(t, matches[3]))
			} else {
				require.Equal(t, "", matches[3])
			}
		})
	}
}

func TestAutoRandom_PatchSchema(t *testing.T) {
	// setAutoRandom should work with a pre-existing placeholder (TiDB v5/v6 path).
	col := &schema.Column{
		Name: "id",
		Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}},
		Attrs: []schema.Attr{
			&AutoRandom{},
		},
	}
	tbl := schema.NewTable("t").AddColumns(col)
	tbl.AddAttrs(&CreateStmt{
		S: "CREATE TABLE `t` (`id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(5) */, PRIMARY KEY (`id`)) CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
	})
	i := &tinspect{}
	err := i.setAutoRandom(tbl)
	require.NoError(t, err)
	ar := &AutoRandom{}
	require.True(t, sqlx.Has(col.Attrs, ar))
	require.Equal(t, 5, ar.ShardBits)
	require.Equal(t, 0, ar.RangeBits)
}

func TestAutoRandom_PatchSchemaNoPlaceholder(t *testing.T) {
	// setAutoRandom should work without a pre-existing placeholder (TiDB v8+ path
	// where EXTRA column is empty).
	col := &schema.Column{
		Name: "id",
		Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}},
	}
	tbl := schema.NewTable("t").AddColumns(col)
	tbl.AddAttrs(&CreateStmt{
		S: "CREATE TABLE `t` (`id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(5) */, PRIMARY KEY (`id`)) CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
	})
	i := &tinspect{}
	err := i.setAutoRandom(tbl)
	require.NoError(t, err)
	ar := &AutoRandom{}
	require.True(t, sqlx.Has(col.Attrs, ar))
	require.Equal(t, 5, ar.ShardBits)
	require.Equal(t, 0, ar.RangeBits)
}

func TestAutoRandom_PatchSchemaWithRange(t *testing.T) {
	col := &schema.Column{
		Name: "id",
		Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}},
	}
	tbl := schema.NewTable("t").AddColumns(col)
	tbl.AddAttrs(&CreateStmt{
		S: "CREATE TABLE `t` (`id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(3, 32) */, PRIMARY KEY (`id`)) CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
	})
	i := &tinspect{}
	err := i.setAutoRandom(tbl)
	require.NoError(t, err)
	ar := &AutoRandom{}
	require.True(t, sqlx.Has(col.Attrs, ar))
	require.Equal(t, 3, ar.ShardBits)
	require.Equal(t, 32, ar.RangeBits)
}

func TestAutoRandom_PatchSchemaDefaultRange(t *testing.T) {
	// RangeBits=64 (the default) should be normalized to 0 so that
	// HCL round-trips are lossless.
	col := &schema.Column{
		Name: "id",
		Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}},
	}
	tbl := schema.NewTable("t").AddColumns(col)
	tbl.AddAttrs(&CreateStmt{
		S: "CREATE TABLE `t` (`id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(5, 64) */, PRIMARY KEY (`id`)) CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
	})
	i := &tinspect{}
	err := i.setAutoRandom(tbl)
	require.NoError(t, err)
	ar := &AutoRandom{}
	require.True(t, sqlx.Has(col.Attrs, ar))
	require.Equal(t, 5, ar.ShardBits)
	require.Equal(t, 0, ar.RangeBits)
}

func TestAutoRandom_PatchSchemaNoAutoRandom(t *testing.T) {
	// If CREATE TABLE has no AUTO_RANDOM, setAutoRandom should be a no-op.
	col := &schema.Column{
		Name: "id",
		Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}},
	}
	tbl := schema.NewTable("t").AddColumns(col)
	tbl.AddAttrs(&CreateStmt{
		S: "CREATE TABLE `t` (`id` bigint NOT NULL, PRIMARY KEY (`id`)) CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
	})
	i := &tinspect{}
	err := i.setAutoRandom(tbl)
	require.NoError(t, err)
	require.False(t, sqlx.Has(col.Attrs, &AutoRandom{}))
}

func TestAutoRandom_ColumnDiff(t *testing.T) {
	differ := newTiDBDiffer()
	fromT := &schema.Table{
		Name:   "t",
		Schema: &schema.Schema{Name: "test"},
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
		},
	}
	toT := &schema.Table{
		Name: "t",
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&AutoRandom{ShardBits: 5}}},
		},
	}
	changes, err := differ.TableDiff(fromT, toT)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	mc, ok := changes[0].(*schema.ModifyColumn)
	require.True(t, ok)
	require.True(t, mc.Change.Is(schema.ChangeAttr))
}

func TestAutoRandom_ColumnDiffShardChange(t *testing.T) {
	differ := newTiDBDiffer()
	fromT := &schema.Table{
		Name:   "t",
		Schema: &schema.Schema{Name: "test"},
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&AutoRandom{ShardBits: 5}}},
		},
	}
	toT := &schema.Table{
		Name: "t",
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&AutoRandom{ShardBits: 10}}},
		},
	}
	changes, err := differ.TableDiff(fromT, toT)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	mc, ok := changes[0].(*schema.ModifyColumn)
	require.True(t, ok)
	require.True(t, mc.Change.Is(schema.ChangeAttr))
}

func TestAutoRandom_ColumnDiffNoChange(t *testing.T) {
	differ := newTiDBDiffer()
	fromT := &schema.Table{
		Name:   "t",
		Schema: &schema.Schema{Name: "test"},
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&AutoRandom{ShardBits: 5}}},
		},
	}
	toT := &schema.Table{
		Name: "t",
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&AutoRandom{ShardBits: 5}}},
		},
	}
	changes, err := differ.TableDiff(fromT, toT)
	require.NoError(t, err)
	require.Empty(t, changes)
}

func TestAutoRandom_ColumnDiffRemovalIgnored(t *testing.T) {
	differ := newTiDBDiffer()
	fromT := &schema.Table{
		Name:   "t",
		Schema: &schema.Schema{Name: "test"},
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&AutoRandom{ShardBits: 5}}},
		},
	}
	toT := &schema.Table{
		Name: "t",
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
		},
	}
	// TiDB does not support dropping AUTO_RANDOM, so no diff should be reported.
	changes, err := differ.TableDiff(fromT, toT)
	require.NoError(t, err)
	require.Empty(t, changes)
}

func TestAutoRandom_ColumnDiffRangeChange(t *testing.T) {
	differ := newTiDBDiffer()
	fromT := &schema.Table{
		Name:   "t",
		Schema: &schema.Schema{Name: "test"},
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&AutoRandom{ShardBits: 5, RangeBits: 64}}},
		},
	}
	toT := &schema.Table{
		Name: "t",
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&AutoRandom{ShardBits: 5, RangeBits: 32}}},
		},
	}
	changes, err := differ.TableDiff(fromT, toT)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	mc, ok := changes[0].(*schema.ModifyColumn)
	require.True(t, ok)
	require.True(t, mc.Change.Is(schema.ChangeAttr))
}

func TestAutoRandom_ParseExtra(t *testing.T) {
	attr, err := parseExtra("auto_random")
	require.NoError(t, err)
	require.True(t, attr.autorandom)
	require.False(t, attr.autoinc)

	// TiDB v7+ may return "auto_random(5)" in the EXTRA column.
	attr, err = parseExtra("auto_random(5)")
	require.NoError(t, err)
	require.True(t, attr.autorandom)
}

func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	require.NoError(t, err, "failed to parse %q as int", s)
	return n
}
