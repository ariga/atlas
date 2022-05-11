// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// A diff provides a PostgreSQL implementation for sqlx.DiffDriver.
type diff struct{ conn }

// SchemaAttrDiff returns a changeset for migrating schema attributes from one state to the other.
func (d *diff) SchemaAttrDiff(_, _ *schema.Schema) []schema.Change {
	// No special schema attribute diffing for PostgreSQL.
	return nil
}

// TableAttrDiff returns a changeset for migrating table attributes from one state to the other.
func (d *diff) TableAttrDiff(from, to *schema.Table) ([]schema.Change, error) {
	var changes []schema.Change
	if change := sqlx.CommentDiff(from.Attrs, to.Attrs); change != nil {
		changes = append(changes, change)
	}
	if err := d.partitionChanged(from, to); err != nil {
		return nil, err
	}
	return append(changes, sqlx.CheckDiff(from, to, func(c1, c2 *schema.Check) bool {
		return sqlx.Has(c1.Attrs, &NoInherit{}) == sqlx.Has(c2.Attrs, &NoInherit{})
	})...), nil
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
	if changed, err = d.defaultChanged(from, to); err != nil {
		return schema.NoChange, err
	}
	if changed {
		change |= schema.ChangeDefault
	}
	if identityChanged(from.Attrs, to.Attrs) {
		change |= schema.ChangeAttr
	}
	if changed, err = d.generatedChanged(from, to); err != nil {
		return schema.NoChange, err
	}
	if changed {
		change |= schema.ChangeGenerated
	}
	return change, nil
}

// defaultChanged reports if the default value of a column was changed.
func (d *diff) defaultChanged(from, to *schema.Column) (bool, error) {
	d1, ok1 := sqlx.DefaultValue(from)
	d2, ok2 := sqlx.DefaultValue(to)
	if ok1 != ok2 {
		return true, nil
	}
	if trimCast(d1) == trimCast(d2) {
		return false, nil
	}
	// Use database comparison in case of mismatch (e.g. `SELECT ARRAY[1] = '{1}'::int[]`).
	equals, err := d.valuesEqual(d1, d2)
	if err != nil {
		return false, err
	}
	return !equals, nil
}

// generatedChanged reports if the generated expression of a column was changed.
func (*diff) generatedChanged(from, to *schema.Column) (bool, error) {
	var fromX, toX schema.GeneratedExpr
	switch fromHas, toHas := sqlx.Has(from.Attrs, &fromX), sqlx.Has(to.Attrs, &toX); {
	case fromHas && toHas && sqlx.MayWrap(fromX.Expr) != sqlx.MayWrap(toX.Expr):
		return false, fmt.Errorf("changing the generation expression for a column %q is not supported", from.Name)
	case !fromHas && toHas:
		return false, fmt.Errorf("changing column %q to generated column is not supported (drop and add is required)", from.Name)
	default:
		// Only DROP EXPRESSION is supported.
		return fromHas && !toHas, nil
	}
}

// partitionChanged checks and returns an error if the partition key of a table was changed.
func (*diff) partitionChanged(from, to *schema.Table) error {
	var fromP, toP Partition
	switch fromHas, toHas := sqlx.Has(from.Attrs, &fromP), sqlx.Has(to.Attrs, &toP); {
	case fromHas && !toHas:
		return fmt.Errorf("partition key cannot be dropped from %q (drop and add is required)", from.Name)
	case !fromHas && toHas:
		return fmt.Errorf("partition key cannot be added to %q (drop and add is required)", to.Name)
	case fromHas && toHas:
		s1, err := formatPartition(fromP)
		if err != nil {
			return err
		}
		s2, err := formatPartition(toP)
		if err != nil {
			return err
		}
		if s1 != s2 {
			return fmt.Errorf("partition key of table %q cannot be changed from %s to %s (drop and add is required)", to.Name, s1, s2)
		}
	}
	return nil
}

