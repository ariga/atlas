package entschema

import (
	"fmt"
	"strconv"

	"ariga.io/atlas/sql/schema"
	sqlschema "entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
)

func Convert(graph *gen.Graph) (*schema.SchemaSpec, error) {
	sch := &schema.SchemaSpec{}
	if err := addTables(sch, graph); err != nil {
		return nil, err
	}
	if err := addForeignKeys(sch, graph); err != nil {
		return nil, err
	}
	return sch, nil
}

func addTables(sch *schema.SchemaSpec, graph *gen.Graph) error {
	for _, etbl := range graph.Tables() {
		tbl := &schema.TableSpec{
			SchemaName: sch.Name,
			Name:       etbl.Name,
		}
		pk := &schema.PrimaryKeySpec{}
		for _, ec := range etbl.Columns {
			col, err := convertColumn(ec)
			if err != nil {
				return err
			}
			tbl.Columns = append(tbl.Columns, col)
			if ec.PrimaryKey() {
				pk.Columns = append(pk.Columns, &schema.ColumnRef{
					Table: etbl.Name,
					Name:  col.Name,
				})
			}
			if ec.Unique {
				tbl.Indexes = append(tbl.Indexes, &schema.IndexSpec{
					Name:   ec.Name,
					Unique: true,
					Columns: []*schema.ColumnRef{
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

func addIndexes(tbl *schema.TableSpec, etbl *sqlschema.Table) error {
	for _, eidx := range etbl.Indexes {
		cols := make([]*schema.ColumnRef, 0, len(eidx.Columns))
		for _, c := range eidx.Columns {
			cols = append(cols, &schema.ColumnRef{
				Name:  c.Name,
				Table: etbl.Name,
			})
		}
		tbl.Indexes = append(tbl.Indexes, &schema.IndexSpec{
			Name:    eidx.Name,
			Unique:  eidx.Unique,
			Columns: cols,
		})
	}
	return nil
}

func addForeignKeys(sch *schema.SchemaSpec, graph *gen.Graph) error {
	for _, etbl := range graph.Tables() {
		if len(etbl.ForeignKeys) == 0 {
			continue
		}

		tbl, ok := tableSpec(sch, etbl.Name)
		if !ok {
			return fmt.Errorf("entschema: could not find table %q", etbl.Name)
		}
		for _, efk := range etbl.ForeignKeys {
			refTable, ok := tableSpec(sch, efk.RefTable.Name)
			if !ok {
				return fmt.Errorf("entschema: could not find ref table %q", refTable.Name)
			}
			cols := make([]*schema.ColumnRef, 0, len(efk.Columns))
			refCols := make([]*schema.ColumnRef, 0, len(efk.RefColumns))
			for _, c := range efk.Columns {
				cols = append(cols, &schema.ColumnRef{
					Name:  c.Name,
					Table: etbl.Name,
				})
			}
			for _, c := range efk.RefColumns {
				refCols = append(refCols, &schema.ColumnRef{
					Name:  c.Name,
					Table: refTable.Name,
				})
			}
			tbl.ForeignKeys = append(tbl.ForeignKeys, &schema.ForeignKeySpec{
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

func convertColumn(col *sqlschema.Column) (*schema.ColumnSpec, error) {
	cspc := &schema.ColumnSpec{
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
		lv := &schema.ListValue{V: col.Enums}
		for i, v := range lv.V {
			lv.V[i] = strconv.Quote(v)
		}
		cspc.Attrs = append(cspc.Attrs, &schema.SpecAttr{K: "values", V: lv})
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
		cspc.Default = &v
	}
	return cspc, nil
}

func tableSpec(sch *schema.SchemaSpec, name string) (*schema.TableSpec, bool) {
	for _, t := range sch.Tables {
		if t.Name == name {
			return t, true
		}
	}
	return nil, false
}

func intAttr(k string, v int) *schema.SpecAttr {
	return &schema.SpecAttr{
		K: k,
		V: &schema.LiteralValue{V: strconv.Itoa(v)},
	}
}
