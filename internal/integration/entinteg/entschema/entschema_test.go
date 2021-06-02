package entschema

import (
	"testing"

	"ariga.io/atlas/sql/schema"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/stretchr/testify/require"
)

func TestConvert(t *testing.T) {
	graph, err := entc.LoadGraph("../ent/schema", &gen.Config{})
	require.NoError(t, err)
	sch, err := Convert(graph)
	require.NoError(t, err)
	users, ok := sch.Table("users")
	require.True(t, ok, "expected users table to exist")
	require.EqualValues(t, "users", users.Name)

	// column types
	for _, tt := range []struct {
		fld      string
		expected *schema.ColumnType
	}{
		{
			fld: "name",
			expected: &schema.ColumnType{
				Type: &schema.StringType{
					T: "string",
				},
			},
		},
		{
			fld: "optional",
			expected: &schema.ColumnType{
				Type: &schema.StringType{
					T: "string",
				},
				Null: true,
			},
		},
		{
			fld: "int",
			expected: &schema.ColumnType{
				Type: &schema.IntegerType{
					T: "integer",
				},
			},
		},
		{
			fld: "uint",
			expected: &schema.ColumnType{
				Type: &schema.IntegerType{
					T:        "integer",
					Unsigned: true,
				},
			},
		},
		{
			fld: "time",
			expected: &schema.ColumnType{
				Type: &schema.TimeType{T: "time"},
			},
		},
		{
			fld: "bool",
			expected: &schema.ColumnType{
				Type: &schema.BoolType{T: "boolean"},
			},
		},
		{
			fld: "enum",
			expected: &schema.ColumnType{
				Type: &schema.EnumType{Values: []string{"1", "2", "3"}},
			},
		},
		{
			fld: "enum_2",
			expected: &schema.ColumnType{
				Type: &schema.EnumType{Values: []string{"1", "2", "3"}},
			},
		},
		{
			fld: "uuid",
			expected: &schema.ColumnType{
				Type: &schema.BinaryType{
					T:    "binary",
					Size: 16,
				},
			},
		},
		{
			fld: "bytes",
			expected: &schema.ColumnType{
				Type: &schema.BinaryType{
					T: "binary",
				},
			},
		},
		{
			fld: "group_id",
			expected: &schema.ColumnType{
				Type: &schema.IntegerType{
					T: "integer",
				},
				Null: true,
			},
		},
	} {
		column, ok := users.Column(tt.fld)
		require.True(t, ok, "expected column to exist")
		require.EqualValues(t, tt.expected, column.Type)
	}

	// primary key
	id, ok := users.Column("id")
	require.True(t, ok, "expected col id to exist")
	require.EqualValues(t, &schema.Index{
		Parts: []*schema.IndexPart{
			{C: id, SeqNo: 0},
		},
	}, users.PrimaryKey)

	// foreign key
	gid, ok := users.Column("group_id")
	require.True(t, ok, "expected column group_id")
	fk, ok := users.ForeignKey("users_groups_group")
	groups, ok := sch.Table("groups")
	require.True(t, ok, "expected table groups")
	refcol, ok := groups.Column("id")
	require.True(t, ok, "expected column id")
	require.EqualValues(t, &schema.ForeignKey{
		Symbol:     "users_groups_group",
		Table:      users,
		Columns:    []*schema.Column{gid},
		RefTable:   groups,
		RefColumns: []*schema.Column{refcol},
		OnUpdate:   "",
		OnDelete:   schema.SetNull,
	}, fk)

	// unique indexes
	uuidc, ok := users.Column("uuid")
	require.True(t, ok, "expected column uuid")
	uniqIdx, ok := users.Index("users_uuid_uniq")
	require.True(t, ok, "expected index users_uuid_uniq")
	require.EqualValues(t, &schema.Index{
		Name:   "users_uuid_uniq",
		Unique: true,
		Table:  users,
		Parts:  []*schema.IndexPart{{C: uuidc, SeqNo: 0}},
	}, uniqIdx)

	// indexes
	timec, ok := users.Column("time")
	require.True(t, ok, "expected column time")
	timeIdx, ok := users.Index("user_time")
	require.True(t, ok, "expected time index")
	require.EqualValues(t, &schema.Index{
		Name:   "user_time",
		Unique: false,
		Table:  users,
		Parts: []*schema.IndexPart{
			{C: timec, SeqNo: 0},
		},
	}, timeIdx)
}
