package mysql

import (
	"testing"

	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/require"
)

func TestBuilder_Build(t *testing.T) {
	b := &Builder{
		SpecConverter: &dummyConverter{},
	}

	spec := &schema.SchemaSpec{
		Name: "schema",
		Tables: []*schema.TableSpec{
			{
				Name: "table",
				Columns: []*schema.ColumnSpec{
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
	exp := &schema.Schema{
		Name: "schema",
		Spec: spec,
	}
	exp.Tables = []*schema.Table{
		{
			Name:   "table",
			Schema: exp,
			Spec:   spec.Tables[0],
			Columns: []*schema.Column{
				{
					Name: "col",
					Type: &schema.ColumnType{},
					Spec: spec.Tables[0].Columns[0],
				},
			},
		},
	}
	require.EqualValues(t, exp, sch)
}

type dummyConverter struct {
}

func (d *dummyConverter) ColumnType(spec *schema.ColumnSpec) (schema.Type, error) {
	return nil, nil
}
