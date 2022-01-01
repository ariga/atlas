// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"fmt"
	"reflect"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// A diff provides a SQLite implementation for sqlx.DiffDriver.
type diff struct{ conn }

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
	return append(changes, sqlx.CheckDiff(from, to, func(c1, c2 *schema.Check) bool {
		return c1.Expr != c2.Expr
	})...)
}

// ColumnChange returns the schema changes (if any) for migrating one column to the other.
func (d *diff) ColumnChange(from, to *schema.Column) (schema.ChangeKind, error) {
	change := sqlx.CommentChange(from.Attrs, to.Attrs)
	if from.Type.Null != to.Type.Null {
		change |= schema.ChangeNull
	}
	changed, err := d.typeChanged(from, to)
	if err != nil {
		return schema.NoChange, err
	}
	if changed {
		change |= schema.ChangeType
	}
	if changed := d.defaultChanged(from, to); changed {
		change |= schema.ChangeDefault
	}
	return change, nil
}

// typeChanged reports if the a column type was changed.
func (d *diff) typeChanged(from, to *schema.Column) (bool, error) {
	fromT, toT := from.Type.Type, to.Type.Type
	if fromT == nil || toT == nil {
		return false, fmt.Errorf("sqlite: missing type infromation for column %q", from.Name)
	}
	// Types are mismatched if they do not have the same "type affinity".
	return reflect.TypeOf(fromT) != reflect.TypeOf(toT), nil
}

// defaultChanged reports if the a default value of a column
// type was changed.
func (d *diff) defaultChanged(from, to *schema.Column) bool {
	d1, ok1 := sqlx.DefaultValue(from)
	d2, ok2 := sqlx.DefaultValue(to)
	return ok1 != ok2 || d1 != d2
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

// Normalize implements the sqlx.Normalizer interface.
func (d *diff) Normalize(from, to *schema.Table) {
	used := make([]bool, len(to.ForeignKeys))
	// In SQLite, there is no easy way to get the foreign-key constraint
	// name, except for parsing the CREATE statement). Therefore, we check
	// if there is a foreign-key with identical properties.
	for _, fk1 := range from.ForeignKeys {
		for i, fk2 := range to.ForeignKeys {
			if used[i] {
				continue
			}
			if fk2.Symbol == fk1.Symbol && !isNumber(fk1.Symbol) || sameFK(fk1, fk2) {
				fk1.Symbol = fk2.Symbol
				used[i] = true
			}
		}
	}
}

func sameFK(fk1, fk2 *schema.ForeignKey) bool {
	if fk1.Table.Name != fk2.Table.Name || fk1.RefTable.Name != fk2.RefTable.Name ||
		len(fk1.Columns) != len(fk2.Columns) || len(fk1.RefColumns) != len(fk2.RefColumns) {
		return false
	}
	for i, c1 := range fk1.Columns {
		if c1.Name != fk2.Columns[i].Name {
			return false
		}
	}
	for i, c1 := range fk1.RefColumns {
		if c1.Name != fk2.RefColumns[i].Name {
			return false
		}
	}
	return true
}
