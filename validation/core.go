package validation

import (
	"context"

	"github.com/kcmvp/strsql"
)

// Event defines the lifecycle event during which validation occurs.
type Event string

const (
	EventBeforeInsert Event = "before_insert"
	EventBeforeUpdate Event = "before_update"
	EventBeforeDelete Event = "before_delete"

	// Backward-compatible aliases
	EventSave   Event = EventBeforeInsert
	EventUpdate Event = EventBeforeUpdate
	EventDelete Event = EventBeforeDelete
)

// RuleContext carries contextual information for custom business rules.
type RuleContext struct {
	context.Context
	Event Event
	Meta  map[string]any
}

// ColumnRule validates a single attribute value.
type ColumnRule[T strsql.Entity] func(ctx RuleContext, attr strsql.Mapping[T], value any) error

// EntityRule validates an entire entity.
type EntityRule[T strsql.Entity] func(ctx RuleContext, entity T) error

// Plugin installs rule sets into a registry.
type Plugin interface {
	Name() string
	Register(r *Registry) error
}
