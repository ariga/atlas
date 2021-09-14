// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemautil

import (
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
	TypeSpecFunc          func(schema.Type) (*schemaspec.Column, error)
	ColumnSpecFunc        func(*schema.Column) (*schemaspec.Column, error)
	TableSpecFunc         func(*schema.Table) (*schemaspec.Table, error)
)

// ConvertSchema converts a schemaspec.Schema with its relevant *schemaspec.Tables
// into a schema.Schema.
func ConvertSchema(spec *schemaspec.Schema, tables []*schemaspec.Table, convertTable ConvertTableFunc) (*schema.Schema, error) {
	sch := &schema.Schema{
		Name: spec.Name,
		Spec: spec,
	}
	for _, ts := range tables {
		table, err := convertTable(ts, sch)
		if err != nil {
			return nil, err
		}
		sch.Tables = append(sch.Tables, table)
	}
	for _, tbl := range sch.Tables {
		if err := linkForeignKeys(tbl, sch); err != nil {
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
		Spec:   spec,
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
		Spec: spec,
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
func linkForeignKeys(tbl *schema.Table, sch *schema.Schema) error {
	for _, spec := range tbl.Spec.ForeignKeys {
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

// Spec converts a schema.Schema into scehmaspec.Schema and []schemaspec.Table.
func Spec(s *schema.Schema, fn TableSpecFunc) (*schemaspec.Schema, []*schemaspec.Table, error) {
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

// TableSpec converts  schema.Table to a schemaspec.Table.
func TableSpec(tab *schema.Table, colSpec ColumnSpecFunc) (*schemaspec.Table, error) {
	tbl := &schemaspec.Table{
		Name: tab.Name,
	}
	for _, csp := range tab.Columns {
		col, err := colSpec(csp)
		if err != nil {
			return nil, err
		}
		tbl.Columns = append(tbl.Columns, col)
	}
	return tbl, nil
}

func ColumnSpec(c *schema.Column, fn TypeSpecFunc) (*schemaspec.Column, error) {
	ct, err := fn(c.Type.Type)
	if err != nil {
		return nil, err
	}
	return &schemaspec.Column{
		Name: c.Name,
		Type: ct.Type,
		Null: ct.Null,
		Resource: schemaspec.Resource{
			Attrs: ct.Attrs,
		},
	}, nil
}
