package validation

import (
	"fmt"
	"reflect"

	"github.com/kcmvp/strsql"
)

// Rule is a column validation function.
// Rules are pure functions: they receive the column value and return an error or nil.
//
// Column rules are always explicitly triggered by the caller. They are never
// bound to persistence lifecycle events — that is the responsibility of
// BusinessValidator hooks (BeforeInsert / BeforeUpdate / BeforeDelete).
type Rule func(value any) error

// ColumnRules[T] carries one or more validation rules for a column attribute.
// Rules are executed serially in the order they are defined (pipeline style).
// Use For to create a ColumnRules and Check or CheckAll to execute them.
type ColumnRules[T strsql.Entity] struct {
	attr  strsql.Attribute[T]
	rules []Rule
}

// For attaches one or more validation rules to a column attribute.
// The returned ColumnRules is stateless and can be reused across calls.
func For[T strsql.Entity](attr strsql.Attribute[T], rules ...Rule) ColumnRules[T] {
	return ColumnRules[T]{attr: attr, rules: append([]Rule(nil), rules...)}
}

// Check explicitly triggers all rules for the given value.
// All rule errors are collected and returned (collect-all behavior).
// Returns nil if all rules pass.
func (c ColumnRules[T]) Check(value any) ValidationErrors {
	errs := runRules(c.attr().Name(), c.rules, value)
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// With binds a runtime value to this ColumnRules, producing a colCheck
// that can be passed to CheckAll.
func (c ColumnRules[T]) With(value any) colCheck {
	return colCheck{field: c.attr().Name(), rules: c.rules, value: value}
}

// colCheck is an internal pair of (field, rules, value) used by CheckAll.
type colCheck struct {
	field string
	rules []Rule
	value any
}

// CheckAll explicitly triggers column rules for multiple column/value pairs and
// aggregates all errors (collect-all behavior). This is the primary entry point
// for validating a set of columns in a single call.
//
// Column rules are always manually triggered; lifecycle coupling is not used here.
// See BusinessValidator for persistence-event hooks.
func CheckAll(pairs ...colCheck) ValidationErrors {
	var errs ValidationErrors
	for _, p := range pairs {
		errs = append(errs, runRules(p.field, p.rules, p.value)...)
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// runRules executes a slice of rules against value and returns all errors.
func runRules(field string, rules []Rule, value any) ValidationErrors {
	var errs ValidationErrors
	for _, rule := range rules {
		if err := rule(value); err != nil {
			if ve, ok := err.(ValidationError); ok {
				ve.Field = field
				errs = append(errs, ve)
			} else {
				errs = append(errs, ValidationError{Field: field, Message: err.Error()})
			}
		}
	}
	return errs
}

// ============================================================================
// Built-in Rule Constructors
// ============================================================================

// Required returns a Rule that fails when the value is nil or the zero value
// for its type (e.g., 0, "", false).
func Required() Rule {
	return func(value any) error {
		if value == nil {
			return ValidationError{Rule: "required", Message: "value is required"}
		}
		if reflect.ValueOf(value).IsZero() {
			return ValidationError{Rule: "required", Message: "value is required"}
		}
		return nil
	}
}

// Min returns a Rule that fails when the numeric value is less than min.
// Non-numeric types are skipped.
func Min(min float64) Rule {
	return func(value any) error {
		if value == nil {
			return nil
		}
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if float64(v.Int()) < min {
				return ValidationError{Rule: "min", Message: fmt.Sprintf("must be >= %v", min)}
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if float64(v.Uint()) < min {
				return ValidationError{Rule: "min", Message: fmt.Sprintf("must be >= %v", min)}
			}
		case reflect.Float32, reflect.Float64:
			if v.Float() < min {
				return ValidationError{Rule: "min", Message: fmt.Sprintf("must be >= %v", min)}
			}
		}
		return nil
	}
}

// Max returns a Rule that fails when the numeric value is greater than max.
// Non-numeric types are skipped.
func Max(max float64) Rule {
	return func(value any) error {
		if value == nil {
			return nil
		}
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if float64(v.Int()) > max {
				return ValidationError{Rule: "max", Message: fmt.Sprintf("must be <= %v", max)}
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if float64(v.Uint()) > max {
				return ValidationError{Rule: "max", Message: fmt.Sprintf("must be <= %v", max)}
			}
		case reflect.Float32, reflect.Float64:
			if v.Float() > max {
				return ValidationError{Rule: "max", Message: fmt.Sprintf("must be <= %v", max)}
			}
		}
		return nil
	}
}

// Len returns a Rule that fails when the length of a string, slice, or array
// falls outside the [minLen, maxLen] range (inclusive).
func Len(minLen, maxLen int) Rule {
	return func(value any) error {
		if value == nil {
			return nil
		}
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.String, reflect.Slice, reflect.Array:
			l := v.Len()
			if l < minLen || l > maxLen {
				return ValidationError{
					Rule:    "len",
					Message: fmt.Sprintf("length must be between %d and %d", minLen, maxLen),
				}
			}
		}
		return nil
	}
}

// OneOf returns a Rule that fails when the value is not in the allowed set.
func OneOf(allowed ...any) Rule {
	return func(value any) error {
		if value == nil {
			return nil
		}
		for _, a := range allowed {
			if reflect.DeepEqual(value, a) {
				return nil
			}
		}
		return ValidationError{Rule: "oneOf", Message: "value is not in the allowed set"}
	}
}
