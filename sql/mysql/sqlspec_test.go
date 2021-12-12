package mysql

import (
	"fmt"
	"log"
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"

	"github.com/stretchr/testify/require"
)

var (
	hclState = schemahcl.New(schemahcl.WithTypes(TypeRegistry.Specs()))
)

func TestSQLSpec(t *testing.T) {
	f := `
schema "schema" {
}

table "table" {
	column "col" {
		type = "int"
	}
	column "age" {
		type = "int"
	}
	column "account_name" {
		type = "string"
		size = 32
	}
	primary_key {
		columns = [table.table.column.col]
	}
	index "index" {
		unique = true
		columns = [
			table.table.column.col,
			table.table.column.age,
		]
	}
	foreign_key "accounts" {
		columns = [
			table.table.column.account_name,
		]
		ref_columns = [
			table.accounts.column.name,
		]
		on_delete = "SET NULL"
	}
}

table "accounts" {
	column "name" {
		type = "string"
		size = 32
	}
	primary_key {
		columns = [table.accounts.column.name]
	}
}
`
	var s schema.Schema
	err := UnmarshalSpec([]byte(f), schemahcl.Unmarshal, &s)
	require.NoError(t, err)

	exp := &schema.Schema{
		Name: "schema",
	}
	exp.Tables = []*schema.Table{
		{
			Name:   "table",
			Schema: exp,
			Columns: []*schema.Column{
				{
					Name: "col",
					Type: &schema.ColumnType{
						Type: &schema.IntegerType{
							T: tInt,
						},
					},
				},
				{
					Name: "age",
					Type: &schema.ColumnType{
						Type: &schema.IntegerType{
							T: tInt,
						},
					},
				},
				{
					Name: "account_name",
					Type: &schema.ColumnType{
						Type: &schema.StringType{
							T:    tVarchar,
							Size: 32,
						},
					},
				},
			},
		},
		{
			Name:   "accounts",
			Schema: exp,
			Columns: []*schema.Column{
				{
					Name: "name",
					Type: &schema.ColumnType{
						Type: &schema.StringType{
							T:    tVarchar,
							Size: 32,
						},
					},
				},
			},
		},
	}
	exp.Tables[0].PrimaryKey = &schema.Index{
		Table: exp.Tables[0],
		Parts: []*schema.IndexPart{
			{SeqNo: 0, C: exp.Tables[0].Columns[0]},
		},
	}
	exp.Tables[0].Indexes = []*schema.Index{
		{
			Name:   "index",
			Table:  exp.Tables[0],
			Unique: true,
			Parts: []*schema.IndexPart{
				{SeqNo: 0, C: exp.Tables[0].Columns[0]},
				{SeqNo: 1, C: exp.Tables[0].Columns[1]},
			},
		},
	}
	exp.Tables[0].ForeignKeys = []*schema.ForeignKey{
		{
			Symbol:     "accounts",
			Table:      exp.Tables[0],
			Columns:    []*schema.Column{exp.Tables[0].Columns[2]},
			RefTable:   exp.Tables[1],
			RefColumns: []*schema.Column{exp.Tables[1].Columns[0]},
			OnDelete:   schema.SetNull,
		},
	}
	exp.Tables[1].PrimaryKey = &schema.Index{
		Table: exp.Tables[1],
		Parts: []*schema.IndexPart{
			{SeqNo: 0, C: exp.Tables[1].Columns[0]},
		},
	}
	require.EqualValues(t, exp, &s)
}

