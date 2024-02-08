// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package specutil

import (
	"math"
	"testing"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestFromSpec_SchemaName(t *testing.T) {
	sc := &schema.Schema{
		Name: "schema",
		Tables: []*schema.Table{
			{},
		},
	}
	sc.Tables[0].Schema = sc
	spec, err := FromSchema(sc, &SchemaFuncs{
		Table: func(*schema.Table) (*sqlspec.Table, error) {
			return &sqlspec.Table{}, nil
		},
		View: func(*schema.View) (*sqlspec.View, error) {
			return &sqlspec.View{}, nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, sc.Name, spec.Schema.Name)
	require.Equal(t, "$schema."+sc.Name, spec.Tables[0].Schema.V)
}

func TestFromForeignKey(t *testing.T) {
	tbl := &schema.Table{
		Name: "users",
		Columns: []*schema.Column{
			{
				Name: "id",
				Type: &schema.ColumnType{
					Type: &schema.IntegerType{
						T: "int",
					},
				},
			},
			{
				Name: "parent_id",
				Type: &schema.ColumnType{
					Type: &schema.IntegerType{
						T: "int",
					},
				},
			},
		},
	}
	fk := &schema.ForeignKey{
		Symbol:     "fk",
		Table:      tbl,
		Columns:    tbl.Columns[1:],
		RefTable:   tbl,
		RefColumns: tbl.Columns[:1],
		OnUpdate:   schema.NoAction,
		OnDelete:   schema.Cascade,
	}
	key, err := FromForeignKey(fk)
	require.NoError(t, err)
	require.EqualValues(t, &sqlspec.ForeignKey{
		Symbol: "fk",
		Columns: []*schemahcl.Ref{
			{V: "$column.parent_id"},
		},
		RefColumns: []*schemahcl.Ref{
			{V: "$column.id"},
		},
		OnUpdate: &schemahcl.Ref{V: "NO_ACTION"},
		OnDelete: &schemahcl.Ref{V: "CASCADE"},
	}, key)

	fk.OnDelete = ""
	fk.OnUpdate = ""
	key, err = FromForeignKey(fk)
	require.NoError(t, err)
	require.EqualValues(t, &sqlspec.ForeignKey{
		Symbol: "fk",
		Columns: []*schemahcl.Ref{
			{V: "$column.parent_id"},
		},
		RefColumns: []*schemahcl.Ref{
			{V: "$column.id"},
		},
	}, key)
}

func TestDefault(t *testing.T) {
	for _, tt := range []struct {
		v cty.Value
		x string
	}{
		{
			v: cty.NumberUIntVal(1),
			x: "1",
		},
		{
			v: cty.NumberIntVal(1),
			x: "1",
		},
		{
			v: cty.NumberFloatVal(1),
			x: "1",
		},
		{
			v: cty.NumberIntVal(-100),
			x: "-100",
		},
		{
			v: cty.NumberFloatVal(-100),
			x: "-100",
		},
		{
			v: cty.NumberUIntVal(math.MaxUint64),
			x: "18446744073709551615",
		},
		{
			v: cty.NumberIntVal(math.MinInt64),
			x: "-9223372036854775808",
		},
		{
			v: cty.NumberFloatVal(-1024.1024),
			x: "-1024.1024",
		},
	} {
		// From cty.Value (HCL) to database literal.
		x, err := Default(tt.v)
		require.NoError(t, err)
		require.Equal(t, tt.x, x.(*schema.Literal).V)
		// From database literal to cty.Value (HCL).
		v, err := ColumnDefault(schema.NewColumn("").SetDefault(&schema.Literal{V: tt.x}))
		require.NoError(t, err)
		require.True(t, tt.v.Equals(v).True())
	}
}
