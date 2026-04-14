// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"context"
	"encoding/binary"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
)

// TiDB-specific constraints and defaults.
const (
	autoRandomShardBitsMin = 1
	autoRandomShardBitsMax = 15
	autoRandomRangeBitsMin = 32
	autoRandomRangeBitsMax = 64 // Also the default range.
	shardRowIDBitsMax      = 15
	autoIDCacheDefault     = 30000
	clustered              = "CLUSTERED"
	nonclustered           = "NONCLUSTERED"
)

type (
	// tplanApply decorates MySQL planApply.
	tplanApply struct{ planApply }
	// tdiff decorates MySQL diff.
	tdiff struct{ diff }
	// tinspect decorates MySQL inspect.
	tinspect struct{ inspect }

	// AutoRandom is a TiDB-specific attribute for AUTO_RANDOM primary key columns.
	AutoRandom struct {
		schema.Attr
		ShardBits int // 1-15, default 5
		RangeBits int // 32-64 or 0 for default (64)
	}
	// ShardRowIDBits distributes implicit _tidb_rowid across shards.
	ShardRowIDBits struct {
		schema.Attr
		N int // 0-15, 0 means disabled
	}
	// PreSplitRegions pre-splits a table into 2^N regions at creation time.
	PreSplitRegions struct {
		schema.Attr
		N int
	}
	// AutoIDCache controls the cache size for auto-increment ID allocation.
	AutoIDCache struct {
		schema.Attr
		N int // >= 1, default 30000
	}
	// ClusteredIndex indicates whether a primary key is CLUSTERED or NONCLUSTERED.
	ClusteredIndex struct {
		schema.Attr
		Clustered bool
	}
)

// priority computes the priority of each change.
//
// TiDB does not support multischema ALTERs (i.e. multiple changes in a single ALTER statement).
// Therefore, we have to break down each alter. This function helps order the ALTERs so they work.
// e.g. priority gives precedence to DropForeignKey over DropColumn, because a column cannot be
// dropped if its foreign key was not dropped before.
func priority(change schema.Change) int {
	switch c := change.(type) {
	case *schema.ModifyTable:
		// each modifyTable should have a single change since we apply `flat` before we sort.
		return priority(c.Changes[0])
	case *schema.ModifySchema:
		// each modifyTable should have a single change since we apply `flat` before we sort.
		return priority(c.Changes[0])
	case *schema.AddColumn:
		return 1
	case *schema.DropIndex, *schema.DropForeignKey, *schema.DropAttr, *schema.DropCheck:
		return 2
	case *schema.ModifyIndex, *schema.ModifyForeignKey:
		return 3
	default:
		return 4
	}
}

// flat takes a list of changes and breaks them down to single atomic changes (e.g: no ModifyTable
// with multiple AddColumn inside it). Note that, the only "changes" that include sub-changes are
// `ModifyTable` and `ModifySchema`.
func flat(changes []schema.Change) []schema.Change {
	var flat []schema.Change
	for _, change := range changes {
		switch m := change.(type) {
		case *schema.ModifyTable:
			for _, c := range m.Changes {
				flat = append(flat, &schema.ModifyTable{
					T:       m.T,
					Changes: []schema.Change{c},
				})
			}
		case *schema.ModifySchema:
			for _, c := range m.Changes {
				flat = append(flat, &schema.ModifySchema{
					S:       m.S,
					Changes: []schema.Change{c},
				})
			}
		default:
			flat = append(flat, change)
		}
	}
	return flat
}

// PlanChanges returns a migration plan for the given schema changes.
func (p *tplanApply) PlanChanges(ctx context.Context, name string, changes []schema.Change, opts ...migrate.PlanOption) (*migrate.Plan, error) {
	planned, err := sqlx.DetachCycles(changes)
	if err != nil {
		return nil, err
	}
	planned = flat(planned)
	if err := checkUnsupportedChanges(planned); err != nil {
		return nil, err
	}
	sort.SliceStable(planned, func(i, j int) bool {
		return priority(planned[i]) < priority(planned[j])
	})
	s := &state{
		conn: p.conn,
		Plan: migrate.Plan{
			Name: name,
			// A plan is reversible, if all
			// its changes are reversible.
			Reversible:    true,
			Transactional: false,
		},
	}
	for _, c := range planned {
		// Use the planner of MySQL with each "atomic" change.
		plan, err := p.planApply.PlanChanges(ctx, name, []schema.Change{c}, opts...)
		if err != nil {
			return nil, err
		}
		if !plan.Reversible {
			s.Plan.Reversible = false
		}
		s.Plan.Changes = append(s.Plan.Changes, plan.Changes...)
	}
	return &s.Plan, nil
}

