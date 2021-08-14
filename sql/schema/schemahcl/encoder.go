// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl

import (
	"bytes"
	"fmt"

	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/schema/schemaspec"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// Encode implements schema.Encoder by taking a Spec and returning a byte slice of an HCL
// document representing the spec.
func Encode(elem schemaspec.Element) ([]byte, error) {
	f := hclwrite.NewFile()
	if err := write(elem, f.Body()); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	_, err := f.WriteTo(&buf)
	return buf.Bytes(), err
}

func writeAttr(attr *schemaspec.Attr, body *hclwrite.Body) error {
	switch v := attr.V.(type) {
	case *schemaspec.LiteralValue:
		body.SetAttributeRaw(attr.K, hclRawTokens(v.V))
	case *schemaspec.ListValue:
		body.SetAttributeRaw(attr.K, hclRawList(v.V))
	default:
		return fmt.Errorf("schemacl: unknown literal type %T", v)
	}
	return nil
}

func writeResource(b *schemaspec.Resource, body *hclwrite.Body) error {
	blk := body.AppendNewBlock(b.Type, []string{b.Name})
	nb := blk.Body()
	return writeCommon(b.Attrs, b.Children, nb)
}

func writeTable(t *schemaspec.Table, body *hclwrite.Body) error {
	blk := body.AppendNewBlock("table", []string{t.Name})
	nb := blk.Body()
	if t.SchemaName != "" {
		nb.SetAttributeRaw("schema", hclRawTokens(fmt.Sprintf("schema.%s", t.SchemaName)))
	}
	for _, col := range t.Columns {
		if err := writeColumn(col, nb); err != nil {
			return err
		}
	}
	if t.PrimaryKey != nil {
		if err := writePk(t.PrimaryKey, nb); err != nil {
			return err
		}
	}
	for _, idx := range t.Indexes {
		if err := writeIndex(idx, nb); err != nil {
			return err
		}
	}
	for _, fk := range t.ForeignKeys {
		if err := writeFk(fk, nb); err != nil {
			return err
		}
	}
	return writeCommon(t.Attrs, t.Children, nb)
}

func writeColumn(c *schemaspec.Column, body *hclwrite.Body) error {
	blk := body.AppendNewBlock("column", []string{c.Name})
	nb := blk.Body()
	nb.SetAttributeValue("type", cty.StringVal(c.Type))
	if c.Default != nil {
		nb.SetAttributeRaw("default", hclRawTokens(c.Default.V))
	}
	if c.Null {
		nb.SetAttributeValue("null", cty.BoolVal(c.Null))
	}
	for _, o := range c.Overrides {
		if err := writeOverride(o, nb); err != nil {
			return err
		}
	}
	return writeCommon(c.Attrs, c.Children, nb)
}

func writeOverride(o *schemaspec.Override, body *hclwrite.Body) error {
	blk := body.AppendNewBlock("dialect", []string{o.Dialect})
	nb := blk.Body()
	nb.SetAttributeValue("version", cty.StringVal(o.Version))
	return writeCommon(o.Attrs, o.Children, nb)
}

func writePk(pk *schemaspec.PrimaryKey, body *hclwrite.Body) error {
	blk := body.AppendNewBlock("primary_key", nil)
	nb := blk.Body()
	columns := make([]string, 0, len(pk.Columns))
	for _, col := range pk.Columns {
		columns = append(columns, fmt.Sprintf("table.%s.column.%s", col.Table, col.Name))
	}
	nb.SetAttributeRaw("columns", hclRawList(columns))
	return writeCommon(pk.Attrs, pk.Children, nb)
}

func writeIndex(idx *schemaspec.Index, body *hclwrite.Body) error {
	blk := body.AppendNewBlock("index", []string{idx.Name})
	nb := blk.Body()
	if idx.Unique {
		nb.SetAttributeValue("unique", cty.True)
	}
	columns := make([]string, 0, len(idx.Columns))
	for _, col := range idx.Columns {
		columns = append(columns, fmt.Sprintf("table.%s.column.%s", col.Table, col.Name))
	}
	nb.SetAttributeRaw("columns", hclRawList(columns))
	return writeCommon(idx.Attrs, idx.Children, nb)
}

func writeFk(fk *schemaspec.ForeignKey, body *hclwrite.Body) error {
	blk := body.AppendNewBlock("foreign_key", []string{fk.Symbol})
	nb := blk.Body()
	columns := make([]string, 0, len(fk.Columns))
	for _, col := range fk.Columns {
		columns = append(columns, fmt.Sprintf("table.%s.column.%s", col.Table, col.Name))
	}
	nb.SetAttributeRaw("columns", hclRawList(columns))
	refCols := make([]string, 0, len(fk.RefColumns))
	for _, col := range fk.RefColumns {
		refCols = append(refCols, fmt.Sprintf("table.%s.column.%s", col.Table, col.Name))
	}
	nb.SetAttributeRaw("references", hclRawList(refCols))
	if fk.OnDelete != "" {
		expr, err := refOptExpr(fk.OnDelete)
		if err != nil {
			return err
		}
		nb.SetAttributeRaw("on_delete", hclRawTokens(expr))
	}
	if fk.OnUpdate != "" {
		expr, err := refOptExpr(fk.OnUpdate)
		if err != nil {
			return err
		}
		nb.SetAttributeRaw("on_update", hclRawTokens(expr))
	}
	return writeCommon(fk.Attrs, fk.Children, nb)
}

func writeFile(spec *schemaspec.File, body *hclwrite.Body) error {
	for _, sch := range spec.Schemas {
		if err := writeSchema(sch, body); err != nil {
			return err
		}
	}
	for _, tbl := range spec.Tables {
		if err := writeTable(tbl, body); err != nil {
			return err
		}
	}
	return writeCommon(spec.Attrs, spec.Children, body)
}

func writeSchema(spec *schemaspec.Schema, body *hclwrite.Body) error {
	if spec.Name != "" {
		body.AppendNewBlock("schema", []string{spec.Name})
	}
	return nil
}

func write(elem schemaspec.Element, body *hclwrite.Body) error {
	switch e := elem.(type) {
	case *schemaspec.File:
		return writeFile(e, body)
	case *schemaspec.Schema:
		return writeSchema(e, body)
	case *schemaspec.Attr:
		return writeAttr(e, body)
	case *schemaspec.Resource:
		return writeResource(e, body)
	case *schemaspec.Table:
		return writeTable(e, body)
	case *schemaspec.Column:
		return writeColumn(e, body)
	case *schemaspec.PrimaryKey:
		return writePk(e, body)
	case *schemaspec.ForeignKey:
		return writeFk(e, body)
	case *schemaspec.Index:
		return writeIndex(e, body)
	default:
		return fmt.Errorf("schemahcl: unsupported element %T", e)
	}
}

func writeCommon(attrs []*schemaspec.Attr, children []*schemaspec.Resource, body *hclwrite.Body) error {
	for _, attr := range attrs {
		if err := writeAttr(attr, body); err != nil {
			return err
		}
	}
	for _, b := range children {
		if err := writeResource(b, body); err != nil {
			return err
		}
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

func hclRawList(items []string) hclwrite.Tokens {
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
	return t
}

func refOptExpr(opt string) (string, error) {
	rf := schema.ReferenceOption(opt)
	switch rf {
	case schema.Restrict:
		return "reference_option.restrict", nil
	case schema.NoAction:
		return "reference_option.no_action", nil
	case schema.Cascade:
		return "reference_option.cascade", nil
	case schema.SetNull:
		return "reference_option.set_null", nil
	case schema.SetDefault:
		return "reference_option.set_default", nil
	default:
		return "", fmt.Errorf("schema: unknown reference option %q", opt)
	}
}
