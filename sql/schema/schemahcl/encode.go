package schemahcl

import (
	"bytes"
	"fmt"

	"ariga.io/atlas/sql/schema/schemacodec"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

func Encode(elem schemacodec.Element) ([]byte, error) {
	f := hclwrite.NewFile()
	if err := write(elem, f.Body()); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	_, err := f.WriteTo(&buf)
	return buf.Bytes(), err
}

func writeAttr(attr *schemacodec.Attr, body *hclwrite.Body) error {
	switch v := attr.V.(type) {
	case schemacodec.String:
		body.SetAttributeValue(attr.K, cty.StringVal(string(v)))
	case schemacodec.Number:
		body.SetAttributeValue(attr.K, cty.NumberFloatVal(float64(v)))
	case schemacodec.Bool:
		body.SetAttributeValue(attr.K, cty.BoolVal(bool(v)))
	default:
		return fmt.Errorf("schemacl: unknown literal type %T", v)
	}
	return nil
}

func writeBlock(b *schemacodec.Block, body *hclwrite.Body) error {
	blk := body.AppendNewBlock(b.Type, b.Labels)
	nb := blk.Body()
	for _, attr := range b.Attrs {
		if err := writeAttr(attr, nb); err != nil {
			return err
		}
	}
	for _, child := range b.Blocks {
		if err := writeBlock(child, nb); err != nil {
			return err
		}
	}
	return nil
}

func writeTable(t *schemacodec.TableSpec, body *hclwrite.Body) error {
	blk := body.AppendNewBlock("table", []string{t.Name})
	nb := blk.Body()
	for _, col := range t.Columns {
		if err := writeColumn(col, nb); err != nil {
			return err
		}
	}
	return nil
}

func writeColumn(c *schemacodec.ColumnSpec, body *hclwrite.Body) error {
	blk := body.AppendNewBlock("column", []string{c.Name})
	nb := blk.Body()
	nb.SetAttributeValue("type", cty.StringVal(c.TypeName))
	if c.Default != nil {
		nb.SetAttributeValue("default", cty.StringVal(*c.Default))
	}
	if c.Null {
		nb.SetAttributeValue("null", cty.BoolVal(c.Null))
	}
	for _, attr := range c.Attrs {
		if err := writeAttr(attr, body); err != nil {
			return err
		}
	}
	for _, b := range c.Blocks {
		if err := writeBlock(b, nb); err != nil {
			return err
		}
	}
	return nil
}

func write(elem schemacodec.Element, body *hclwrite.Body) error {
	switch e := elem.(type) {
	case *schemacodec.Attr:
		return writeAttr(e, body)
	case *schemacodec.Block:
		return writeBlock(e, body)
	case *schemacodec.TableSpec:
		return writeTable(e, body)
	case *schemacodec.ColumnSpec:
		return writeColumn(e, body)
	}
	return nil
}
