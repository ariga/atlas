// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// A diff provides a SQLite implementation for sqlx.DiffDriver.
type diff struct {
	version string
}

// Diff returns a SQLite schema differ.
func (d Driver) Diff() schema.Differ {
	return &sqlx.Diff{
		DiffDriver: &diff{version: d.version},
	}
}

// SchemaAttrDiff returns a changeset for migrating schema attributes from one state to the other.
func (d *diff) SchemaAttrDiff(_, _ *schema.Schema) []schema.Change {
	// No special schema attribute diffing for SQLite.
	return nil
}

// TableAttrDiff returns a changeset for migrating table attributes from one state to the other.
func (d *diff) TableAttrDiff(from, to *schema.Table) []schema.Change {
	var changes []schema.Change
	switch {
	case sqlx.Has(from.Attrs, &WithoutRowID{}) && !sqlx.Has(to.Attrs, &WithoutRowID{}):
		changes = append(changes, &schema.DropAttr{
			A: &WithoutRowID{},
		})
	case !sqlx.Has(from.Attrs, &WithoutRowID{}) && sqlx.Has(to.Attrs, &WithoutRowID{}):
		changes = append(changes, &schema.AddAttr{
			A: &WithoutRowID{},
		})
	}
	// TODO: support diffing constraints after it's supported.
	return changes
}

// ColumnTypeChanged reports if the a column type was changed.
func (d *diff) ColumnTypeChanged(c1, c2 *schema.Column) (bool, error) {
	changed, err := sqlx.ColumnTypeChanged(c1, c2)
	if sqlx.IsUnsupportedTypeError(err) {
		// All integer types have the same "type affinity".
		if _, ok := c1.Type.Type.(*schema.IntegerType); ok {
			return false, nil
		}
	}
	return changed, err
}

// ColumnDefaultChanged reports if the a default value of a column
// type was changed.
func (d *diff) ColumnDefaultChanged(from, to *schema.Column) bool {
	d1, ok1 := from.Default.(*schema.RawExpr)
	d2, ok2 := to.Default.(*schema.RawExpr)
	return ok1 != ok2 || ok1 && d1.X != d2.X
}

// IndexAttrChanged reports if the index attributes were changed.
func (*diff) IndexAttrChanged(from, to []schema.Attr) bool {
	var p1, p2 IndexPredicate
	if sqlx.Has(from, &p1) != sqlx.Has(to, &p2) || p1.P != p2.P {
		return true
	}
	return false
}

// IndexPartAttrChanged reports if the index-part attributes were changed.
func (*diff) IndexPartAttrChanged(_, _ []schema.Attr) bool {
	return false
}

// ReferenceChanged reports if the foreign key referential action was changed.
func (*diff) ReferenceChanged(from, to schema.ReferenceOption) bool {
	// According to SQLite, if an action is not explicitly
	// specified, it defaults to "NO ACTION".
	if from == "" {
		from = schema.NoAction
	}
	if to == "" {
		to = schema.NoAction
	}
	return from != to
}
