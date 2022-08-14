// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schema

import (
	"time"

	"ariga.io/atlas/sql/migrate"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Revision holds the schema definition for the Revision entity.
type Revision struct {
	ent.Schema
}

// Fields of the Revision.
func (Revision) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			StorageKey("version").
			Immutable(),
		field.String("description").
			Immutable(),
		field.Uint("type").
			GoType(migrate.RevisionType(0)),
		field.Int("applied").
			NonNegative(),
		field.Int("total").
			NonNegative(),
		field.Time("executed_at").
			Immutable(),
		field.Int64("execution_time").
			GoType(time.Duration(0)),
		field.Text("error").
			Optional(),
		field.String("hash"),
		field.Strings("partial_hashes").
			Optional(),
		field.String("operator_version"),
	}
}

// Annotations of the Revision.
func (Revision) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "atlas_schema_revisions"},
	}
}
