package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuilder_Build(t *testing.T) {
	b := &Builder{
		SpecConverter: &DefaultSpecConverter{},
	}

	spec := &SchemaSpec{
		Name: "schema",
		Tables: []*TableSpec{
			{
				Name: "table",
				Columns: []*ColumnSpec{
					{
						Name:     "col",
						TypeName: "int",
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
					Type: &ColumnType{}, // TODO(rotemtam): in next PR
					Spec: spec.Tables[0].Columns[0],
				},
			},
		},
	}
	require.EqualValues(t, exp, sch)
}
