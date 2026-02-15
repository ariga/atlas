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

func TestAutoRandom_RegexOnlyMatchesTiDBComment(t *testing.T) {
	// The regex should only match AUTO_RANDOM in TiDB's special comment format,
	// not in regular SQL comments or other contexts.
	tests := []struct {
		name    string
		input   string
		matches bool
	}{
		{
			name:    "valid TiDB comment",
			input:   "`id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(5) */",
			matches: true,
		},
		{
			name:    "valid TiDB comment with range",
			input:   "`id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(5, 32) */",
			matches: true,
		},
		{
			name:    "SQL line comment should not match",
			input:   "-- This table uses AUTO_RANDOM(5)\n`id` bigint NOT NULL",
			matches: false,
		},
		{
			name:    "SQL block comment should not match",
			input:   "/* AUTO_RANDOM(5) */ `id` bigint NOT NULL",
			matches: false,
		},
		{
			name:    "plain text AUTO_RANDOM should not match",
			input:   "`id` bigint NOT NULL AUTO_RANDOM(5)",
			matches: false,
		},
		{
			name:    "different TiDB comment feature should not match",
			input:   "`id` bigint NOT NULL /*T![other_feature] AUTO_RANDOM(5) */",
			matches: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := reAutoRandom.FindStringSubmatch(tt.input)
			if tt.matches {
				require.NotNil(t, matches, "expected regex to match")
			} else {
				require.Nil(t, matches, "expected regex NOT to match")
			}
		})
	}
}

func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	require.NoError(t, err, "failed to parse %q as int", s)
	return n
}

func TestShardRowIDBits_ParseCreateTable(t *testing.T) {
	tests := []struct {
		name          string
		create        string
		wantShard     int
		wantPreSplit  int
		wantClustered *bool // nil = not present
	}{
		{
			name:         "shard bits only",
			create:       "CREATE TABLE `t` (`id` int NOT NULL, PRIMARY KEY (`id`) /*T![clustered_index] NONCLUSTERED */) /*T! SHARD_ROW_ID_BITS=4 */",
			wantShard:    4,
			wantPreSplit: 0,
		},
		{
			name:         "shard and pre-split",
			create:       "CREATE TABLE `t` (`id` int NOT NULL, PRIMARY KEY (`id`) /*T![clustered_index] NONCLUSTERED */) /*T! SHARD_ROW_ID_BITS=4 PRE_SPLIT_REGIONS=2 */",
			wantShard:    4,
			wantPreSplit: 2,
		},
		{
			name:      "clustered",
			create:    "CREATE TABLE `t` (`id` bigint NOT NULL, PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */)",
			wantShard: 0,
		},
		{
			name:      "nonclustered",
			create:    "CREATE TABLE `t` (`id` int NOT NULL, PRIMARY KEY (`id`) /*T![clustered_index] NONCLUSTERED */)",
			wantShard: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantShard > 0 {
				matches := reShardRowID.FindStringSubmatch(tt.create)
				require.NotNil(t, matches)
				require.Equal(t, tt.wantShard, mustAtoi(t, matches[1]))
			}
			if tt.wantPreSplit > 0 {
				matches := rePreSplitRegions.FindStringSubmatch(tt.create)
				require.NotNil(t, matches)
				require.Equal(t, tt.wantPreSplit, mustAtoi(t, matches[1]))
			}
		})
	}
}

