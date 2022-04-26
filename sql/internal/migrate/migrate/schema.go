// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/schema/field"
)

var (
	// AtlasSchemaRevisionsColumns holds the columns for the "atlas_schema_revisions" table.
	AtlasSchemaRevisionsColumns = []*schema.Column{
		{Name: "version", Type: field.TypeString},
		{Name: "description", Type: field.TypeString},
		{Name: "execution_state", Type: field.TypeEnum, Enums: []string{"ongoing", "ok", "error"}},
		{Name: "executed_at", Type: field.TypeTime},
		{Name: "execution_time", Type: field.TypeInt64},
		{Name: "hash", Type: field.TypeString},
		{Name: "operator_version", Type: field.TypeString},
		{Name: "meta", Type: field.TypeJSON},
	}
	// AtlasSchemaRevisionsTable holds the schema information for the "atlas_schema_revisions" table.
	AtlasSchemaRevisionsTable = &schema.Table{
		Name:       "atlas_schema_revisions",
		Columns:    AtlasSchemaRevisionsColumns,
		PrimaryKey: []*schema.Column{AtlasSchemaRevisionsColumns[0]},
	}
	// Tables holds all the tables in the schema.
	Tables = []*schema.Table{
		AtlasSchemaRevisionsTable,
	}
)

func init() {
	AtlasSchemaRevisionsTable.Annotation = &entsql.Annotation{
		Table: "atlas_schema_revisions",
	}
}
