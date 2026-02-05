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

func (p *tplanApply) ApplyChanges(ctx context.Context, changes []schema.Change, opts ...migrate.PlanOption) error {
	return sqlx.ApplyChanges(ctx, changes, p, opts...)
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
	switch {
	case !fromHas && toHas:
		// AUTO_RANDOM was added.
	case fromHas && toHas && (fromAR.ShardBits != toAR.ShardBits || fromAR.RangeBits != toAR.RangeBits):
		// AUTO_RANDOM parameters changed.
	default:
		// No AUTO_RANDOM change detected. Note that we intentionally
		// skip the case where AUTO_RANDOM is removed (fromHas && !toHas),
		// because TiDB does not support dropping AUTO_RANDOM from a column.
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
//
// The [^`]* ensures we match the column name closest to AUTO_RANDOM (not the table name).
// Group 1: column name, Group 2: shard bits, Group 3: optional range bits.
var reAutoRandom = regexp.MustCompile("`([^`]+)`[^`]*AUTO_RANDOM\\((\\d+)(?:\\s*,\\s*(\\d+))?\\)")

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
		return err
	}
	ar := &AutoRandom{ShardBits: shard}
	if matches[3] != "" {
		rangeBits, err := strconv.Atoi(matches[3])
		if err != nil {
			return err
		}
		// Normalize the default range (64) to 0 so that HCL round-trips
		// are lossless: columnSpec omits auto_random_range when it equals
		// the default, and convertColumn reads the absence as 0.
		if rangeBits != 64 {
			ar.RangeBits = rangeBits
		}
	}
	schema.ReplaceOrAppend(&target.Attrs, ar)
	return nil
}

// setAutoIncrement extracts the actual AUTO_INCREMENT value from the CREATE TABLE statement.
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
