package action

import (
	"context"
	"fmt"

	"ariga.io/atlas/sql/schema"
)

type changeDesc struct {
	typ, subject string
	queries      []string
}

func changeDescriptor(ctx context.Context, c schema.Change, d *Driver) (*changeDesc, error) {
	desc := &changeDesc{}
	switch c := c.(type) {
	case *schema.AddTable:
		desc.typ = "Add Table"
		desc.subject = c.T.Name
	case *schema.DropTable:
		desc.typ = "Drop Table"
		desc.subject = c.T.Name
	case *schema.ModifyTable:
		desc.typ = "Modify Table"
		desc.subject = c.T.Name
	case *schema.AddColumn:
		desc.typ = "Add Column"
		desc.subject = c.C.Name
	case *schema.DropColumn:
		desc.typ = "Drop Column"
		desc.subject = c.C.Name
	case *schema.ModifyColumn:
		desc.typ = "Modify Column"
		desc.subject = c.From.Name
	case *schema.AddAttr:
		desc.typ = "Add Attr"
	case *schema.ModifyAttr:
		desc.typ = "Modify Attr"
	case *schema.DropAttr:
		desc.typ = "Drop Attr"
	case *schema.AddIndex:
		desc.typ = "Add Index"
		desc.subject = c.I.Name
	case *schema.DropIndex:
		desc.typ = "Drop Index"
		desc.subject = c.I.Name
	case *schema.ModifyIndex:
		desc.typ = "Modify Index"
		desc.subject = c.From.Name
	case *schema.AddForeignKey:
		desc.typ = "Add ForeignKey"
		desc.subject = c.F.Symbol
	case *schema.DropForeignKey:
		desc.typ = "Drop ForeignKey"
		desc.subject = c.F.Symbol
	case *schema.ModifyForeignKey:
		desc.typ = "Modify ForeignKey"
		desc.subject = c.From.Symbol
	}
	d.interceptor.on()
	defer d.interceptor.clear()
	defer d.interceptor.off()
	if err := d.Exec(ctx, []schema.Change{c}); err != nil {
		return nil, fmt.Errorf("atlas: failed getting planned sql: %Web", err)
	}
	desc.queries = d.interceptor.history
	return desc, nil
}
