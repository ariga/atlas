// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"testing"

	"ariga.io/atlas/sql/internal/schemautil"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/schema/schemaspec"
	"github.com/stretchr/testify/require"
)

func TestConvertColumnType(t *testing.T) {
	for _, tt := range []struct {
		spec        *schemaspec.Column
		expected    schema.Type
		expectedErr string
	}{
		{
			spec: schemautil.ColSpec("int", "int"),
			expected: &schema.IntegerType{
				T:        "integer",
				Unsigned: false,
			},
		},
		{
			spec:        schemautil.ColSpec("uint", "uint"),
			expectedErr: "sqlite: unsigned integers currently not supported",
		},
		{
			spec: schemautil.ColSpec("int64", "int64"),
			expected: &schema.IntegerType{
				T:        "integer",
				Unsigned: false,
			},
		},
		{
			spec:        schemautil.ColSpec("uint64", "uint64"),
			expectedErr: "sqlite: unsigned integers currently not supported",
		},
		{
			spec: schemautil.ColSpec("string_varchar", "string", schemautil.LitAttr("size", "255")),
			expected: &schema.StringType{
				T:    "text",
				Size: 255,
			},
		},
		{
			spec: schemautil.ColSpec("string_test", "string", schemautil.LitAttr("size", "10485761")),
			expected: &schema.StringType{
				T:    "text",
				Size: 10_485_761,
			},
		},
		{
			spec: schemautil.ColSpec("varchar(255)", "varchar(255)"),
			expected: &schema.StringType{
				T:    "varchar",
				Size: 255,
			},
		},
		{
			spec: schemautil.ColSpec("decimal(10, 2)", "decimal(10, 2)"),
			expected: &schema.DecimalType{
				T:         "decimal",
				Scale:     2,
				Precision: 10,
			},
		},
		{
			spec:     schemautil.ColSpec("enum", "enum", schemautil.ListAttr("values", "a", "b", "c")),
			expected: &schema.StringType{T: "text"},
		},
		{
			spec:     schemautil.ColSpec("bool", "boolean"),
			expected: &schema.BoolType{T: "boolean"},
		},
		{
			spec:     schemautil.ColSpec("decimal", "decimal", schemautil.LitAttr("precision", "10"), schemautil.LitAttr("scale", "2")),
			expected: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2},
		},
		{
			spec:     schemautil.ColSpec("float", "float", schemautil.LitAttr("precision", "10")),
			expected: &schema.FloatType{T: "real", Precision: 10},
		},
	} {
		t.Run(tt.spec.Name, func(t *testing.T) {
			columnType, err := ConvertColumnType(tt.spec)
			if tt.expectedErr != "" && err != nil {
				require.Equal(t, tt.expectedErr, err.Error())
				return
			}
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, columnType)
		})
	}
}
