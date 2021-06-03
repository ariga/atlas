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
