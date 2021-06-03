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

func (s *ConvertSuite) SetupSuite() {
	graph, err := entc.LoadGraph("../ent/schema", &gen.Config{})
	s.Require().NoError(err)
	sch, err := Convert(graph)
	s.Require().NoError(err)
	s.graph = graph
	s.schema = sch
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(ConvertSuite))
}

func (s *ConvertSuite) TestTables() {
	for _, n := range []string{
		"users",
		"groups",
		"activities",
		"user_activities",
	} {
		s.Run(n, func() {
			_, ok := s.schema.Table(n)
			s.Require().Truef(ok, "expected table %q to exist", n)
		})
	}
}

func (s *ConvertSuite) TestUserColumns() {
	users, _ := s.schema.Table("users")
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
		s.Require().True(ok, "expected column to exist")
		s.Require().EqualValues(tt.expected, column.Type)
	}
}

func (s *ConvertSuite) TestPrimaryKey() {
	users, _ := s.schema.Table("users")
	id, ok := users.Column("id")
	s.Require().True(ok, "expected col id to exist")
	s.Require().EqualValues(&schema.Index{
		Parts: []*schema.IndexPart{
			{C: id, SeqNo: 0},
		},
	}, users.PrimaryKey)
}

func (s *ConvertSuite) TestForeignKey() {
	users, _ := s.schema.Table("users")
	gid, ok := users.Column("group_id")
	s.Require().True(ok, "expected column group_id")
	fk, ok := users.ForeignKey("users_groups_group")
	groups, ok := s.schema.Table("groups")
	s.Require().True(ok, "expected table groups")
	refcol, ok := groups.Column("id")
	s.Require().True(ok, "expected column id")
	s.Require().EqualValues(&schema.ForeignKey{
		Symbol:     "users_groups_group",
		Table:      users,
		Columns:    []*schema.Column{gid},
		RefTable:   groups,
		RefColumns: []*schema.Column{refcol},
		OnUpdate:   "",
		OnDelete:   schema.SetNull,
	}, fk)
}

func (s *ConvertSuite) TestUnique() {
	users, _ := s.schema.Table("users")
	uuidc, ok := users.Column("uuid")
	s.Require().True(ok, "expected column uuid")
	uniqIdx, ok := users.Index("uuid")
	s.Require().True(ok, "expected index users_uuid_uniq")
	s.Require().EqualValues(&schema.Index{
		Name:   "uuid",
		Unique: true,
		Table:  users,
		Parts:  []*schema.IndexPart{{C: uuidc, SeqNo: 0}},
	}, uniqIdx)
}

func (s *ConvertSuite) TestIndexes() {
	users, _ := s.schema.Table("users")
	timec, ok := users.Column("time")
	s.Require().True(ok, "expected column time")
	timeIdx, ok := users.Index("user_time")
	s.Require().True(ok, "expected time index")
	s.Require().EqualValues(&schema.Index{
		Name:   "user_time",
		Unique: false,
		Table:  users,
		Parts: []*schema.IndexPart{
			{C: timec, SeqNo: 0},
		},
	}, timeIdx)
}

func (s *ConvertSuite) TestRelationTable() {
	relTable, ok := s.schema.Table("user_activities")
	s.Require().True(ok, "expected relation table user_activities")
	s.Require().Len(relTable.Columns, 2)
	s.Require().Len(relTable.ForeignKeys, 2)
	_, ok = relTable.Column("user_id")
	s.Require().True(ok, "expected user_id column")
	_, ok = relTable.Column("activity_id")
	s.Require().True(ok, "expected activity_id column")
}

func (s *ConvertSuite) TestDefault() {
	tbl, ok := s.schema.Table("default_containers")
	s.Require().True(ok, "expected default_containers table")
	for _, tt := range []struct {
		col      string
		expected string
	}{
		{col: "stringdef", expected: `"default"`},
		{col: "int", expected: `1`},
		{col: "bool", expected: `true`},
		{col: "enum", expected: `"1"`},
		{col: "float", expected: `1.5`},
	} {
		col, ok := tbl.Column(tt.col)
		s.Require().Truef(ok, "expected col %q", tt.col)
		s.Require().EqualValues(tt.expected, col.Default.(*schema.RawExpr).X)
	}
}