func TestMarshalSpec_Charset(t *testing.T) {
	s := &schema.Schema{
		Name: "test",
		Attrs: []schema.Attr{
			&schema.Charset{V: "utf8mb4"},
			&schema.Collation{V: "utf8mb4_0900_ai_ci"},
		},
		Tables: []*schema.Table{
			{
				Name: "users",
				Attrs: []schema.Attr{
					&schema.Charset{V: "utf8mb4"},
					&schema.Collation{V: "utf8mb4_0900_ai_ci"},
				},
				Columns: []*schema.Column{
					{
						Name: "a",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_swedish_ci"},
						},
					},
					{
						Name: "b",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}},
						Attrs: []schema.Attr{
							&schema.Charset{V: "utf8mb4"},
							&schema.Collation{V: "utf8mb4_0900_ai_ci"},
						},
					},
				},
			},
			{
				Name: "posts",
				Attrs: []schema.Attr{
					&schema.Charset{V: "latin1"},
					&schema.Collation{V: "latin1_swedish_ci"},
				},
				Columns: []*schema.Column{
					{
						Name: "a",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_swedish_ci"},
						},
					},
					{
						Name: "b",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}},
						Attrs: []schema.Attr{
							&schema.Charset{V: "utf8mb4"},
							&schema.Collation{V: "utf8mb4_0900_ai_ci"},
						},
					},
				},
			},
		},
	}
	s.Tables[0].Schema = s
	s.Tables[1].Schema = s
	buf, err := MarshalSpec(s, schemahcl.Marshal)
	require.NoError(t, err)
	// Charset and collate that are identical to their parent elements
	// should not be printed as they are inherited by default from it.
	const expected = `table "users" {
  schema = schema.test
  column "a" {
    null      = false
    type      = "text"
    charset   = "latin1"
    collation = "latin1_swedish_ci"
  }
  column "b" {
    null = false
    type = "text"
  }
}
table "posts" {
  schema    = schema.test
  charset   = "latin1"
  collation = "latin1_swedish_ci"
  column "a" {
    null = false
    type = "text"
  }
  column "b" {
    null      = false
    type      = "text"
    charset   = "utf8mb4"
    collation = "utf8mb4_0900_ai_ci"
  }
}
schema "test" {
  charset   = "utf8mb4"
  collation = "utf8mb4_0900_ai_ci"
}
`
	require.EqualValues(t, expected, buf)

	var (
		s2    schema.Schema
		latin = []schema.Attr{
			&schema.Charset{V: "latin1"},
			&schema.Collation{V: "latin1_swedish_ci"},
		}
		utf8mb4 = []schema.Attr{
			&schema.Charset{V: "utf8mb4"},
			&schema.Collation{V: "utf8mb4_0900_ai_ci"},
		}
	)
	require.NoError(t, UnmarshalSpec(buf, schemahcl.Unmarshal, &s2))
	require.Equal(t, utf8mb4, s2.Attrs)
	posts, ok := s2.Table("posts")
	require.True(t, ok)
	require.Equal(t, latin, posts.Attrs)
	users, ok := s2.Table("users")
	require.True(t, ok)
	require.Empty(t, users.Attrs)
	a, ok := users.Column("a")
	require.True(t, ok)
	require.Equal(t, latin, a.Attrs)
	b, ok := posts.Column("b")
	require.True(t, ok)
	require.Equal(t, utf8mb4, b.Attrs)
}