func TestShardRowIDBits_RegexOnlyMatchesTiDBComment(t *testing.T) {
	// The regex should only match SHARD_ROW_ID_BITS in TiDB's special comment format,
	// not in regular SQL comments or other contexts.
	tests := []struct {
		name    string
		input   string
		matches bool
	}{
		{
			name:    "valid TiDB comment",
			input:   ") /*T! SHARD_ROW_ID_BITS=4 */",
			matches: true,
		},
		{
			name:    "valid TiDB comment with PRE_SPLIT_REGIONS",
			input:   ") /*T! SHARD_ROW_ID_BITS=4 PRE_SPLIT_REGIONS=2 */",
			matches: true,
		},
		{
			name:    "SQL line comment should not match",
			input:   "-- SHARD_ROW_ID_BITS=4\n`id` int NOT NULL",
			matches: false,
		},
		{
			name:    "SQL block comment should not match",
			input:   "/* SHARD_ROW_ID_BITS=4 */ `id` int NOT NULL",
			matches: false,
		},
		{
			name:    "plain text should not match",
			input:   ") SHARD_ROW_ID_BITS=4",
			matches: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := reShardRowID.FindStringSubmatch(tt.input)
			if tt.matches {
				require.NotNil(t, matches, "expected regex to match")
			} else {
				require.Nil(t, matches, "expected regex NOT to match")
			}
		})
	}
}

func TestPreSplitRegions_RegexOnlyMatchesTiDBComment(t *testing.T) {
	// The regex should only match PRE_SPLIT_REGIONS in TiDB's special comment format.
	tests := []struct {
		name    string
		input   string
		matches bool
	}{
		{
			name:    "valid TiDB comment",
			input:   ") /*T! SHARD_ROW_ID_BITS=4 PRE_SPLIT_REGIONS=2 */",
			matches: true,
		},
		{
			name:    "valid TiDB comment PRE_SPLIT only",
			input:   ") /*T! PRE_SPLIT_REGIONS=3 */",
			matches: true,
		},
		{
			name:    "SQL line comment should not match",
			input:   "-- PRE_SPLIT_REGIONS=2\n`id` int NOT NULL",
			matches: false,
		},
		{
			name:    "SQL block comment should not match",
			input:   "/* PRE_SPLIT_REGIONS=2 */ `id` int NOT NULL",
			matches: false,
		},
		{
			name:    "plain text should not match",
			input:   ") PRE_SPLIT_REGIONS=2",
			matches: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := rePreSplitRegions.FindStringSubmatch(tt.input)
			if tt.matches {
				require.NotNil(t, matches, "expected regex to match")
			} else {
				require.Nil(t, matches, "expected regex NOT to match")
			}
		})
	}
}

func TestShardRowIDBits_PatchSchema(t *testing.T) {
	tbl := schema.NewTable("t")
	tbl.AddAttrs(&CreateStmt{
		S: "CREATE TABLE `t` (`id` int NOT NULL, PRIMARY KEY (`id`) /*T![clustered_index] NONCLUSTERED */) /*T! SHARD_ROW_ID_BITS=4 PRE_SPLIT_REGIONS=2 */",
	})
	i := &tinspect{}
	err := i.setShardRowIDBits(tbl)
	require.NoError(t, err)
	shard := &ShardRowIDBits{}
	require.True(t, sqlx.Has(tbl.Attrs, shard))
	require.Equal(t, 4, shard.N)
	preSplit := &PreSplitRegions{}
	require.True(t, sqlx.Has(tbl.Attrs, preSplit))
	require.Equal(t, 2, preSplit.N)
}

func TestClusteredIndex_PatchSchema(t *testing.T) {
	tests := []struct {
		name        string
		create      string
		wantPresent bool
		wantValue   bool
	}{
		{
			name:        "clustered",
			create:      "CREATE TABLE `t` (`id` bigint NOT NULL, PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */)",
			wantPresent: true,
			wantValue:   true,
		},
		{
			name:        "nonclustered",
			create:      "CREATE TABLE `t` (`id` int NOT NULL, PRIMARY KEY (`id`) /*T![clustered_index] NONCLUSTERED */)",
			wantPresent: true,
			wantValue:   false,
		},
		{
			name:        "no annotation",
			create:      "CREATE TABLE `t` (`id` int NOT NULL, PRIMARY KEY (`id`))",
			wantPresent: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := &schema.Column{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}}
			tbl := schema.NewTable("t").AddColumns(col)
			tbl.PrimaryKey = schema.NewIndex("PRIMARY").AddColumns(col)
			tbl.AddAttrs(&CreateStmt{S: tt.create})
			i := &tinspect{}
			err := i.setClusteredIndex(tbl)
			require.NoError(t, err)
			ci := &ClusteredIndex{}
			if tt.wantPresent {
				require.True(t, sqlx.Has(tbl.PrimaryKey.Attrs, ci))
				require.Equal(t, tt.wantValue, ci.Clustered)
			} else {
				require.False(t, sqlx.Has(tbl.PrimaryKey.Attrs, ci))
			}
		})
	}
}

