package hclsqlspec

import (
	"strconv"
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"
	"ariga.io/atlas/sql/sqlspec"

	"github.com/stretchr/testify/require"
)

func TestHCL_SQL(t *testing.T) {
	file, err := decode(`
schema "hi" {

}

table "users" {
	schema = schema.hi
	
	column "id" {
		type = int
		null = false
		default = 123
	}
	column "age" {
		type = int
		null = false
		default = 10
	}
	column "active" {
		type = boolean
		default = true
	}

	column "account_active" {
		type = boolean
		default = true
	}

	primary_key {
		columns = [table.users.column.id, table.users.column.age]
	}
	
	index "age" {
		unique = true
		columns = [table.users.column.age]
	}
	index "active" {
		unique = false
		columns = [table.users.column.active]
	}

	foreign_key "fk" {
		columns = [table.users.column.account_active]
		ref_columns = [table.accounts.column.active]
		on_delete = "SET NULL"
	}
}

table "accounts" {
	schema = schema.hi
	
	column "id" {
		type = int
		null = false
		default = 123
	}
	column "age" {
		type = int
		null = false
		default = 10
	}
	column "active" {
		type = boolean
		default = true
	}

	column "user_active" {
		type = boolean
		default = true
	}

	primary_key {
		columns = [table.accounts.column.id]
	}
	
	index "age" {
		unique = true
		columns = [table.accounts.column.age]
	}
	index "active" {
		unique = false
		columns = [table.accounts.column.active]
	}

	foreign_key "fk" {
		columns = [table.accounts.column.user_active]
		ref_columns = [table.users.column.active]
		on_delete = "SET NULL"
	}
}

`)
	require.NoError(t, err)
	expected := &db{
		Schemas: []*sqlspec.Schema{
			{Name: "hi"},
		},
		Tables: []*sqlspec.Table{
			{
				Name:   "users",
				Schema: &schemaspec.Ref{V: "$schema.hi"},
				Columns: []*sqlspec.Column{
					{
						Name:    "id",
						Type:    &schemaspec.Type{T: "int"},
						Null:    false,
						Default: &schemaspec.LiteralValue{V: "123"},
					},
					{
						Name:    "age",
						Type:    &schemaspec.Type{T: "int"},
						Null:    false,
						Default: &schemaspec.LiteralValue{V: "10"},
					},
					{
						Name:    "active",
						Type:    &schemaspec.Type{T: "boolean"},
						Null:    false,
						Default: &schemaspec.LiteralValue{V: "true"},
					},
					{
						Name:    "account_active",
						Type:    &schemaspec.Type{T: "boolean"},
						Null:    false,
						Default: &schemaspec.LiteralValue{V: "true"},
					},
				},
				PrimaryKey: &sqlspec.PrimaryKey{
					Columns: []*schemaspec.Ref{
						{
							V: "$table.users.$column.id",
						},
						{
							V: "$table.users.$column.age",
						},
					},
				},
				Indexes: []*sqlspec.Index{
					{
						Name:   "age",
						Unique: true,
						Columns: []*schemaspec.Ref{
							{
								V: "$table.users.$column.age",
							},
						},
					},
					{
						Name:   "active",
						Unique: false,
						Columns: []*schemaspec.Ref{
							{
								V: "$table.users.$column.active",
							},
						},
					},
				},
				ForeignKeys: []*sqlspec.ForeignKey{
					{
						Symbol: "fk",
						Columns: []*schemaspec.Ref{
							{
								V: "$table.users.$column.account_active",
							},
						},
						RefColumns: []*schemaspec.Ref{
							{
								V: "$table.accounts.$column.active",
							},
						},
						OnDelete: &schemaspec.Ref{V: string(schema.SetNull)},
					},
				},
			},
			{
				Name:   "accounts",
				Schema: &schemaspec.Ref{V: "$schema.hi"},
				Columns: []*sqlspec.Column{
					{
						Name:    "id",
						Type:    &schemaspec.Type{T: "int"},
						Null:    false,
						Default: &schemaspec.LiteralValue{V: "123"},
					},
					{
						Name:    "age",
						Type:    &schemaspec.Type{T: "int"},
						Null:    false,
						Default: &schemaspec.LiteralValue{V: "10"},
					},
					{
						Name:    "active",
						Type:    &schemaspec.Type{T: "boolean"},
						Null:    false,
						Default: &schemaspec.LiteralValue{V: "true"},
					},
					{
						Name:    "user_active",
						Type:    &schemaspec.Type{T: "boolean"},
						Null:    false,
						Default: &schemaspec.LiteralValue{V: "true"},
					},
				},
				PrimaryKey: &sqlspec.PrimaryKey{
					Columns: []*schemaspec.Ref{
						{
							V: "$table.accounts.$column.id",
						},
					},
				},
				Indexes: []*sqlspec.Index{
					{
						Name:   "age",
						Unique: true,
						Columns: []*schemaspec.Ref{
							{
								V: "$table.accounts.$column.age",
							},
						},
					},
					{
						Name:   "active",
						Unique: false,
						Columns: []*schemaspec.Ref{
							{
								V: "$table.accounts.$column.active",
							},
						},
					},
				},
				ForeignKeys: []*sqlspec.ForeignKey{
					{
						Symbol: "fk",
						Columns: []*schemaspec.Ref{
							{
								V: "$table.accounts.$column.user_active",
							},
						},
						RefColumns: []*schemaspec.Ref{
							{
								V: "$table.users.$column.active",
							},
						},
						OnDelete: &schemaspec.Ref{V: string(schema.SetNull)},
					},
				},
			},
		},
	}
	require.EqualValues(t, expected, file)
}

