package mysql

import (
	"fmt"
	"log"
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/stretchr/testify/require"
)

var hclState = schemahcl.New(schemahcl.WithTypes(TypeRegistry.Specs()))

func TestSQLSpec(t *testing.T) {
	f := `
schema "schema" {
}

table "table" {
	column "col" {
		type = int
	}
	column "age" {
		type = int
	}
	column "account_name" {
		type = varchar(32)
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
		type = varchar(32)
	}
	primary_key {
		columns = [table.accounts.column.name]
	}
}
`
	var s schema.Schema
	err := UnmarshalSpec([]byte(f), hclState, &s)
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
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	// Charset and collate that are identical to their parent elements
	// should not be printed as they are inherited by default from it.
	const expected = `table "users" {
  schema = schema.test
  column "a" {
    null      = false
    type      = text
    charset   = "latin1"
    collation = "latin1_swedish_ci"
  }
  column "b" {
    null = false
    type = text
  }
}
table "posts" {
  schema    = schema.test
  charset   = "latin1"
  collation = "latin1_swedish_ci"
  column "a" {
    null = false
    type = text
  }
  column "b" {
    null      = false
    type      = text
    charset   = "utf8mb4"
    collation = "utf8mb4_0900_ai_ci"
  }
}
schema "test" {
  charset   = "utf8mb4"
  collation = "utf8mb4_0900_ai_ci"
}
`
	require.EqualValues(t, expected, string(buf))

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
	require.NoError(t, UnmarshalSpec(buf, hclState, &s2))
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
		{
			typeExpr: "bit(10)",
			expected: &BitType{T: tBit},
		},
		{
			typeExpr: "int(10)",
			expected: &schema.IntegerType{T: tInt},
		},
		{
			typeExpr: "tinyint(10)",
			expected: &schema.IntegerType{T: tTinyInt},
		},
		{
			typeExpr: "smallint(10)",
			expected: &schema.IntegerType{T: tSmallInt},
		},
		{
			typeExpr: "mediumint(10)",
			expected: &schema.IntegerType{T: tMediumInt},
		},
		{
			typeExpr: "bigint(10)",
			expected: &schema.IntegerType{T: tBigInt},
		},
		{
			typeExpr: "decimal",
			expected: &schema.DecimalType{T: tDecimal},
		},
		{
			typeExpr: "numeric",
			expected: &schema.DecimalType{T: tNumeric},
		},
		{
			typeExpr: "float(10,0)",
			expected: &schema.FloatType{T: tFloat, Precision: 10},
		},
		{
			typeExpr: "double(10,0)",
			expected: &schema.FloatType{T: tDouble, Precision: 10},
		},
		{
			typeExpr: "real",
			expected: &schema.FloatType{T: tReal},
		},
		{
			typeExpr: "timestamp",
			expected: &schema.TimeType{T: tTimestamp},
		},
		{
			typeExpr: "date",
			expected: &schema.TimeType{T: tDate},
		},
		{
			typeExpr: "time",
			expected: &schema.TimeType{T: tTime},
		},
		{
			typeExpr: "datetime",
			expected: &schema.TimeType{T: tDateTime},
		},
		{
			typeExpr: "year",
			expected: &schema.TimeType{T: tYear},
		},
		{
			typeExpr: "varchar(10)",
			expected: &schema.StringType{T: tVarchar, Size: 10},
		},
		{
			typeExpr: "char(25)",
			expected: &schema.StringType{T: tChar, Size: 25},
		},
		{
			typeExpr: "varbinary(30)",
			expected: &schema.BinaryType{T: tVarBinary, Size: 30},
		},
		{
			typeExpr: "binary(5)",
			expected: &schema.BinaryType{T: tBinary, Size: 5},
		},
		{
			typeExpr: "blob(5)",
			expected: &schema.StringType{T: tBlob},
		},
		{
			typeExpr: "tinyblob",
			expected: &schema.StringType{T: tTinyBlob},
		},
		{
			typeExpr: "mediumblob",
			expected: &schema.StringType{T: tMediumBlob},
		},
		{
			typeExpr: "longblob",
			expected: &schema.StringType{T: tLongBlob},
		},
		{
			typeExpr: "text(13)",
			expected: &schema.StringType{T: tText},
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
			typeExpr: `enum("a","b")`,
			expected: &schema.EnumType{Values: []string{"a", "b"}},
		},
		{
			typeExpr: `set("a","b")`,
			expected: &SetType{Values: []string{"a", "b"}},
		},
		{
			typeExpr: "geometry",
			expected: &schema.SpatialType{T: tGeometry},
		},
		{
			typeExpr: "point",
			expected: &schema.SpatialType{T: tPoint},
		},
		{
			typeExpr: "multipoint",
			expected: &schema.SpatialType{T: tMultiPoint},
		},
		{
			typeExpr: "linestring",
			expected: &schema.SpatialType{T: tLineString},
		},
		{
			typeExpr: "multilinestring",
			expected: &schema.SpatialType{T: tMultiLineString},
		},
		{
			typeExpr: "polygon",
			expected: &schema.SpatialType{T: tPolygon},
		},
		{
			typeExpr: "multipolygon",
			expected: &schema.SpatialType{T: tMultiPolygon},
		},
		{
			typeExpr: "geometrycollection",
			expected: &schema.SpatialType{T: tGeometryCollection},
		},
	} {
		t.Run(tt.typeExpr, func(t *testing.T) {
			// simulates sqlspec.Column until we change its Type field.
			type col struct {
				Type *schemaspec.Type `spec:"type"`
				schemaspec.DefaultExtension
			}
			var test struct {
				Columns []*col `spec:"column"`
			}
			doc := fmt.Sprintf(`column {
	type = %s%s
}
`, tt.typeExpr, lineIfSet(tt.extraAttr))
			err := hclState.UnmarshalSpec([]byte(doc), &test)
			require.NoError(t, err)
			column := test.Columns[0]
			typ, err := TypeRegistry.Type(column.Type, column.Extra.Attrs, parseRawType)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, typ)
			spec, err := hclState.MarshalSpec(&test)
			require.NoError(t, err)
			hclEqual(t, []byte(doc), spec)
		})
	}
}

func hclEqual(t *testing.T, expected, actual []byte) {
	require.EqualValues(t, string(hclwrite.Format(expected)), string(hclwrite.Format(actual)))
}

func lineIfSet(s string) string {
	if s != "" {
		return "\n" + s
	}
	return s
}