func TestUnmarshalSpecColumnTypes(t *testing.T) {
	for _, tt := range []struct {
		spec     *sqlspec.Column
		expected schema.Type
	}{
		{
			spec: specutil.NewCol("int", "int"),
			expected: &schema.IntegerType{
				T:        tInt,
				Unsigned: false,
			},
		},
		{
			spec: specutil.NewCol("uint", "uint"),
			expected: &schema.IntegerType{
				T:        tInt,
				Unsigned: true,
			},
		},
		{
			spec: specutil.NewCol("int8", "int8"),
			expected: &schema.IntegerType{
				T:        tTinyInt,
				Unsigned: false,
			},
		},
		{
			spec: specutil.NewCol("int64", "int64"),
			expected: &schema.IntegerType{
				T:        tBigInt,
				Unsigned: false,
			},
		},
		{
			spec: specutil.NewCol("uint64", "uint64"),
			expected: &schema.IntegerType{
				T:        tBigInt,
				Unsigned: true,
			},
		},
		{
			spec: specutil.NewCol("string_varchar", "string", specutil.LitAttr("size", "255")),
			expected: &schema.StringType{
				T:    tVarchar,
				Size: 255,
			},
		},
		{
			spec: specutil.NewCol("string_mediumtext", "string", specutil.LitAttr("size", "100000")),
			expected: &schema.StringType{
				T:    tMediumText,
				Size: 100_000,
			},
		},
		{
			spec: specutil.NewCol("string_longtext", "string", specutil.LitAttr("size", "17000000")),
			expected: &schema.StringType{
				T:    tLongText,
				Size: 17_000_000,
			},
		},
		{
			spec: specutil.NewCol("varchar(255)", "varchar(255)"),
			expected: &schema.StringType{
				T:    tVarchar,
				Size: 255,
			},
		},
		{
			spec: specutil.NewCol("decimal(10, 2) unsigned", "decimal(10, 2) unsigned"),
			expected: &schema.DecimalType{
				T:         tDecimal,
				Scale:     2,
				Precision: 10,
			},
		},
		{
			spec: specutil.NewCol("blob", "binary"),
			expected: &schema.BinaryType{
				T: tBlob,
			},
		},
		{
			spec: specutil.NewCol("tinyblob", "binary", specutil.LitAttr("size", "16")),
			expected: &schema.BinaryType{
				T:    tTinyBlob,
				Size: 16,
			},
		},
		{
			spec: specutil.NewCol("mediumblob", "binary", specutil.LitAttr("size", "100000")),
			expected: &schema.BinaryType{
				T:    tMediumBlob,
				Size: 100_000,
			},
		},
		{
			spec: specutil.NewCol("longblob", "binary", specutil.LitAttr("size", "20000000")),
			expected: &schema.BinaryType{
				T:    tLongBlob,
				Size: 20_000_000,
			},
		},
		{
			spec:     specutil.NewCol("enum", "enum", specutil.ListAttr("values", `"a"`, `"b"`, `"c"`)),
			expected: &schema.EnumType{Values: []string{"a", "b", "c"}},
		},
		{
			spec:     specutil.NewCol("bool", "boolean"),
			expected: &schema.BoolType{T: "boolean"},
		},
		{
			spec:     specutil.NewCol("decimal", "decimal", specutil.LitAttr("precision", "10"), specutil.LitAttr("scale", "2")),
			expected: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2},
		},
		{
			spec:     specutil.NewCol("float", "float", specutil.LitAttr("precision", "10")),
			expected: &schema.FloatType{T: "float", Precision: 10},
		},
		{
			spec:     specutil.NewCol("float", "float", specutil.LitAttr("precision", "25")),
			expected: &schema.FloatType{T: "double", Precision: 25},
		},
	} {
		t.Run(tt.spec.Name, func(t *testing.T) {
			var s schema.Schema
			err := UnmarshalSpec(hcl(tt.spec), schemahcl.Unmarshal, &s)
			require.NoError(t, err)
			tbl, ok := s.Table("table")
			require.True(t, ok)
			col, ok := tbl.Column(tt.spec.Name)
			require.True(t, ok)
			require.EqualValues(t, tt.expected, col.Type.Type)
		})
	}
}

// hcl returns an Atlas HCL document containing the column spec.
func hcl(c *sqlspec.Column) []byte {
	mm, err := schemahcl.Marshal(c)
	if err != nil {
		log.Fatalln(err)
	}
	tmpl := `
schema "default" {
}
table "table" {
	schema = schema.default
	%s
}
`
	body := fmt.Sprintf(tmpl, string(mm))
	return []byte(body)
}

