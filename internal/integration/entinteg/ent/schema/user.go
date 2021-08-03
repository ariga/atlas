// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.String("optional").
			Optional(),
		field.Int("int"),
		field.Uint("uint"),
		field.Uint64("uint64"),
		field.Int64("int64"),
		field.Time("time"),
		field.Bool("bool"),
		field.Enum("enum").
			Values("1", "2", "3"),
		field.Enum("named_enum").
			NamedValues("a", "1", "b", "2", "c", "3"),
		field.UUID("uuid", uuid.New()).
			Unique(),
		field.Bytes("bytes"),
		field.Int("group_id").
			Optional(),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("group", Group.Type).
			Unique().
			Field("group_id"),
		edge.To("activities", Activity.Type),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("time"),
	}
}
