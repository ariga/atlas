package entschema

import (
	"testing"

	"ariga.io/atlas/sql/schema"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/stretchr/testify/suite"
)

type ConvertSuite struct {
	suite.Suite
	graph  *gen.Graph
	schema *schema.Schema
}

func (suite *ConvertSuite) SetupSuite() {
	graph, err := entc.LoadGraph("../ent/schema", &gen.Config{})
	suite.Require().NoError(err)
	sch, err := Convert(graph)
	suite.Require().NoError(err)
	suite.graph = graph
	suite.schema = sch
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(ConvertSuite))
}

func (suite *ConvertSuite) TestTables() {
	for _, n := range []string{
		"users",
		"groups",
		"activities",
		"user_activities",
	} {
		suite.Run(n, func() {
			_, ok := suite.schema.Table(n)
			suite.Require().Truef(ok, "expected table %q to exist", n)
		})
	}
}

func (suite *ConvertSuite) TestUserColumns() {
	users, _ := suite.schema.Table("users")
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
			fld: "named_enum",
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
		suite.Require().True(ok, "expected column to exist")
		suite.Require().EqualValues(tt.expected, column.Type)
	}
}

func (suite *ConvertSuite) TestPrimaryKey() {
	users, _ := suite.schema.Table("users")
	id, ok := users.Column("id")
	suite.Require().True(ok, "expected col id to exist")
	suite.Require().EqualValues(&schema.Index{
		Parts: []*schema.IndexPart{
			{C: id, SeqNo: 0},
		},
	}, users.PrimaryKey)
}

func (suite *ConvertSuite) TestForeignKey() {
	users, _ := suite.schema.Table("users")
	gid, ok := users.Column("group_id")
	suite.Require().True(ok, "expected column group_id")
	fk, ok := users.ForeignKey("users_groups_group")
	groups, ok := suite.schema.Table("groups")
	suite.Require().True(ok, "expected table groups")
	refcol, ok := groups.Column("id")
	suite.Require().True(ok, "expected column id")
	suite.Require().EqualValues(&schema.ForeignKey{
		Symbol:     "users_groups_group",
		Table:      users,
		Columns:    []*schema.Column{gid},
		RefTable:   groups,
		RefColumns: []*schema.Column{refcol},
		OnUpdate:   "",
		OnDelete:   schema.SetNull,
	}, fk)
}

func (suite *ConvertSuite) TestUnique() {
	users, _ := suite.schema.Table("users")
	uuidc, ok := users.Column("uuid")
	suite.Require().True(ok, "expected column uuid")
	uniqIdx, ok := users.Index("users_uuid_uniq")
	suite.Require().True(ok, "expected index users_uuid_uniq")
	suite.Require().EqualValues(&schema.Index{
		Name:   "users_uuid_uniq",
		Unique: true,
		Table:  users,
		Parts:  []*schema.IndexPart{{C: uuidc, SeqNo: 0}},
	}, uniqIdx)
}

func (suite *ConvertSuite) TestIndexes() {
	users, _ := suite.schema.Table("users")
	timec, ok := users.Column("time")
	suite.Require().True(ok, "expected column time")
	timeIdx, ok := users.Index("user_time")
	suite.Require().True(ok, "expected time index")
	suite.Require().EqualValues(&schema.Index{
		Name:   "user_time",
		Unique: false,
		Table:  users,
		Parts: []*schema.IndexPart{
			{C: timec, SeqNo: 0},
		},
	}, timeIdx)
}

func (suite *ConvertSuite) TestRelationTable() {
	relTable, ok := suite.schema.Table("user_activities")
	suite.Require().True(ok, "expected relation table user_activities")
	suite.Require().Len(relTable.Columns, 2)
	suite.Require().Len(relTable.ForeignKeys, 2)
	_, ok = relTable.Column("user_id")
	suite.Require().True(ok, "expected user_id column")
	_, ok = relTable.Column("activity_id")
	suite.Require().True(ok, "expected activity_id column")
}
