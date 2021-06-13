package mysql

import (
	"fmt"
	"math"
	"strings"

	"ariga.io/atlas/sql/schema"
)

// ConvertSchema converts a SchemaSpec into a Schema.
func ConvertSchema(spec *schema.SchemaSpec) (*schema.Schema, error) {
	sch := &schema.Schema{
		Name: spec.Name,
		Spec: spec,
	}
	for _, ts := range spec.Tables {
		table, err := ConvertTable(ts, sch)
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

// ConvertTable converts a TableSpec to a Table. Table conversion is done without converting
// ForeignKeySpecs into ForeignKeys, as the target tables do not necessarily exist in the schema
// at this point. Instead, the linking is done by the ConvertSchema function.
func ConvertTable(spec *schema.TableSpec, parent *schema.Schema) (*schema.Table, error) {
	tbl := &schema.Table{
		Name:   spec.Name,
		Schema: parent,
		Spec:   spec,
	}
	for _, csp := range spec.Columns {
		col, err := ConvertColumn(csp, tbl)
		if err != nil {
			return nil, err
		}
		tbl.Columns = append(tbl.Columns, col)
	}
	if spec.PrimaryKey != nil {
		pk, err := ConvertPrimaryKey(spec.PrimaryKey, tbl)
		if err != nil {
			return nil, err
		}
		tbl.PrimaryKey = pk
	}
	for _, idx := range spec.Indexes {
		i, err := ConvertIndex(idx, tbl)
		if err != nil {
			return nil, err
		}
		tbl.Indexes = append(tbl.Indexes, i)
	}
	return tbl, nil
}

// ConvertPrimaryKey converts a PrimaryKeySpec to an Index.
func ConvertPrimaryKey(spec *schema.PrimaryKeySpec, parent *schema.Table) (*schema.Index, error) {
	parts := make([]*schema.IndexPart, 0, len(spec.Columns))
	for seqno, c := range spec.Columns {
		pkc, ok := parent.Column(c.Name)
		if !ok {
			return nil, fmt.Errorf("mysql: cannot set column %q as primary key for table %q", c.Name, parent.Name)
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

// ConvertIndex converts an IndexSpec to an Index.
func ConvertIndex(spec *schema.IndexSpec, parent *schema.Table) (*schema.Index, error) {
	parts := make([]*schema.IndexPart, 0, len(spec.Columns))
	for seqno, c := range spec.Columns {
		cn := c.Name
		col, ok := parent.Column(cn)
		if !ok {
			return nil, fmt.Errorf("mysql: unknown column %q in table %q", cn, parent.Name)
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

// ConvertColumn converts a ColumnSpec into a Column.
func ConvertColumn(spec *schema.ColumnSpec, parent *schema.Table) (*schema.Column, error) {
	out := &schema.Column{
		Name: spec.Name,
		Spec: spec,
		Type: &schema.ColumnType{
			Null: spec.Null,
		},
	}
	if spec.Default != nil {
		out.Default = &schema.Literal{V: *spec.Default}
	}
	ct, err := ConvertColumnType(spec)
	if err != nil {
		return nil, err
	}
	out.Type.Type = ct
	return out, err
}

func ConvertColumnType(spec *schema.ColumnSpec) (schema.Type, error) {
	switch spec.Type {
	case "int", "int8", "int16", "int64", "uint", "uint8", "uint16", "uint64":
		return convertInteger(spec)
	case "string":
		return convertString(spec)
	case "binary":
		return convertBinary(spec)
	case "enum":
		return convertEnum(spec)
	case "boolean":
		return convertBoolean(spec)
	case "decimal":
		return convertDecimal(spec)
	case "float":
		return convertFloat(spec)
	case "time":
		return convertTime(spec)
	}
	return parseRawType(spec.Type)
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

func resolveCol(ref *schema.ColumnRef, sch *schema.Schema) (*schema.Column, error) {
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

func convertInteger(spec *schema.ColumnSpec) (schema.Type, error) {
	typ := &schema.IntegerType{
		Unsigned: strings.HasPrefix(spec.Type, "u"),
	}
	switch spec.Type {
	case "int8", "uint8":
		typ.Size = 1
		typ.T = tTinyInt
	case "int16", "uint16":
		typ.Size = 2
		typ.T = tSmallInt
	case "int32", "uint32", "int", "integer", "uint":
		typ.Size = 4
		typ.T = tInt
	case "int64", "uint64":
		typ.Size = 8
		typ.T = tBigInt
	default:
		return nil, fmt.Errorf("mysql: unknown integer column type %q", spec.Type)
	}
	return typ, nil
}

func convertBinary(spec *schema.ColumnSpec) (schema.Type, error) {
	bt := &schema.BinaryType{}
	if attr, ok := spec.Attr("size"); ok {
		s, err := attr.Int()
		if err != nil {
			return nil, err
		}
		bt.Size = s
	}
	switch {
	case bt.Size == 0:
		bt.T = "blob"
	case bt.Size <= math.MaxUint8:
		bt.T = "tinyblob"
	case bt.Size > math.MaxUint8 && bt.Size <= math.MaxUint16:
		bt.T = "blob"
	case bt.Size > math.MaxUint16 && bt.Size <= 1<<24-1:
		bt.T = "mediumblob"
	case bt.Size > 1<<24-1 && bt.Size <= math.MaxUint32:
		bt.T = "longblob"
	default:
		return nil, fmt.Errorf("mysql: blob fields can be up to 4GB long")
	}
	return bt, nil
}

func convertString(spec *schema.ColumnSpec) (schema.Type, error) {
	st := &schema.StringType{
		Size: 255,
	}
	if attr, ok := spec.Attr("size"); ok {
		s, err := attr.Int()
		if err != nil {
			return nil, err
		}
		st.Size = s
	}
	switch {
	case st.Size <= math.MaxUint16:
		st.T = "varchar"
	case st.Size > math.MaxUint16 && st.Size <= (1<<24-1):
		st.T = "mediumtext"
	case st.Size > (1<<24-1) && st.Size <= math.MaxUint32:
		st.T = "longtext"
	default:
		return nil, fmt.Errorf("mysql: string fields can be up to 4GB long")
	}
	return st, nil
}

func convertEnum(spec *schema.ColumnSpec) (schema.Type, error) {
	attr, ok := spec.Attr("values")
	if !ok {
		return nil, fmt.Errorf("mysql: expected enum fields to have values")
	}
	list, err := attr.Strings()
	if err != nil {
		return nil, err
	}
	return &schema.EnumType{Values: list}, nil
}

func convertBoolean(spec *schema.ColumnSpec) (schema.Type, error) {
	return &schema.BoolType{T: "boolean"}, nil
}

func convertTime(spec *schema.ColumnSpec) (schema.Type, error) {
	return &schema.TimeType{T: "timestamp"}, nil
}

func convertDecimal(spec *schema.ColumnSpec) (schema.Type, error) {
	dt := &schema.DecimalType{
		T: tDecimal,
	}
	if precision, ok := spec.Attr("precision"); ok {
		p, err := precision.Int()
		if err != nil {
			return nil, err
		}
		dt.Precision = p
	}
	if scale, ok := spec.Attr("scale"); ok {
		s, err := scale.Int()
		if err != nil {
			return nil, err
		}
		dt.Scale = s
	}
	return dt, nil
}

func convertFloat(spec *schema.ColumnSpec) (schema.Type, error) {
	ft := &schema.FloatType{
		T: tFloat,
	}
	if precision, ok := spec.Attr("precision"); ok {
		p, err := precision.Int()
		if err != nil {
			return nil, err
		}
		ft.Precision = p
	}
	// A precision from 0 to 23 results in a 4-byte single-precision FLOAT column.
	// A precision from 24 to 53 results in an 8-byte double-precision DOUBLE column:
	// https://dev.mysql.com/doc/refman/8.0/en/floating-point-types.html
	if ft.Precision > 23 {
		ft.T = "double"
	}
	return ft, nil
}