func TestCheckUnsupportedChanges_TiDBSpecific(t *testing.T) {
	t.Run("ModifyPreSplitRegions", func(t *testing.T) {
		changes := []schema.Change{
			&schema.ModifyTable{
				T: schema.NewTable("users"),
				Changes: []schema.Change{
					&schema.ModifyAttr{
						From: &PreSplitRegions{N: 2},
						To:   &PreSplitRegions{N: 4},
					},
				},
			},
		}
		err := checkUnsupportedChanges(changes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot modify pre_split_regions")
		require.Contains(t, err.Error(), "users")
	})

	t.Run("AddPreSplitRegions", func(t *testing.T) {
		changes := []schema.Change{
			&schema.ModifyTable{
				T: schema.NewTable("users"),
				Changes: []schema.Change{
					&schema.AddAttr{
						A: &PreSplitRegions{N: 4},
					},
				},
			},
		}
		err := checkUnsupportedChanges(changes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot add pre_split_regions")
	})

	t.Run("DropPreSplitRegions", func(t *testing.T) {
		changes := []schema.Change{
			&schema.ModifyTable{
				T: schema.NewTable("users"),
				Changes: []schema.Change{
					&schema.DropAttr{
						A: &PreSplitRegions{N: 4},
					},
				},
			},
		}
		err := checkUnsupportedChanges(changes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot drop pre_split_regions")
	})

	t.Run("ShardRowIDBitsChangeAllowed", func(t *testing.T) {
		changes := []schema.Change{
			&schema.ModifyTable{
				T: schema.NewTable("users"),
				Changes: []schema.Change{
					&schema.ModifyAttr{
						From: &ShardRowIDBits{N: 2},
						To:   &ShardRowIDBits{N: 4},
					},
				},
			},
		}
		err := checkUnsupportedChanges(changes)
		require.NoError(t, err)
	})

	t.Run("ClusteredIndexChangeBlocked", func(t *testing.T) {
		fromIdx := schema.NewIndex("PRIMARY").AddColumns(&schema.Column{Name: "id"})
		fromIdx.AddAttrs(&ClusteredIndex{Clustered: true})
		toIdx := schema.NewIndex("PRIMARY").AddColumns(&schema.Column{Name: "id"})
		toIdx.AddAttrs(&ClusteredIndex{Clustered: false})
		changes := []schema.Change{
			&schema.ModifyTable{
				T: schema.NewTable("users"),
				Changes: []schema.Change{
					&schema.ModifyIndex{
						From: fromIdx,
						To:   toIdx,
					},
				},
			},
		}
		err := checkUnsupportedChanges(changes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot change primary key clustering mode")
	})

	t.Run("AutoRandomModificationBlocked", func(t *testing.T) {
		fromCol := &schema.Column{
			Name: "id",
			Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}},
		}
		fromCol.AddAttrs(&AutoRandom{ShardBits: 5})
		toCol := &schema.Column{
			Name: "id",
			Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}},
		}
		toCol.AddAttrs(&AutoRandom{ShardBits: 10})
		changes := []schema.Change{
			&schema.ModifyTable{
				T: schema.NewTable("users"),
				Changes: []schema.Change{
					&schema.ModifyColumn{
						From:   fromCol,
						To:     toCol,
						Change: schema.ChangeAttr,
					},
				},
			},
		}
		err := checkUnsupportedChanges(changes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot modify AUTO_RANDOM")
	})

	t.Run("AutoRandomRemovalBlocked", func(t *testing.T) {
		fromCol := &schema.Column{
			Name: "id",
			Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}},
		}
		fromCol.AddAttrs(&AutoRandom{ShardBits: 5})
		toCol := &schema.Column{
			Name: "id",
			Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}},
		}
		changes := []schema.Change{
			&schema.ModifyTable{
				T: schema.NewTable("users"),
				Changes: []schema.Change{
					&schema.ModifyColumn{
						From:   fromCol,
						To:     toCol,
						Change: schema.ChangeAttr,
					},
				},
			},
		}
		err := checkUnsupportedChanges(changes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot remove AUTO_RANDOM")
	})

	t.Run("AutoRandomAdditionAllowed", func(t *testing.T) {
		fromCol := &schema.Column{
			Name: "id",
			Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}},
		}
		toCol := &schema.Column{
			Name: "id",
			Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}},
		}
		toCol.AddAttrs(&AutoRandom{ShardBits: 5})
		changes := []schema.Change{
			&schema.ModifyTable{
				T: schema.NewTable("users"),
				Changes: []schema.Change{
					&schema.ModifyColumn{
						From:   fromCol,
						To:     toCol,
						Change: schema.ChangeAttr,
					},
				},
			},
		}
		err := checkUnsupportedChanges(changes)
		require.NoError(t, err)
	})

	t.Run("AutoRandomRequiresBigInt", func(t *testing.T) {
		fromCol := &schema.Column{
			Name: "id",
			Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}},
		}
		toCol := &schema.Column{
			Name: "id",
			Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}},
		}
		toCol.AddAttrs(&AutoRandom{ShardBits: 5})
		changes := []schema.Change{
			&schema.ModifyTable{
				T: schema.NewTable("users"),
				Changes: []schema.Change{
					&schema.ModifyColumn{
						From:   fromCol,
						To:     toCol,
						Change: schema.ChangeAttr,
					},
				},
			},
		}
		err := checkUnsupportedChanges(changes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "only supported on BIGINT columns")
	})
}

