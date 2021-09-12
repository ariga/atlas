// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"fmt"
	"strings"
	"unicode"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// A diff provides a PostgreSQL implementation for sqlx.DiffDriver.
type diff struct{ *Driver }

// Diff returns a PostgreSQL schema differ.
func (d *Driver) Diff() schema.Differ {
	return &sqlx.Diff{
		DiffDriver: &diff{Driver: d},
	}
}

// SchemaAttrDiff returns a changeset for migrating schema attributes from one state to the other.
func (d *diff) SchemaAttrDiff(_, _ *schema.Schema) []schema.Change {
	// No special schema attribute diffing for PostgreSQL.
	return nil
}

// TableAttrDiff returns a changeset for migrating table attributes from one state to the other.
func (d *diff) TableAttrDiff(from, to *schema.Table) []schema.Change {
	var changes []schema.Change
	// Drop or modify checks.
	for _, c1 := range checks(from.Attrs) {
		switch c2, ok := checkByName(to.Attrs, c1.Name); {
		case !ok:
			changes = append(changes, &schema.DropAttr{
				A: c1,
			})
		case c1.Clause != c2.Clause || c1.NoInherit != c2.NoInherit:
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

// ColumnDefaultChanged reports if the a default value of a column
// type was changed.
func (d *diff) ColumnDefaultChanged(from, to *schema.Column) bool {
	d1, ok1 := from.Default.(*schema.RawExpr)
	d2, ok2 := to.Default.(*schema.RawExpr)
	if ok1 != ok2 {
		return true
	}
	if d1 == nil || d1.X == d2.X || trimCast(d1.X) == trimCast(d2.X) {
		return false
	}
	// Use database comparison in case of mismatch (e.g. `SELECT ARRAY[1] = '{1}'::int[]`).
	return !d.valuesEqual(d1.X, d2.X)
}

// IndexAttrChanged reports if the index attributes were changed.
// The default type is BTREE if no type was specified.
func (*diff) IndexAttrChanged(from, to []schema.Attr) bool {
	t1 := &IndexType{T: "BTREE"}
	if sqlx.Has(from, t1) {
		t1.T = strings.ToUpper(t1.T)
	}
	t2 := &IndexType{T: "BTREE"}
	if sqlx.Has(to, t2) {
		t2.T = strings.ToUpper(t2.T)
	}
	if t1.T != t2.T {
		return true
	}
	var p1, p2 IndexPredicate
	if sqlx.Has(from, &p1) != sqlx.Has(to, &p2) || p1.P != p2.P {
		return true
	}
	return false
}

// IndexPartAttrChanged reports if the index-part attributes were changed.
func (*diff) IndexPartAttrChanged(from, to []schema.Attr) bool {
	// By default, B-tree indexes store rows
	// in ascending order with nulls last.
	p1 := &IndexColumnProperty{Asc: true, NullsLast: true}
	sqlx.Has(from, p1)
	p2 := &IndexColumnProperty{Asc: true, NullsLast: true}
	sqlx.Has(to, p2)
	return p1.Asc != p2.Asc || p1.Desc != p2.Desc || p1.NullsFirst != p2.NullsFirst || p1.NullsLast != p2.NullsLast
}

// ReferenceChanged reports if the foreign key referential action was changed.
func (*diff) ReferenceChanged(from, to schema.ReferenceOption) bool {
	// According to PostgreSQL, the NO ACTION rule is set
	// if no referential action was defined in foreign key.
	if from == "" {
		from = schema.NoAction
	}
	if to == "" {
		to = schema.NoAction
	}
	return from != to
}

func (d *diff) typeChanged(from, to *schema.Column) (bool, error) {
	fromT, toT := from.Type.Type, to.Type.Type
	var (
		changed bool
		x, y    = from.Type.Raw, to.Type.Raw
	)
	switch fromT := fromT.(type) {
	case *schema.BinaryType:
		toT := toT.(*schema.BinaryType)
		changed = fromT.T != toT.T
	case *EnumType:
		toT := toT.(*schema.EnumType)
		changed = fromT.T != toT.T || !sqlx.ValuesEqual(fromT.Values, toT.Values)
	case *schema.IntegerType:
		toT := toT.(*schema.IntegerType)
		// Unsigned integers are not supported.
		changed = fromT.T != toT.T
	case *NetworkType:
		toT := toT.(*NetworkType)
		changed = fromT.T != toT.T
	case *SerialType:
		toT := toT.(*SerialType)
		changed = fromT.T != toT.T || fromT.Precision != toT.Precision
	case *BitType:
		toT := toT.(*BitType)
		changed = fromT.T != toT.T || fromT.Len != toT.Len
	case *CurrencyType:
		toT := toT.(*CurrencyType)
		changed = fromT.T != toT.T
	case *UUIDType:
		toT := toT.(*UUIDType)
		changed = fromT.T != toT.T
	case *XMLType:
		toT := toT.(*XMLType)
		changed = fromT.T != toT.T
	case *ArrayType:
		toT := toT.(*ArrayType)
		changed = fromT.T != toT.T
		x, y = fromT.T, toT.T
	case *UserDefinedType:
		toT := toT.(*UserDefinedType)
		changed = fromT.T != toT.T
	default:
		return false, &sqlx.UnsupportedTypeError{Type: fromT}
	}
	// If we assume that a value was changed, we compare
	// its underlying data type name.
	if !changed {
		return false, nil
	}
	return !d.typesEqual(x, y), nil
}

// Normalize implements the sqlx.Normalizer interface.
func (d *diff) Normalize(tables ...*schema.Table) {
	for _, t := range tables {
		d.normalize(t)
	}
}

func (d *diff) normalize(table *schema.Table) {
	for _, c := range table.Columns {
		switch t := c.Type.Type.(type) {
		case nil:
		case *schema.TimeType:
			// "timestamp" and "timestamptz" are accepted as
			// abbreviations for timestamp with(out) time zone.
			switch t.T {
			case "timestamp with time zone":
				t.T = "timestamptz"
			case "timestamp without time zone":
				t.T = "timestamp"
			}
		case *schema.FloatType:
			// The same numeric precision is used in all platform.
			// See: https://www.postgresql.org/docs/current/datatype-numeric.html
			switch {
			case t.T == "float" && t.Precision < 25:
				// float(1) to float(24) are selected as "real" type.
				t.T = "real"
				fallthrough
			case t.T == "real":
				t.Precision = 24
			case t.T == "float" && t.Precision >= 25:
				// float(25) to float(53) are selected as "double precision" type.
				t.T = "double precision"
				fallthrough
			case t.T == "double precision":
				t.Precision = 53
			}
		case *schema.StringType:
			switch t.T {
			case "character", "char":
				// Character without length specifier
				// is equivalent to character(1).
				t.Size = 1
			}
		case *EnumType:
			c.Type.Type = &schema.EnumType{T: t.T, Values: t.Values}
		case *SerialType:
			// The smallserial, serial and bigserial data types are not true types, but merely a
			// notational convenience for creating integers types with AUTO_INCREMENT property.
			it := &schema.IntegerType{}
			switch t.T {
			case "smallserial":
				it.T = "smallint"
			case "serial":
				it.T = "integer"
			case "bigserial":
				it.T = "bigint"
			default:
				panic(fmt.Sprintf("unexpected serial type: %q", it.T))
			}
			// The definition of "<column> <serial type>" is equivalent to specifying:
			// "<column> <int type> NOT NULL DEFAULT nextval('<table>_<column>_seq')".
			c.Default = &SeqFuncExpr{
				X: fmt.Sprintf("nextval('%s_%s_seq')", table.Name, c.Name),
			}
			c.Type.Type = it
			c.Type.Null = false
		}
	}
}

// valuesEqual reports if the DEFAULT values x and y
// equal according to the database engine.
func (d *diff) valuesEqual(x, y string) (b bool) {
	// The DEFAULT expressions are safe to be inlined in the SELECT
	// statement same as we inline them in the CREATE TABLE statement.
	if err := d.Driver.QueryRow(fmt.Sprintf("SELECT %s = %s", x, y)).Scan(&b); err != nil {
		return false
	}
	return b
}

// valuesEqual reports if the data types x and y
// equal according to the database engine.
func (d *diff) typesEqual(x, y string) (b bool) {
	// The datatype are safe to be inlined in the SELECT statement
	// same as we inline them in the CREATE TABLE statement.
	if err := d.Driver.QueryRow(fmt.Sprintf("SELECT '%s'::regtype = '%s'::regtype", x, y)).Scan(&b); err != nil {
		return false
	}
	return b
}

func checks(attr []schema.Attr) (checks []*Check) {
	for i := range attr {
		if c, ok := attr[i].(*Check); ok {
			checks = append(checks, c)
		}
	}
	return checks
}

func checkByName(attr []schema.Attr, name string) (*Check, bool) {
	for i := range attr {
		if c, ok := attr[i].(*Check); ok && c.Name == name {
			return c, true
		}
	}
	return nil, false
}

func trimCast(s string) string {
	i := strings.LastIndex(s, "::")
	if i == -1 {
		return s
	}
	for _, r := range s[i+2:] {
		if r != ' ' && !unicode.IsLetter(r) {
			return s
		}
	}
	return s[:i]
}
