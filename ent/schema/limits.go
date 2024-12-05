package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Limits holds the schema definition for the Limits entity.
type Limits struct {
	ent.Schema
}

// Fields of the Limits.
func (Limits) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("user_id"),
		field.Int64("tenant_id"),
	}
}

// Edges of the Limits.
func (Limits) Edges() []ent.Edge {
	return nil
}
