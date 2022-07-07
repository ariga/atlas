// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package hclsqlspec

import (
	"testing"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/sqlspec"

	"github.com/stretchr/testify/require"
)

var hcl = schemahcl.New(schemahcl.WithTypes(postgres.TypeRegistry.Specs()))

func TestMigrate(t *testing.T) {
	f := `
modify_table {
	table = "users"
	add_column {
		column "id" {
			type = int
		}
	}
}
`
	var test struct {
		Changes []sqlspec.Change `spec:""`
	}
	err := hcl.EvalBytes([]byte(f), &test, nil)
	require.NoError(t, err)
	require.EqualValues(t, &sqlspec.ModifyTable{
		Table: "users",
		Changes: []sqlspec.Change{
			&sqlspec.AddColumn{
				Column: &sqlspec.Column{
					Name: "id",
					Null: false,
					Type: &schemahcl.Type{T: "int"},
				},
			},
		},
	}, test.Changes[0])
}
