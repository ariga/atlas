package schema

import (
	"time"

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
			StorageKey("version"),
		field.String("description"),
		field.Enum("execution_state").
			Values("ongoing", "ok", "error"),
		field.Time("executed_at"),
		field.Int64("execution_time").
			GoType(time.Duration(0)),
		field.String("hash"),
		field.String("operator_version"),
		field.JSON("meta", make(map[string]string)),
	}
}

// Annotations of the Revision.
func (Revision) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "atlas_schema_revisions"},
	}
}
