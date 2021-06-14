package entschema

import (
	"testing"

	"ariga.io/atlas/sql/schema"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ConvertSuite struct {
	suite.Suite
	graph  *gen.Graph
	schema *schema.SchemaSpec
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
			_, ok := tableSpec(s.schema, n)
			s.Require().Truef(ok, "expected table %q to exist", n)
		})
	}
}

func (s *ConvertSuite) TestUserColumns() {
	users, _ := tableSpec(s.schema, "users")
	for _, tt := range []struct {
		fld      string
		expected *schema.ColumnType
		exp      *schema.ColumnSpec
	}{
		{
			fld: "name",
			exp: &schema.ColumnSpec{
				Name: "name",
				Type: "string",
			},
		},
		{
			fld: "optional",
			exp: &schema.ColumnSpec{
				Name: "optional",
				Type: "string",
				Null: true,
			},
		},
		{
			fld: "int",
			exp: &schema.ColumnSpec{
				Name: "int",
				Type: "int",
			},
		},
		{
			fld: "uint",
			exp: &schema.ColumnSpec{
				Name: "uint",
				Type: "uint",
			},
		},
		{
			fld: "int64",
			exp: &schema.ColumnSpec{
				Name: "int64",
				Type: "int64",
			},
		},
		{
			fld: "uint64",
			exp: &schema.ColumnSpec{
				Name: "uint64",
				Type: "uint64",
			},
		},
		{
			fld: "time",
			exp: &schema.ColumnSpec{
				Name: "time",
				Type: "time",
			},
		},
		{
			fld: "bool",
			exp: &schema.ColumnSpec{
				Name: "bool",
				Type: "boolean",
			},
		},
		{
			fld: "enum",
			exp: &schema.ColumnSpec{
				Name: "enum",
				Type: "enum",
				Attrs: []*schema.SpecAttr{
					{K: "values", V: &schema.ListValue{V: []string{`"1"`, `"2"`, `"3"`}}},
				},
			},
		},
		{
			fld: "named_enum",
			exp: &schema.ColumnSpec{
				Name: "named_enum",
				Type: "enum",
				Attrs: []*schema.SpecAttr{
					{K: "values", V: &schema.ListValue{V: []string{`"1"`, `"2"`, `"3"`}}},
				},
			},
		},
		{
			fld: "uuid",
			exp: &schema.ColumnSpec{
				Name: "uuid",
				Type: "binary",
				Attrs: []*schema.SpecAttr{
					intAttr("size", 16),
				},
			},
		},
		{
			fld: "bytes",
			exp: &schema.ColumnSpec{
				Name: "bytes",
				Type: "binary",
			},
		},
		{
			fld: "group_id",
			exp: &schema.ColumnSpec{
				Name: "group_id",
				Null: true,
				Type: "int",
			},
		},
	} {
		s.T().Run(tt.fld, func(t *testing.T) {
			column, ok := columnSpec(users, tt.fld)
			require.True(t, ok, "expected column to exist")
			require.EqualValues(t, tt.exp, column)
		})
	}
}

func (s *ConvertSuite) TestPrimaryKey() {
	users, _ := tableSpec(s.schema, "users")
	s.Require().EqualValues(&schema.PrimaryKeySpec{
		Columns: []*schema.ColumnRef{
			{Table: "users", Name: "id"},
		},
	}, users.PrimaryKey)
}

func (s *ConvertSuite) TestForeignKey() {
	users, _ := tableSpec(s.schema, "users")
	fk, ok := fkSpec(users, "users_groups_group")
	s.Require().True(ok, "expected column id")
	s.Require().EqualValues(&schema.ForeignKeySpec{
		Symbol: "users_groups_group",
		Columns: []*schema.ColumnRef{
			{Table: "users", Name: "group_id"},
		},
		RefColumns: []*schema.ColumnRef{
			{Table: "groups", Name: "id"},
		},
		OnUpdate: "",
		OnDelete: string(schema.SetNull),
	}, fk)
}

func (s *ConvertSuite) TestUnique() {
	users, _ := tableSpec(s.schema, "users")
	fk, ok := fkSpec(users, "users_groups_group")
	s.Require().True(ok, "expected column id")
	s.Require().EqualValues(&schema.ForeignKeySpec{
		Symbol: "users_groups_group",
		Columns: []*schema.ColumnRef{
			{Table: "users", Name: "group_id"},
		},
		RefColumns: []*schema.ColumnRef{
			{Table: "groups", Name: "id"},
		},
		OnDelete: string(schema.SetNull),
	}, fk)
}

func (s *ConvertSuite) TestIndexes() {
	users, _ := tableSpec(s.schema, "users")
	timeIdx, ok := indexSpec(users, "user_time")
	s.Require().True(ok, "expected time index")
	s.Require().EqualValues(&schema.IndexSpec{
		Name:   "user_time",
		Unique: false,
		Columns: []*schema.ColumnRef{
			{Table: "users", Name: "time"},
		},
	}, timeIdx)
}

func (s *ConvertSuite) TestRelationTable() {
	relTable, ok := tableSpec(s.schema, "user_activities")
	s.Require().True(ok, "expected relation table user_activities")
	s.Require().Len(relTable.Columns, 2)
	s.Require().Len(relTable.ForeignKeys, 2)
	_, ok = columnSpec(relTable, "user_id")
	s.Require().True(ok, "expected user_id column")
	_, ok = columnSpec(relTable, "activity_id")
	s.Require().True(ok, "expected activity_id column")
}

func (s *ConvertSuite) TestDefault() {
	tbl, ok := tableSpec(s.schema, "default_containers")
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
		col, ok := columnSpec(tbl, tt.col)
		s.Require().Truef(ok, "expected col %q", tt.col)
		s.Require().EqualValues(schema.LiteralValue{V: tt.expected}, *col.Default)
	}
}

func columnSpec(t *schema.TableSpec, name string) (*schema.ColumnSpec, bool) {
	for _, c := range t.Columns {
		if c.Name == name {
			return c, true
		}
	}
	return nil, false
}

func fkSpec(t *schema.TableSpec, symbol string) (*schema.ForeignKeySpec, bool) {
	for _, fk := range t.ForeignKeys {
		if fk.Symbol == symbol {
			return fk, true
		}
	}
	return nil, false
}

func indexSpec(t *schema.TableSpec, name string) (*schema.IndexSpec, bool) {
	for _, idx := range t.Indexes {
		if idx.Name == name {
			return idx, true
		}
	}
	return nil, false
}