func TestMarshalSpecColumnType(t *testing.T) {
	for _, tt := range []struct {
		schem    schema.Type
		expected *sqlspec.Column
	}{
		{
			schem: &schema.IntegerType{
				T:        tInt,
				Unsigned: false,
			},
			expected: specutil.NewCol("column", "int"),
		},
		{
			schem: &schema.IntegerType{
				T:        tInt,
				Unsigned: true,
			},
			expected: specutil.NewCol("column", "uint"),
		},
		{
			schem: &schema.IntegerType{
				T:        tTinyInt,
				Unsigned: false,
			},
			expected: specutil.NewCol("column", "int8"),
		},
		{
			schem: &schema.IntegerType{
				T:        tMediumInt,
				Unsigned: false,
			},
			expected: specutil.NewCol("column", tMediumInt),
		},
		{
			schem: &schema.IntegerType{
				T:        tSmallInt,
				Unsigned: false,
			},
			expected: specutil.NewCol("column", tSmallInt),
		},
		{
			schem: &schema.IntegerType{
				T:        tBigInt,
				Unsigned: false,
			},
			expected: specutil.NewCol("column", "int64"),
		},
		{
			schem: &schema.IntegerType{
				T:        tBigInt,
				Unsigned: true,
			},
			expected: specutil.NewCol("column", "uint64"),
		},
		{
			schem: &schema.StringType{
				T:    tVarchar,
				Size: 255,
			},
			expected: specutil.NewCol("column", "string", specutil.LitAttr("size", "255")),
		},
		{
			schem: &schema.StringType{
				T:    tTinyText,
				Size: 255,
			},
			expected: specutil.NewCol("column", "string", specutil.LitAttr("size", "255")),
		},
		{
			schem: &schema.StringType{
				T:    tText,
				Size: 255,
			},
			expected: specutil.NewCol("column", "string", specutil.LitAttr("size", "255")),
		},
		{
			schem: &schema.StringType{
				T:    tChar,
				Size: 255,
			},
			expected: specutil.NewCol("column", "string", specutil.LitAttr("size", "255")),
		},
		{
			schem: &schema.StringType{
				T:    tMediumText,
				Size: 100_000,
			},
			expected: specutil.NewCol("column", "string", specutil.LitAttr("size", "100000")),
		},
		{
			schem: &schema.StringType{
				T:    tLongText,
				Size: 17_000_000,
			},
			expected: specutil.NewCol("column", "string", specutil.LitAttr("size", "17000000")),
		},
		{
			schem:    &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2},
			expected: specutil.NewCol("column", "decimal", specutil.LitAttr("precision", "10"), specutil.LitAttr("scale", "2")),
		},
		{
			schem: &schema.BinaryType{
				T: tBlob,
			},
			expected: specutil.NewCol("column", "binary"),
		},
		{
			schem: &schema.BinaryType{
				T:    tTinyBlob,
				Size: 16,
			},
			expected: specutil.NewCol("column", "binary", specutil.LitAttr("size", "16")),
		},
		{
			schem: &schema.BinaryType{
				T:    tMediumBlob,
				Size: 100_000,
			},
			expected: specutil.NewCol("column", "binary", specutil.LitAttr("size", "100000")),
		},
		{
			schem: &schema.BinaryType{
				T:    tLongBlob,
				Size: 20_000_000,
			},
			expected: specutil.NewCol("column", "binary", specutil.LitAttr("size", "20000000")),
		},
		{
			schem:    &schema.EnumType{Values: []string{"a", "b", "c"}},
			expected: specutil.NewCol("column", "enum", specutil.ListAttr("values", `"a"`, `"b"`, `"c"`)),
		},
		{
			schem:    &schema.BoolType{T: "boolean"},
			expected: specutil.NewCol("column", "boolean"),
		},
		{
			schem:    &schema.FloatType{T: "float", Precision: 10},
			expected: specutil.NewCol("column", "float", specutil.LitAttr("precision", "10")),
		},
		{
			schem:    &schema.FloatType{T: "double", Precision: 25},
			expected: specutil.NewCol("column", "float", specutil.LitAttr("precision", "25")),
		},
		{
			schem:    &schema.TimeType{T: "date"},
			expected: specutil.NewCol("column", "date"),
		},
		{
			schem:    &schema.TimeType{T: "datetime"},
			expected: specutil.NewCol("column", "datetime"),
		},
		{
			schem:    &schema.TimeType{T: "time"},
			expected: specutil.NewCol("column", "time"),
		},
		{
			schem:    &schema.TimeType{T: "timestamp"},
			expected: specutil.NewCol("column", "timestamp"),
		},
		{
			schem:    &schema.TimeType{T: "year"},
			expected: specutil.NewCol("column", "year"),
		},
		{
			schem:    &schema.TimeType{T: "year(4)"},
			expected: specutil.NewCol("column", "year(4)"),
		},
	} {
		t.Run(tt.expected.Type, func(t *testing.T) {
			s := schema.Schema{
				Tables: []*schema.Table{
					{
						Name: "table",
						Columns: []*schema.Column{
							{
								Name: "column",
								Type: &schema.ColumnType{Type: tt.schem},
							},
						},
					},
				},
			}
			s.Tables[0].Schema = &s
			ddl, err := MarshalSpec(&s, schemahcl.Marshal)
			require.NoError(t, err)
			var test struct {
				Table *sqlspec.Table `spec:"table"`
			}
			err = schemahcl.Unmarshal(ddl, &test)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected.Type, test.Table.Columns[0].Type)
			require.ElementsMatch(t, tt.expected.Extra.Attrs, test.Table.Columns[0].Extra.Attrs)
		})
	}
}

