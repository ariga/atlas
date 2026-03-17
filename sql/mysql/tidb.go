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
	// AutoRandomShardBitsMin is the minimum value for AUTO_RANDOM shard bits.
	AutoRandomShardBitsMin = 1
	// AutoRandomShardBitsMax is the maximum value for AUTO_RANDOM shard bits.
	AutoRandomShardBitsMax = 15
	// AutoRandomRangeBitsMin is the minimum value for AUTO_RANDOM range bits.
	AutoRandomRangeBitsMin = 32
	// AutoRandomRangeBitsMax is the maximum/default value for AUTO_RANDOM range bits.
	AutoRandomRangeBitsMax = 64
	// ShardRowIDBitsMax is the maximum value for SHARD_ROW_ID_BITS.
	ShardRowIDBitsMax = 15
	// AutoIDCacheDefault is the default value for AUTO_ID_CACHE in TiDB.
	AutoIDCacheDefault = 30000
)

type (
	// tplanApply decorates MySQL planApply.
	tplanApply struct{ planApply }
	// tdiff decorates MySQL diff.
	tdiff struct{ diff }
	// tinspect decorates MySQL inspect.
	tinspect struct{ inspect }
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
		// Each ModifyTable should have a single change since we apply `flat` before we sort.
		// Defensive check: if Changes is empty, return default priority.
		if len(c.Changes) == 0 {
			return 4
		}
		return priority(c.Changes[0])
	case *schema.ModifySchema:
		// Each ModifySchema should have a single change since we apply `flat` before we sort.
		// Defensive check: if Changes is empty, return default priority.
		if len(c.Changes) == 0 {
			return 4
		}
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
	var result []schema.Change
	for _, change := range changes {
		switch m := change.(type) {
		case *schema.ModifyTable:
			for _, c := range m.Changes {
				result = append(result, &schema.ModifyTable{
					T:       m.T,
					Changes: []schema.Change{c},
				})
			}
		case *schema.ModifySchema:
			for _, c := range m.Changes {
				result = append(result, &schema.ModifySchema{
					S:       m.S,
					Changes: []schema.Change{c},
				})
			}
		default:
			result = append(result, change)
		}
	}
	return result
}

// PlanChanges returns a migration plan for the given schema changes.
func (p *tplanApply) PlanChanges(ctx context.Context, name string, changes []schema.Change, opts ...migrate.PlanOption) (*migrate.Plan, error) {
	planned, err := sqlx.DetachCycles(changes)
	if err != nil {
		return nil, err
	}
	// Check for unsupported TiDB-specific changes before planning.
	if err := checkUnsupportedChanges(planned); err != nil {
		return nil, err
	}
	planned = flat(planned)
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
		// PRE_SPLIT_REGIONS cannot be changed after table creation.
		// ModifyAttr.From and .To are always the same type, so checking either suffices.
		if _, ok := c.From.(*PreSplitRegions); ok {
			return fmt.Errorf(
				"cannot modify pre_split_regions for table %q: "+
					"TiDB does not support changing PRE_SPLIT_REGIONS after table creation; "+
					"you must recreate the table to change this setting",
				tableName,
			)
		}
	case *schema.AddAttr:
		// PRE_SPLIT_REGIONS cannot be added after table creation.
		if _, ok := c.A.(*PreSplitRegions); ok {
			return fmt.Errorf(
				"cannot add pre_split_regions to table %q: "+
					"TiDB does not support adding PRE_SPLIT_REGIONS after table creation; "+
					"you must recreate the table to add this setting",
				tableName,
			)
		}
	case *schema.DropAttr:
		// PRE_SPLIT_REGIONS cannot be dropped after table creation.
		if _, ok := c.A.(*PreSplitRegions); ok {
			return fmt.Errorf(
				"cannot drop pre_split_regions from table %q: "+
					"TiDB does not support removing PRE_SPLIT_REGIONS after table creation; "+
					"you must recreate the table to remove this setting",
				tableName,
			)
		}
	case *schema.ModifyIndex:
		// Check if ClusteredIndex attribute is being changed.
		if c.From == nil || c.To == nil {
			break
		}
		var fromCI, toCI ClusteredIndex
		fromHas, toHas := sqlx.Has(c.From.Attrs, &fromCI), sqlx.Has(c.To.Attrs, &toCI)
		if fromHas && toHas && fromCI.Clustered != toCI.Clustered {
			return fmt.Errorf(
				"cannot change primary key clustering mode for table %q: "+
					"TiDB does not support changing between CLUSTERED and NONCLUSTERED after table creation; "+
					"you must recreate the table to change this setting",
				tableName,
			)
		}
	case *schema.ModifyColumn:
		// Check for AUTO_RANDOM modifications (not additions).
		if c.From == nil || c.To == nil {
			break
		}
		var fromAR, toAR AutoRandom
		fromHas, toHas := sqlx.Has(c.From.Attrs, &fromAR), sqlx.Has(c.To.Attrs, &toAR)
		// Adding AUTO_RANDOM requires BIGINT column type.
		if !fromHas && toHas {
			if !isBigIntColumn(c.To) {
				return fmt.Errorf(
					"cannot add AUTO_RANDOM to column %q in table %q: "+
						"AUTO_RANDOM is only supported on BIGINT columns",
					c.To.Name, tableName,
				)
			}
		}
		// Modifying existing AUTO_RANDOM parameters is not supported.
		if fromHas && toHas && (fromAR.ShardBits != toAR.ShardBits || fromAR.RangeBits != toAR.RangeBits) {
			return fmt.Errorf(
				"cannot modify AUTO_RANDOM for column %q in table %q: "+
					"TiDB does not support changing AUTO_RANDOM shard bits or range bits after column creation; "+
					"you must recreate the table to change this setting",
				c.From.Name, tableName,
			)
		}
		// Removing AUTO_RANDOM is not supported.
		if fromHas && !toHas {
			return fmt.Errorf(
				"cannot remove AUTO_RANDOM from column %q in table %q: "+
					"TiDB does not support removing AUTO_RANDOM after it has been set; "+
					"you must recreate the table to remove this setting",
				c.From.Name, tableName,
			)
		}
	}
	return nil
}

