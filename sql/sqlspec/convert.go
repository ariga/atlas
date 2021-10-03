// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlspec

import (
	"errors"
	"fmt"
	"strings"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/schema"
)

// List of convert function types.
type (
	ConvertTableFunc      func(*Table, *schema.Schema) (*schema.Table, error)
	ConvertColumnFunc     func(*Column, *schema.Table) (*schema.Column, error)
	ConvertTypeFunc       func(*Column) (schema.Type, error)
	ConvertPrimaryKeyFunc func(*PrimaryKey, *schema.Table) (*schema.Index, error)
	ConvertIndexFunc      func(*Index, *schema.Table) (*schema.Index, error)
	ColumnSpecFunc        func(*schema.Column) (*Column, error)
	TableSpecFunc         func(*schema.Table) (*Table, error)
	PrimaryKeySpecFunc    func(index *schema.Index) (*PrimaryKey, error)
	IndexSpecFunc         func(index *schema.Index) (*Index, error)
	ForeignKeySpecFunc    func(fk *schema.ForeignKey) (*ForeignKey, error)
)

// ConvertSchema converts a Schema with its relevant *Tables
// into a schema.Schema.
func ConvertSchema(spec *Schema, tables []*Table, convertTable ConvertTableFunc) (*schema.Schema, error) {
	sch := &schema.Schema{
		Name: spec.Name,
	}
	m := make(map[*schema.Table]*Table)
	for _, ts := range tables {
		table, err := convertTable(ts, sch)
		if err != nil {
			return nil, err
		}
		sch.Tables = append(sch.Tables, table)
		m[table] = ts
	}
	for _, tbl := range sch.Tables {
		if err := linkForeignKeys(tbl, sch, m[tbl]); err != nil {
			return nil, err
		}
	}
	return sch, nil
}

// ConvertTable converts a Table to a schema.Table. Table conversion is done without converting
// ForeignKeySpecs into ForeignKeys, as the target tables do not necessarily exist in the schema
// at this point. Instead, the linking is done by the ConvertSchema function.
func ConvertTable(spec *Table, parent *schema.Schema, convertColumn ConvertColumnFunc,
	convertPk ConvertPrimaryKeyFunc, convertIndex ConvertIndexFunc) (*schema.Table, error) {
	tbl := &schema.Table{
		Name:   spec.Name,
		Schema: parent,
	}
	for _, csp := range spec.Columns {
		col, err := convertColumn(csp, tbl)
		if err != nil {
			return nil, err
		}
		tbl.Columns = append(tbl.Columns, col)
	}
	if spec.PrimaryKey != nil {
		pk, err := convertPk(spec.PrimaryKey, tbl)
		if err != nil {
			return nil, err
		}
		tbl.PrimaryKey = pk
	}
	for _, idx := range spec.Indexes {
		i, err := convertIndex(idx, tbl)
		if err != nil {
			return nil, err
		}
		tbl.Indexes = append(tbl.Indexes, i)
	}
	return tbl, nil
}

// ConvertColumn converts a Column into a schema.Column.
func ConvertColumn(spec *Column, conv ConvertTypeFunc) (*schema.Column, error) {
	out := &schema.Column{
		Name: spec.Name,
		Type: &schema.ColumnType{
			Null: spec.Null,
		},
	}
	if spec.Default != nil {
		out.Default = &schema.Literal{V: spec.Default.V}
	}
	ct, err := conv(spec)
	if err != nil {
		return nil, err
	}
	out.Type.Type = ct
	return out, err
}

// ConvertIndex converts an Index to a schema.Index.
func ConvertIndex(spec *Index, parent *schema.Table) (*schema.Index, error) {
	parts := make([]*schema.IndexPart, 0, len(spec.Columns))
	for seqno, c := range spec.Columns {
		cn := c.V
		col, ok := parent.Column(cn)
		if !ok {
			return nil, fmt.Errorf("sqlspec: unknown column %q in table %q", cn, parent.Name)
		}
		parts = append(parts, &schema.IndexPart{
			SeqNo: seqno,
			C:     col,
		})
	}
	return &schema.Index{
		Name:   spec.Name,
		Unique: spec.Unique,
		Table:  parent,
		Parts:  parts,
	}, nil
}

// ConvertPrimaryKey converts a PrimaryKey to a schema.Index.
func ConvertPrimaryKey(spec *PrimaryKey, parent *schema.Table) (*schema.Index, error) {
	parts := make([]*schema.IndexPart, 0, len(spec.Columns))
	for seqno, c := range spec.Columns {
		n, err := columnName(c)
		if err != nil {
			return nil, fmt.Errorf("sqlspec: cannot get column name %q as primary key for table %q", c.V, parent.Name)
		}
		pkc, ok := parent.Column(n)
		if !ok {
			return nil, fmt.Errorf("sqlspec: cannot set column %q as primary key for table %q", n, parent.Name)
		}
		parts = append(parts, &schema.IndexPart{
			SeqNo: seqno,
			C:     pkc,
		})
	}
	return &schema.Index{
		Table: parent,
		Parts: parts,
	}, nil
}

