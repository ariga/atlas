package schema

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// Unmarshaler is used to read textual representations of schemas into schema.Schema elements.
type Unmarshaler struct {
}

// UnmarshalHCL converts HCL .schema documents into a slice of Table elements.
func (u *Unmarshaler) UnmarshalHCL(body []byte, filename string) ([]*Table, error) {
	parser := hclparse.NewParser()
	srcHCL, diag := parser.ParseHCL(body, filename)
	if diag.HasErrors() {
		return nil, diag
	}
	ctx := &hcl.EvalContext{}
	schemaHCL := &schemaHCL{}
	if diag := gohcl.DecodeBody(srcHCL.Body, ctx, schemaHCL); diag.HasErrors() {
		return nil, diag
	}
	out := make([]*Table, 0, len(schemaHCL.Tables))
	for _, tableHCL := range schemaHCL.Tables {
		table := &Table{
			Name: tableHCL.Name,
		}
		for _, colHCL := range tableHCL.Columns {
			column, err := u.toColumn(colHCL)
			if err != nil {
				return nil, err
			}
			table.Columns = append(table.Columns, column)
		}
		out = append(out, table)
	}
	return out, nil
}

func (u *Unmarshaler) toColumn(c *columnHCL) (*Column, error) {
	// TODO: handle column types and attributes
	return &Column{
		Name: c.Name,
	}, nil
}

type schemaHCL struct {
	Tables []*tableHCL `hcl:"table,block"`
}

type tableHCL struct {
	Name    string       `hcl:",label"`
	Columns []*columnHCL `hcl:"column,block"`
}

type columnHCL struct {
	Name          string `hcl:",label"`
	TypeName      string `hcl:"type"`
	AttributesHCL *struct {
		HCL hcl.Body `hcl:",remain"`
	} `hcl:"attributes,block"`
}
