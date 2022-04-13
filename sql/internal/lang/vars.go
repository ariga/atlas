package lang

import (
	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
)

// Var is an input variable to an Atlas DDL document.
type Var struct {
	Name    string                   `spec:",name"`
	Type    *schemaspec.Type         `spec:"type"`
	Default *schemaspec.LiteralValue `spec:"default"`
}

type doc struct {
	Vars []*Var `spec:"variable"`
}

// ExtractVarsHCL extracts the variable definitions from an Atlas DDL HCL document.
func ExtractVarsHCL(body []byte) ([]*Var, error) {
	d := doc{}
	if err := hclState.UnmarshalSpec(body, &d); err != nil {
		return nil, err
	}
	return d.Vars, nil
}

var (
	hclState = schemahcl.New(schemahcl.WithTypes(
		specutil.NewRegistry(
			specutil.WithSpecs(
				specutil.TypeSpec("int"),
				specutil.TypeSpec("bool"),
				specutil.TypeSpec("float"),
			),
		).Specs(),
	))
)

func init() {
	schemaspec.Register("variable", &Var{})
}