func TestTypes(t *testing.T) {

	for _, tt := range []struct {
		typeExpr  string
		extraAttr string
		expected  schema.Type
	}{
		{
			typeExpr: "varchar(255)",
			expected: &schema.StringType{T: tVarchar, Size: 255},
		},
		{
			typeExpr: "char(255)",
			expected: &schema.StringType{T: tChar, Size: 255},
		},
		{
			typeExpr: "binary(255)",
			expected: &schema.BinaryType{T: tBinary, Size: 255},
		},
		{
			typeExpr: "varbinary(255)",
			expected: &schema.BinaryType{T: tVarBinary, Size: 255},
		},
		{
			typeExpr: "int",
			expected: &schema.IntegerType{T: tInt},
		},
		{
			typeExpr:  "int",
			extraAttr: "unsigned=true",
			expected:  &schema.IntegerType{T: tInt, Unsigned: true},
		},
		{
			typeExpr: "bigint",
			expected: &schema.IntegerType{T: tBigInt},
		},
		{
			typeExpr:  "bigint",
			extraAttr: "unsigned=true",
			expected:  &schema.IntegerType{T: tBigInt, Unsigned: true},
		},
		{
			typeExpr: "tinyint",
			expected: &schema.IntegerType{T: tTinyInt},
		},
		{
			typeExpr:  "tinyint",
			extraAttr: "unsigned=true",
			expected:  &schema.IntegerType{T: tTinyInt, Unsigned: true},
		},
		{
			typeExpr: "smallint",
			expected: &schema.IntegerType{T: tSmallInt},
		},
		{
			typeExpr:  "smallint",
			extraAttr: "unsigned=true",
			expected:  &schema.IntegerType{T: tSmallInt, Unsigned: true},
		},
		{
			typeExpr: "mediumint",
			expected: &schema.IntegerType{T: tMediumInt},
		},
		{
			typeExpr:  "mediumint",
			extraAttr: "unsigned=true",
			expected:  &schema.IntegerType{T: tMediumInt, Unsigned: true},
		},
		{
			typeExpr: "tinytext",
			expected: &schema.StringType{T: tTinyText},
		},
		{
			typeExpr: "mediumtext",
			expected: &schema.StringType{T: tMediumText},
		},
		{
			typeExpr: "longtext",
			expected: &schema.StringType{T: tLongText},
		},
		{
			typeExpr: "text",
			expected: &schema.StringType{T: tText},
		},
		{
			typeExpr: `enum("on","off")`,
			expected: &schema.EnumType{Values: []string{"on", "off"}},
		},
	} {
		t.Run(tt.typeExpr, func(t *testing.T) {
			type col struct {
				Type *schemaspec.Type `spec:"type"`
				schemaspec.DefaultExtension
			}
			var test struct {
				Columns []*col `spec:"column"`
			}
			doc := fmt.Sprintf(`
column "test" {
	type = %s
	%s
}
`, tt.typeExpr, tt.extraAttr)
			err := hclState.UnmarshalSpec([]byte(doc), &test)
			require.NoError(t, err)
			col := test.Columns[0]
			convertColumnType()
			//err := UnmarshalSpec([]byte(doc), hclState, &test)
			//require.NoError(t, err)
			//table, ok := test.Table("test")
			//require.True(t, ok)
			//column, ok := table.Column("test")
			//require.True(t, ok)
			//require.EqualValues(t, tt.expected, column.Type.Type)
			//spec, err := MarshalSpec(&test, hclState)
			//require.NoError(t, err)
			//var after schema.Schema
			//err = UnmarshalSpec(spec, hclState, &after)
			//require.NoError(t, err)
			//require.EqualValues(t, test, after)
		})
	}
}
