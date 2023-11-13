package schema

import (
	"time"

	"gitlab.calendaria.team/services/media/ent/mixins"

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
		field.Int64("owner_id").Immutable(),
		field.String("file_name").Immutable(),
		field.String("extension").Immutable().MinLen(2).MaxLen(10),
		field.String("path").Immutable().Unique(),
		field.String("url").Nillable().Optional(),
		field.Int32("size"),                             // in bytes
		field.Int32("width").Nillable().Optional(),      // in pixels
		field.Int32("height").Nillable().Optional(),     // in pixels
		field.Float32("duration").Nillable().Optional(), // in seconds
		field.Bool("is_activated").Default(false),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("uploaded_at").Nillable().Optional(),
	}
}

// Edges of the Media.
func (Media) Edges() []ent.Edge {
	return nil
}

func (Media) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.SoftDeleteMixin{},
	}
}
