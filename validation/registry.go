package validation

import (
	"reflect"
)

type columnRuleDef struct {
	attrName string
	attr     any // holds strsql.Mapping[T]
	rule     any // holds ColumnRule[T]
	events   []Event
}

type entityRuleDef struct {
	name   string
	rule   any // holds EntityRule[T]
	events []Event
}

// Registry stores validation rules for entities.
type Registry struct {
	colRules map[reflect.Type][]columnRuleDef
	entRules map[reflect.Type][]entityRuleDef
}

// NewRegistry creates a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		colRules: make(map[reflect.Type][]columnRuleDef),
		entRules: make(map[reflect.Type][]entityRuleDef),
	}
}

// AddPlugin applies a plugin to the registry.
func (r *Registry) AddPlugin(p Plugin) error {
	return p.Register(r)
}
