// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemautil

import (
	"errors"
	"fmt"

	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/schema/schemaspec"
)

// List of convert function types.
type (
	ConvertTableFunc      func(*schemaspec.Table, *schema.Schema) (*schema.Table, error)
	ConvertColumnFunc     func(*schemaspec.Column, *schema.Table) (*schema.Column, error)
	ConvertTypeFunc       func(*schemaspec.Column) (schema.Type, error)
	ConvertPrimaryKeyFunc func(*schemaspec.PrimaryKey, *schema.Table) (*schema.Index, error)
	ConvertIndexFunc      func(*schemaspec.Index, *schema.Table) (*schema.Index, error)
	ColumnSpecFunc        func(*schema.Column) (*schemaspec.Column, error)
	TableSpecFunc         func(*schema.Table) (*schemaspec.Table, error)
	PrimaryKeySpecFunc    func(index *schema.Index) (*schemaspec.PrimaryKey, error)
	IndexSpecFunc         func(index *schema.Index) (*schemaspec.Index, error)
	ForeignKeySpecFunc    func(fk *schema.ForeignKey) (*schemaspec.ForeignKey, error)
)

// ConvertSchema converts a schemaspec.Schema with its relevant *schemaspec.Tables
// into a schema.Schema.
func ConvertSchema(spec *schemaspec.Schema, tables []*schemaspec.Table, convertTable ConvertTableFunc) (*schema.Schema, error) {
	sch := &schema.Schema{
		Name: spec.Name,
	}
	m := make(map[*schema.Table]*schemaspec.Table)
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

// ConvertTable converts a schemaspec.Table to a schema.Table. Table conversion is done without converting
// ForeignKeySpecs into ForeignKeys, as the target tables do not necessarily exist in the schema
// at this point. Instead, the linking is done by the ConvertSchema function.
func ConvertTable(spec *schemaspec.Table, parent *schema.Schema, convertColumn ConvertColumnFunc,
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

// ConvertColumn converts a schemaspec.Column into a schema.Column.
func ConvertColumn(spec *schemaspec.Column, conv ConvertTypeFunc) (*schema.Column, error) {
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

// ConvertIndex converts an schemaspec.Index to a schema.Index.
func ConvertIndex(spec *schemaspec.Index, parent *schema.Table) (*schema.Index, error) {
	parts := make([]*schema.IndexPart, 0, len(spec.Columns))
	for seqno, c := range spec.Columns {
		cn := c.Name
		col, ok := parent.Column(cn)
		if !ok {
			return nil, fmt.Errorf("schemautil: unknown column %q in table %q", cn, parent.Name)
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

// ConvertPrimaryKey converts a schemaspec.PrimaryKey to a schema.Index.
func ConvertPrimaryKey(spec *schemaspec.PrimaryKey, parent *schema.Table) (*schema.Index, error) {
	parts := make([]*schema.IndexPart, 0, len(spec.Columns))
	for seqno, c := range spec.Columns {
		pkc, ok := parent.Column(c.Name)
		if !ok {
			return nil, fmt.Errorf("schemautil: cannot set column %q as primary key for table %q", c.Name, parent.Name)
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
func linkForeignKeys(tbl *schema.Table, sch *schema.Schema, table *schemaspec.Table) error {
	for _, spec := range table.ForeignKeys {
		fk := &schema.ForeignKey{
			Symbol:   spec.Symbol,
			Table:    tbl,
			OnUpdate: schema.ReferenceOption(spec.OnUpdate),
			OnDelete: schema.ReferenceOption(spec.OnDelete),
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

func resolveCol(ref *schemaspec.ColumnRef, sch *schema.Schema) (*schema.Column, error) {
	tbl, ok := sch.Table(ref.Table)
	if !ok {
		return nil, fmt.Errorf("mysql: table %q not found", ref.Table)
	}
	col, ok := tbl.Column(ref.Name)
	if !ok {
		return nil, fmt.Errorf("mysql: column %q not found int table %q", ref.Name, ref.Table)
	}
	return col, nil
}

// SchemaSpec converts a schema.Schema into scehmaspec.Schema and []schemaspec.Table.
func SchemaSpec(s *schema.Schema, fn TableSpecFunc) (*schemaspec.Schema, []*schemaspec.Table, error) {
	spec := &schemaspec.Schema{
		Name: s.Name,
	}
	tables := make([]*schemaspec.Table, 0, len(s.Tables))
	for _, t := range s.Tables {
		table, err := fn(t)
		if err != nil {
			return nil, nil, err
		}
		tables = append(tables, table)
	}
	return spec, tables, nil
}

// TableSpec converts schema.Table to a schemaspec.Table.
func TableSpec(t *schema.Table, colFn ColumnSpecFunc, pkFn PrimaryKeySpecFunc, idxFn IndexSpecFunc, fkFn ForeignKeySpecFunc) (*schemaspec.Table, error) {
	spec := &schemaspec.Table{
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

// PrimaryKeySpec converts schema.Index to a schemaspec.PrimaryKey.
func PrimaryKeySpec(s *schema.Index) (*schemaspec.PrimaryKey, error) {
	c := make([]*schemaspec.ColumnRef, 0, len(s.Parts))
	for _, v := range s.Parts {
		c = append(c, &schemaspec.ColumnRef{
			Name:  v.C.Name,
			Table: s.Table.Name,
		})
	}
	return &schemaspec.PrimaryKey{
		Columns: c,
	}, nil
}

// IndexSpec converts schema.Index to schemaspec.Index
func IndexSpec(idx *schema.Index) (*schemaspec.Index, error) {
	c := make([]*schemaspec.ColumnRef, 0, len(idx.Parts))
	for _, p := range idx.Parts {
		if p.C == nil {
			return nil, errors.New("index expression is not supported")
		}
		c = append(c, &schemaspec.ColumnRef{
			Name:  p.C.Name,
			Table: idx.Table.Name,
		})
	}
	return &schemaspec.Index{
		Name:    idx.Name,
		Unique:  idx.Unique,
		Columns: c,
	}, nil
}

// ForeignKeySpec converts schema.ForeignKey to schemaspec.ForeignKey
func ForeignKeySpec(s *schema.ForeignKey) (*schemaspec.ForeignKey, error) {
	c := make([]*schemaspec.ColumnRef, 0, len(s.Columns))
	for _, v := range s.Columns {
		c = append(c, &schemaspec.ColumnRef{
			Name:  v.Name,
			Table: s.Table.Name,
		})
	}
	r := make([]*schemaspec.ColumnRef, 0, len(s.RefColumns))
	for _, v := range s.RefColumns {
		r = append(r, &schemaspec.ColumnRef{
			Name:  v.Name,
			Table: s.Symbol,
		})
	}
	return &schemaspec.ForeignKey{
		Symbol:     s.Symbol,
		Columns:    c,
		RefColumns: r,
		OnDelete:   string(s.OnDelete),
		OnUpdate:   string(s.OnUpdate),
	}, nil
}
