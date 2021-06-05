package mysql

import (
	"fmt"
	"testing"

	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/require"
)

func TestConcretizer(t *testing.T) {
	for _, tt := range []struct {
		version  string
		body     string
		expected *schema.ColumnType
	}{
		{
			version: "8.0.24",
			body:    `type = "int"`,
			expected: &schema.ColumnType{
				Raw: "int",
				Type: &schema.IntegerType{
					T:    "int",
					Size: 4,
				},
				Null: false,
			},
		},
		{
			version: "8.0.24",
			body:    `type = "int8"`,
			expected: &schema.ColumnType{
				Raw: "tinyint",
				Type: &schema.IntegerType{
					T:    "tinyint",
					Size: 1,
				},
				Null: false,
			},
		},
		{
			version: "8.0.24",
			body:    `type = "int16"`,
			expected: &schema.ColumnType{
				Raw: "smallint",
				Type: &schema.IntegerType{
					T:    "smallint",
					Size: 2,
				},
				Null: false,
			},
		},
		{
			version: "8.0.24",
			body:    `type = "uint64"`,
			expected: &schema.ColumnType{
				Raw: "bigint unsigned",
				Type: &schema.IntegerType{
					T:        "bigint",
					Unsigned: true,
					Size:     8,
				},
				Null: false,
			},
		},
		{
			version: "8.0.24",
			body:    `type = "string"`,
			expected: &schema.ColumnType{
				Raw: "varchar(255)",
				Type: &schema.StringType{
					T:    "varchar",
					Size: 255,
				},
			},
		},
		{
			version: "8.0.24",
			body: `type = "string"
size = 100`,
			expected: &schema.ColumnType{
				Raw: "varchar(100)",
				Type: &schema.StringType{
					T:    "varchar",
					Size: 100,
				},
			},
		},
		{
			version: "8.0.24",
			body: `type = "string"
size = 100000`,
			expected: &schema.ColumnType{
				Raw: "mediumtext",
				Type: &schema.StringType{
					T:    "mediumtext",
					Size: 100000,
				},
			},
		},
		{
			version: "8.0.24",
			body: `type = "string"
size = 16777216`,
			expected: &schema.ColumnType{
				Raw: "longtext",
				Type: &schema.StringType{
					T:    "longtext",
					Size: 16777216,
				},
			},
		},
		{
			version: "8.0.24",
			body:    `type = "boolean"`,
			expected: &schema.ColumnType{
				Raw: "boolean",
				Type: &schema.BoolType{
					T: "boolean",
				},
			},
		},
	} {
		t.Run(tt.version+"::"+tt.body, func(t *testing.T) {
			c := &Concretizer{version: tt.version}
			col := parseHCLColumn(t, tt.body)
			err := c.Concretize(col)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, col.Type)
		})
	}
}

func parseHCLColumn(t *testing.T, body string) *schema.Column {
	s := `
schema "test" {
}

table "test" {
	schema = schema.test
	column "test" {
		%s
	}
	
}
`
	body = fmt.Sprintf(s, body)
	schemas, err := schema.UnmarshalHCL([]byte(body), "test.hcl")
	require.NoError(t, err)
	return schemas[0].Tables[0].Columns[0]
}
