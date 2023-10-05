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
		field.String("file_name").Immutable().MaxLen(255),
		field.Int64("user_id").Positive(),
		field.String("extension").Immutable().MinLen(2).MaxLen(10),
		field.String("path").Immutable().Unique(),
		field.String("location").Nillable().Optional(),
		field.Int64("size"),                         // in bytes
		field.Int64("width").Nillable().Optional(),  // in pixels
		field.Int64("height").Nillable().Optional(), // in pixels
		field.Time("created_at").Default(time.Now),
		field.Time("uploaded_at").Nillable().Optional(),
	}
}

// Edges of the Media.
func (Media) Edges() []ent.Edge {
	return nil
}