func TestTableAttrDiff_ShardRowIDBits(t *testing.T) {
	differ := newTiDBDiffer()
	testSchema := &schema.Schema{Name: "test"}

	t.Run("AddShardRowIDBits", func(t *testing.T) {
		from := schema.NewTable("t")
		from.Schema = testSchema
		to := schema.NewTable("t")
		to.Schema = testSchema
		to.AddAttrs(&ShardRowIDBits{N: 4})
		changes, err := differ.TableDiff(from, to)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		// Adding SHARD_ROW_ID_BITS generates a ModifyAttr (from 0 → desired)
		// so that the reverse SQL is correctly produced as SHARD_ROW_ID_BITS = 0.
		modAttr, ok := changes[0].(*schema.ModifyAttr)
		require.True(t, ok, "expected ModifyAttr, got %T", changes[0])
		fromS, ok := modAttr.From.(*ShardRowIDBits)
		require.True(t, ok)
		require.Equal(t, 0, fromS.N)
		toS, ok := modAttr.To.(*ShardRowIDBits)
		require.True(t, ok)
		require.Equal(t, 4, toS.N)
	})

	t.Run("ModifyShardRowIDBits", func(t *testing.T) {
		from := schema.NewTable("t")
		from.Schema = testSchema
		from.AddAttrs(&ShardRowIDBits{N: 2})
		to := schema.NewTable("t")
		to.Schema = testSchema
		to.AddAttrs(&ShardRowIDBits{N: 4})
		changes, err := differ.TableDiff(from, to)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		modAttr, ok := changes[0].(*schema.ModifyAttr)
		require.True(t, ok, "expected ModifyAttr, got %T", changes[0])
		fromS, ok := modAttr.From.(*ShardRowIDBits)
		require.True(t, ok)
		require.Equal(t, 2, fromS.N)
		toS, ok := modAttr.To.(*ShardRowIDBits)
		require.True(t, ok)
		require.Equal(t, 4, toS.N)
	})

	t.Run("DropShardRowIDBits", func(t *testing.T) {
		from := schema.NewTable("t")
		from.Schema = testSchema
		from.AddAttrs(&ShardRowIDBits{N: 4})
		to := schema.NewTable("t")
		to.Schema = testSchema
		changes, err := differ.TableDiff(from, to)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		// Dropping is represented as ModifyAttr with To.N=0
		modAttr, ok := changes[0].(*schema.ModifyAttr)
		require.True(t, ok, "expected ModifyAttr, got %T", changes[0])
		fromS, ok := modAttr.From.(*ShardRowIDBits)
		require.True(t, ok)
		require.Equal(t, 4, fromS.N)
		toS, ok := modAttr.To.(*ShardRowIDBits)
		require.True(t, ok)
		require.Equal(t, 0, toS.N)
	})
}