// checkUnsupportedChanges checks for TiDB-specific changes that cannot be
// applied via ALTER TABLE and returns an error with guidance.
func checkUnsupportedChanges(changes []schema.Change) error {
	for _, c := range changes {
		switch c := c.(type) {
		case *schema.ModifyTable:
			for _, tc := range c.Changes {
				if err := checkUnsupportedTableChange(c.T.Name, tc); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func checkUnsupportedTableChange(tableName string, c schema.Change) error {
	switch c := c.(type) {
	case *schema.ModifyAttr:
		if _, ok := c.From.(*PreSplitRegions); ok {
			return fmt.Errorf("cannot modify pre_split_regions for table %q: recreate the table to apply this change", tableName)
		}
	case *schema.AddAttr:
		if _, ok := c.A.(*PreSplitRegions); ok {
			return fmt.Errorf("cannot add pre_split_regions to table %q: recreate the table to apply this change", tableName)
		}
	case *schema.DropAttr:
		if _, ok := c.A.(*PreSplitRegions); ok {
			return fmt.Errorf("cannot drop pre_split_regions from table %q: recreate the table to apply this change", tableName)
		}
	case *schema.ModifyIndex:
		if c.From == nil || c.To == nil {
			break
		}
		var fromCI, toCI ClusteredIndex
		fromHas, toHas := sqlx.Has(c.From.Attrs, &fromCI), sqlx.Has(c.To.Attrs, &toCI)
		if fromHas && toHas && fromCI.Clustered != toCI.Clustered {
			return fmt.Errorf("cannot change clustering mode for table %q: recreate the table to apply this change", tableName)
		}
	case *schema.ModifyColumn:
		if c.From == nil || c.To == nil {
			break
		}
		var fromAR, toAR AutoRandom
		fromHas, toHas := sqlx.Has(c.From.Attrs, &fromAR), sqlx.Has(c.To.Attrs, &toAR)
		if !fromHas && toHas && !isBigIntColumn(c.To) {
			return fmt.Errorf("cannot add AUTO_RANDOM to column %q in table %q: AUTO_RANDOM requires BIGINT", c.To.Name, tableName)
		}
		if fromHas && toHas && (fromAR.ShardBits != toAR.ShardBits || fromAR.RangeBits != toAR.RangeBits) {
			return fmt.Errorf("cannot modify AUTO_RANDOM for column %q in table %q: recreate the table to apply this change", c.From.Name, tableName)
		}
		if fromHas && !toHas {
			return fmt.Errorf("cannot remove AUTO_RANDOM from column %q in table %q: recreate the table to apply this change", c.From.Name, tableName)
		}
	}
	return nil
}

// isBigIntColumn reports whether the column type is BIGINT (signed or unsigned).
func isBigIntColumn(c *schema.Column) bool {
	if c == nil || c.Type == nil || c.Type.Type == nil {
		return false
	}
	it, ok := c.Type.Type.(*schema.IntegerType)
	if !ok {
		return false
	}
	// Case-insensitive comparison to handle variations like "BIGINT", "bigint", etc.
	return strings.EqualFold(it.T, "bigint")
}

func (p *tplanApply) ApplyChanges(ctx context.Context, changes []schema.Change, opts ...migrate.PlanOption) error {
	return sqlx.ApplyChanges(ctx, changes, p, opts...)
}

// TableAttrDiff returns a changeset for migrating table attributes from one state to the other,
// including TiDB-specific attributes like SHARD_ROW_ID_BITS and PRE_SPLIT_REGIONS.
func (d *tdiff) TableAttrDiff(from, to *schema.Table, opts *schema.DiffOptions) ([]schema.Change, error) {
	changes, err := d.diff.TableAttrDiff(from, to, opts)
	if err != nil {
		return nil, err
	}
	// Compare ShardRowIDBits.
	var fromS, toS ShardRowIDBits
	fromHasS, toHasS := sqlx.Has(from.Attrs, &fromS), sqlx.Has(to.Attrs, &toS)
	switch {
	case !fromHasS && toHasS && toS.N > 0:
		// Adding SHARD_ROW_ID_BITS. Use ModifyAttr (from 0 → desired) instead of
		// AddAttr so that the reverse SQL is correctly generated as SHARD_ROW_ID_BITS = 0.
		changes = append(changes, &schema.ModifyAttr{
			From: &ShardRowIDBits{N: 0},
			To:   &ShardRowIDBits{N: toS.N},
		})
	case fromHasS && !toHasS && fromS.N > 0:
		// Dropping SHARD_ROW_ID_BITS (setting to 0).
		changes = append(changes, &schema.ModifyAttr{
			From: &ShardRowIDBits{N: fromS.N},
			To:   &ShardRowIDBits{N: 0},
		})
	case fromHasS && toHasS && fromS.N != toS.N:
		// Modifying SHARD_ROW_ID_BITS value.
		changes = append(changes, &schema.ModifyAttr{
			From: &ShardRowIDBits{N: fromS.N},
			To:   &ShardRowIDBits{N: toS.N},
		})
	}
	// Compare AutoIDCache.
	var fromAIC, toAIC AutoIDCache
	fromHasAIC, toHasAIC := sqlx.Has(from.Attrs, &fromAIC), sqlx.Has(to.Attrs, &toAIC)
	switch {
	case !fromHasAIC && toHasAIC && toAIC.N > 0 && toAIC.N != autoIDCacheDefault:
		// Adding AUTO_ID_CACHE. Use ModifyAttr (from default → desired) instead of
		// AddAttr so that the reverse SQL is correctly generated as a restore to default.
		changes = append(changes, &schema.ModifyAttr{
			From: &AutoIDCache{N: autoIDCacheDefault},
			To:   &AutoIDCache{N: toAIC.N},
		})
	case fromHasAIC && !toHasAIC && fromAIC.N > 0 && fromAIC.N != autoIDCacheDefault:
		// Dropping AUTO_ID_CACHE (restoring to TiDB default).
		// Unlike SHARD_ROW_ID_BITS (which can be set to 0), AUTO_ID_CACHE
		// requires a minimum value of 1, so we restore the default (30000).
		// Skip if the current value is already the default to avoid no-op ALTERs.
		changes = append(changes, &schema.ModifyAttr{
			From: &AutoIDCache{N: fromAIC.N},
			To:   &AutoIDCache{N: autoIDCacheDefault},
		})
	case fromHasAIC && toHasAIC && fromAIC.N != toAIC.N:
		// Modifying AUTO_ID_CACHE value.
		changes = append(changes, &schema.ModifyAttr{
			From: &AutoIDCache{N: fromAIC.N},
			To:   &AutoIDCache{N: toAIC.N},
		})
	}
	// Compare PreSplitRegions.
	// Note: PRE_SPLIT_REGIONS can only be set at table creation time in TiDB,
	// so changes to it after creation are not supported. We generate the appropriate
	// change type so that checkUnsupportedChanges can provide a helpful error message.
	var fromP, toP PreSplitRegions
	fromHasP, toHasP := sqlx.Has(from.Attrs, &fromP), sqlx.Has(to.Attrs, &toP)
	switch {
	case !fromHasP && toHasP && toP.N > 0:
		// Adding PRE_SPLIT_REGIONS (not supported by TiDB after table creation).
		changes = append(changes, &schema.AddAttr{A: &PreSplitRegions{N: toP.N}})
	case fromHasP && !toHasP && fromP.N > 0:
		// Dropping PRE_SPLIT_REGIONS (not supported by TiDB).
		changes = append(changes, &schema.DropAttr{A: &PreSplitRegions{N: fromP.N}})
	case fromHasP && toHasP && fromP.N != toP.N:
		// Modifying PRE_SPLIT_REGIONS (not supported by TiDB).
		changes = append(changes, &schema.ModifyAttr{
			From: &PreSplitRegions{N: fromP.N},
			To:   &PreSplitRegions{N: toP.N},
		})
	}
	return changes, nil
}

// IndexAttrChanged reports if the index attributes were changed.
// For TiDB, we also compare the ClusteredIndex attribute.
//
// Note: We intentionally do NOT report a change when ClusteredIndex is added
// or dropped (i.e., when one side has the attribute and the other doesn't).
// This is because:
//  1. TiDB's default clustering mode depends on @@tidb_enable_clustered_index,
//     column types, and TiDB version. The inspected schema may not have the
//     attribute if the default is used.
//  2. TiDB does not support changing clustering mode after table creation,
//     so reporting such a change would only produce an unsupported migration.
//  3. Comparing explicit CLUSTERED/NONCLUSTERED against the absence of the
//     attribute could cause false positives across different environments.
func (d *tdiff) IndexAttrChanged(from, to []schema.Attr) bool {
	if d.diff.IndexAttrChanged(from, to) {
		return true
	}
	var fromCI, toCI ClusteredIndex
	fromHas, toHas := sqlx.Has(from, &fromCI), sqlx.Has(to, &toCI)
	// Only report a change if both sides have the attribute and they differ.
	if !fromHas || !toHas {
		return false
	}
	return fromCI.Clustered != toCI.Clustered
}

// ColumnChange returns the schema changes (if any) for migrating one column to the other,
// including AUTO_RANDOM attribute changes specific to TiDB.
func (d *tdiff) ColumnChange(fromT *schema.Table, from, to *schema.Column, opts *schema.DiffOptions) (schema.Change, error) {
	change, err := d.diff.ColumnChange(fromT, from, to, opts)
	if err != nil {
		return sqlx.NoChange, err
	}
	var fromAR, toAR AutoRandom
	fromHas, toHas := sqlx.Has(from.Attrs, &fromAR), sqlx.Has(to.Attrs, &toAR)
	// Check if AUTO_RANDOM attribute changed.
	autoRandomChanged := false
	switch {
	case !fromHas && toHas:
		// AUTO_RANDOM was added.
		autoRandomChanged = true
	case fromHas && toHas && (fromAR.ShardBits != toAR.ShardBits || fromAR.RangeBits != toAR.RangeBits):
		// AUTO_RANDOM parameters changed (not supported by TiDB, will be caught by checkUnsupportedChanges).
		autoRandomChanged = true
	case fromHas && !toHas:
		// AUTO_RANDOM was removed (not supported by TiDB, will be caught by checkUnsupportedChanges).
		autoRandomChanged = true
	}
	if !autoRandomChanged {
		return change, nil
	}
	if change == sqlx.NoChange {
		return &schema.ModifyColumn{
			Change: schema.ChangeAttr,
			From:   from,
			To:     to,
		}, nil
	}
	if mc, ok := change.(*schema.ModifyColumn); ok {
		mc.Change |= schema.ChangeAttr
	}
	return change, nil
}

func (i *tinspect) InspectSchema(ctx context.Context, name string, opts *schema.InspectOptions) (*schema.Schema, error) {
	s, err := i.inspect.InspectSchema(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	return i.patchSchema(ctx, s)
}

func (i *tinspect) InspectRealm(ctx context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	r, err := i.inspect.InspectRealm(ctx, opts)
	if err != nil {
		return nil, err
	}
	for _, s := range r.Schemas {
		if _, err := i.patchSchema(ctx, s); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (i *tinspect) patchSchema(ctx context.Context, s *schema.Schema) (*schema.Schema, error) {
	for _, t := range s.Tables {
		var createStmt CreateStmt
		if ok := sqlx.Has(t.Attrs, &createStmt); !ok {
			if _, err := i.createStmt(ctx, t); err != nil {
				return nil, err
			}
		}
		if err := i.setCollate(t); err != nil {
			return nil, err
		}
		if err := i.setAutoIncrement(t); err != nil {
			return nil, err
		}
		if err := i.setAutoRandom(t); err != nil {
			return nil, err
		}
		if err := i.setShardRowIDBits(t); err != nil {
			return nil, err
		}
		if err := i.setClusteredIndex(t); err != nil {
			return nil, err
		}
		if err := i.setAutoIDCache(t); err != nil {
			return nil, err
		}
		for _, c := range t.Columns {
			i.patchColumn(ctx, c)
		}
	}
	return s, nil
}

func (i *tinspect) patchColumn(_ context.Context, c *schema.Column) {
	_, ok := c.Type.Type.(*BitType)
	if !ok {
		return
	}
	// TiDB has a bug where it does not format bit default value correctly.
	if lit, ok := c.Default.(*schema.Literal); ok && !strings.HasPrefix(lit.V, "b'") {
		lit.V = bytesToBitLiteral([]byte(lit.V))
	}
}

// bytesToBitLiteral converts a bytes to MySQL bit literal.
// e.g. []byte{4} -> b'100', []byte{2,1} -> b'1000000001'.
// See: https://github.com/pingcap/tidb/issues/32655.
func bytesToBitLiteral(b []byte) string {
	bytes := make([]byte, 8)
	for i := 0; i < len(b); i++ {
		bytes[8-len(b)+i] = b[i]
	}
	val := binary.BigEndian.Uint64(bytes)
	return fmt.Sprintf("b'%b'", val)
}

// e.g CHARSET=utf8mb4 COLLATE=utf8mb4_bin
var reColl = regexp.MustCompile(`(?i)CHARSET\s*=\s*(\w+)\s*COLLATE\s*=\s*(\w+)`)

// setCollate extracts the updated Collation from CREATE TABLE statement.
func (i *tinspect) setCollate(t *schema.Table) error {
	var c CreateStmt
	if !sqlx.Has(t.Attrs, &c) {
		return fmt.Errorf("missing CREATE TABLE statement in attributes for %q", t.Name)
	}
	matches := reColl.FindStringSubmatch(c.S)
	if len(matches) != 3 {
		return fmt.Errorf("missing COLLATE and/or CHARSET information on CREATE TABLE statement for %q", t.Name)
	}
	t.SetCharset(matches[1])
	t.SetCollation(matches[2])
	return nil
}

// reAutoRandom matches AUTO_RANDOM(S) or AUTO_RANDOM(S, R) in a CREATE TABLE column
// definition and captures the column name. TiDB wraps this in a special comment:
//
//	`id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(5) */
//	`id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(5, 64) */
//
// Captures: (1) column name, (2) shard bits, (3) optional range bits.
var reAutoRandom = regexp.MustCompile("`([^`]+)`[^`\n]*/\\*T!\\[auto_rand\\]\\s*AUTO_RANDOM\\((\\d+)(?:\\s*,\\s*(\\d+))?\\)")

// setAutoRandom extracts AUTO_RANDOM shard and range bits from the CREATE TABLE statement.
func (i *tinspect) setAutoRandom(t *schema.Table) error {
	var c CreateStmt
	if !sqlx.Has(t.Attrs, &c) {
		return nil
	}
	matches := reAutoRandom.FindStringSubmatch(c.S)
	if matches == nil {
		return nil
	}
	colName := matches[1]
	target, ok := t.Column(colName)
	if !ok {
		return fmt.Errorf("column %q referenced by AUTO_RANDOM not found in table %q", colName, t.Name)
	}
	shard, err := strconv.Atoi(matches[2])
	if err != nil {
		return fmt.Errorf("parsing AUTO_RANDOM shard bits for column %q in table %q: %w", colName, t.Name, err)
	}
	ar := &AutoRandom{ShardBits: shard}
	if matches[3] != "" {
		rangeBits, err := strconv.Atoi(matches[3])
		if err != nil {
			return fmt.Errorf("parsing AUTO_RANDOM range bits for column %q in table %q: %w", colName, t.Name, err)
		}
		// Normalize the default range (64) to 0 so that HCL round-trips
		// are lossless: columnSpec omits auto_random_range when it equals
		// the default, and convertColumn reads the absence as 0.
		if rangeBits != autoRandomRangeBitsMax {
			ar.RangeBits = rangeBits
		}
	}
	schema.ReplaceOrAppend(&target.Attrs, ar)
	return nil
}

func (i *tinspect) setAutoIncrement(t *schema.Table) error {
	// patch only it is set (set falsely to '1' due to this bug:https://github.com/pingcap/tidb/issues/24702).
	ai := &AutoIncrement{}
	if !sqlx.Has(t.Attrs, ai) {
		return nil
	}
	var c CreateStmt
	if !sqlx.Has(t.Attrs, &c) {
		return fmt.Errorf("missing CREATE TABLE statement in attributes for %q", t.Name)
	}
	matches := reAutoinc.FindStringSubmatch(c.S)
	if len(matches) != 2 {
		return nil
	}
	v, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return err
	}
	ai.V = v
	schema.ReplaceOrAppend(&t.Attrs, ai)
	return nil
}

// reShardRowID matches SHARD_ROW_ID_BITS=N inside TiDB's /*T! ... */ comment block.
var reShardRowID = regexp.MustCompile(`/\*T!.*?SHARD_ROW_ID_BITS\s*=\s*(\d+)`)

// rePreSplitRegions matches PRE_SPLIT_REGIONS=N inside TiDB's /*T! ... */ comment block.
var rePreSplitRegions = regexp.MustCompile(`/\*T!.*?PRE_SPLIT_REGIONS\s*=\s*(\d+)`)

// reAutoIDCache matches AUTO_ID_CACHE=N inside TiDB's /*T![auto_id_cache] ... */ comment block.
var reAutoIDCache = regexp.MustCompile(`/\*T!\[auto_id_cache\]\s*AUTO_ID_CACHE\s*=\s*(\d+)`)

// reClustered matches CLUSTERED or NONCLUSTERED inside TiDB's /*T![clustered_index] ... */ comment block.
// "NONCLUSTERED" is listed first in the alternation because it contains "CLUSTERED" as a substring.
var reClustered = regexp.MustCompile(`PRIMARY\s+KEY\s*\([^)]+\)\s*/\*T!\[clustered_index\]\s*(NONCLUSTERED|CLUSTERED)\s*\*/`)

// setShardRowIDBits extracts SHARD_ROW_ID_BITS and PRE_SPLIT_REGIONS from CREATE TABLE.
func (i *tinspect) setShardRowIDBits(t *schema.Table) error {
	var c CreateStmt
	if !sqlx.Has(t.Attrs, &c) {
		return nil
	}
	// Extract SHARD_ROW_ID_BITS.
	if matches := reShardRowID.FindStringSubmatch(c.S); matches != nil {
		n, err := strconv.Atoi(matches[1])
		if err != nil {
			return fmt.Errorf("parsing SHARD_ROW_ID_BITS for table %q: %w", t.Name, err)
		}
		if n > 0 {
			t.AddAttrs(&ShardRowIDBits{N: n})
		}
	}
	// Extract PRE_SPLIT_REGIONS.
	if matches := rePreSplitRegions.FindStringSubmatch(c.S); matches != nil {
		n, err := strconv.Atoi(matches[1])
		if err != nil {
			return fmt.Errorf("parsing PRE_SPLIT_REGIONS for table %q: %w", t.Name, err)
		}
		if n > 0 {
			t.AddAttrs(&PreSplitRegions{N: n})
		}
	}
	return nil
}

// setAutoIDCache extracts AUTO_ID_CACHE from CREATE TABLE statement.
func (i *tinspect) setAutoIDCache(t *schema.Table) error {
	var c CreateStmt
	if !sqlx.Has(t.Attrs, &c) {
		return nil
	}
	matches := reAutoIDCache.FindStringSubmatch(c.S)
	if matches == nil {
		return nil
	}
	n, err := strconv.Atoi(matches[1])
	if err != nil {
		return fmt.Errorf("parsing AUTO_ID_CACHE for table %q: %w", t.Name, err)
	}
	if n > 0 {
		t.AddAttrs(&AutoIDCache{N: n})
	}
	return nil
}

// setClusteredIndex extracts CLUSTERED/NONCLUSTERED from PRIMARY KEY definition.
func (i *tinspect) setClusteredIndex(t *schema.Table) error {
	if t.PrimaryKey == nil {
		return nil
	}
	var c CreateStmt
	if !sqlx.Has(t.Attrs, &c) {
		return nil
	}
	matches := reClustered.FindStringSubmatch(c.S)
	if matches == nil {
		return nil
	}
	t.PrimaryKey.AddAttrs(&ClusteredIndex{Clustered: matches[1] == clustered})
	return nil
}
