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
	case *schema.ListValue:
		body.SetAttributeRaw(attr.K, hclRawList(v.V))
	default:
		return fmt.Errorf("schemacl: unknown literal type %T", v)
	}
	return nil
}

func writeResource(b *schema.ResourceSpec, body *hclwrite.Body) error {
	blk := body.AppendNewBlock(b.Type, []string{b.Name})
	nb := blk.Body()
	return writeCommon(b.Attrs, b.Children, nb)
}

func writeTable(t *schema.TableSpec, body *hclwrite.Body) error {
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

func writeColumn(c *schema.ColumnSpec, body *hclwrite.Body) error {
	blk := body.AppendNewBlock("column", []string{c.Name})
	nb := blk.Body()
	nb.SetAttributeValue("type", cty.StringVal(c.Type))
	if c.Default != nil {
		nb.SetAttributeRaw("default", hclRawTokens(c.Default.V))
	}
	if c.Null {
		nb.SetAttributeValue("null", cty.BoolVal(c.Null))
	}
	return writeCommon(c.Attrs, c.Children, nb)
}

func writePk(pk *schema.PrimaryKeySpec, body *hclwrite.Body) error {
	blk := body.AppendNewBlock("primary_key", nil)
	nb := blk.Body()
	columns := make([]string, 0, len(pk.Columns))
	for _, col := range pk.Columns {
		columns = append(columns, fmt.Sprintf("table.%s.column.%s", col.Table, col.Name))
	}
	nb.SetAttributeRaw("columns", hclRawList(columns))
	return writeCommon(pk.Attrs, pk.Children, nb)
}

func writeIndex(idx *schema.IndexSpec, body *hclwrite.Body) error {
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

func writeFk(fk *schema.ForeignKeySpec, body *hclwrite.Body) error {
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

func writeSchema(spec *schema.SchemaSpec, body *hclwrite.Body) error {
	if spec.Name != "" {
		body.AppendNewBlock("schema", []string{spec.Name})
	}
	for _, tbl := range spec.Tables {
		if err := writeTable(tbl, body); err != nil {
			return err
		}
	}
	return nil
}

func write(elem schema.Element, body *hclwrite.Body) error {
	switch e := elem.(type) {
	case *schema.SchemaSpec:
		return writeSchema(e, body)
	case *schema.SpecAttr:
		return writeAttr(e, body)
	case *schema.ResourceSpec:
		return writeResource(e, body)
	case *schema.TableSpec:
		return writeTable(e, body)
	case *schema.ColumnSpec:
		return writeColumn(e, body)
	case *schema.PrimaryKeySpec:
		return writePk(e, body)
	case *schema.ForeignKeySpec:
		return writeFk(e, body)
	case *schema.IndexSpec:
		return writeIndex(e, body)
	}
	return nil
}

func writeCommon(attrs []*schema.SpecAttr, children []*schema.ResourceSpec, body *hclwrite.Body) error {
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
