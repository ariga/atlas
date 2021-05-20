package mysql

import (
	"fmt"
	"reflect"
	"sort"

	"ariga.io/atlas/sql/schema"
)

// A Diff provides diff capabilities for schema elements.
type Diff struct {
	Version string
}

// SchemaDiff implements the schema.Differ interface and returns a list of
// changes that need to be applied in order to move from one state to the other.
func (d *Diff) SchemaDiff(from, to *schema.Schema) ([]schema.Change, error) {
	var changes []schema.Change
	if change := d.collationChange(from.Attrs, to.Attrs); change != noChange {
		changes = append(changes, change)
	}
	// Drop or modify tables.
	for _, t1 := range from.Tables {
		t2, ok := to.Table(t1.Name)
		if !ok {
			changes = append(changes, &schema.DropTable{T: t1})
			continue
		}
		change, err := d.TableDiff(t1, t2)
		if err != nil {
			return nil, err
		}
		if len(change) > 0 {
			changes = append(changes, &schema.ModifyTable{
				T:       t1,
				Changes: change,
			})
		}
	}
	// Add tables.
	for _, t1 := range to.Tables {
		if _, ok := from.Table(t1.Name); !ok {
			changes = append(changes, &schema.AddTable{T: t1})
		}
	}
	return changes, nil
}