func TestTableAttrDiff_PreSplitRegions(t *testing.T) {
	differ := newTiDBDiffer()
	testSchema := &schema.Schema{Name: "test"}

	t.Run("AddPreSplitRegions", func(t *testing.T) {
		from := schema.NewTable("t")
		from.Schema = testSchema
		to := schema.NewTable("t")
		to.Schema = testSchema
		to.AddAttrs(&PreSplitRegions{N: 4})
		changes, err := differ.TableDiff(from, to)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		addAttr, ok := changes[0].(*schema.AddAttr)
		require.True(t, ok, "expected AddAttr, got %T", changes[0])
		psr, ok := addAttr.A.(*PreSplitRegions)
		require.True(t, ok)
		require.Equal(t, 4, psr.N)
	})

	t.Run("DropPreSplitRegions", func(t *testing.T) {
		from := schema.NewTable("t")
		from.Schema = testSchema
		from.AddAttrs(&PreSplitRegions{N: 4})
		to := schema.NewTable("t")
		to.Schema = testSchema
		changes, err := differ.TableDiff(from, to)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		dropAttr, ok := changes[0].(*schema.DropAttr)
		require.True(t, ok, "expected DropAttr, got %T", changes[0])
		psr, ok := dropAttr.A.(*PreSplitRegions)
		require.True(t, ok)
		require.Equal(t, 4, psr.N)
	})
}

func TestAutoIDCache_ParseCreateTable(t *testing.T) {
	tests := []struct {
		name    string
		create  string
		wantN   int
		wantNil bool
	}{
		{
			name:   "auto_id_cache=1",
			create: "CREATE TABLE `t` (`id` bigint NOT NULL AUTO_INCREMENT, PRIMARY KEY (`id`)) /*T![auto_id_cache] AUTO_ID_CACHE=1 */ CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
			wantN:  1,
		},
		{
			name:   "auto_id_cache=100",
			create: "CREATE TABLE `t` (`id` bigint NOT NULL AUTO_INCREMENT, PRIMARY KEY (`id`)) /*T![auto_id_cache] AUTO_ID_CACHE=100 */ CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
			wantN:  100,
		},
		{
			name:    "no auto_id_cache",
			create:  "CREATE TABLE `t` (`id` bigint NOT NULL, PRIMARY KEY (`id`)) CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := reAutoIDCache.FindStringSubmatch(tt.create)
			if tt.wantNil {
				require.Nil(t, matches)
			} else {
				require.NotNil(t, matches)
				require.Equal(t, tt.wantN, mustAtoi(t, matches[1]))
			}
		})
	}
}

func TestAutoIDCache_RegexOnlyMatchesTiDBComment(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		matches bool
	}{
		{
			name:    "valid TiDB comment",
			input:   "/*T![auto_id_cache] AUTO_ID_CACHE=1 */",
			matches: true,
		},
		{
			name:    "valid TiDB comment with spaces",
			input:   "/*T![auto_id_cache]  AUTO_ID_CACHE = 100 */",
			matches: true,
		},
		{
			name:    "SQL block comment should not match",
			input:   "/* AUTO_ID_CACHE=1 */",
			matches: false,
		},
		{
			name:    "plain text should not match",
			input:   "AUTO_ID_CACHE=1",
			matches: false,
		},
		{
			name:    "different TiDB feature tag should not match",
			input:   "/*T![other_feature] AUTO_ID_CACHE=1 */",
			matches: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := reAutoIDCache.FindStringSubmatch(tt.input)
			if tt.matches {
				require.NotNil(t, matches, "expected regex to match")
			} else {
				require.Nil(t, matches, "expected regex NOT to match")
			}
		})
	}
}