// IsGeneratedIndexName reports if the index name was generated by the database.
func (d *diff) IsGeneratedIndexName(t *schema.Table, idx *schema.Index) bool {
	names := make([]string, len(idx.Parts))
	for i, p := range idx.Parts {
		if p.C == nil {
			return false
		}
		names[i] = p.C.Name
	}
	// Auto-generate index names will have the following format: <table>_<c1>_..._key.
	// In case of conflict, PostgreSQL adds additional index at the end (e.g. "key1").
	p := fmt.Sprintf("%s_%s_key", t.Name, strings.Join(names, "_"))
	if idx.Name == p {
		return true
	}
	i, err := strconv.ParseInt(strings.TrimPrefix(idx.Name, p), 10, 64)
	return err == nil && i > 0
}

// IndexAttrChanged reports if the index attributes were changed.
// The default type is BTREE if no type was specified.
func (*diff) IndexAttrChanged(from, to []schema.Attr) bool {
	t1 := &IndexType{T: IndexTypeBTree}
	if sqlx.Has(from, t1) {
		t1.T = strings.ToUpper(t1.T)
	}
	t2 := &IndexType{T: IndexTypeBTree}
	if sqlx.Has(to, t2) {
		t2.T = strings.ToUpper(t2.T)
	}
	if t1.T != t2.T {
		return true
	}
	var p1, p2 IndexPredicate
	if sqlx.Has(from, &p1) != sqlx.Has(to, &p2) || (p1.P != p2.P && p1.P != sqlx.MayWrap(p2.P)) {
		return true
	}
	sp := func(attrs []schema.Attr) IndexStorageParams {
		var s IndexStorageParams
		sqlx.Has(attrs, &s)
		if s.PagesPerRange == 0 {
			// Default size is 128.
			s.PagesPerRange = 128
		}
		return s
	}
	s1, s2 := sp(from), sp(to)
	return s1.AutoSummarize != s2.AutoSummarize || s1.PagesPerRange != s2.PagesPerRange
}

// IndexPartAttrChanged reports if the index-part attributes were changed.
func (*diff) IndexPartAttrChanged(from, to *schema.IndexPart) bool {
	p1 := &IndexColumnProperty{NullsFirst: from.Desc, NullsLast: !from.Desc}
	sqlx.Has(from.Attrs, p1)
	p2 := &IndexColumnProperty{NullsFirst: to.Desc, NullsLast: !to.Desc}
	sqlx.Has(to.Attrs, p2)
	return p1.NullsFirst != p2.NullsFirst || p1.NullsLast != p2.NullsLast
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
	if fromT == nil || toT == nil {
		return false, fmt.Errorf("postgres: missing type information for column %q", from.Name)
	}
	// Skip checking SERIAL types as they are not real types in the database, but more
	// like a convenience way for creating integers types with AUTO_INCREMENT property.
	if s, ok := to.Type.Type.(*SerialType); ok {
		i, ok := from.Type.Type.(*schema.IntegerType)
		if !ok {
			return true, nil
		}
		var it string
		switch s.T {
		case TypeSmallSerial:
			it = TypeSmallInt
		case TypeSerial:
			it = TypeInteger
		case TypeBigSerial:
			it = TypeBigInt
		}
		return i.T != it, nil
	}
	if reflect.TypeOf(fromT) != reflect.TypeOf(toT) {
		return true, nil
	}
	var changed bool
	switch fromT := fromT.(type) {
	case *schema.BinaryType, *schema.BoolType, *schema.DecimalType, *schema.FloatType,
		*schema.IntegerType, *schema.JSONType, *schema.SpatialType, *schema.StringType,
		*schema.TimeType, *BitType, *NetworkType, *UserDefinedType:
		t1, err := FormatType(toT)
		if err != nil {
			return false, err
		}
		t2, err := FormatType(fromT)
		if err != nil {
			return false, err
		}
		changed = t1 != t2
	case *enumType:
		toT := toT.(*schema.EnumType)
		changed = fromT.T != toT.T || !sqlx.ValuesEqual(fromT.Values, toT.Values)
	case *schema.EnumType:
		toT := toT.(*schema.EnumType)
		changed = fromT.T != toT.T || !sqlx.ValuesEqual(fromT.Values, toT.Values)
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
		// Array types can be defined differently, but they may represent the same type.
		// Therefore, in case of mismatch, we verify it using the database engine.
		if changed {
			equals, err := d.typesEqual(fromT.T, toT.T)
			return !equals, err
		}
	default:
		return false, &sqlx.UnsupportedTypeError{Type: fromT}
	}
	return changed, nil
}

