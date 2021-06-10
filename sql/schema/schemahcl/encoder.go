package schemahcl

import (
	"bytes"
	"fmt"

	"ariga.io/atlas/sql/schema"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// Encode implements schema.Encoder by taking a Spec and returning a byte slice of an HCL
// document representing the spec.
func Encode(elem schema.Element) ([]byte, error) {
	f := hclwrite.NewFile()
	if err := write(elem, f.Body()); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	_, err := f.WriteTo(&buf)
	return buf.Bytes(), err
}

func writeAttr(attr *schema.SpecAttr, body *hclwrite.Body) error {
	switch v := attr.V.(type) {
	case *schema.LiteralValue:
		body.SetAttributeRaw(attr.K, hclRawTokens(v.V))
	default:
		return fmt.Errorf("schemacl: unknown literal type %T", v)
	}
	return nil
}

func writeResource(b *schema.ResourceSpec, body *hclwrite.Body) error {
	blk := body.AppendNewBlock(b.Type, []string{b.Name})
	nb := blk.Body()
	for _, attr := range b.Attrs {
		if err := writeAttr(attr, nb); err != nil {
			return err
		}
	}
	for _, child := range b.Children {
		if err := writeResource(child, nb); err != nil {
			return err
		}
	}
	return nil
}

func writeTable(t *schema.TableSpec, body *hclwrite.Body) error {
	blk := body.AppendNewBlock("table", []string{t.Name})
	nb := blk.Body()
	for _, attr := range t.Attrs {
		if err := writeAttr(attr, nb); err != nil {
			return err
		}
	}
	for _, col := range t.Columns {
		if err := writeColumn(col, nb); err != nil {
			return err
		}
	}
	for _, child := range t.Children {
		if err := writeResource(child, nb); err != nil {
			return err
		}
	}
	return nil
}

func writeColumn(c *schema.ColumnSpec, body *hclwrite.Body) error {
	blk := body.AppendNewBlock("column", []string{c.Name})
	nb := blk.Body()
	nb.SetAttributeValue("type", cty.StringVal(c.Type))
	if c.Default != nil {
		nb.SetAttributeValue("default", cty.StringVal(*c.Default))
	}
	if c.Null {
		nb.SetAttributeValue("null", cty.BoolVal(c.Null))
	}
	for _, attr := range c.Attrs {
		if err := writeAttr(attr, nb); err != nil {
			return err
		}
	}
	for _, b := range c.Children {
		if err := writeResource(b, nb); err != nil {
			return err
		}
	}
	return nil
}

func write(elem schema.Element, body *hclwrite.Body) error {
	switch e := elem.(type) {
	case *schema.SpecAttr:
		return writeAttr(e, body)
	case *schema.ResourceSpec:
		return writeResource(e, body)
	case *schema.TableSpec:
		return writeTable(e, body)
	case *schema.ColumnSpec:
		return writeColumn(e, body)
	}
	return nil
}

func hclRawTokens(s string) hclwrite.Tokens {
	return hclwrite.Tokens{
		&hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(s),
		},
	}
}
