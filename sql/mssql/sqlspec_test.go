// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mssql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ariga.io/atlas/sql/internal/spectest"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

func TestRegistrySanity(t *testing.T) {
	spectest.RegistrySanityTest(t, TypeRegistry, []string{
		// skip the following types as they are have different sizes in input and output
		// nchar(50) and nvarchar(50) have Size attribute as 100
		"nchar", "nvarchar",
	})
}

func TestParseType_NCharNVarchar(t *testing.T) {
	for _, tt := range []struct {
		input   string
		wantT   *schema.StringType
		wantErr bool
	}{
		{
			input: "nchar",
			wantT: &schema.StringType{T: TypeNChar, Size: 2},
		},
		{
			input: "nchar(1)",
			wantT: &schema.StringType{T: TypeNChar, Size: 2},
		},
		{
			input: "nchar(2)",
			wantT: &schema.StringType{T: TypeNChar, Size: 4},
		},
		{
			input: "nvarchar",
			wantT: &schema.StringType{T: TypeNVarchar, Size: 2},
		},
		{
			input: "nvarchar(1)",
			wantT: &schema.StringType{T: TypeNVarchar, Size: 2},
		},
		{
			input: "nvarchar(2)",
			wantT: &schema.StringType{T: TypeNVarchar, Size: 4},
		},
	} {
		d, err := ParseType(tt.input)
		require.Equal(t, tt.wantErr, err != nil)
		require.Equal(t, tt.wantT, d)
	}
}

func TestParseType_MAX(t *testing.T) {
	for _, tt := range []struct {
		input   string
		wantT   schema.Type
		wantErr bool
	}{
		{
			input: "varbinary(MAX)",
			wantT: &schema.BinaryType{T: TypeVarBinary, Size: sqlx.P(-1)},
		},
		{
			input: "varchar(MAX)",
			wantT: &schema.StringType{T: TypeVarchar, Size: -1},
		},
		{
			input: "nvarchar(MAX)",
			wantT: &schema.StringType{T: TypeNVarchar, Size: -1},
		},
	} {
		d, err := ParseType(tt.input)
		require.Equal(t, tt.wantErr, err != nil)
		require.Equal(t, tt.wantT, d)
	}
}

func TestMarshalSpec_Identity(t *testing.T) {
	s := &schema.Schema{
		Name: "dbo",
		Tables: []*schema.Table{
			{
				Name: "t1",
				Columns: []*schema.Column{
					{
						Name: "id",
						Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}},
						Attrs: []schema.Attr{
							&Identity{Seek: 701, Increment: 1000},
						},
					},
				},
			},
		},
	}
	s.Tables[0].Schema = s
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	const expected = `table "t1" {
  schema = schema.dbo
  column "id" {
    null = false
    type = bigint
    identity {
      seek      = 701
      increment = 1000
    }
  }
}
schema "dbo" {
}
`
	require.EqualValues(t, expected, string(buf))
}

func TestSQLSpec(t *testing.T) {
	f := `
schema "dbo" {
}

table "t1" {
	schema = schema.dbo
	column "c1" {
		type = int
		identity {
			seek      = 701
			increment = 1000
		}
	}
}
`
	var s schema.Schema
	err := EvalHCLBytes([]byte(f), &s, nil)
	require.NoError(t, err)

	exp := &schema.Schema{
		Name: "dbo",
	}
	tables := []*schema.Table{
		{
			Name: "t1",
			Columns: []*schema.Column{
				{
					Name: "c1",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}},
					Attrs: []schema.Attr{
						&Identity{Seek: 701, Increment: 1000},
					},
				},
			},
		},
	}
	tables[0].Schema = exp
	exp.Tables = tables
	require.EqualValues(t, exp, &s)
}