// TableDiff implements the schema.TableDiffer interface and returns a list of
// changes that need to be applied in order to move from one state to the other.
func (d *Diff) TableDiff(from, to *schema.Table) ([]schema.Change, error) {
	var (
		changes  []schema.Change
		pk1, pk2 = pk(from), pk(to)
	)
	// PK modification is not support.
	if n, m := len(pk1), len(pk2); n != m {
		return nil, fmt.Errorf("mismatch number of columns for table %q primary key: %d != %d", to.Name, n, m)
	}
	for i := range pk1 {
		if pk1[i].Name != pk2[i].Name {
			return nil, fmt.Errorf("changing primary key of table %q is not supported", to.Name)
		}
	}

	// Collation change.
	if change := d.collationChange(from.Attrs, to.Attrs); change != noChange {
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

	// Drop or modify columns.
	for _, c1 := range from.Columns {
		c2, ok := to.Column(c1.Name)
		if !ok {
			changes = append(changes, &schema.DropColumn{C: c1})
			continue
		}
		change, err := d.columnChange(c1, c2)
		if err != nil {
			return nil, err
		}
		if change != schema.NoChange {
			changes = append(changes, &schema.ModifyColumn{
				From:   c1,
				To:     c2,
				Change: change,
			})
		}
	}
	// Add columns.
	for _, c1 := range to.Columns {
		if _, ok := from.Column(c1.Name); !ok {
			changes = append(changes, &schema.AddColumn{C: c1})
		}
	}

	// Drop or modify indexes.
	for _, idx1 := range from.Indexes {
		idx2, ok := to.Index(idx1.Name)
		if !ok {
			changes = append(changes, &schema.DropIndex{I: idx1})
			continue
		}
		if change := d.indexChange(idx1, idx2); change != schema.NoChange {
			changes = append(changes, &schema.ModifyIndex{
				From:   idx1,
				To:     idx2,
				Change: change,
			})
		}
	}
	// Add indexes.
	for _, idx1 := range to.Indexes {
		if _, ok := from.Index(idx1.Name); !ok {
			changes = append(changes, &schema.AddIndex{I: idx1})
		}
	}

	// Drop or modify foreign-keys.
	for _, fk1 := range from.ForeignKeys {
		fk2, ok := to.ForeignKey(fk1.Symbol)
		if !ok {
			changes = append(changes, &schema.DropForeignKey{F: fk1})
			continue
		}
		if change := d.fkChange(fk1, fk2); change != schema.NoChange {
			changes = append(changes, &schema.ModifyForeignKey{
				From:   fk1,
				To:     fk2,
				Change: change,
			})
		}
	}
	// Add foreign-keys.
	for _, fk1 := range to.ForeignKeys {
		if _, ok := from.ForeignKey(fk1.Symbol); !ok {
			changes = append(changes, &schema.AddForeignKey{F: fk1})
		}
	}
	return changes, nil
}

// columnChange returns the schema changes (if any) for migrating one column to the other.
func (d *Diff) columnChange(from, to *schema.Column) (schema.ChangeKind, error) {
	var change schema.ChangeKind
	if from.Type.Null != to.Type.Null {
		change |= schema.ChangeNull
	}
	change |= commentChange(from.Attrs, to.Attrs)
	c1, c2 := collate(from.Attrs), collate(to.Attrs)
	if (c1 != nil) != (c2 != nil) || (c1 != nil && c1.V != c2.V) {
		change |= schema.ChangeCollation
	}
	s1, s2 := charset(from.Attrs), charset(to.Attrs)
	if (s1 != nil) != (s2 != nil) || (s1 != nil && s1.V != s2.V) {
		change |= schema.ChangeCharset
	}
	t1, t2 := from.Type.Type, to.Type.Type
	if t1 == nil || t2 == nil {
		return 0, fmt.Errorf("missing type infromation for column %q", from.Name)
	}
	if reflect.TypeOf(t1) != reflect.TypeOf(t2) || from.Type.Raw != to.Type.Raw {
		change |= schema.ChangeType
	}
	d1, d2 := from.Default, to.Default
	if (d1 != nil) != (d2 != nil) || (d1 != nil && d1.(*schema.RawExpr).X != d2.(*schema.RawExpr).X) {
		change |= schema.ChangeDefault
	}
	return change, nil
}

// indexChange returns the schema changes (if any) for migrating one index to the other.
func (d *Diff) indexChange(from, to *schema.Index) schema.ChangeKind {
	var change schema.ChangeKind
	change |= partsChange(from.Parts, to.Parts)
	change |= commentChange(from.Attrs, to.Attrs)
	if from.Unique != to.Unique {
		change |= schema.ChangeUnique
	}
	return change
}

// collationChange returns the schema change (if any) for migrating the collation.
func (d *Diff) collationChange(from, to []schema.Attr) schema.Change {
	switch c1, c2 := collate(from), collate(to); {
	case c1 == nil && c2 == nil:
	case c2 == nil:
		return &schema.DropAttr{
			A: c1,
		}
	case c1 == nil:
		return &schema.AddAttr{
			A: c2,
		}
	case c1.V != c2.V:
		return &schema.ModifyAttr{
			From: c1,
			To:   c2,
		}
	}
	return noChange
}

// fkChange returns the schema changes (if any) for migrating one index to the other.
func (d *Diff) fkChange(from, to *schema.ForeignKey) schema.ChangeKind {
	var change schema.ChangeKind
	switch {
	case from.Table.Name != to.Table.Name || from.Table.Schema != to.Table.Schema:
		change |= schema.ChangeRefTable | schema.ChangeRefColumn
	case len(from.RefColumns) != len(to.RefColumns):
		change |= schema.ChangeRefColumn
	default:
		for i := range from.RefColumns {
			if from.RefColumns[i].Name != to.RefColumns[i].Name {
				change |= schema.ChangeRefColumn
			}
		}
	}
	switch {
	case len(from.Columns) != len(to.Columns):
		change |= schema.ChangeColumn
	default:
		for i := range from.Columns {
			if from.Columns[i].Name != to.Columns[i].Name {
				change |= schema.ChangeColumn
			}
		}
	}
	if from.OnUpdate != to.OnUpdate {
		change |= schema.ChangeUpdateAction
	}
	if from.OnDelete != to.OnDelete {
		change |= schema.ChangeDeleteAction
	}
	return change
}

func commentChange(from, to []schema.Attr) schema.ChangeKind {
	c1, c2 := comment(from), comment(to)
	if (c1 != nil) != (c2 != nil) || (c1 != nil && c1.Text != c2.Text) {
		return schema.ChangeComment
	}
	return schema.NoChange
}

func partsChange(from, to []*schema.IndexPart) schema.ChangeKind {
	if len(from) != len(to) {
		return schema.ChangeParts
	}
	for i := range from {
		switch {
		case from[i].SeqNo != to[i].SeqNo || len(from[i].Attrs) != len(to[i].Attrs):
			return schema.ChangeParts
		case from[i].C != nil && to[i].C != nil:
			if from[i].C.Name != to[i].C.Name {
				return schema.ChangeParts
			}
			s1, s2 := subpart(from[i].Attrs), subpart(to[i].Attrs)
			if (s1 != nil) != (s2 != nil) || (s1 != nil && s2 != nil && s1.Len != s2.Len) {
				return schema.ChangeParts
			}
		case from[i].X != nil && to[i].X != nil:
			if from[i].X.(*schema.RawExpr).X != to[i].X.(*schema.RawExpr).X {
				return schema.ChangeParts
			}
		default: // (C1 != nil) != (C2 != nil) || (X1 != nil) != (X2 != nil).
			return schema.ChangeParts
		}
	}
	return schema.NoChange
}

// noChange describes a zero change.
var noChange struct{ schema.Change }

func pk(t *schema.Table) []*schema.Column {
	pk := make([]*schema.Column, len(t.PrimaryKey))
	copy(pk, t.PrimaryKey)
	sort.Slice(pk, func(i, j int) bool { return pk[i].Name < pk[j].Name })
	return pk
}

func collate(attr []schema.Attr) *schema.Collation {
	for i := range attr {
		if c, ok := attr[i].(*schema.Collation); ok {
			return c
		}
	}
	return nil
}

func charset(attr []schema.Attr) *schema.Charset {
	for i := range attr {
		if c, ok := attr[i].(*schema.Charset); ok {
			return c
		}
	}
	return nil
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

func comment(attr []schema.Attr) *schema.Comment {
	for i := range attr {
		if c, ok := attr[i].(*schema.Comment); ok {
			return c
		}
	}
	return nil
}

func subpart(attr []schema.Attr) *SubPart {
	for i := range attr {
		if s, ok := attr[i].(*SubPart); ok {
			return s
		}
	}
	return nil
}