// isBigIntColumn checks if the column is a BIGINT type.
// TiDB's AUTO_RANDOM only supports BIGINT columns (signed or unsigned).
// The IntegerType.T field contains the base type name (e.g., "bigint"),
// while the Unsigned field indicates if it's unsigned.
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
	case !fromHasAIC && toHasAIC && toAIC.N > 0 && toAIC.N != AutoIDCacheDefault:
		// Adding AUTO_ID_CACHE. Use ModifyAttr (from default → desired) instead of
		// AddAttr so that the reverse SQL is correctly generated as a restore to default.
		changes = append(changes, &schema.ModifyAttr{
			From: &AutoIDCache{N: AutoIDCacheDefault},
			To:   &AutoIDCache{N: toAIC.N},
		})
	case fromHasAIC && !toHasAIC && fromAIC.N > 0 && fromAIC.N != AutoIDCacheDefault:
		// Dropping AUTO_ID_CACHE (restoring to TiDB default).
		// Unlike SHARD_ROW_ID_BITS (which can be set to 0), AUTO_ID_CACHE
		// requires a minimum value of 1, so we restore the default (30000).
		// Skip if the current value is already the default to avoid no-op ALTERs.
		changes = append(changes, &schema.ModifyAttr{
			From: &AutoIDCache{N: fromAIC.N},
			To:   &AutoIDCache{N: AutoIDCacheDefault},
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
	if c == nil || c.Type == nil || c.Type.Type == nil {
		return
	}
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
	// MySQL BIT type supports up to 64 bits (8 bytes).
	// If input exceeds 8 bytes, truncate to the last 8 bytes.
	if len(b) > 8 {
		b = b[len(b)-8:]
	}
	buf := make([]byte, 8)
	copy(buf[8-len(b):], b)
	val := binary.BigEndian.Uint64(buf)
	return fmt.Sprintf("b'%b'", val)
}

// e.g CHARSET=utf8mb4 COLLATE=utf8mb4_bin
var reColl = regexp.MustCompile(`(?i)CHARSET\s*=\s*(\w+)\s*COLLATE\s*=\s*(\w+)`)

// setCollate extracts the updated Collation from CREATE TABLE statement.
func (i *tinspect) setCollate(t *schema.Table) error {
	var c CreateStmt
	if !sqlx.Has(t.Attrs, &c) {
		return fmt.Errorf("mysql: missing CREATE TABLE statement in attributes for table %q; "+
			"this may indicate an internal error during schema inspection", t.Name)
	}
	matches := reColl.FindStringSubmatch(c.S)
	if len(matches) != 3 {
		return fmt.Errorf("mysql: could not extract CHARSET and COLLATE from CREATE TABLE statement for table %q; "+
			"expected format 'CHARSET=... COLLATE=...' but got: %s", t.Name, c.S)
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
// Pattern breakdown:
//   - `([^`]+)` captures the column name (any chars except backtick)
//   - [^`]*/\*T!\[auto_rand\] matches the TiDB-specific comment marker
//     This ensures we only match AUTO_RANDOM in TiDB comments, not in SQL comments
//   - \s*AUTO_RANDOM\((\d+) captures the shard bits (required, 1-15)
//   - (?:\s*,\s*(\d+))? optionally captures range bits (32-64, default 64)
//
// Group 1: column name, Group 2: shard bits, Group 3: optional range bits.
//
// Note: Column names with escaped backticks are extremely rare
// and not supported by this pattern.
var reAutoRandom = regexp.MustCompile("`([^`]+)`[^`]*/\\*T!\\[auto_rand\\]\\s*AUTO_RANDOM\\((\\d+)(?:\\s*,\\s*(\\d+))?\\)")

// setAutoRandom extracts the shard and range bits from CREATE TABLE statement.
// TiDB allows at most one AUTO_RANDOM column per table. Unlike older TiDB versions
// (v5/v6) which set "auto_random" in the EXTRA column of INFORMATION_SCHEMA, newer
// versions (v8+) leave EXTRA empty. Therefore, this function identifies the column
// directly from the CREATE TABLE statement rather than relying on a placeholder.
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
		if rangeBits != AutoRandomRangeBitsMax {
			ar.RangeBits = rangeBits
		}
	}
	schema.ReplaceOrAppend(&target.Attrs, ar)
	return nil
}

// setAutoIncrement extracts the actual AUTO_INCREMENT value from the CREATE TABLE statement.
func (i *tinspect) setAutoIncrement(t *schema.Table) error {
	// patch only it is set (set falsely to '1' due to this bug: https://github.com/pingcap/tidb/issues/24702).
	ai := &AutoIncrement{}
	if !sqlx.Has(t.Attrs, ai) {
		return nil
	}
	var c CreateStmt
	if !sqlx.Has(t.Attrs, &c) {
		return fmt.Errorf("mysql: missing CREATE TABLE statement in attributes for table %q", t.Name)
	}
	matches := reAutoinc.FindStringSubmatch(c.S)
	if len(matches) != 2 {
		return nil
	}
	v, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return fmt.Errorf("parsing AUTO_INCREMENT for table %q: %w", t.Name, err)
	}
	ai.V = v
	schema.ReplaceOrAppend(&t.Attrs, ai)
	return nil
}

