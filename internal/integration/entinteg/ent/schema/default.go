package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type DefaultContainer struct {
	ent.Schema
}

func (DefaultContainer) Fields() []ent.Field {
	return []ent.Field{
		field.String("string").Default("default"),
		field.Int("int").Default(1),
		field.Bool("bool").Default(true),
		field.Enum("enum").Values("1", "2").Default("1"),
		field.Float("float").Default(1.5),
	}
}