func TestAutoIDCache_PatchSchema(t *testing.T) {
	tbl := schema.NewTable("t")
	tbl.AddAttrs(&CreateStmt{
		S: "CREATE TABLE `t` (`id` bigint NOT NULL AUTO_INCREMENT, PRIMARY KEY (`id`)) /*T![auto_id_cache] AUTO_ID_CACHE=1 */ CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
	})
	i := &tinspect{}
	err := i.setAutoIDCache(tbl)
	require.NoError(t, err)
	aic := &AutoIDCache{}
	require.True(t, sqlx.Has(tbl.Attrs, aic))
	require.Equal(t, 1, aic.N)
}

func TestAutoIDCache_PatchSchemaDefault(t *testing.T) {
	// When TiDB outputs AUTO_ID_CACHE=30000 (the default), setAutoIDCache
	// should still add the attribute so that round-trips are consistent.
	tbl := schema.NewTable("t")
	tbl.AddAttrs(&CreateStmt{
		S: "CREATE TABLE `t` (`id` bigint NOT NULL AUTO_INCREMENT, PRIMARY KEY (`id`)) /*T![auto_id_cache] AUTO_ID_CACHE=30000 */ CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
	})
	i := &tinspect{}
	err := i.setAutoIDCache(tbl)
	require.NoError(t, err)
	aic := &AutoIDCache{}
	require.True(t, sqlx.Has(tbl.Attrs, aic))
	require.Equal(t, AutoIDCacheDefault, aic.N)
}

func TestAutoIDCache_PatchSchemaNoAutoIDCache(t *testing.T) {
	tbl := schema.NewTable("t")
	tbl.AddAttrs(&CreateStmt{
		S: "CREATE TABLE `t` (`id` bigint NOT NULL, PRIMARY KEY (`id`)) CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
	})
	i := &tinspect{}
	err := i.setAutoIDCache(tbl)
	require.NoError(t, err)
	require.False(t, sqlx.Has(tbl.Attrs, &AutoIDCache{}))
}