// linkForeignKeys creates the foreign keys defined in the Table's spec by creating references
// to column in the provided Schema. It is assumed that the schema contains all of the tables
// referenced by the FK definitions in the spec.
func linkForeignKeys(tbl *schema.Table, sch *schema.Schema, table *Table) error {
	for _, spec := range table.ForeignKeys {
		fk := &schema.ForeignKey{
			Symbol:   spec.Symbol,
			Table:    tbl,
			OnUpdate: spec.OnUpdate,
			OnDelete: spec.OnDelete,
		}
		for _, ref := range spec.Columns {
			col, err := resolveCol(ref, sch)
			if err != nil {
				return err
			}
			fk.Columns = append(fk.Columns, col)
		}
		for _, ref := range spec.RefColumns {
			col, err := resolveCol(ref, sch)
			if err != nil {
				return err
			}
			fk.RefColumns = append(fk.RefColumns, col)
		}
		tbl.ForeignKeys = append(tbl.ForeignKeys, fk)
	}
	return nil
}

func resolveCol(ref *schemaspec.Ref, sch *schema.Schema) (*schema.Column, error) {
	t, err := tableName(ref)
	if err != nil {
		return nil, fmt.Errorf("sqlspec: table %q not found", ref.V)
	}
	tbl, ok := sch.Table(t)
	if !ok {
		return nil, fmt.Errorf("sqlspec: table %q not found", t)
	}
	c, err := columnName(ref)
	if err != nil {
		return nil, fmt.Errorf("sqlspec: column %q not found", ref.V)
	}
	col, ok := tbl.Column(c)
	if !ok {
		return nil, fmt.Errorf("sqlspec: column %q not found in table %q", c, t)
	}
	return col, nil
}

// FromSchema converts a schema.Schema into Schema and []Table.
func FromSchema(s *schema.Schema, fn TableSpecFunc) (*Schema, []*Table, error) {
	spec := &Schema{
		Name: s.Name,
	}
	tables := make([]*Table, 0, len(s.Tables))
	for _, t := range s.Tables {
		table, err := fn(t)
		if err != nil {
			return nil, nil, err
		}
		tables = append(tables, table)
	}
	return spec, tables, nil
}

// FromTable converts a schema.Table to a Table.
func FromTable(t *schema.Table, colFn ColumnSpecFunc, pkFn PrimaryKeySpecFunc, idxFn IndexSpecFunc, fkFn ForeignKeySpecFunc) (*Table, error) {
	spec := &Table{
		Name: t.Name,
	}
	for _, c := range t.Columns {
		col, err := colFn(c)
		if err != nil {
			return nil, err
		}
		spec.Columns = append(spec.Columns, col)
	}
	if t.PrimaryKey != nil {
		pk, err := pkFn(t.PrimaryKey)
		if err != nil {
			return nil, err
		}
		spec.PrimaryKey = pk
	}
	for _, idx := range t.Indexes {
		i, err := idxFn(idx)
		if err != nil {
			return nil, err
		}
		spec.Indexes = append(spec.Indexes, i)
	}
	for _, fk := range t.ForeignKeys {
		f, err := fkFn(fk)
		if err != nil {
			return nil, err
		}
		spec.ForeignKeys = append(spec.ForeignKeys, f)
	}
	return spec, nil
}

// FromPrimaryKey converts schema.Index to a PrimaryKey.
func FromPrimaryKey(s *schema.Index) (*PrimaryKey, error) {
	c := make([]*schemaspec.Ref, 0, len(s.Parts))
	for _, v := range s.Parts {
		c = append(c, toReference(v.C.Name, s.Table.Name))
	}
	return &PrimaryKey{
		Columns: c,
	}, nil
}

// FromIndex converts schema.Index to Index
func FromIndex(idx *schema.Index) (*Index, error) {
	c := make([]*schemaspec.Ref, 0, len(idx.Parts))
	for _, p := range idx.Parts {
		if p.C == nil {
			return nil, errors.New("index expression is not supported")
		}
		c = append(c, toReference(p.C.Name, idx.Table.Name))
	}
	return &Index{
		Name:    idx.Name,
		Unique:  idx.Unique,
		Columns: c,
	}, nil
}

// FromForeignKey converts schema.ForeignKey to ForeignKey
func FromForeignKey(s *schema.ForeignKey) (*ForeignKey, error) {
	c := make([]*schemaspec.Ref, 0, len(s.Columns))
	for _, v := range s.Columns {
		c = append(c, toReference(v.Name, s.Table.Name))
	}
	r := make([]*schemaspec.Ref, 0, len(s.RefColumns))
	for _, v := range s.RefColumns {
		r = append(r, toReference(v.Name, s.Symbol))
	}
	return &ForeignKey{
		Symbol:     s.Symbol,
		Columns:    c,
		RefColumns: r,
		OnDelete:   s.OnDelete,
		OnUpdate:   s.OnUpdate,
	}, nil
}

func columnName(ref *schemaspec.Ref) (string, error) {
	s := strings.Split(ref.V, "$column.")
	if len(s) != 2 {
		return "", fmt.Errorf("sqlspec: failed to extract column name from %q", ref)

	}
	return s[1], nil
}

func tableName(ref *schemaspec.Ref) (string, error) {
	s := strings.Split(ref.V, "$column.")
	if len(s) != 2 {
		return "", fmt.Errorf("sqlspec: failed to split by column name from %q", ref)

	}
	s = strings.Split(s[0], ".")
	if len(s) != 3 {
		return "", fmt.Errorf("sqlspec: failed to extract table name from %q", s)
	}
	return s[1], nil
}

func toReference(cName string, tName string) *schemaspec.Ref {
	v := "$table." + tName + ".$column." + cName
	return &schemaspec.Ref{
		V: v,
	}
}
