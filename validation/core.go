package validation

import (
	"github.com/kcmvp/strsql"
)

// Event defines the lifecycle event during which validation occurs.
type Event string

const (
	EventSave   Event = "save"
	EventUpdate Event = "update"
	EventDelete Event = "delete"
)

// ColumnRule validates a single attribute value.
type ColumnRule[T strsql.Entity] func(attr strsql.Mapping[T], value any) error

// EntityRule validates an entire entity.
type EntityRule[T strsql.Entity] func(entity T) error

// Plugin installs rule sets into a registry.
type Plugin interface {
	Name() string
	Register(r *Registry) error
}
