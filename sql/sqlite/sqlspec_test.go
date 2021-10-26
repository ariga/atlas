// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"fmt"
	"testing"

	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/require"
)

func TestSQLSpec(t *testing.T) {
	f := `
schema "schema" {
}

table "table" {
	column "col" {
		type = "int"
	}
	
}
`
	b := `
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
	fmt.Println(b)
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
							T: "integer",
						},
					},
				},
				//{
				//	Name: "age",
				//	Type: &schema.ColumnType{
				//		Type: &schema.IntegerType{
				//			T: "integer",
				//		},
				//	},
				//},
				//{
				//	Name: "account_name",
				//	Type: &schema.ColumnType{
				//		Type: &schema.StringType{
				//			T:    "varchar",
				//			Size: 32,
				//		},
				//	},
				//},
			},
		},
		//{
		//	Name:   "accounts",
		//	Schema: exp,
		//	Columns: []*schema.Column{
		//		{
		//			Name: "name",
		//			Type: &schema.ColumnType{
		//				Type: &schema.StringType{
		//					T:    "varchar",
		//					Size: 32,
		//				},
		//			},
		//		},
		//	},
		//},
	}
	//exp.Tables[0].PrimaryKey = &schema.Index{
	//	Table: exp.Tables[0],
	//	Parts: []*schema.IndexPart{
	//		{SeqNo: 0, C: exp.Tables[0].Columns[0]},
	//	},
	//}
	//exp.Tables[0].Indexes = []*schema.Index{
	//	{
	//		Name:   "index",
	//		Table:  exp.Tables[0],
	//		Unique: true,
	//		Parts: []*schema.IndexPart{
	//			{SeqNo: 0, C: exp.Tables[0].Columns[0]},
	//			{SeqNo: 1, C: exp.Tables[0].Columns[1]},
	//		},
	//	},
	//}
	//exp.Tables[0].ForeignKeys = []*schema.ForeignKey{
	//	{
	//		Symbol:     "accounts",
	//		Table:      exp.Tables[0],
	//		Columns:    []*schema.Column{exp.Tables[0].Columns[2]},
	//		RefColumns: []*schema.Column{exp.Tables[1].Columns[0]},
	//		OnDelete:   schema.SetNull,
	//	},
	//}

	s := schema.Schema{}
	err := UnmarshalSpec([]byte(f), schemahcl.Unmarshal, &s)
	require.NoError(t, err)
	require.EqualValues(t, exp, &s)
}
