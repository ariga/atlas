package mysql

import (
	"fmt"
	"testing"

	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/require"
)

func TestSQLSpec(t *testing.T) {
	f := `
schema "schema" {
}

table "table" {
	column "col" {
		type = int
		comment = "column comment"
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
		comment = "index comment"
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
	comment = "table comment"
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
	err := UnmarshalHCL([]byte(f), &s)
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
							T: TypeInt,
						},
					},
					Attrs: []schema.Attr{
						&schema.Comment{Text: "column comment"},
					},
				},
				{
					Name: "age",
					Type: &schema.ColumnType{
						Type: &schema.IntegerType{
							T: TypeInt,
						},
					},
				},
				{
					Name: "account_name",
					Type: &schema.ColumnType{
						Type: &schema.StringType{
							T:    TypeVarchar,
							Size: 32,
						},
					},
				},
			},
			Attrs: []schema.Attr{
				&schema.Comment{Text: "table comment"},
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
							T:    TypeVarchar,
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
			Attrs: []schema.Attr{
				&schema.Comment{Text: "index comment"},
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
	require.NoError(t, UnmarshalHCL(buf, &s2))
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

func TestMarshalSpec_Comment(t *testing.T) {
	s := &schema.Schema{
		Name: "test",
		Tables: []*schema.Table{
			{
				Name: "users",
				Attrs: []schema.Attr{
					&schema.Comment{Text: "table comment"},
				},
				Columns: []*schema.Column{
					{
						Name: "a",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}},
						Attrs: []schema.Attr{
							&schema.Comment{Text: "column comment"},
						},
					},
				},
			},
			{
				Name: "posts",
				Columns: []*schema.Column{
					{
						Name: "a",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}},
					},
				},
			},
		},
	}
	s.Tables[0].Schema = s
	s.Tables[1].Schema = s
	s.Tables[0].Indexes = []*schema.Index{
		{
			Name:   "index",
			Table:  s.Tables[0],
			Unique: true,
			Parts:  []*schema.IndexPart{{SeqNo: 0, C: s.Tables[0].Columns[0]}},
			Attrs: []schema.Attr{
				&schema.Comment{Text: "index comment"},
			},
		},
	}
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	// We expect a zero value comment to not be present in the marshaled HCL.
	const expected = `table "users" {
  schema  = schema.test
  comment = "table comment"
  column "a" {
    null    = false
    type    = text
    comment = "column comment"
  }
  index "index" {
    unique  = true
    columns = [table.users.column.a, ]
    comment = "index comment"
  }
}
table "posts" {
  schema = schema.test
  column "a" {
    null = false
    type = text
  }
}
schema "test" {
}
`
	require.EqualValues(t, expected, string(buf))
}

func TestTypes(t *testing.T) {
	tests := []struct {
		typeExpr  string
		extraAttr string
		expected  schema.Type
	}{
		{
			typeExpr: "varchar(255)",
			expected: &schema.StringType{T: TypeVarchar, Size: 255},
		},
		{
			typeExpr: "char(255)",
			expected: &schema.StringType{T: TypeChar, Size: 255},
		},
		{
			typeExpr: `sql("custom")`,
			expected: &schema.UnsupportedType{T: "custom"},
		},
		{
			typeExpr: "binary(255)",
			expected: &schema.BinaryType{T: TypeBinary, Size: 255},
		},
		{
			typeExpr: "varbinary(255)",
			expected: &schema.BinaryType{T: TypeVarBinary, Size: 255},
		},
		{
			typeExpr: "int",
			expected: &schema.IntegerType{T: TypeInt},
		},
		{
			typeExpr:  "int",
			extraAttr: "unsigned=true",
			expected:  &schema.IntegerType{T: TypeInt, Unsigned: true},
		},
		{
			typeExpr: "int",
			expected: &schema.IntegerType{T: TypeInt},
		},
		{
			typeExpr: "bigint",
			expected: &schema.IntegerType{T: TypeBigInt},
		},
		{
			typeExpr:  "bigint",
			extraAttr: "unsigned=true",
			expected:  &schema.IntegerType{T: TypeBigInt, Unsigned: true},
		},
		{
			typeExpr: "tinyint",
			expected: &schema.IntegerType{T: TypeTinyInt},
		},
		{
			typeExpr:  "tinyint",
			extraAttr: "unsigned=true",
			expected:  &schema.IntegerType{T: TypeTinyInt, Unsigned: true},
		},
		{
			typeExpr: "smallint",
			expected: &schema.IntegerType{T: TypeSmallInt},
		},
		{
			typeExpr:  "smallint",
			extraAttr: "unsigned=true",
			expected:  &schema.IntegerType{T: TypeSmallInt, Unsigned: true},
		},
		{
			typeExpr: "mediumint",
			expected: &schema.IntegerType{T: TypeMediumInt},
		},
		{
			typeExpr:  "mediumint",
			extraAttr: "unsigned=true",
			expected:  &schema.IntegerType{T: TypeMediumInt, Unsigned: true},
		},
		{
			typeExpr: "tinytext",
			expected: &schema.StringType{T: TypeTinyText},
		},
		{
			typeExpr: "mediumtext",
			expected: &schema.StringType{T: TypeMediumText},
		},
		{
			typeExpr: "longtext",
			expected: &schema.StringType{T: TypeLongText},
		},
		{
			typeExpr: "text",
			expected: &schema.StringType{T: TypeText},
		},
		{
			typeExpr: `enum("on","off")`,
			expected: &schema.EnumType{T: TypeEnum, Values: []string{"on", "off"}},
		},
		{
			typeExpr: "bit(10)",
			expected: &BitType{T: TypeBit},
		},
		{
			typeExpr: "int(10)",
			expected: &schema.IntegerType{T: TypeInt},
		},
		{
			typeExpr: "tinyint(10)",
			expected: &schema.IntegerType{T: TypeTinyInt},
		},
		{
			typeExpr: "smallint(10)",
			expected: &schema.IntegerType{T: TypeSmallInt},
		},
		{
			typeExpr: "mediumint(10)",
			expected: &schema.IntegerType{T: TypeMediumInt},
		},
		{
			typeExpr: "bigint(10)",
			expected: &schema.IntegerType{T: TypeBigInt},
		},
		{
			typeExpr: "decimal",
			expected: &schema.DecimalType{T: TypeDecimal},
		},
		{
			typeExpr: "decimal(10)",
			expected: &schema.DecimalType{T: TypeDecimal, Precision: 10},
		},
		{
			typeExpr: "decimal(10,2)",
			expected: &schema.DecimalType{T: TypeDecimal, Precision: 10, Scale: 2},
		},
		{
			typeExpr: "numeric",
			expected: &schema.DecimalType{T: TypeNumeric},
		},
		{
			typeExpr: "numeric(10)",
			expected: &schema.DecimalType{T: TypeNumeric, Precision: 10},
		},
		{
			typeExpr: "numeric(10,2)",
			expected: &schema.DecimalType{T: TypeNumeric, Precision: 10, Scale: 2},
		},
		{
			typeExpr: "float(10,0)",
			expected: &schema.FloatType{T: TypeFloat, Precision: 10},
		},
		{
			typeExpr: "double(10,0)",
			expected: &schema.FloatType{T: TypeDouble, Precision: 10},
		},
		{
			typeExpr: "real",
			expected: &schema.FloatType{T: TypeReal},
		},
		{
			typeExpr: "timestamp",
			expected: &schema.TimeType{T: TypeTimestamp},
		},
		{
			typeExpr: "date",
			expected: &schema.TimeType{T: TypeDate},
		},
		{
			typeExpr: "time",
			expected: &schema.TimeType{T: TypeTime},
		},
		{
			typeExpr: "datetime",
			expected: &schema.TimeType{T: TypeDateTime},
		},
		{
			typeExpr: "year",
			expected: &schema.TimeType{T: TypeYear},
		},
		{
			typeExpr: "varchar(10)",
			expected: &schema.StringType{T: TypeVarchar, Size: 10},
		},
		{
			typeExpr: "char(25)",
			expected: &schema.StringType{T: TypeChar, Size: 25},
		},
		{
			typeExpr: "varbinary(30)",
			expected: &schema.BinaryType{T: TypeVarBinary, Size: 30},
		},
		{
			typeExpr: "binary(5)",
			expected: &schema.BinaryType{T: TypeBinary, Size: 5},
		},
		{
			typeExpr: "blob(5)",
			expected: &schema.StringType{T: TypeBlob},
		},
		{
			typeExpr: "tinyblob",
			expected: &schema.StringType{T: TypeTinyBlob},
		},
		{
			typeExpr: "mediumblob",
			expected: &schema.StringType{T: TypeMediumBlob},
		},
		{
			typeExpr: "longblob",
			expected: &schema.StringType{T: TypeLongBlob},
		},
		{
			typeExpr: "json",
			expected: &schema.JSONType{T: TypeJSON},
		},
		{
			typeExpr: "text(13)",
			expected: &schema.StringType{T: TypeText},
		},
		{
			typeExpr: "tinytext",
			expected: &schema.StringType{T: TypeTinyText},
		},
		{
			typeExpr: "mediumtext",
			expected: &schema.StringType{T: TypeMediumText},
		},
		{
			typeExpr: "longtext",
			expected: &schema.StringType{T: TypeLongText},
		},
		{
			typeExpr: `enum("a","b")`,
			expected: &schema.EnumType{T: "enum", Values: []string{"a", "b"}},
		},
		{
			typeExpr: `set("a","b")`,
			expected: &SetType{Values: []string{"a", "b"}},
		},
		{
			typeExpr: "geometry",
			expected: &schema.SpatialType{T: TypeGeometry},
		},
		{
			typeExpr: "point",
			expected: &schema.SpatialType{T: TypePoint},
		},
		{
			typeExpr: "multipoint",
			expected: &schema.SpatialType{T: TypeMultiPoint},
		},
		{
			typeExpr: "linestring",
			expected: &schema.SpatialType{T: TypeLineString},
		},
		{
			typeExpr: "multilinestring",
			expected: &schema.SpatialType{T: TypeMultiLineString},
		},
		{
			typeExpr: "polygon",
			expected: &schema.SpatialType{T: TypePolygon},
		},
		{
			typeExpr: "multipolygon",
			expected: &schema.SpatialType{T: TypeMultiPolygon},
		},
		{
			typeExpr: "geometrycollection",
			expected: &schema.SpatialType{T: TypeGeometryCollection},
		},
	}
	for _, tt := range tests {
		t.Run(tt.typeExpr, func(t *testing.T) {
			doc := fmt.Sprintf(`table "test" {
	schema = schema.test
	column "test" {
		null = false
		type = %s%s
	}
}
schema "test" {
}
`, tt.typeExpr, lineIfSet(tt.extraAttr))
			var test schema.Schema
			err := UnmarshalHCL([]byte(doc), &test)
			require.NoError(t, err)
			colspec := test.Tables[0].Columns[0]
			require.EqualValues(t, tt.expected, colspec.Type.Type)
			spec, err := MarshalHCL(&test)
			require.NoError(t, err)
			var after schema.Schema
			err = UnmarshalHCL(spec, &after)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, after.Tables[0].Columns[0].Type.Type)
		})
	}
}

func lineIfSet(s string) string {
	if s != "" {
		return "\n" + s
	}
	return s
}
