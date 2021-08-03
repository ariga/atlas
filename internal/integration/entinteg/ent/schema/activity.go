// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Activity struct {
	ent.Schema
}

func (Activity) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
	}
}

func (Activity) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("users", User.Type).Ref("activities"),
	}
}
