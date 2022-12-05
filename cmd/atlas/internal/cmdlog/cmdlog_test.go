// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdlog_test

import (
	"encoding/json"
	"io"
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/cmdlog"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestSchemaInspect_MarshalJSON(t *testing.T) {
	report := &cmdlog.SchemaInspect{
		Realm: schema.NewRealm(
			schema.New("test").
				AddTables(
					schema.NewTable("users").
						AddColumns(
							&schema.Column{
								Name: "id",
								Type: &schema.ColumnType{Raw: "bigint"},
							},
							&schema.Column{
								Name: "name",
								Type: &schema.ColumnType{Raw: "varchar(255)"},
							},
						),
					schema.NewTable("posts").
						AddColumns(
							&schema.Column{
								Name: "id",
								Type: &schema.ColumnType{Raw: "bigint"},
							},
							&schema.Column{
								Name: "text",
								Type: &schema.ColumnType{Raw: "text"},
							},
						),
				),
			schema.New("temp"),
		),
	}
	b, err := report.MarshalJSON()
	require.NoError(t, err)
	ident, err := json.MarshalIndent(json.RawMessage(b), "", "  ")
	require.NoError(t, err)
	require.Equal(t, `{
  "schemas": [
    {
      "name": "test",
      "tables": [
        {
          "name": "users",
          "columns": [
            {
              "name": "id",
              "type": "bigint"
            },
            {
              "name": "name",
              "type": "varchar(255)"
            }
          ]
        },
        {
          "name": "posts",
          "columns": [
            {
              "name": "id",
              "type": "bigint"
            },
            {
              "name": "text",
              "type": "text"
            }
          ]
        }
      ]
    },
    {
      "name": "temp"
    }
  ]
}`, string(ident))

	report = &cmdlog.SchemaInspect{Error: io.EOF}
	b, err = report.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, `{"Error":"EOF"}`, string(b))
}