func TestTableAttrDiff_AutoIDCache(t *testing.T) {
	differ := newTiDBDiffer()
	testSchema := &schema.Schema{Name: "test"}

	t.Run("AddAutoIDCache", func(t *testing.T) {
		from := schema.NewTable("t")
		from.Schema = testSchema
		to := schema.NewTable("t")
		to.Schema = testSchema
		to.AddAttrs(&AutoIDCache{N: 1})
		changes, err := differ.TableDiff(from, to)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		// Adding AUTO_ID_CACHE generates a ModifyAttr (from default → desired)
		// so that the reverse SQL is correctly produced.
		modAttr, ok := changes[0].(*schema.ModifyAttr)
		require.True(t, ok, "expected ModifyAttr, got %T", changes[0])
		fromAIC, ok := modAttr.From.(*AutoIDCache)
		require.True(t, ok)
		require.Equal(t, AutoIDCacheDefault, fromAIC.N)
		toAIC, ok := modAttr.To.(*AutoIDCache)
		require.True(t, ok)
		require.Equal(t, 1, toAIC.N)
	})

	t.Run("ModifyAutoIDCache", func(t *testing.T) {
		from := schema.NewTable("t")
		from.Schema = testSchema
		from.AddAttrs(&AutoIDCache{N: 1})
		to := schema.NewTable("t")
		to.Schema = testSchema
		to.AddAttrs(&AutoIDCache{N: 100})
		changes, err := differ.TableDiff(from, to)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		modAttr, ok := changes[0].(*schema.ModifyAttr)
		require.True(t, ok, "expected ModifyAttr, got %T", changes[0])
		fromAIC, ok := modAttr.From.(*AutoIDCache)
		require.True(t, ok)
		require.Equal(t, 1, fromAIC.N)
		toAIC, ok := modAttr.To.(*AutoIDCache)
		require.True(t, ok)
		require.Equal(t, 100, toAIC.N)
	})

	t.Run("DropAutoIDCache", func(t *testing.T) {
		from := schema.NewTable("t")
		from.Schema = testSchema
		from.AddAttrs(&AutoIDCache{N: 100})
		to := schema.NewTable("t")
		to.Schema = testSchema
		changes, err := differ.TableDiff(from, to)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		modAttr, ok := changes[0].(*schema.ModifyAttr)
		require.True(t, ok, "expected ModifyAttr, got %T", changes[0])
		fromAIC, ok := modAttr.From.(*AutoIDCache)
		require.True(t, ok)
		require.Equal(t, 100, fromAIC.N)
		// Drop restores to TiDB default (30000), not 0,
		// because AUTO_ID_CACHE requires a minimum value of 1.
		toAIC, ok := modAttr.To.(*AutoIDCache)
		require.True(t, ok)
		require.Equal(t, AutoIDCacheDefault, toAIC.N)
	})

	t.Run("AddAutoIDCacheDefaultNoOp", func(t *testing.T) {
		// Setting AUTO_ID_CACHE to the default value (30000) should not
		// generate any change, since the table already uses that default.
		from := schema.NewTable("t")
		from.Schema = testSchema
		to := schema.NewTable("t")
		to.Schema = testSchema
		to.AddAttrs(&AutoIDCache{N: AutoIDCacheDefault})
		changes, err := differ.TableDiff(from, to)
		require.NoError(t, err)
		require.Empty(t, changes)
	})

	t.Run("DropAutoIDCacheNoOpWhenDefault", func(t *testing.T) {
		// If the current value is already the default, dropping it
		// should not generate any change (no-op).
		from := schema.NewTable("t")
		from.Schema = testSchema
		from.AddAttrs(&AutoIDCache{N: AutoIDCacheDefault})
		to := schema.NewTable("t")
		to.Schema = testSchema
		changes, err := differ.TableDiff(from, to)
		require.NoError(t, err)
		require.Empty(t, changes)
	})

	t.Run("NoChangeAutoIDCache", func(t *testing.T) {
		from := schema.NewTable("t")
		from.Schema = testSchema
		from.AddAttrs(&AutoIDCache{N: 100})
		to := schema.NewTable("t")
		to.Schema = testSchema
		to.AddAttrs(&AutoIDCache{N: 100})
		changes, err := differ.TableDiff(from, to)
		require.NoError(t, err)
		require.Empty(t, changes)
	})

	t.Run("ModifyFromDefaultToNonDefault", func(t *testing.T) {
		// Both sides have the attribute, but from is the default value.
		// This exercises the fromHasAIC && toHasAIC branch with the default on from.
		from := schema.NewTable("t")
		from.Schema = testSchema
		from.AddAttrs(&AutoIDCache{N: AutoIDCacheDefault})
		to := schema.NewTable("t")
		to.Schema = testSchema
		to.AddAttrs(&AutoIDCache{N: 1})
		changes, err := differ.TableDiff(from, to)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		modAttr, ok := changes[0].(*schema.ModifyAttr)
		require.True(t, ok, "expected ModifyAttr, got %T", changes[0])
		fromAIC, ok := modAttr.From.(*AutoIDCache)
		require.True(t, ok)
		require.Equal(t, AutoIDCacheDefault, fromAIC.N)
		toAIC, ok := modAttr.To.(*AutoIDCache)
		require.True(t, ok)
		require.Equal(t, 1, toAIC.N)
	})
}

