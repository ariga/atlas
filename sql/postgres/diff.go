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

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// DefaultDiff provides basic diffing capabilities for PostgreSQL dialects.
// Note, it is recommended to call Open, create a new Driver and use its Differ
// when a database connection is available.
var DefaultDiff schema.Differ = &sqlx.Diff{DiffDriver: &diff{&conn{ExecQuerier: sqlx.NoRows}}}

// A diff provides a PostgreSQL implementation for sqlx.DiffDriver.
type diff struct{ *conn }

// SchemaAttrDiff returns a changeset for migrating schema attributes from one state to the other.
func (*diff) SchemaAttrDiff(from, to *schema.Schema) []schema.Change {
	var changes []schema.Change
	if change := sqlx.CommentDiff(skipDefaultComment(from), skipDefaultComment(to)); change != nil {
		changes = append(changes, change)
	}
	return changes
}

func skipDefaultComment(s *schema.Schema) []schema.Attr {
	attrs := s.Attrs
	if c := (schema.Comment{}); sqlx.Has(attrs, &c) && c.Text == "standard public schema" && (s.Name == "" || s.Name == "public") {
		attrs = schema.RemoveAttr[*schema.Comment](attrs)
	}
	return attrs
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
func (d *diff) ColumnChange(_ *schema.Table, from, to *schema.Column) (schema.ChangeKind, error) {
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
	if !ok1 && !ok2 || trimCast(d1) == trimCast(d2) || quote(d1) == quote(d2) {
		return false, nil
	}
	var (
		_, fromX = from.Default.(*schema.RawExpr)
		_, toX   = to.Default.(*schema.RawExpr)
	)
	// In case one of the DEFAULT values is an expression, and a database connection is
	// available (not DefaultDiff), we use the database to compare in case of mismatch.
	//
	//	SELECT ARRAY[1] = '{1}'::int[]
	//	SELECT lower('X'::text) = lower('X')
	//
	if (fromX || toX) && d.conn.ExecQuerier != nil {
		equals, err := d.defaultEqual(from.Default, to.Default)
		return !equals, err
	}
	return true, nil
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
	if indexNullsDistinct(to) != indexNullsDistinct(from) {
		return true
	}
	var p1, p2 IndexPredicate
	if sqlx.Has(from, &p1) != sqlx.Has(to, &p2) || (p1.P != p2.P && p1.P != sqlx.MayWrap(p2.P)) {
		return true
	}
	if indexIncludeChanged(from, to) {
		return true
	}
	s1, ok1 := indexStorageParams(from)
	s2, ok2 := indexStorageParams(to)
	return ok1 != ok2 || ok1 && *s1 != *s2
}

// IndexPartAttrChanged reports if the index-part attributes were changed.
func (*diff) IndexPartAttrChanged(fromI, toI *schema.Index, i int) bool {
	from, to := fromI.Parts[i], toI.Parts[i]
	p1 := &IndexColumnProperty{NullsFirst: from.Desc, NullsLast: !from.Desc}
	sqlx.Has(from.Attrs, p1)
	p2 := &IndexColumnProperty{NullsFirst: to.Desc, NullsLast: !to.Desc}
	sqlx.Has(to.Attrs, p2)
	if p1.NullsFirst != p2.NullsFirst || p1.NullsLast != p2.NullsLast {
		return true
	}
	var fromOp, toOp IndexOpClass
	switch fromHas, toHas := sqlx.Has(from.Attrs, &fromOp), sqlx.Has(to.Attrs, &toOp); {
	case fromHas && toHas:
		return !fromOp.Equal(&toOp)
	case toHas:
		// Report a change if a non-default operator class was added.
		d, err := toOp.DefaultFor(toI, toI.Parts[i])
		return !d && err == nil
	case fromHas:
		// Report a change if a non-default operator class was removed.
		d, err := fromOp.DefaultFor(fromI, fromI.Parts[i])
		return !d && err == nil
	default:
		return false
	}
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

// DiffOptions defines PostgreSQL specific schema diffing process.
type DiffOptions struct {
	ConcurrentIndex struct {
		Drop bool `spec:"drop"`
		// Allow config "CREATE" both with "add" and "create"
		// as the documentation used both terms (accidentally).
		Add    bool `spec:"add"`
		Create bool `spec:"create"`
	} `spec:"concurrent_index"`
}

// AnnotateChanges implements the sqlx.ChangeAnnotator interface.
func (*diff) AnnotateChanges(changes []schema.Change, opts *schema.DiffOptions) error {
	var extra DiffOptions
	switch ex := opts.Extra.(type) {
	case nil:
		return nil
	case schemahcl.DefaultExtension:
		if err := ex.Extra.As(&extra); err != nil {
			return err
		}
	default:
		return fmt.Errorf("postgres: unexpected DiffOptions.Extra type %T", opts.Extra)
	}
	for _, c := range changes {
		m, ok := c.(*schema.ModifyTable)
		if !ok {
			continue
		}
		for i := range m.Changes {
			switch c := m.Changes[i].(type) {
			case *schema.AddIndex:
				if extra.ConcurrentIndex.Add || extra.ConcurrentIndex.Create {
					c.Extra = append(c.Extra, &Concurrently{})
				}
			case *schema.DropIndex:
				if extra.ConcurrentIndex.Drop {
					c.Extra = append(c.Extra, &Concurrently{})
				}
			}
		}
	}
	return nil
}

func (d *diff) typeChanged(from, to *schema.Column) (bool, error) {
	return typeChanged(from, to, d.conn.schema)
}

func typeChanged(from, to *schema.Column, ns string) (bool, error) {
	fromT, toT := from.Type.Type, to.Type.Type
	if fromT == nil || toT == nil {
		return false, fmt.Errorf("postgres: missing type information for column %q", from.Name)
	}
	if reflect.TypeOf(fromT) != reflect.TypeOf(toT) {
		return true, nil
	}
	var changed bool
	switch fromT := fromT.(type) {
	case *schema.BinaryType, *BitType, *schema.BoolType, *schema.DecimalType, *schema.FloatType, *IntervalType,
		*schema.IntegerType, *schema.JSONType, *OIDType, *RangeType, *SerialType, *schema.SpatialType,
		*schema.StringType, *schema.TimeType, *TextSearchType, *NetworkType, *schema.UUIDType:
		t1, err := FormatType(toT)
		if err != nil {
			return false, err
		}
		t2, err := FormatType(fromT)
		if err != nil {
			return false, err
		}
		changed = t1 != t2
	case *UserDefinedType:
		toT := toT.(*UserDefinedType)
		changed = toT.T != fromT.T &&
			// In case the type is defined with schema qualifier, but returned without
			// (inspecting a schema scope), or vice versa, remove before comparing.
			ns != "" && trimSchema(toT.T, ns) != trimSchema(toT.T, ns)
	case *DomainType:
		toT := toT.(*DomainType)
		changed = toT.T != fromT.T ||
			(toT.Schema != nil && fromT.Schema != nil && toT.Schema.Name != fromT.Schema.Name)
	case *schema.EnumType:
		toT := toT.(*schema.EnumType)
		// Column type was changed if the underlying enum type was changed.
		changed = fromT.T != toT.T || (toT.Schema != nil && fromT.Schema != nil && fromT.Schema.Name != toT.Schema.Name)
	case *CurrencyType:
		toT := toT.(*CurrencyType)
		changed = fromT.T != toT.T
	case *XMLType:
		toT := toT.(*XMLType)
		changed = fromT.T != toT.T
	case *ArrayType:
		toT := toT.(*ArrayType)
		// In case the desired schema is not normalized, the string type can look different even
		// if the two strings represent the same array type (varchar(1), character varying (1)).
		// Therefore, we try by comparing the underlying types if they were defined.
		if fromT.Type != nil && toT.Type != nil {
			t1, err := FormatType(fromT.Type)
			if err != nil {
				return false, err
			}
			t2, err := FormatType(toT.Type)
			if err != nil {
				return false, err
			}
			// Same underlying type.
			changed = t1 != t2
		}
	default:
		return false, &sqlx.UnsupportedTypeError{Type: fromT}
	}
	return changed, nil
}

// trimSchema returns the given type without the schema qualifier.
func trimSchema(t string, ns string) string {
	if strings.HasPrefix(t, `"`) {
		return strings.TrimPrefix(t, fmt.Sprintf("%q.", ns))
	}
	return strings.TrimPrefix(t, fmt.Sprintf("%s.", ns))
}

// defaultEqual reports if the DEFAULT values x and y
// equal according to the database engine.
func (d *diff) defaultEqual(from, to schema.Expr) (bool, error) {
	var (
		b      bool
		d1, d2 string
	)
	switch from := from.(type) {
	case *schema.Literal:
		d1 = quote(from.V)
	case *schema.RawExpr:
		d1 = from.X
	}
	switch to := to.(type) {
	case *schema.Literal:
		d2 = quote(to.V)
	case *schema.RawExpr:
		d2 = to.X
	}
	// The DEFAULT expressions are safe to be inlined in the SELECT
	// statement same as we inline them in the CREATE TABLE statement.
	rows, err := d.QueryContext(context.Background(), fmt.Sprintf("SELECT %s = %s", d1, d2))
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
	b := &sqlx.Builder{QuoteOpening: '"', QuoteClosing: '"'}
	b.P("PARTITION BY")
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

// indexStorageParams returns the index storage parameters from the attributes
// in case it is there, and it is not the default.
func indexStorageParams(attrs []schema.Attr) (*IndexStorageParams, bool) {
	s := &IndexStorageParams{}
	if !sqlx.Has(attrs, s) {
		return nil, false
	}
	if !s.AutoSummarize && (s.PagesPerRange == 0 || s.PagesPerRange == defaultPagePerRange) {
		return nil, false
	}
	return s, true
}

// indexIncludeChanged reports if the INCLUDE attribute clause was changed.
func indexIncludeChanged(from, to []schema.Attr) bool {
	var fromI, toI IndexInclude
	if sqlx.Has(from, &fromI) != sqlx.Has(to, &toI) || len(fromI.Columns) != len(toI.Columns) {
		return true
	}
	for i := range fromI.Columns {
		if fromI.Columns[i].Name != toI.Columns[i].Name {
			return true
		}
	}
	return false
}

// indexNullsDistinct returns the NULLS [NOT] DISTINCT value from the index attributes.
func indexNullsDistinct(attrs []schema.Attr) bool {
	if i := (IndexNullsDistinct{}); sqlx.Has(attrs, &i) {
		return i.V
	}
	// The default PostgreSQL behavior. The inverse of
	// "indnullsnotdistinct" in pg_index which is false.
	return true
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
