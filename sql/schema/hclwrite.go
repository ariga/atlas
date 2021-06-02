package schema

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// MarshalHCL returns a byte slice containing an HCL representation of the Schema
func MarshalHCL(schema *Schema) ([]byte, error) {
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
				return nil, err
			}
			if err := block.addToBody(cbody); err != nil {
				return nil, err
			}
		}
		if tbl.PrimaryKey != nil {
			pkbody := tbody.AppendNewBlock("primary_key", nil).Body()
			block, err := hclIndex(tbl.PrimaryKey)
			if err != nil {
				return nil, err
			}
			if err := block.addToBody(pkbody); err != nil {
				return nil, err
			}
		}
		for _, fk := range tbl.ForeignKeys {
			fkbody := tbody.AppendNewBlock("foreign_key", []string{fk.Symbol}).Body()
			block, err := hclForeignKey(fk)
			if err != nil {
				return nil, err
			}
			if err := block.addToBody(fkbody); err != nil {
				return nil, err
			}
		}
		for _, idx := range tbl.Indexes {
			ibody := tbody.AppendNewBlock("index", []string{idx.Name}).Body()
			block, err := hclIndex(idx)
			if err != nil {
				return nil, err
			}
			if err := block.addToBody(ibody); err != nil {
				return nil, err
			}
		}
	}
	var buf bytes.Buffer
	_, err := f.WriteTo(&buf)
	return buf.Bytes(), err
}

func fmtColRef(table, col string) string {
	return fmt.Sprintf("table.%s.column.%s", table, col)
}

type blockElement struct {
	name  string
	value interface{}
}
type hclBlock struct {
	elements []blockElement
}

func (b *hclBlock) addToBody(body *hclwrite.Body) error {
	for _, elem := range b.elements {
		switch v := elem.value.(type) {
		case cty.Value:
			body.SetAttributeValue(elem.name, v)
		case *hclwrite.Block:
			body.AppendBlock(v)
		case hclwrite.Tokens:
			body.SetAttributeRaw(elem.name, v)
		default:
			return fmt.Errorf("schema: unknown element type %T", v)
		}
	}
	return nil
}

func (b *hclBlock) setAttr(name string, val cty.Value) {
	b.elements = append(b.elements, blockElement{
		name:  name,
		value: val,
	})
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
	b.elements = append(b.elements, blockElement{
		name:  attr,
		value: t,
	})
}

func (b *hclBlock) setRawAttr(name, token string) {
	b.elements = append(b.elements, blockElement{
		name:  name,
		value: hclRawTokens(token),
	})
}

func hclForeignKey(fk *ForeignKey) (*hclBlock, error) {
	def := &hclBlock{}
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
	idx := &hclBlock{}
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

func hclColumn(c *Column) (*hclBlock, error) {
	col := &hclBlock{}
	if c.Type.Null {
		col.setBoolAttr("null", true)
	}
	if c.Default != nil {
		exp, ok := c.Default.(*RawExpr)
		if !ok {
			return nil, fmt.Errorf("schema: default expression of type %T unsupported", exp)
		}
		col.setStrAttr("default", exp.X)
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
