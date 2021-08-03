// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"

	"golang.org/x/mod/semver"
)

// A diff provides a MySQL implementation for sqlx.DiffDriver.
type diff struct {
	version string
}

// Diff returns a MySQL schema differ.
func (d Driver) Diff() schema.Differ {
	return &sqlx.Diff{
		DiffDriver: &diff{version: d.version},
	}
}

// SchemaAttrDiff returns a changeset for migrating schema attributes from one state to the other.
func (d *diff) SchemaAttrDiff(from, to *schema.Schema) []schema.Change {
	var changes []schema.Change
	// Charset change.
	if change := d.charsetChange(from.Attrs, from.Realm.Attrs, to.Attrs); change != noChange {
		changes = append(changes, change)
	}
	// Collation change.
	if change := d.collationChange(from.Attrs, from.Realm.Attrs, to.Attrs); change != noChange {
		changes = append(changes, change)
	}
	return changes
}

// TableAttrDiff returns a changeset for migrating table attributes from one state to the other.
func (d *diff) TableAttrDiff(from, to *schema.Table) []schema.Change {
	var changes []schema.Change
	// Charset change.
	if change := d.charsetChange(from.Attrs, from.Schema.Attrs, to.Attrs); change != noChange {
		changes = append(changes, change)
	}
	// Collation change.
	if change := d.collationChange(from.Attrs, from.Schema.Attrs, to.Attrs); change != noChange {
		changes = append(changes, change)
	}
	// Drop or modify checks.
	for _, c1 := range checks(from.Attrs) {
		switch c2, ok := checkByName(to.Attrs, c1.Name); {
		case !ok:
			changes = append(changes, &schema.DropAttr{
				A: c1,
			})
		case c1.Clause != c2.Clause || c1.Enforced != c2.Enforced:
			changes = append(changes, &schema.ModifyAttr{
				From: c1,
				To:   c2,
			})
		}
	}
	// Add checks.
	for _, c1 := range checks(to.Attrs) {
		if _, ok := checkByName(from.Attrs, c1.Name); !ok {
			changes = append(changes, &schema.AddAttr{
				A: c1,
			})
		}
	}
	return changes
}

// ColumnTypeChanged reports if the a column type was changed.
func (d *diff) ColumnTypeChanged(c1, c2 *schema.Column) (bool, error) {
	changed, err := sqlx.ColumnTypeChanged(c1, c2)
	if sqlx.IsUnsupportedTypeError(err) {
		return d.typeChanged(c1, c2)
	}
	return changed, err
}

// IndexAttrChanged reports if the index attributes were changed.
func (*diff) IndexAttrChanged(from, to []schema.Attr) bool {
	return indexType(from).T != indexType(to).T
}

// IndexPartAttrChanged reports if the index-part attributes were changed.
func (*diff) IndexPartAttrChanged(from, to []schema.Attr) bool {
	return indexCollation(from).V != indexCollation(to).V
}

// ReferenceChanged reports if the foreign key referential action was changed.
func (*diff) ReferenceChanged(from, to schema.ReferenceOption) bool {
	// According to MySQL docs, foreign key constraints are checked
	// immediately, so NO ACTION is the same as RESTRICT. Specifying
	// RESTRICT (or NO ACTION) is the same as omitting the ON DELETE
	// or ON UPDATE clause.
	if from == "" || from == schema.Restrict {
		from = schema.NoAction
	}
	if to == "" || to == schema.Restrict {
		to = schema.NoAction
	}
	return from != to
}

// collationChange returns the schema change for migrating the collation if
// it was changed and its not the default attribute inherited from its parent.
func (d *diff) collationChange(from, top, to []schema.Attr) schema.Change {
	var fromC, topC, toC schema.Collation
	switch fromHas, topHas, toHas := sqlx.Has(from, &fromC), sqlx.Has(top, &topC), sqlx.Has(to, &toC); {
	case !fromHas && !toHas:
	case !fromHas:
		return &schema.AddAttr{
			A: &toC,
		}
	case !toHas:
		if !topHas || fromC.V != topC.V {
			return &schema.DropAttr{
				A: &fromC,
			}
		}
	case fromC.V != toC.V:
		return &schema.ModifyAttr{
			From: &fromC,
			To:   &toC,
		}
	}
	return noChange
}

