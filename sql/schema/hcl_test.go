package schema

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicSchemaUnmarshal(t *testing.T) {
	schema := getTestSchema(t, "basic_schema.hcl")
	require.EqualValues(t, schema.Name, "todo")

	for _, tt := range []struct {
		table, column string
		expected      *ColumnType
	}{
		{
			table:  "users",
			column: "id",
			expected: &ColumnType{
				Null: false,
				Type: &IntegerType{
					T:            "integer",
					Unsigned:     true,
					StorageBytes: Unspecified,
				},
			},
		},
		{
			table:  "users",
			column: "name",
			expected: &ColumnType{
				Null: true,
				Type: &StringType{
					T:    "string",
					Size: 255,
				},
			},
		},
		{
			table:  "roles",
			column: "id",
			expected: &ColumnType{
				Null: false,
				Type: &IntegerType{
					T: "integer",
				},
			},
		},
		{
			table:  "roles",
			column: "name",
			expected: &ColumnType{
				Type: &StringType{
					T: "string",
				},
			},
		},
		{
			table:  "todos",
			column: "status",
			expected: &ColumnType{
				Type: &EnumType{
					Values: []string{"pending", "in_progress", "done"},
				},
			},
		},
		{
			table:  "todos",
			column: "signature",
			expected: &ColumnType{
				Type: &BinaryType{
					T:    "binary",
					Size: 128,
				},
			},
		},
		{
			table:  "todos",
			column: "visible",
			expected: &ColumnType{
				Type: &BoolType{
					T: "boolean",
				},
			},
		},
		{
			table:  "todos",
			column: "decimal_col",
			expected: &ColumnType{
				Type: &DecimalType{
					T:         "decimal",
					Precision: 2,
					Scale:     5,
				},
			},
		},
		{
			table:  "todos",
			column: "float_col",
			expected: &ColumnType{
				Type: &FloatType{
					T:         "float",
					Precision: 2,
				},
			},
		},
		{
			table:  "todos",
			column: "created",
			expected: &ColumnType{
				Type: &TimeType{
					T: "time",
				},
			},
		},
		{
			table:  "todos",
			column: "json_col",
			expected: &ColumnType{
				Type: &JSONType{
					T: "json",
				},
			},
		},
		{
			table:  "todos",
			column: "storage_bytes",
			expected: &ColumnType{
				Type: &IntegerType{
					T:            "integer",
					StorageBytes: FourBytes,
				},
			},
		},
	} {
		t.Run(fmt.Sprintf("%s_%s", tt.table, tt.column), func(t *testing.T) {
			tbl, ok := schema.Table(tt.table)
			require.Truef(t, ok, "expected table named %q", tt.table)
			col, ok := tbl.Column(tt.column)
			require.Truef(t, ok, "expected column named %q", tt.column)
			require.EqualValues(t, tt.expected, col.Type)
		})
	}
}

func TestDefault(t *testing.T) {
	schema := getTestSchema(t, "defaults.hcl")
	tbl, ok := schema.Table("tasks")
	require.True(t, ok, "expected table tasks")
	require.EqualValues(t, tbl, &Table{
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
	schema := getTestSchema(t, "attributes.hcl")
	tbl, ok := schema.Table("tasks")
	require.True(t, ok, "expected table tasks")
	require.EqualValues(t, tbl, &Table{
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
	schema := getTestSchema(t, "indexes.hcl")
	tbl1, ok := schema.Table("tasks")
	require.True(t, ok, "expected table tasks")
	tbl2, ok := schema.Table("group_vals")
	require.True(t, ok, "expected table group_vals")
	require.EqualValues(t, tbl1.Columns[0], tbl1.PrimaryKey.Parts[0].C)
	require.EqualValues(t, tbl2.Columns[0], tbl2.PrimaryKey.Parts[0].C)
	require.EqualValues(t, tbl2.Columns[1], tbl2.PrimaryKey.Parts[1].C)
}

func TestForeignKey(t *testing.T) {
	schema := getTestSchema(t, "indexes.hcl")
	tasks, ok := schema.Table("tasks")
	require.True(t, ok, "expected table tasks")
	resources, ok := schema.Table("resources")
	require.True(t, ok, "expected table resources")
	require.EqualValues(t, &ForeignKey{
		Symbol:     "resource_task",
		Table:      resources,
		Columns:    []*Column{resources.Columns[1]},
		RefTable:   tasks,
		RefColumns: []*Column{tasks.Columns[0]},
		OnDelete:   Cascade,
	}, resources.ForeignKeys[0])
}

func TestIndex(t *testing.T) {
	schema := getTestSchema(t, "indexes.hcl")
	tasks, ok := schema.Table("tasks")
	require.True(t, ok, "expected table tasks")
	textCol, ok := tasks.Column("text")
	require.True(t, ok)
	require.EqualValues(t, &Index{
		Name:   "idx_text",
		Unique: true,
		Table:  tasks,
		Attrs:  nil,
		Parts: []*IndexPart{
			{
				SeqNo: 0,
				C:     textCol,
			},
		},
	}, tasks.Indexes[0])
}

func TestRewriteHCL(t *testing.T) {
	dir, err := ioutil.ReadDir("testdata/")
	require.NoError(t, err)
	for _, tt := range dir {
		if tt.IsDir() {
			continue
		}
		filename := filepath.Join("testdata", tt.Name())
		t.Run(filename, func(t *testing.T) {
			fb, err := ioutil.ReadFile(filename)
			require.NoError(t, err)
			fromFile, err := UnmarshalHCL(fb, filename)
			require.NoError(t, err)
			out, err := MarshalHCL(fromFile[0])
			require.NoError(t, err)
			generated, err := UnmarshalHCL(out, filename)
			require.NoError(t, err)
			require.EqualValues(t, fromFile, generated)
		})
	}
}

func getTestSchema(t *testing.T, filename string) *Schema {
	path := filepath.Join("testdata", filename)
	bytes, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	schemas, err := UnmarshalHCL(bytes, path)
	require.NoError(t, err)
	require.Len(t, schemas, 1)
	return schemas[0]
}
