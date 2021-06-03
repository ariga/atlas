package entschema

import (
	"fmt"

	"ariga.io/atlas/sql/schema"
	sqlschema "entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
)

func Convert(graph *gen.Graph) (*schema.Schema, error) {
	sch := &schema.Schema{}
	if err := addTables(sch, graph); err != nil {
		return nil, err
	}
	if err := addForeignKeys(sch, graph); err != nil {
		return nil, err
	}
	return sch, nil
}

func addTables(sch *schema.Schema, graph *gen.Graph) error {
	for _, etbl := range graph.Tables() {
		tbl := &schema.Table{
			Schema: sch,
			Name:   etbl.Name,
		}
		pk := &schema.Index{}
		for _, ec := range etbl.Columns {
			col, err := convertColumn(ec)
			if err != nil {
				return err
			}
			tbl.Columns = append(tbl.Columns, col)
			if ec.PrimaryKey() {
				pk.Parts = append(pk.Parts, &schema.IndexPart{C: col, SeqNo: len(pk.Parts)})
			}
			if ec.Unique {
				tbl.Indexes = append(tbl.Indexes, &schema.Index{
					Name:   ec.Name,
					Unique: true,
					Table:  tbl,
					Attrs:  nil,
					Parts: []*schema.IndexPart{
						{C: col, SeqNo: 0},
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

func addIndexes(tbl *schema.Table, etbl *sqlschema.Table) error {
	for _, eidx := range etbl.Indexes {
		parts := make([]*schema.IndexPart, 0, len(eidx.Columns))
		for _, c := range eidx.Columns {
			col, ok := tbl.Column(c.Name)
			if !ok {
				return fmt.Errorf("entschema: could not find column %q in table %q", c.Name, tbl.Name)
			}
			parts = append(parts, &schema.IndexPart{
				SeqNo: len(parts),
				C:     col,
			})
		}
		tbl.Indexes = append(tbl.Indexes, &schema.Index{
			Name:   eidx.Name,
			Unique: eidx.Unique,
			Table:  tbl,
			Parts:  parts,
		})
	}
	return nil
}

func addForeignKeys(sch *schema.Schema, graph *gen.Graph) error {
	for _, etbl := range graph.Tables() {
		if len(etbl.ForeignKeys) == 0 {
			continue
		}
		tbl, ok := sch.Table(etbl.Name)
		if !ok {
			return fmt.Errorf("entschema: could not find table %q", etbl.Name)
		}
		for _, efk := range etbl.ForeignKeys {
			refTable, ok := sch.Table(efk.RefTable.Name)
			if !ok {
				return fmt.Errorf("entschema: could not find ref table %q", refTable.Name)
			}
			cols := make([]*schema.Column, 0, len(efk.Columns))
			refCols := make([]*schema.Column, 0, len(efk.RefColumns))
			for _, c := range efk.Columns {
				col, ok := tbl.Column(c.Name)
				if !ok {
					return fmt.Errorf("entschema: could not find column %q in table %q", c.Name, etbl.Name)
				}
				cols = append(cols, col)
			}
			for _, c := range efk.RefColumns {
				col, ok := refTable.Column(c.Name)
				if !ok {
					return fmt.Errorf("entschema: could not find column %q in ref table %q", c.Name, etbl.Name)
				}
				refCols = append(refCols, col)
			}
			tbl.ForeignKeys = append(tbl.ForeignKeys, &schema.ForeignKey{
				Symbol:     efk.Symbol,
				Table:      tbl,
				Columns:    cols,
				RefTable:   refTable,
				RefColumns: refCols,
				OnUpdate:   schema.ReferenceOption(efk.OnUpdate),
				OnDelete:   schema.ReferenceOption(efk.OnDelete),
			})
		}
	}
	return nil
}

func convertColumn(col *sqlschema.Column) (*schema.Column, error) {
	ct := &schema.ColumnType{
		Null: col.Nullable,
	}
	switch col.Type {
	case field.TypeString:
		ct.Type = &schema.StringType{
			T: "string",
		}
	case field.TypeInt, field.TypeInt8, field.TypeInt16, field.TypeInt32, field.TypeInt64:
		ct.Type = &schema.IntegerType{
			T: "integer",
		}
	case field.TypeUint, field.TypeUint8, field.TypeUint16, field.TypeUint32, field.TypeUint64:
		ct.Type = &schema.IntegerType{
			T:        "integer",
			Unsigned: true,
		}
	case field.TypeBool:
		ct.Type = &schema.BoolType{T: "boolean"}
	case field.TypeTime:
		ct.Type = &schema.TimeType{T: "time"}
	case field.TypeEnum:
		ct.Type = &schema.EnumType{Values: col.Enums}
	case field.TypeFloat32, field.TypeFloat64:
		ct.Type = &schema.FloatType{
			T: "float",
		}
	case field.TypeUUID:
		ct.Type = &schema.BinaryType{
			T:    "binary",
			Size: 16,
		}
	case field.TypeBytes:
		ct.Type = &schema.BinaryType{
			T: "binary",
		}
	case field.TypeJSON:
		ct.Type = &schema.JSONType{
			T: "json",
		}
	default:
		return nil, fmt.Errorf("entschema: unsupported type %q", col.Type)
	}
	out := &schema.Column{
		Name: col.Name,
		Type: ct,
	}
	if col.Default != nil {
		out.Default = &schema.RawExpr{X: fmt.Sprint(col.Default)}
	}
	return out, nil
}
