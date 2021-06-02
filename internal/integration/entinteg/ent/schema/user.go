package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
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
		field.String("optional").Optional(),
		field.Int("int"),
		field.Uint("uint"),
		field.Time("time"),
		field.Bool("bool"),
		field.Enum("enum").Values("1", "2", "3"),
		field.Enum("enum_2").NamedValues("a", "1", "b", "2", "c", "3"),
		field.UUID("uuid", uuid.New()),
		field.Bytes("bytes"),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return nil
}
