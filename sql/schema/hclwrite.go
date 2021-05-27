package schema

import (
	"fmt"
	"io"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// WriteHCL writes an HCL representation of the Schema
func WriteHCL(schema *Schema, w io.Writer) error {
	f := hclwrite.NewEmptyFile()
	body := f.Body()
	body.AppendNewBlock("schema", []string{schema.Name})
	for _, tbl := range schema.Tables {
		tbody := body.AppendNewBlock("table", []string{tbl.Name}).Body()
		tbody.SetAttributeRaw("schema", hclRawTokens("schema."+schema.Name))
		for _, col := range tbl.Columns {
			cbody := tbody.AppendNewBlock("column", []string{col.Name}).Body()
			block, err := hclColumn(col)
			if err != nil {
				return err
			}
			block.addToBody(cbody)
		}
		if len(tbl.PrimaryKey) > 0 {
			pkbody := tbody.AppendNewBlock("primary_key", nil).Body()
			block, err := hclPrimaryKey(tbl)
			if err != nil {
				return err
			}
			block.addToBody(pkbody)
		}
		for _, fk := range tbl.ForeignKeys {
			fkbody := tbody.AppendNewBlock("foreign_key", []string{fk.Symbol}).Body()
			block, err := hclForeignKey(fk)
			if err != nil {
				return err
			}
			block.addToBody(fkbody)
		}
		for _, idx := range tbl.Indexes {
			ibody := tbody.AppendNewBlock("index", []string{idx.Name}).Body()
			block, err := hclIndex(idx)
			if err != nil {
				return err
			}
			block.addToBody(ibody)
		}
	}
	_, err := f.WriteTo(w)
	return err
}

func fmtColRef(table, col string) string {
	return fmt.Sprintf("table.%s.column.%s", table, col)
}

type hclBlock struct {
	attrs  map[string]cty.Value
	raw    map[string]hclwrite.Tokens
	blocks []*hclwrite.Block
}

func newHCLBlock() *hclBlock {
	return &hclBlock{
		attrs: make(map[string]cty.Value),
		raw:   make(map[string]hclwrite.Tokens),
	}
}

func (b *hclBlock) addToBody(body *hclwrite.Body) {
	for name, raw := range b.raw {
		body.SetAttributeRaw(name, raw)
	}
	for name, attr := range b.attrs {
		body.SetAttributeValue(name, attr)
	}
	for _, block := range b.blocks {
		body.AppendBlock(block)
	}
}

func (b *hclBlock) setAttr(name string, val cty.Value) {
	b.attrs[name] = val
}

func (b *hclBlock) setStrAttr(name, val string) {
	b.setAttr(name, cty.StringVal(val))
}

func (b *hclBlock) setNumAttr(name string, val int64) {
	b.setAttr(name, cty.NumberIntVal(val))
}

func (b *hclBlock) setBoolAttr(name string, val bool) {
	b.setAttr(name, cty.BoolVal(val))
}

func hclRawTokens(s string) hclwrite.Tokens {
	return hclwrite.Tokens{
		&hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(s),
		},
	}
}

func (b *hclBlock) setRawList(attr string, items []string) {
	t := hclwrite.Tokens{&hclwrite.Token{
		Type:  hclsyntax.TokenOBrack,
		Bytes: []byte("["),
	}}
	for _, item := range items {
		t = append(t, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(item),
		}, &hclwrite.Token{
			Type:  hclsyntax.TokenComma,
			Bytes: []byte(","),
		})
	}
	t = append(t, &hclwrite.Token{
		Type:  hclsyntax.TokenCBrack,
		Bytes: []byte("]"),
	})
	b.raw[attr] = t
}

func (b *hclBlock) setRawAttr(name, token string) {
	b.raw[name] = hclRawTokens(token)
}