func TestAutoIDCache_PatchSchemaNoCreateStmt(t *testing.T) {
	// setAutoIDCache should be a no-op when CreateStmt is absent.
	tbl := schema.NewTable("t")
	i := &tinspect{}
	err := i.setAutoIDCache(tbl)
	require.NoError(t, err)
	require.False(t, sqlx.Has(tbl.Attrs, &AutoIDCache{}))
}

func TestAutoIDCache_CombinedInspection(t *testing.T) {
	// Verify patchSchema correctly extracts both SHARD_ROW_ID_BITS and
	// AUTO_ID_CACHE from the same CREATE TABLE statement.
	tbl := schema.NewTable("t")
	tbl.AddAttrs(&CreateStmt{
		S: "CREATE TABLE `t` (`id` int NOT NULL, PRIMARY KEY (`id`) " +
			"/*T![clustered_index] NONCLUSTERED */) " +
			"/*T! SHARD_ROW_ID_BITS=4 */ " +
			"/*T![auto_id_cache] AUTO_ID_CACHE=1 */ " +
			"CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
	})
	i := &tinspect{}
	err := i.setShardRowIDBits(tbl)
	require.NoError(t, err)
	err = i.setAutoIDCache(tbl)
	require.NoError(t, err)

	shard := &ShardRowIDBits{}
	require.True(t, sqlx.Has(tbl.Attrs, shard))
	require.Equal(t, 4, shard.N)

	aic := &AutoIDCache{}
	require.True(t, sqlx.Has(tbl.Attrs, aic))
	require.Equal(t, 1, aic.N)
}

func TestTableAttrDiff_CombinedAutoIDCacheAndShardRowIDBits(t *testing.T) {
	// When both ShardRowIDBits and AutoIDCache change simultaneously,
	// the diff should produce two separate changes.
	differ := newTiDBDiffer()
	testSchema := &schema.Schema{Name: "test"}

	from := schema.NewTable("t")
	from.Schema = testSchema
	from.AddAttrs(&ShardRowIDBits{N: 2}, &AutoIDCache{N: 100})
	to := schema.NewTable("t")
	to.Schema = testSchema
	to.AddAttrs(&ShardRowIDBits{N: 4}, &AutoIDCache{N: 1})
	changes, err := differ.TableDiff(from, to)
	require.NoError(t, err)
	require.Len(t, changes, 2)

	// First change: ShardRowIDBits modification.
	modShard, ok := changes[0].(*schema.ModifyAttr)
	require.True(t, ok, "expected ModifyAttr for ShardRowIDBits, got %T", changes[0])
	_, ok = modShard.From.(*ShardRowIDBits)
	require.True(t, ok)

	// Second change: AutoIDCache modification.
	modAIC, ok := changes[1].(*schema.ModifyAttr)
	require.True(t, ok, "expected ModifyAttr for AutoIDCache, got %T", changes[1])
	_, ok = modAIC.From.(*AutoIDCache)
	require.True(t, ok)
}

func TestCheckUnsupportedChanges_AutoIDCacheAllowed(t *testing.T) {
	// Unlike PreSplitRegions, modifying AUTO_ID_CACHE via ALTER TABLE is
	// supported by TiDB. Verify it passes through without error.
	changes := []schema.Change{
		&schema.ModifyTable{
			T: schema.NewTable("users"),
			Changes: []schema.Change{
				&schema.ModifyAttr{
					From: &AutoIDCache{N: AutoIDCacheDefault},
					To:   &AutoIDCache{N: 1},
				},
			},
		},
	}
	err := checkUnsupportedChanges(changes)
	require.NoError(t, err)
}

func TestAutoRandom_ColumnDiffRemovalDetected(t *testing.T) {
	// Now that we detect AUTO_RANDOM removal, verify the diff is generated.
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
	changes, err := differ.TableDiff(fromT, toT)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	mc, ok := changes[0].(*schema.ModifyColumn)
	require.True(t, ok)
	require.True(t, mc.Change.Is(schema.ChangeAttr))
}
