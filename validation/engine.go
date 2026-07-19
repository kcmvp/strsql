package validation

import (
	"context"
	"reflect"

	"github.com/kcmvp/strsql"
)

// Engine executes validations.
type Engine struct {
	registry *Registry
	failFast bool
}

// NewEngine creates a new validation engine.
func NewEngine(registry *Registry, failFast bool) *Engine {
	return &Engine{
		registry: registry,
		failFast: failFast,
	}
}

// BeforeInsert validates an entity for insertion.
func (e *Engine) BeforeInsert(entity strsql.Entity) error {
	return e.Validate(context.Background(), entity, EventBeforeInsert)
}

// BeforeUpdate validates an entity for update. It accepts optional fields for partial validation.
func (e *Engine) BeforeUpdate(entity strsql.Entity, fields ...string) error {
	return e.Validate(context.Background(), entity, EventBeforeUpdate, fields...)
}

// BeforeDelete validates an entity for deletion.
func (e *Engine) BeforeDelete(entity strsql.Entity) error {
	return e.Validate(context.Background(), entity, EventBeforeDelete)
}

// Validate executes all applicable rules for the entity and event.
// If fields are provided, it only validates those fields (partial validation).
func (e *Engine) Validate(ctx context.Context, entity strsql.Entity, event Event, fields ...string) error {
	var errs ValidationErrors
	typ := reflect.TypeOf(entity)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	rc := RuleContext{Context: ctx, Event: event, Meta: map[string]any{}}

	fieldSet := make(map[string]bool)
	for _, f := range fields {
		fieldSet[f] = true
	}
	partial := len(fields) > 0

	// Entity Name (usually struct name)
	entityName := typ.Name()

	// 1. Column rules
	if crs, ok := e.registry.colRules[typ]; ok {
		val := reflect.ValueOf(entity)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		for _, cr := range crs {
			if !hasEvent(cr.events, event) {
				continue
			}
			if partial && !fieldSet[cr.attrName] {
				continue
			}

			fieldVal := val.FieldByName(cr.attrName)
			var fv any
			if fieldVal.IsValid() {
				fv = fieldVal.Interface()
			}

			fnVal := reflect.ValueOf(cr.rule)
			res := fnVal.Call([]reflect.Value{
				reflect.ValueOf(rc),
				reflect.ValueOf(cr.attr),
				reflect.ValueOf(fv),
			})

			if !res[0].IsNil() {
				err := res[0].Interface().(error)
				if valErr, ok := err.(ValidationError); ok {
					valErr.Entity = entityName
					valErr.Field = cr.attrName
					errs = append(errs, valErr)
				} else {
					errs = append(errs, ValidationError{
						Entity:  entityName,
						Field:   cr.attrName,
						Message: err.Error(),
					})
				}
				if e.failFast {
					return errs
				}
			}
		}
	}

	// 2. Entity rules
	if !partial {
		if ers, ok := e.registry.entRules[typ]; ok {
			for _, er := range ers {
				if !hasEvent(er.events, event) {
					continue
				}

				fnVal := reflect.ValueOf(er.rule)
				res := fnVal.Call([]reflect.Value{reflect.ValueOf(rc), reflect.ValueOf(entity)})

				if !res[0].IsNil() {
					err := res[0].Interface().(error)
					if valErr, ok := err.(ValidationError); ok {
						valErr.Entity = entityName
						valErr.Rule = er.name
						errs = append(errs, valErr)
					} else {
						errs = append(errs, ValidationError{
							Entity:  entityName,
							Rule:    er.name,
							Message: err.Error(),
						})
					}
					if e.failFast {
						return errs
					}
				}
			}
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func hasEvent(events []Event, target Event) bool {
	for _, e := range events {
		if e == target {
			return true
		}
	}
	return false
}
