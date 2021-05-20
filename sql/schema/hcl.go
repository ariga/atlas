package schema

import (
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// Unmarshaler is used to read textual representations of schemas into schema.Schema elements.
type Unmarshaler struct {
}

// UnmarshalHCL converts HCL .schema documents into a slice of Table elements.
func (u *Unmarshaler) UnmarshalHCL(body []byte, filename string) ([]*Schema, error) {
	parser := hclparse.NewParser()
	srcHCL, diag := parser.ParseHCL(body, filename)
	if diag.HasErrors() {
		return nil, diag
	}
	ctx, err := u.evalContext(srcHCL)
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
			Name:   tableHCL.Name,
			Schema: tableHCL.Schema,
		}
		if _, ok := schemas[table.Schema]; !ok {
			return nil, fmt.Errorf("schema: unknown schema %q for table %q", table.Schema, table.Name)
		}
		for _, colHCL := range tableHCL.Columns {
			column, err := u.toColumn(colHCL)
			if err != nil {
				return nil, err
			}
			table.Columns = append(table.Columns, column)
		}
		schemas[table.Schema].Tables = append(schemas[table.Schema].Tables, table)
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

// evalContext does an initial pass through the hcl.File f and returns a context with populated
// variables that can be used in the actual file evaluation
func (u *Unmarshaler) evalContext(f *hcl.File) (*hcl.EvalContext, error) {
	fi := &struct {
		Schemas []*struct {
			Name string `hcl:",label"`
		} `hcl:"schema,block"`
		Remain hcl.Body `hcl:",remain"`
	}{}
	if diag := gohcl.DecodeBody(f.Body, &hcl.EvalContext{}, fi); diag.HasErrors() {
		return nil, diag
	}
	schemas := make(map[string]cty.Value)
	for _, sch := range fi.Schemas {
		schemas[sch.Name] = cty.ObjectVal(map[string]cty.Value{
			"name": cty.StringVal(sch.Name),
		})
	}
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"schemas": cty.MapVal(schemas),
		},
	}, nil
}

func (u *Unmarshaler) toColumn(c *columnHCL) (*Column, error) {
	// TODO: handle column types and attributes
	return &Column{
		Name: c.Name,
	}, nil
}

type fileHCL struct {
	Tables  []*tableHCL  `hcl:"table,block"`
	Schemas []*schemaHCL `hcl:"schema,block"`
}

type schemaHCL struct {
	Name string `hcl:",label"`
}

type tableHCL struct {
	Name    string       `hcl:",label"`
	Schema  string       `hcl:"schema"`
	Columns []*columnHCL `hcl:"column,block"`
}

type columnHCL struct {
	Name          string `hcl:",label"`
	TypeName      string `hcl:"type"`
	AttributesHCL *struct {
		HCL hcl.Body `hcl:",remain"`
	} `hcl:"attributes,block"`
}
