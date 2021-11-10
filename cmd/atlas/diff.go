package main

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
		desc.typ = "AddTable"
		desc.subject = c.T.Name
	case *schema.DropTable:
		desc.typ = "DropTable"
		desc.subject = c.T.Name
	case *schema.ModifyTable:
		desc.typ = "ModifyTable"
		desc.subject = c.T.Name
	case *schema.AddColumn:
		desc.typ = "AddColumn"
		desc.subject = c.C.Name
	case *schema.DropColumn:
		desc.typ = "DropColumn"
		desc.subject = c.C.Name
	case *schema.ModifyColumn:
		desc.typ = "ModifyColumn"
		desc.subject = c.From.Name
	case *schema.AddAttr:
		desc.typ = "AddAttr"
	case *schema.ModifyAttr:
		desc.typ = "ModifyAttr"
	case *schema.DropAttr:
		desc.typ = "DropAttr"
	case *schema.AddIndex:
		desc.typ = "AddIndex"
	case *schema.DropIndex:
		desc.typ = "DropIndex"
	case *schema.ModifyIndex:
		desc.typ = "ModifyIndex"
	case *schema.AddForeignKey:
		desc.typ = "AddForeignKey"
	case *schema.DropForeignKey:
		desc.typ = "DropForeignKey"
	case *schema.ModifyForeignKey:
		desc.typ = "ModifyForeignKey"
	}
	d.interceptor.on()
	defer d.interceptor.clear()
	defer d.interceptor.off()
	if err := d.Exec(ctx, []schema.Change{c}); err != nil {
		return nil, fmt.Errorf("atlas: failed getting planned sql: %w", err)
	}
	desc.queries = d.interceptor.history
	return desc, nil
}
