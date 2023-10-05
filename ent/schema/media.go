package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Media holds the schema definition for the Media entity.
type Media struct {
	ent.Schema
}

// Fields of the Media.
func (Media) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").Positive(),
		field.String("file_name").Immutable(),
		field.String("extension").Immutable().MinLen(2).MaxLen(10),
		field.String("path").Immutable().Unique(),
		field.String("location").Nillable().Optional(),
		field.Int32("size"),                         // in bytes
		field.Int32("width").Nillable().Optional(),  // in pixels
		field.Int32("height").Nillable().Optional(), // in pixels
		field.Time("created_at").Default(time.Now),
		field.Time("uploaded_at").Nillable().Optional(),
	}
}

// Edges of the Media.
func (Media) Edges() []ent.Edge {
	return nil
}
