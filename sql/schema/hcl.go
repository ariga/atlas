package schema

import (
	"fmt"

	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

type HCLConverter interface {
	ConvertType(*hcl.EvalContext, *ColumnHCL) (*ColumnType, error)
}

type DefaultHCLConverter struct {
}

func (c *DefaultHCLConverter) ConvertType(ctx *hcl.EvalContext, column *ColumnHCL) (*ColumnType, error) {
	switch column.TypeName {
	case "integer":
		return c.convertInteger(ctx, column)
	case "string":
		return c.convertString(ctx, column)
	default:
		return nil, fmt.Errorf("schema: unsupported column type %q", column.TypeName)
	}
}

func (*DefaultHCLConverter) convertString(ctx *hcl.EvalContext, col *ColumnHCL) (*ColumnType, error) {
	var v struct {
		Size int `hcl:"size,optional"`
	}
	if col.Attributes != nil {
		if diag := gohcl.DecodeBody(col.Attributes.HCL, ctx, &v); diag.HasErrors() {
			return nil, diag
		}
	}
	return &ColumnType{
		Type: &StringType{
			T:    col.TypeName,
			Size: v.Size,
		},
		Null: col.Null,
	}, nil
}

func (*DefaultHCLConverter) convertInteger(ctx *hcl.EvalContext, col *ColumnHCL) (*ColumnType, error) {
	var v struct {
		Size     int  `hcl:"size,optional"`
		Unsigned bool `hcl:"unsigned,optional"`
	}
	if col.Attributes != nil {
		if diag := gohcl.DecodeBody(col.Attributes.HCL, ctx, &v); diag.HasErrors() {
			return nil, diag
		}
	}
	return &ColumnType{
		Type: &IntegerType{
			T:        col.TypeName,
			Size:     v.Size,
			Unsigned: v.Unsigned,
		},
		Null: false,
	}, nil
}

// UnmarshalHCL converts HCL .schema documents into a slice of Table elements.
func UnmarshalHCL(body []byte, filename string) ([]*Schema, error) {
	parser := hclparse.NewParser()
	srcHCL, diag := parser.ParseHCL(body, filename)
	if diag.HasErrors() {
		return nil, diag
	}
	ctx, err := evalContext(srcHCL)
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
		conv := &DefaultHCLConverter{}
		for _, colHCL := range tableHCL.Columns {
			column, err := toColumn(ctx, colHCL, conv)
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
func evalContext(f *hcl.File) (*hcl.EvalContext, error) {
	var fi struct {
		Schemas []struct {
			Name string `hcl:",label"`
		} `hcl:"schema,block"`
		Remain hcl.Body `hcl:",remain"`
	}
	if diag := gohcl.DecodeBody(f.Body, &hcl.EvalContext{}, &fi); diag.HasErrors() {
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
			"schema": cty.MapVal(schemas),
		},
	}, nil
}

func toColumn(ctx *hcl.EvalContext, c *ColumnHCL, converter HCLConverter) (*Column, error) {
	columnType, err := converter.ConvertType(ctx, c)
	if err != nil {
		return nil, err
	}
	columnType.Null = c.Null
	return &Column{
		Name: c.Name,
		Type: columnType,
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
	Columns []*ColumnHCL `hcl:"column,block"`
}

type ColumnHCL struct {
	Name       string `hcl:",label"`
	TypeName   string `hcl:"type"`
	Null       bool   `hcl:"null,optional"`
	Attributes *struct {
		HCL hcl.Body `hcl:",remain"`
	} `hcl:"attributes,block"`
}