func TestWithRemain(t *testing.T) {
	file, err := decode(`
schema "hi" {
	x = 1
}`)
	require.NoError(t, err)
	require.EqualValues(t, &db{
		Schemas: []*sqlspec.Schema{
			{
				Name: "hi",
				DefaultExtension: schemaspec.DefaultExtension{
					Extra: schemaspec.Resource{
						Attrs: []*schemaspec.Attr{
							{K: "x", V: &schemaspec.LiteralValue{V: "1"}},
						},
					},
				},
			},
		},
	}, file)
}

func TestMultiTable(t *testing.T) {
	_, err := decode(`
schema "hi" {

}

table "users" {
	schema = schema.hi
	column "id" {
		type = int
		unsigned = true
		null = false
		default = 123
	}
}

table "accounts" {
	schema = schema.hi
	column "id" {
		type = varchar(255)
	}
	index "name" {
		unique = true
	}
}

`)
	require.NoError(t, err)
}

func TestMarshalTopLevel(t *testing.T) {
	c := &sqlspec.Schema{
		Name: "schema",
	}
	h, err := hcl.MarshalSpec(c)
	require.NoError(t, err)
	require.EqualValues(t, `schema "schema" {
}
`, string(h))
}

func TestRealm(t *testing.T) {
	f := `schema "account_a" {
}
table "t1" {
	schema = schema.account_a
}
schema "account_b" {
}
table "t2" {
	schema = schema.account_b
}
`
	for _, tt := range []struct {
		name string
		schemaspec.Marshaler
		schemaspec.Unmarshaler
	}{
		{
			name:        "mysql",
			Marshaler:   mysql.MarshalHCL,
			Unmarshaler: mysql.UnmarshalHCL,
		},
		{
			name:        "postgres",
			Marshaler:   postgres.MarshalHCL,
			Unmarshaler: postgres.UnmarshalHCL,
		},
		{
			name:        "sqlite",
			Marshaler:   sqlite.MarshalHCL,
			Unmarshaler: sqlite.UnmarshalHCL,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var r schema.Realm
			err := tt.UnmarshalSpec([]byte(f), &r)
			require.NoError(t, err)
			exp := &schema.Realm{
				Schemas: []*schema.Schema{
					{
						Name: "account_a",
						Tables: []*schema.Table{
							{Name: "t1"},
						},
					},
					{
						Name: "account_b",
						Tables: []*schema.Table{
							{Name: "t2"},
						},
					},
				},
			}
			exp.Schemas[0].Tables[0].Schema = exp.Schemas[0]
			exp.Schemas[1].Tables[0].Schema = exp.Schemas[1]
			require.EqualValues(t, exp, &r)
			hcl, err := tt.MarshalSpec(&r)
			require.NoError(t, err)
			var after schema.Realm
			err = tt.UnmarshalSpec(hcl, &after)
			require.NoError(t, err)
			require.EqualValues(t, exp, &after)
		})
	}
}

func TestUnsignedImmutability(t *testing.T) {
	f := `table "users" {
	column "id" {
		type = bigint
		unsigned = true
	}
	column "shouldnt" {
		type = bigint
	}
}
schema "test" {
}`
	var s schema.Schema
	err := mysql.UnmarshalHCL([]byte(f), &s)
	require.NoError(t, err)
	tbl := s.Tables[0]
	require.EqualValues(t, &schema.IntegerType{T: "bigint", Unsigned: true}, tbl.Columns[0].Type.Type)
	require.EqualValues(t, &schema.IntegerType{T: "bigint", Unsigned: false}, tbl.Columns[1].Type.Type)
}

func decode(f string) (*db, error) {
	d := &db{}
	if err := hcl.UnmarshalSpec([]byte(f), d); err != nil {
		return nil, err
	}
	return d, nil
}

type db struct {
	Schemas []*sqlspec.Schema `spec:"schema"`
	Tables  []*sqlspec.Table  `spec:"table"`
}

func attrs(attrs ...*schemaspec.Attr) []*schemaspec.Attr {
	return attrs
}

func size(n int) *schemaspec.Attr {
	return &schemaspec.Attr{
		K: "size",
		V: &schemaspec.LiteralValue{V: strconv.Itoa(n)},
	}
}

func unsigned(b bool) *schemaspec.Attr {
	return &schemaspec.Attr{
		K: "unsigned",
		V: &schemaspec.LiteralValue{V: strconv.FormatBool(b)},
	}
}