func hclForeignKey(fk *ForeignKey) (*hclBlock, error) {
	def := newHCLBlock()
	var cols []string
	for _, col := range fk.Columns {
		cols = append(cols, fmtColRef(fk.Table.Name, col.Name))
	}
	def.setRawList("columns", cols)
	var refCols []string
	for _, rc := range fk.RefColumns {
		refCols = append(refCols, fmtColRef(fk.RefTable.Name, rc.Name))
	}
	def.setRawList("references", refCols)
	if fk.OnDelete != "" {
		expr, err := refOptExpr(fk.OnDelete)
		if err != nil {
			return nil, err
		}
		def.setRawAttr("on_delete", expr)
	}
	if fk.OnUpdate != "" {
		expr, err := refOptExpr(fk.OnUpdate)
		if err != nil {
			return nil, err
		}
		def.setRawAttr("on_update", expr)
	}
	return def, nil
}

func refOptExpr(opt ReferenceOption) (string, error) {
	switch opt {
	case Restrict:
		return "reference_option.restrict", nil
	case NoAction:
		return "reference_option.no_action", nil
	case Cascade:
		return "reference_option.cascade", nil
	case SetNull:
		return "reference_option.set_null", nil
	case SetDefault:
		return "reference_option.set_default", nil
	default:
		return "", fmt.Errorf("schema: unknown reference option %q", opt)
	}
}

func hclIndex(i *Index) (*hclBlock, error) {
	idx := newHCLBlock()
	if i.Unique {
		idx.setBoolAttr("unique", true)
	}
	var cols []string
	for _, part := range i.Parts {
		cols = append(cols, fmtColRef(i.Table.Name, part.C.Name))
	}
	idx.setRawList("columns", cols)
	return idx, nil
}

func hclPrimaryKey(table *Table) (*hclBlock, error) {
	pk := newHCLBlock()
	var cols []string
	for _, col := range table.PrimaryKey {
		cols = append(cols, fmtColRef(table.Name, col.Name))
	}
	pk.setRawList("columns", cols)
	return pk, nil
}

func hclColumn(c *Column) (*hclBlock, error) {
	col := newHCLBlock()
	if c.Type.Null {
		col.setBoolAttr("null", true)
	}
	if c.Default != nil {
		switch exp := c.Default.(type) {
		case *RawExpr:
			col.setStrAttr("default", exp.X)
		default:
			return nil, fmt.Errorf("schema: default expression of type %T unsupported", exp)
		}
	}
	for _, attr := range c.Attrs {
		switch a := attr.(type) {
		case *Collation:
			col.setStrAttr("collation", a.V)
		case *Charset:
			col.setStrAttr("charset", a.V)
		case *Comment:
			col.setStrAttr("comment", a.Text)
		default:
			return nil, fmt.Errorf("schema: unsupported column attribute %T", a)
		}
	}
	switch ct := c.Type.Type.(type) {
	case *BoolType:
		col.setStrAttr("type", "boolean")
	case *IntegerType:
		col.setStrAttr("type", "integer")
		if ct.Unsigned {
			col.setBoolAttr("unsigned", ct.Unsigned)
		}
	case *FloatType:
		col.setStrAttr("type", "float")
		col.setNumAttr("precision", int64(ct.Precision))
	case *DecimalType:
		col.setStrAttr("type", "decimal")
		col.setNumAttr("precision", int64(ct.Precision))
		col.setNumAttr("scale", int64(ct.Scale))
	case *EnumType:
		col.setStrAttr("type", "enum")
		vals := make([]cty.Value, 0, len(ct.Values))
		for _, v := range ct.Values {
			vals = append(vals, cty.StringVal(v))
		}
		col.setAttr("values", cty.ListVal(vals))
	case *BinaryType:
		col.setStrAttr("type", "binary")
		col.setNumAttr("size", int64(ct.Size))
	case *StringType:
		col.setStrAttr("type", "string")
		col.setNumAttr("size", int64(ct.Size))
	case *TimeType:
		col.setStrAttr("type", "time")
	case *JSONType:
		col.setStrAttr("type", "json")
	default:
		return nil, fmt.Errorf("schema: unmapped col type %T", c.Type.Type)
	}
	return col, nil
}
