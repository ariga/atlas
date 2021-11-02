// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package entschema

import (
	"fmt"
	"strconv"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/schema"
	sqlschema "entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
)

func Convert(graph *gen.Graph) (*schema.Schema, error) {
	var f schema.Schema
	if err := addTables(&f, graph); err != nil {
		return nil, err
	}
	if err := addForeignKeys(&f, graph); err != nil {
		return nil, err
	}
	return f, nil
}

func addTables(sch *schema.Schema, graph *gen.Graph) error {
	for _, etbl := range graph.Tables() {
		tbl := &schema.Table{
			Schema: sch,
			Name:       etbl.Name,
		}
		pk := &schemaspec.PrimaryKey{}
		for _, ec := range etbl.Columns {
			col, err := convertColumn(ec)
			if err != nil {
				return err
			}
			tbl.Columns = append(tbl.Columns, col)
			if ec.PrimaryKey() {
				pk.Columns = append(pk.Columns, &schemaspec.ColumnRef{
					Table: etbl.Name,
					Name:  col.Name,
				})
			}
			if ec.Unique {
				tbl.Indexes = append(tbl.Indexes, &schemaspec.Index{
					Name:   ec.Name,
					Unique: true,
					Columns: []*schemaspec.ColumnRef{
						{Table: etbl.Name, Name: col.Name},
					},
				})
			}
		}
		tbl.PrimaryKey = pk
		if err := addIndexes(tbl, etbl); err != nil {
			return err
		}
		sch.Tables = append(sch.Tables, tbl)
	}
	return nil
}

func addIndexes(tbl *schemaspec.Table, etbl *sqlschema.Table) error {
	for _, eidx := range etbl.Indexes {
		cols := make([]*schemaspec.ColumnRef, 0, len(eidx.Columns))
		for _, c := range eidx.Columns {
			cols = append(cols, &schemaspec.ColumnRef{
				Name:  c.Name,
				Table: etbl.Name,
			})
		}
		tbl.Indexes = append(tbl.Indexes, &schemaspec.Index{
			Name:    eidx.Name,
			Unique:  eidx.Unique,
			Columns: cols,
		})
	}
	return nil
}

func addForeignKeys(f *schema.Schema,, graph *gen.Graph) error {
	for _, etbl := range graph.Tables() {
		if len(etbl.ForeignKeys) == 0 {
			continue
		}

		tbl, ok := f.Table(etbl.Name, "")
		if !ok {
			return fmt.Errorf("entschema: could not find table %q", etbl.Name)
		}
		for _, efk := range etbl.ForeignKeys {
			refTable, ok := f.Table(efk.RefTable.Name, "")
			if !ok {
				return fmt.Errorf("entschema: could not find ref table %q", refTable.Name)
			}
			cols := make([]*schemaspec.ColumnRef, 0, len(efk.Columns))
			refCols := make([]*schemaspec.ColumnRef, 0, len(efk.RefColumns))
			for _, c := range efk.Columns {
				cols = append(cols, &schemaspec.ColumnRef{
					Name:  c.Name,
					Table: etbl.Name,
				})
			}
			for _, c := range efk.RefColumns {
				refCols = append(refCols, &schemaspec.ColumnRef{
					Name:  c.Name,
					Table: refTable.Name,
				})
			}
			tbl.ForeignKeys = append(tbl.ForeignKeys, &schemaspec.ForeignKey{
				Symbol:     efk.Symbol,
				Columns:    cols,
				RefColumns: refCols,
				OnUpdate:   string(efk.OnUpdate),
				OnDelete:   string(efk.OnDelete),
			})
		}
	}
	return nil
}

func convertColumn(col *sqlschema.Column) (*schemaspec.Column, error) {
	cspc := &schemaspec.Column{
		Name: col.Name,
		Null: col.Nullable,
	}
	switch col.Type {
	case field.TypeString:
		cspc.Type = "string"
		if col.Size != 0 {
			cspc.Attrs = append(cspc.Attrs, intAttr("size", int(col.Size)))
		}
	case field.TypeInt, field.TypeInt8, field.TypeInt16, field.TypeInt32, field.TypeInt64,
		field.TypeUint, field.TypeUint8, field.TypeUint16, field.TypeUint32, field.TypeUint64:
		cspc.Type = col.Type.String()
	case field.TypeBool:
		cspc.Type = "boolean"
	case field.TypeTime:
		cspc.Type = "time"
	case field.TypeEnum:
		cspc.Type = "enum"
		lv := &schemaspec.ListValue{V: col.Enums}
		for i, v := range lv.V {
			lv.V[i] = strconv.Quote(v)
		}
		cspc.Attrs = append(cspc.Attrs, &schemaspec.Attr{K: "values", V: lv})
	case field.TypeFloat32, field.TypeFloat64:
		// TODO: when dialect specific attributes are supported, set precision on the dialect level
		//  according to the byte-size of the float.
		cspc.Type = "float"
	case field.TypeUUID:
		// TODO: when dialect specific attributes are supported, set a dialect specific column type
		cspc.Type = "binary"
		cspc.Attrs = append(cspc.Attrs, intAttr("size", 16))
	case field.TypeBytes:
		cspc.Type = "binary"
	case field.TypeJSON:
		cspc.Type = "json"
	default:
		return nil, fmt.Errorf("entschema: unsupported type %q", col.Type)
	}
	if col.Default != nil {
		v := fmt.Sprint(col.Default)
		cspc.Default = &schemaspec.LiteralValue{V: v}
	}
	return cspc, nil
}

func intAttr(k string, v int) *schemaspec.Attr {
	return &schemaspec.Attr{
		K: k,
		V: &schemaspec.LiteralValue{V: strconv.Itoa(v)},
	}
}
