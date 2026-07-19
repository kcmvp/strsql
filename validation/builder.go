package validation

import (
	"reflect"

	"github.com/kcmvp/strsql"
)

// Builder provides a fluent API for defining validation rules.
type Builder[T strsql.Entity] struct {
	registry *Registry
	typ      reflect.Type
}

// Validate creates a new Builder for the given entity type.
func Validate[T strsql.Entity](r *Registry) *Builder[T] {
	return &Builder[T]{
		registry: r,
		typ:      reflect.TypeOf(*new(T)),
	}
}

// Col begins defining rules for a specific column.
func (b *Builder[T]) Col(attr strsql.Attribute[T]) *ColBuilder[T] {
	return &ColBuilder[T]{
		builder: b,
		attr:    attr(),
	}
}

// Entity begins defining rules for the entire entity.
func (b *Builder[T]) Entity() *EntityBuilder[T] {
	return &EntityBuilder[T]{
		builder: b,
	}
}

// ColBuilder provides a fluent API for adding rules to a column.
type ColBuilder[T strsql.Entity] struct {
	builder *Builder[T]
	attr    strsql.Mapping[T]
}

// Rule adds a custom rule.
func (cb *ColBuilder[T]) Rule(rule ColumnRule[T], events ...Event) *ColBuilder[T] {
	if len(events) == 0 {
		events = []Event{EventBeforeInsert, EventBeforeUpdate} // default events
	}
	cb.builder.registry.colRules[cb.builder.typ] = append(cb.builder.registry.colRules[cb.builder.typ], columnRuleDef{
		attrName: cb.attr.Name(),
		attr:     cb.attr,
		rule:     rule,
		events:   events,
	})
	return cb
}

// Col chains to another column.
func (cb *ColBuilder[T]) Col(attr strsql.Attribute[T]) *ColBuilder[T] {
	return cb.builder.Col(attr)
}

// Entity chains to entity rules.
func (cb *ColBuilder[T]) Entity() *EntityBuilder[T] {
	return cb.builder.Entity()
}

// EntityBuilder provides a fluent API for adding entity-level rules.
type EntityBuilder[T strsql.Entity] struct {
	builder *Builder[T]
}

// Rule adds a custom entity-level rule.
func (eb *EntityBuilder[T]) Rule(name string, rule EntityRule[T], events ...Event) *EntityBuilder[T] {
	if len(events) == 0 {
		events = []Event{EventBeforeInsert, EventBeforeUpdate} // default events
	}
	eb.builder.registry.entRules[eb.builder.typ] = append(eb.builder.registry.entRules[eb.builder.typ], entityRuleDef{
		name:   name,
		rule:   rule,
		events: events,
	})
	return eb
}

// Col chains to a column.
func (eb *EntityBuilder[T]) Col(attr strsql.Attribute[T]) *ColBuilder[T] {
	return eb.builder.Col(attr)
}

// Entity chains to another entity rule.
func (eb *EntityBuilder[T]) Entity() *EntityBuilder[T] {
	return eb.builder.Entity()
}
