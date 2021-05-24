package schema

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicSchemaUnmarshal(t *testing.T) {
	filename := "testdata/basic_schema.hcl"
	bytes, err := ioutil.ReadFile(filename)
	require.NoError(t, err)
	schemas, err := UnmarshalHCL(bytes, filename)
	require.NoError(t, err)
	require.EqualValues(t, schemas[0].Name, "todo")
	tables := schemas[0].Tables
	require.EqualValues(t, tables[0].Name, "users")
	require.EqualValues(t, tables[0].Columns[0].Name, "id")
	require.EqualValues(t, &ColumnType{
		Null: false,
		Type: &IntegerType{
			T:        "integer",
			Size:     1,
			Unsigned: true,
		},
	}, tables[0].Columns[0].Type)
	require.EqualValues(t, tables[0].Columns[1].Name, "name")
	require.EqualValues(t, &ColumnType{
		Null: false,
		Type: &IntegerType{
			T:    "integer",
			Size: 0,
		},
	}, tables[1].Columns[0].Type)
	require.EqualValues(t, tables[1].Name, "roles")
	require.EqualValues(t, tables[1].Columns[0].Name, "id")
	require.EqualValues(t, tables[1].Columns[1].Name, "name")
	require.EqualValues(t, &ColumnType{
		Null: false,
		Type: &StringType{
			T:    "string",
			Size: 0,
		},
	}, tables[1].Columns[1].Type)
	require.EqualValues(t, tables[2].Name, "todos")
	require.EqualValues(t, &EnumType{
		Values: []string{"pending", "in_progress", "done"},
	}, tables[2].Columns[2].Type.Type)
	require.EqualValues(t, &BinaryType{
		T:    "binary",
		Size: 128,
	}, tables[2].Columns[3].Type.Type)
	require.EqualValues(t, &BoolType{
		T: "boolean",
	}, tables[2].Columns[4].Type.Type)
	require.EqualValues(t, &DecimalType{
		T:         "decimal",
		Precision: 2,
		Scale:     5,
	}, tables[2].Columns[5].Type.Type)
	require.EqualValues(t, &FloatType{
		T:         "float",
		Precision: 2,
	}, tables[2].Columns[6].Type.Type)
	require.EqualValues(t, &TimeType{
		T: "time",
	}, tables[2].Columns[7].Type.Type)
	require.EqualValues(t, &TimeType{
		T: "json",
	}, tables[2].Columns[8].Type.Type)
}

func TestDefault(t *testing.T) {
	filename := "testdata/defaults.hcl"
	bytes, err := ioutil.ReadFile(filename)
	require.NoError(t, err)
	schemas, err := UnmarshalHCL(bytes, filename)
	require.NoError(t, err)
	require.EqualValues(t, schemas[0].Tables[0], &Table{
		Name: "tasks",
		Schema: &Schema{
			Name: "todo",
		},
		Columns: []*Column{
			{
				Name: "uuid",
				Type: &ColumnType{
					Type: &StringType{
						T: "string",
					},
				},
				Default: &RawExpr{X: "uuid()"},
			},
			{
				Name: "text",
				Type: &ColumnType{
					Type: &StringType{
						T: "string",
					},
				},
			},
		},
	})
}

func TestAttributes(t *testing.T) {
	filename := "testdata/attributes.hcl"
	bytes, err := ioutil.ReadFile(filename)
	require.NoError(t, err)
	schemas, err := UnmarshalHCL(bytes, filename)
	require.NoError(t, err)
	require.EqualValues(t, schemas[0].Tables[0], &Table{
		Name: "tasks",
		Schema: &Schema{
			Name: "todo",
		},
		Columns: []*Column{
			{
				Name: "text",
				Type: &ColumnType{
					Type: &StringType{
						T: "string",
					},
				},
				Attrs: []Attr{
					&Comment{Text: "comment"},
					&Charset{V: "charset"},
					&Collation{V: "collation"},
				},
			},
		},
	})
}

func TestPrimaryKey(t *testing.T) {
	filename := "testdata/indexes.hcl"
	bytes, err := ioutil.ReadFile(filename)
	require.NoError(t, err)
	schemas, err := UnmarshalHCL(bytes, filename)
	require.NoError(t, err)
	tbl1 := schemas[0].Tables[0]
	tbl2 := schemas[0].Tables[1]
	require.EqualValues(t, tbl1.Columns[0], tbl1.PrimaryKey[0])
	require.EqualValues(t, tbl2.Columns[0], tbl2.PrimaryKey[0])
	require.EqualValues(t, tbl2.Columns[1], tbl2.PrimaryKey[1])
}

func TestForeignKey(t *testing.T) {
	filename := "testdata/indexes.hcl"
	bytes, err := ioutil.ReadFile(filename)
	require.NoError(t, err)
	schemas, err := UnmarshalHCL(bytes, filename)
	require.NoError(t, err)
	tasks := schemas[0].Tables[0]
	resources := schemas[0].Tables[2]
	require.EqualValues(t, &ForeignKey{
		Symbol:     "resource_task",
		Table:      resources,
		Columns:    []*Column{resources.Columns[1]},
		RefTable:   tasks,
		RefColumns: []*Column{tasks.Columns[0]},
		OnDelete:   Cascade,
	}, resources.ForeignKeys[0])
}
