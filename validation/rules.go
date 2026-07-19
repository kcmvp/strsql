package validation

import (
	"fmt"
	"reflect"

	"github.com/kcmvp/strsql"
)

// Required adds a rule that the column value must not be its zero value.
func (cb *ColBuilder[T]) Required(events ...Event) *ColBuilder[T] {
	return cb.Rule(func(ctx RuleContext, attr strsql.Mapping[T], val any) error {
		if val == nil {
			return ValidationError{Rule: "Required", Message: "value is required"}
		}
		v := reflect.ValueOf(val)
		if v.IsZero() {
			return ValidationError{Rule: "Required", Message: "value is required"}
		}
		return nil
	}, events...)
}

// Min adds a numeric min rule.
func (cb *ColBuilder[T]) Min(min float64, events ...Event) *ColBuilder[T] {
	return cb.Rule(func(ctx RuleContext, attr strsql.Mapping[T], val any) error {
		if val == nil {
			return nil // delegate to Required if needed
		}
		v := reflect.ValueOf(val)
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if float64(v.Int()) < min {
				return ValidationError{Rule: "Min", Message: fmt.Sprintf("must be >= %v", min)}
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if float64(v.Uint()) < min {
				return ValidationError{Rule: "Min", Message: fmt.Sprintf("must be >= %v", min)}
			}
		case reflect.Float32, reflect.Float64:
			if v.Float() < min {
				return ValidationError{Rule: "Min", Message: fmt.Sprintf("must be >= %v", min)}
			}
		}
		return nil
	}, events...)
}

// Max adds a numeric max rule.
func (cb *ColBuilder[T]) Max(max float64, events ...Event) *ColBuilder[T] {
	return cb.Rule(func(ctx RuleContext, attr strsql.Mapping[T], val any) error {
		if val == nil {
			return nil
		}
		v := reflect.ValueOf(val)
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if float64(v.Int()) > max {
				return ValidationError{Rule: "Max", Message: fmt.Sprintf("must be <= %v", max)}
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if float64(v.Uint()) > max {
				return ValidationError{Rule: "Max", Message: fmt.Sprintf("must be <= %v", max)}
			}
		case reflect.Float32, reflect.Float64:
			if v.Float() > max {
				return ValidationError{Rule: "Max", Message: fmt.Sprintf("must be <= %v", max)}
			}
		}
		return nil
	}, events...)
}

// Len adds a string/slice length rule.
func (cb *ColBuilder[T]) Len(min, max int, events ...Event) *ColBuilder[T] {
	return cb.Rule(func(ctx RuleContext, attr strsql.Mapping[T], val any) error {
		if val == nil {
			return nil
		}
		v := reflect.ValueOf(val)
		if v.Kind() == reflect.String || v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			l := v.Len()
			if l < min || l > max {
				return ValidationError{Rule: "Len", Message: fmt.Sprintf("length must be between %d and %d", min, max)}
			}
		}
		return nil
	}, events...)
}

// OneOf adds an enum rule.
func (cb *ColBuilder[T]) OneOf(allowed []any, events ...Event) *ColBuilder[T] {
	return cb.Rule(func(ctx RuleContext, attr strsql.Mapping[T], val any) error {
		if val == nil {
			return nil
		}
		for _, a := range allowed {
			if reflect.DeepEqual(val, a) {
				return nil
			}
		}
		return ValidationError{Rule: "OneOf", Message: "value is not allowed"}
	}, events...)
}