// Normalize implements the sqlx.Normalizer interface.
func (d *diff) Normalize(from, to *schema.Table) {
	d.normalize(from)
	d.normalize(to)
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
		case *enumType:
			c.Type.Type = &schema.EnumType{T: t.T, Values: t.Values}
		case *SerialType:
			// The definition of "<column> <serial type>" is equivalent to specifying:
			// "<column> <int type> NOT NULL DEFAULT nextval('<table>_<column>_seq')".
			c.Default = &schema.RawExpr{
				X: fmt.Sprintf("nextval('%s_%s_seq'::regclass)", table.Name, c.Name),
			}
		}
	}
}

// valuesEqual reports if the DEFAULT values x and y
// equal according to the database engine.
func (d *diff) valuesEqual(x, y string) (bool, error) {
	var b bool
	// The DEFAULT expressions are safe to be inlined in the SELECT
	// statement same as we inline them in the CREATE TABLE statement.
	rows, err := d.QueryContext(context.Background(), fmt.Sprintf("SELECT %s = %s", x, y))
	if err != nil {
		return false, err
	}
	if err := sqlx.ScanOne(rows, &b); err != nil {
		return false, err
	}
	return b, nil
}

// typesEqual reports if the data types x and y
// equal according to the database engine.
func (d *diff) typesEqual(x, y string) (bool, error) {
	var b bool
	// The datatype are safe to be inlined in the SELECT statement
	// same as we inline them in the CREATE TABLE statement.
	rows, err := d.QueryContext(context.Background(), fmt.Sprintf("SELECT '%s'::regtype = '%s'::regtype", x, y))
	if err != nil {
		return false, err
	}
	if err := sqlx.ScanOne(rows, &b); err != nil {
		return false, err
	}
	return b, nil
}

// Default IDENTITY attributes.
const (
	defaultIdentityGen  = "BY DEFAULT"
	defaultSeqStart     = 1
	defaultSeqIncrement = 1
)

// identityChanged reports if one of the identity attributes was changed.
func identityChanged(from, to []schema.Attr) bool {
	i1, ok1 := identity(from)
	i2, ok2 := identity(to)
	if !ok1 && !ok2 || ok1 != ok2 {
		return ok1 != ok2
	}
	return i1.Generation != i2.Generation || i1.Sequence.Start != i2.Sequence.Start || i1.Sequence.Increment != i2.Sequence.Increment
}

func identity(attrs []schema.Attr) (*Identity, bool) {
	i := &Identity{}
	if !sqlx.Has(attrs, i) {
		return nil, false
	}
	if i.Generation == "" {
		i.Generation = defaultIdentityGen
	}
	if i.Sequence == nil {
		i.Sequence = &Sequence{Start: defaultSeqStart, Increment: defaultSeqIncrement}
		return i, true
	}
	if i.Sequence.Start == 0 {
		i.Sequence.Start = defaultSeqStart
	}
	if i.Sequence.Increment == 0 {
		i.Sequence.Increment = defaultSeqIncrement
	}
	return i, true
}

// formatPartition returns the string representation of the
// partition key according to the PostgreSQL format/grammar.
func formatPartition(p Partition) (string, error) {
	b := Build("PARTITION BY")
	switch t := strings.ToUpper(p.T); t {
	case PartitionTypeRange, PartitionTypeList, PartitionTypeHash:
		b.P(t)
	default:
		return "", fmt.Errorf("unknown partition type: %q", t)
	}
	if len(p.Parts) == 0 {
		return "", errors.New("missing parts for partition key")
	}
	b.Wrap(func(b *sqlx.Builder) {
		b.MapComma(p.Parts, func(i int, b *sqlx.Builder) {
			switch k := p.Parts[i]; {
			case k.C != nil:
				b.Ident(k.C.Name)
			case k.X != nil:
				b.P(sqlx.MayWrap(k.X.(*schema.RawExpr).X))
			}
		})
	})
	return b.String(), nil
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
