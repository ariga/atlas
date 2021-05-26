package schema

import (
	"fmt"

	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// HCLConverter converts column definitions an HCL file to schema components.
type HCLConverter interface {
	// ConvertType returns the ColumnType of the column.
	ConvertType(*hcl.EvalContext, *ColumnHCL) (*ColumnType, error)

	// ConvertDefault returns the default value Expr for the column.
	ConvertDefault(*hcl.EvalContext, *ColumnHCL) (Expr, error)

	// ConvertAttrs returns a slice of Attr elements describing the column.
	ConvertAttrs(*hcl.EvalContext, *ColumnHCL) ([]Attr, error)
}

type DefaultHCLConverter struct {
}

func (c *DefaultHCLConverter) ConvertType(ctx *hcl.EvalContext, column *ColumnHCL) (*ColumnType, error) {
	typ, err := c.convertType(ctx, column)
	if err != nil {
		return nil, err
	}
	return &ColumnType{
		Type: typ,
		Null: column.Null,
	}, nil
}

func (c *DefaultHCLConverter) ConvertDefault(ctx *hcl.EvalContext, column *ColumnHCL) (Expr, error) {
	if column.Default != nil {
		return &RawExpr{X: *column.Default}, nil
	}
	return nil, nil
}

func (c *DefaultHCLConverter) ConvertAttrs(ctx *hcl.EvalContext, column *ColumnHCL) ([]Attr, error) {
	var v struct {
		Comment   *string  `hcl:"comment,optional"`
		Charset   *string  `hcl:"charset,optional"`
		Collation *string  `hcl:"collation,optional"`
		Remain    hcl.Body `hcl:",remain"`
	}
	var out []Attr
	if column.Remain != nil {
		if diag := gohcl.DecodeBody(column.Remain, ctx, &v); diag.HasErrors() {
			return nil, diag
		}
	}
	if v.Comment != nil {
		out = append(out, &Comment{Text: *v.Comment})
	}
	if v.Charset != nil {
		out = append(out, &Charset{V: *v.Charset})
	}
	if v.Collation != nil {
		out = append(out, &Collation{V: *v.Collation})
	}
	return out, nil
}

func (c *DefaultHCLConverter) convertType(ctx *hcl.EvalContext, column *ColumnHCL) (Type, error) {
	switch column.TypeName {
	case "integer":
		return c.convertInteger(ctx, column)
	case "string":
		return c.convertString(ctx, column)
	case "enum":
		return c.convertEnum(ctx, column)
	case "binary":
		return c.convertBinary(ctx, column)
	case "boolean":
		return c.convertBool(ctx, column)
	case "decimal":
		return c.convertDecimal(ctx, column)
	case "float":
		return c.convertFloat(ctx, column)
	case "time":
		return c.convertTime(ctx, column)
	case "json":
		return c.convertJSON(ctx, column)
	default:
		return nil, fmt.Errorf("schema: unsupported column type %q", column.TypeName)
	}
}

func (*DefaultHCLConverter) convertString(ctx *hcl.EvalContext, col *ColumnHCL) (Type, error) {
	var v struct {
		Size   int      `hcl:"size,optional"`
		Remain hcl.Body `hcl:",remain"`
	}
	if col.Remain != nil {
		if diag := gohcl.DecodeBody(col.Remain, ctx, &v); diag.HasErrors() {
			return nil, diag
		}
	}
	return &StringType{
		T:    col.TypeName,
		Size: v.Size,
	}, nil
}

func (*DefaultHCLConverter) convertBinary(ctx *hcl.EvalContext, col *ColumnHCL) (Type, error) {
	var v struct {
		Size   int      `hcl:"size,optional"`
		Remain hcl.Body `hcl:",remain"`
	}
	if col.Remain != nil {
		if diag := gohcl.DecodeBody(col.Remain, ctx, &v); diag.HasErrors() {
			return nil, diag
		}
	}
	return &BinaryType{
		T:    col.TypeName,
		Size: v.Size,
	}, nil
}

func (*DefaultHCLConverter) convertInteger(ctx *hcl.EvalContext, col *ColumnHCL) (Type, error) {
	var v struct {
		Size     int      `hcl:"size,optional"`
		Unsigned bool     `hcl:"unsigned,optional"`
		Remain   hcl.Body `hcl:",remain"`
	}
	if col.Remain != nil {
		if diag := gohcl.DecodeBody(col.Remain, ctx, &v); diag.HasErrors() {
			return nil, diag
		}
	}
	return &IntegerType{
		T:        col.TypeName,
		Size:     v.Size,
		Unsigned: v.Unsigned,
	}, nil
}

func (*DefaultHCLConverter) convertDecimal(ctx *hcl.EvalContext, col *ColumnHCL) (Type, error) {
	var v struct {
		Precision int `hcl:"precision"`
		Scale     int `hcl:"scale"`
	}
	if col.Remain != nil {
		if diag := gohcl.DecodeBody(col.Remain, ctx, &v); diag.HasErrors() {
			return nil, diag
		}
	}
	return &DecimalType{
		T:         col.TypeName,
		Precision: v.Precision,
		Scale:     v.Scale,
	}, nil
}

func (*DefaultHCLConverter) convertFloat(ctx *hcl.EvalContext, col *ColumnHCL) (Type, error) {
	var v struct {
		Precision int `hcl:"precision"`
	}
	if col.Remain != nil {
		if diag := gohcl.DecodeBody(col.Remain, ctx, &v); diag.HasErrors() {
			return nil, diag
		}
	}
	return &FloatType{
		T:         col.TypeName,
		Precision: v.Precision,
	}, nil
}

func (*DefaultHCLConverter) convertEnum(ctx *hcl.EvalContext, col *ColumnHCL) (Type, error) {
	var v struct {
		Values []string `hcl:"values"`
		Remain hcl.Body `hcl:",remain"`
	}
	if col.Remain != nil {
		if diag := gohcl.DecodeBody(col.Remain, ctx, &v); diag.HasErrors() {
			return nil, diag
		}
	}
	return &EnumType{
		Values: v.Values,
	}, nil
}

func (*DefaultHCLConverter) convertBool(ctx *hcl.EvalContext, col *ColumnHCL) (Type, error) {
	return &BoolType{
		T: col.TypeName,
	}, nil
}

func (*DefaultHCLConverter) convertTime(ctx *hcl.EvalContext, col *ColumnHCL) (Type, error) {
	return &TimeType{
		T: col.TypeName,
	}, nil
}

func (*DefaultHCLConverter) convertJSON(ctx *hcl.EvalContext, col *ColumnHCL) (Type, error) {
	return &JSONType{
		T: col.TypeName,
	}, nil
}

// UnmarshalHCL converts HCL .schema documents into a slice of Table elements.
func UnmarshalHCL(body []byte, filename string) ([]*Schema, error) {
	parser := hclparse.NewParser()
	srcHCL, diag := parser.ParseHCL(body, filename)
	if diag.HasErrors() {
		return nil, diag
	}
	if srcHCL == nil {
		return nil, fmt.Errorf("schema: file %q contents is nil", filename)
	}
	ctx, err := evalContext(srcHCL)
	if err != nil {
		return nil, err
	}
	f := &fileHCL{}
	if diag := gohcl.DecodeBody(srcHCL.Body, ctx, f); diag.HasErrors() {
		return nil, diag
	}
	schemas := make(map[string]*Schema)
	for _, schemaHCL := range f.Schemas {
		schemas[schemaHCL.Name] = &Schema{
			Name: schemaHCL.Name,
		}
	}
	for _, tableHCL := range f.Tables {
		table := &Table{
			Name: tableHCL.Name,
			Schema: &Schema{
				Name: tableHCL.Schema,
			},
		}
		if _, ok := schemas[table.Schema.Name]; !ok {
			return nil, fmt.Errorf("schema: unknown schema %q for table %q", table.Schema.Name, table.Name)
		}
		conv := &DefaultHCLConverter{}
		for _, colHCL := range tableHCL.Columns {
			column, err := toColumn(ctx, colHCL, conv)
			if err != nil {
				return nil, err
			}
			table.Columns = append(table.Columns, column)
		}
		if tableHCL.PrimaryKey != nil {
			if err := addPrimaryKeys(tableHCL, table); err != nil {
				return nil, err
			}
		}
		if err := addIndexes(tableHCL, table); err != nil {
			return nil, err
		}
		schemas[table.Schema.Name].Tables = append(schemas[table.Schema.Name].Tables, table)
	}
	for _, tableHCL := range f.Tables {
		if err := linkForeignKeys(schemas, tableHCL); err != nil {
			return nil, fmt.Errorf("schema: failed linking foreign keys for table %q: %w", tableHCL.Name, err)
		}
	}
	out := make([]*Schema, 0, len(schemas))
	for _, s := range schemas {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func addIndexes(tableHCL *tableHCL, table *Table) error {
	for _, idx := range tableHCL.Indexes {
		parts := make([]*IndexPart, 0, len(idx.Columns))
		for seqno, c := range idx.Columns {
			cn := c.GetAttr("name").AsString()
			col, ok := table.Column(cn)
			if !ok {
				return fmt.Errorf("schema: unknown column %q in table %q", cn, table.Name)
			}
			parts = append(parts, &IndexPart{
				SeqNo: seqno,
				C:     col,
			})
		}
		table.Indexes = append(table.Indexes, &Index{
			Name:   idx.Name,
			Unique: idx.Unique,
			Table:  table,
			Parts:  parts,
		})
	}
	return nil
}

func addPrimaryKeys(tableHCL *tableHCL, table *Table) error {
	for _, pk := range tableHCL.PrimaryKey.Columns {
		cn := pk.GetAttr("name").AsString()
		pkc, ok := table.Column(cn)
		if !ok {
			return fmt.Errorf("schema: cannot set column %q as primary key for table %q", cn, table.Name)
		}
		table.PrimaryKey = append(table.PrimaryKey, pkc)
	}
	return nil
}

func linkForeignKeys(schemas map[string]*Schema, tableHCL *tableHCL) error {
	sch, ok := schemas[tableHCL.Schema]
	if !ok {
		return fmt.Errorf("schema: unknown schema %q", tableHCL.Schema)
	}
	table, ok := sch.Table(tableHCL.Name)
	if !ok {
		return fmt.Errorf("schema: did not find table %q in schemas", tableHCL.Name)
	}
	for _, fk := range tableHCL.ForeignKeys {
		cols := make([]*Column, 0, len(fk.Columns))
		for _, col := range fk.Columns {
			cn := col.GetAttr("name").AsString()
			fkc, ok := table.Column(cn)
			if !ok {
				return fmt.Errorf("schema: unknown column %q for table %q", cn, table.Name)
			}
			cols = append(cols, fkc)
		}
		var (
			refTable   *Table
			refColumns []*Column
		)
		for _, refCol := range fk.RefColumns {
			refTableName := refCol.GetAttr("table").AsString()
			if refTable == nil {
				tbl, ok := sch.Table(refTableName)
				if !ok {
					return fmt.Errorf("schema: unknown table %q", refTableName)
				}
				refTable = tbl
			}
			if refTable.Name != refTableName {
				return fmt.Errorf(
					"schema: cannot apply foreign key %q, all referenced columns must belong to the same table",
					fk.Symbol)
			}
			refColName := refCol.GetAttr("name").AsString()
			col, ok := refTable.Column(refColName)
			if !ok {
				return fmt.Errorf("schema: unknown column %q in table %q", refColName, refTableName)
			}
			refColumns = append(refColumns, col)
		}
		table.ForeignKeys = append(table.ForeignKeys, &ForeignKey{
			Symbol:     fk.Symbol,
			Table:      table,
			Columns:    cols,
			RefTable:   refTable,
			RefColumns: refColumns,
			OnUpdate:   ReferenceOption(fk.OnUpdate),
			OnDelete:   ReferenceOption(fk.OnDelete),
		})
	}
	return nil
}

// evalContext does an initial pass through the hcl.File f and returns a context with populated
// variables that can be used in the actual file evaluation
func evalContext(f *hcl.File) (*hcl.EvalContext, error) {
	var fi struct {
		Schemas []struct {
			Name string `hcl:",label"`
		} `hcl:"schema,block"`
		Tables []struct {
			Name    string `hcl:",label"`
			Columns []struct {
				Name   string   `hcl:",label"`
				Remain hcl.Body `hcl:",remain"`
			} `hcl:"column,block"`
			Remain hcl.Body `hcl:",remain"`
		} `hcl:"table,block"`
		Remain hcl.Body `hcl:",remain"`
	}
	if diag := gohcl.DecodeBody(f.Body, &hcl.EvalContext{}, &fi); diag.HasErrors() {
		return nil, diag
	}
	schemas := make(map[string]cty.Value)
	for _, sch := range fi.Schemas {
		schemas[sch.Name] = cty.ObjectVal(map[string]cty.Value{
			"name": cty.StringVal(sch.Name),
		})
	}
	tables := make(map[string]cty.Value)
	for _, tab := range fi.Tables {
		cols := make(map[string]cty.Value)
		for _, col := range tab.Columns {
			cols[col.Name] = cty.ObjectVal(map[string]cty.Value{
				"name":  cty.StringVal(col.Name),
				"table": cty.StringVal(tab.Name),
			})
		}
		tables[tab.Name] = cty.ObjectVal(map[string]cty.Value{
			"column": cty.MapVal(cols),
		})
	}
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"schema": cty.MapVal(schemas),
			"table":  cty.MapVal(tables),
			"reference_option": cty.MapVal(map[string]cty.Value{
				"no_action":   cty.StringVal(string(NoAction)),
				"restrict":    cty.StringVal(string(Restrict)),
				"cascade":     cty.StringVal(string(Cascade)),
				"set_null":    cty.StringVal(string(SetNull)),
				"set_default": cty.StringVal(string(SetDefault)),
			}),
		},
	}, nil
}

func toColumn(ctx *hcl.EvalContext, column *ColumnHCL, converter HCLConverter) (*Column, error) {
	columnType, err := converter.ConvertType(ctx, column)
	if err != nil {
		return nil, err
	}
	def, err := converter.ConvertDefault(ctx, column)
	if err != nil {
		return nil, err
	}
	attrs, err := converter.ConvertAttrs(ctx, column)
	if err != nil {
		return nil, err
	}
	columnType.Null = column.Null
	return &Column{
		Name:    column.Name,
		Type:    columnType,
		Default: def,
		Attrs:   attrs,
	}, nil
}

type fileHCL struct {
	Tables  []*tableHCL  `hcl:"table,block"`
	Schemas []*schemaHCL `hcl:"schema,block"`
}

type schemaHCL struct {
	Name string `hcl:",label"`
}

type foreignKeyHCL struct {
	Symbol     string      `hcl:",label"`
	Columns    []cty.Value `hcl:"columns"`
	RefColumns []cty.Value `hcl:"references"`
	OnUpdate   string      `hcl:"on_update,optional"`
	OnDelete   string      `hcl:"on_delete,optional"`
	Remain     hcl.Body    `hcl:",remain"`
}

type primaryKeyHCL struct {
	Columns []cty.Value `hcl:"columns"`
}

type indexHCL struct {
	Name    string      `hcl:",label"`
	Columns []cty.Value `hcl:"columns"`
	Unique  bool        `hcl:"unique"`
}

type tableHCL struct {
	Name        string           `hcl:",label"`
	Schema      string           `hcl:"schema"`
	Columns     []*ColumnHCL     `hcl:"column,block"`
	PrimaryKey  *primaryKeyHCL   `hcl:"primary_key,block"`
	ForeignKeys []*foreignKeyHCL `hcl:"foreign_key,block"`
	Indexes     []*indexHCL      `hcl:"index,block"`
}

type ColumnHCL struct {
	Name     string   `hcl:",label"`
	TypeName string   `hcl:"type"`
	Null     bool     `hcl:"null,optional"`
	Default  *string  `hcl:"default,optional"`
	Remain   hcl.Body `hcl:",remain"`
}