// charsetChange returns the schema change for migrating the collation if
// it was changed and its not the default attribute inherited from its parent.
func (d *diff) charsetChange(from, top, to []schema.Attr) schema.Change {
	var fromC, topC, toC schema.Charset
	switch fromHas, topHas, toHas := sqlx.Has(from, &fromC), sqlx.Has(top, &topC), sqlx.Has(to, &toC); {
	case !fromHas && !toHas:
	case !fromHas:
		return &schema.AddAttr{
			A: &toC,
		}
	case !toHas:
		if !topHas || fromC.V != topC.V {
			return &schema.DropAttr{
				A: &fromC,
			}
		}
	case fromC.V != toC.V:
		return &schema.ModifyAttr{
			From: &fromC,
			To:   &toC,
		}
	}
	return noChange
}

// indexCollation returns the index collation from its attribute.
// The default collation is ascending if no order was specified.
func indexCollation(attr []schema.Attr) *schema.Collation {
	c := &schema.Collation{V: "A"}
	if sqlx.Has(attr, c) {
		c.V = strings.ToUpper(c.V)
	}
	return c
}

// indexType returns the index type from its attribute.
// The default type is BTREE if no type was specified.
func indexType(attr []schema.Attr) *IndexType {
	t := &IndexType{T: "BTREE"}
	if sqlx.Has(attr, t) {
		t.T = strings.ToUpper(t.T)
	}
	return t
}

// noChange describes a zero change.
var noChange struct{ schema.Change }

func checks(attr []schema.Attr) (checks []*Check) {
	for i := range attr {
		if c, ok := attr[i].(*Check); ok {
			checks = append(checks, c)
		}
	}
	return checks
}

func (d *diff) typeChanged(from, to *schema.Column) (bool, error) {
	fromT, toT := from.Type.Type, to.Type.Type
	var changed bool
	switch fromT := fromT.(type) {
	case *schema.IntegerType:
		toT := toT.(*schema.IntegerType)
		// MySQL v8.0.19 dropped the display width
		// information from the information schema.
		if semver.Compare("v"+d.version, "v8.0.19") == -1 {
			ft, _, _, err := parseColumn(fromT.T)
			if err != nil {
				return false, err
			}
			tt, _, _, err := parseColumn(toT.T)
			if err != nil {
				return false, err
			}
			fromT.T, toT.T = ft[0], tt[0]
		}
		fromW, toW := displayWidth(fromT.Attrs), displayWidth(toT.Attrs)
		changed = fromT.T != toT.T || fromT.Unsigned != toT.Unsigned ||
			(fromW != nil) != (toW != nil) || (fromW != nil && fromW.N != toW.N)
	case *schema.JSONType:
		toT := toT.(*schema.JSONType)
		changed = fromT.T != toT.T
	case *schema.StringType:
		toT := toT.(*schema.StringType)
		changed = fromT.T != toT.T || fromT.Size != toT.Size
	case *schema.SpatialType:
		toT := toT.(*schema.SpatialType)
		changed = fromT.T != toT.T
	case *schema.TimeType:
		toT := toT.(*schema.TimeType)
		changed = fromT.T != toT.T
	case *BitType:
		toT := toT.(*BitType)
		changed = fromT.T != toT.T
	case *SetType:
		toT := toT.(*SetType)
		changed = !sqlx.ValuesEqual(fromT.Values, toT.Values)
	default:
		return false, &sqlx.UnsupportedTypeError{Type: fromT}
	}
	return changed, nil
}

func checkByName(attr []schema.Attr, name string) (*Check, bool) {
	for i := range attr {
		if c, ok := attr[i].(*Check); ok && c.Name == name {
			return c, true
		}
	}
	return nil, false
}

func displayWidth(attr []schema.Attr) *DisplayWidth {
	var (
		z *ZeroFill
		d *DisplayWidth
	)
	for i := range attr {
		switch at := attr[i].(type) {
		case *ZeroFill:
			z = at
		case *DisplayWidth:
			d = at
		}
	}
	// Accept the display width only if
	// the zerofill attribute is defined.
	if z == nil || d == nil {
		return nil
	}
	return d
}