// reShardRowID matches SHARD_ROW_ID_BITS=N in CREATE TABLE statement.
// TiDB wraps these in a special comment block:
//
//	) /*T! SHARD_ROW_ID_BITS=4 PRE_SPLIT_REGIONS=2 */
//
// The pattern requires the TiDB comment marker /*T! to ensure we only match
// TiDB-specific attributes, not values in SQL comments or other contexts.
// Valid range: 0-15 (0 means disabled).
var reShardRowID = regexp.MustCompile(`/\*T![^*]*SHARD_ROW_ID_BITS\s*=\s*(\d+)`)

// rePreSplitRegions matches PRE_SPLIT_REGIONS=N in CREATE TABLE statement.
// TiDB wraps these in a special comment block:
//
//	) /*T! SHARD_ROW_ID_BITS=4 PRE_SPLIT_REGIONS=2 */
//
// The pattern requires the TiDB comment marker /*T! to ensure we only match
// TiDB-specific attributes. Must be used together with SHARD_ROW_ID_BITS or AUTO_RANDOM.
// The value must be <= SHARD_ROW_ID_BITS (or AUTO_RANDOM shard bits).
// Creates 2^N regions at table creation time.
var rePreSplitRegions = regexp.MustCompile(`/\*T![^*]*PRE_SPLIT_REGIONS\s*=\s*(\d+)`)

// reAutoIDCache matches AUTO_ID_CACHE=N in CREATE TABLE statement.
// TiDB wraps this in a feature-tagged comment block:
//
//	/*T![auto_id_cache] AUTO_ID_CACHE=100 */
//
// Note: Unlike SHARD_ROW_ID_BITS/PRE_SPLIT_REGIONS which use the generic /*T! ... */
// format, AUTO_ID_CACHE uses the feature-tagged /*T![feature_name] ... */ format.
// The pattern matches the specific tag to avoid false positives.
var reAutoIDCache = regexp.MustCompile(`/\*T!\[auto_id_cache\]\s*AUTO_ID_CACHE\s*=\s*(\d+)`)

// reClustered matches CLUSTERED or NONCLUSTERED in PRIMARY KEY definition.
// TiDB wraps this in a special comment:
//
//	PRIMARY KEY (`id`) /*T![clustered_index] NONCLUSTERED */
//	PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */
//
// Pattern captures "NONCLUSTERED" first (if present), otherwise "CLUSTERED".
// This is important because "NONCLUSTERED" contains "CLUSTERED" as a substring.
// The pattern explicitly matches the TiDB comment format /*T![...] ... */.
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
	clustered := matches[1] == "CLUSTERED"
	t.PrimaryKey.AddAttrs(&ClusteredIndex{Clustered: clustered})
	return nil
}
