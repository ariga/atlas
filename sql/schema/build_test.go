package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuilder_Build(t *testing.T) {
	b := &Builder{
		SpecConverter: &dummyConverter{},
	}

	spec := &SchemaSpec{
		Name: "schema",
		Tables: []*TableSpec{
			{
				Name: "table",
				Columns: []*ColumnSpec{
					{
						Name: "col",
						Type: "int",
					},
				},
			},
		},
	}
	sch, err := b.Build(spec)
	require.NoError(t, err)
	exp := &Schema{
		Name: "schema",
		Spec: spec,
	}
	exp.Tables = []*Table{
		{
			Name:   "table",
			Schema: exp,
			Spec:   spec.Tables[0],
			Columns: []*Column{
				{
					Name: "col",
					Type: &ColumnType{},
					Spec: spec.Tables[0].Columns[0],
				},
			},
		},
	}
	require.EqualValues(t, exp, sch)
}

type dummyConverter struct {
}

func (d *dummyConverter) ColumnType(spec *ColumnSpec) (Type, error) {
	return nil, nil
}
