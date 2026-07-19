// Package validation provides a minimal, function-first column rule carrier
// for strsql.
//
// # Design Principles
//
//   - Column rules are explicit/manual triggered by the caller. They are never
//     automatically invoked by any lifecycle event.
//   - Rules are plain functions (Rule = func(value any) error). No heavy OOP,
//     no plugin registry, no event bus.
//   - Rules execute serially in the order they are defined (pipeline style).
//   - Errors are aggregated across all columns (collect-all behavior) and
//     returned as ValidationErrors for easy frontend consumption.
//   - Lifecycle hooks (BeforeInsert / BeforeUpdate / BeforeDelete) are
//     business-layer concerns only. Implement BusinessValidator to add
//     persistence-event logic; this is entirely separate from column rules.
//
// # Quick Start
//
//	import "github.com/kcmvp/strsql/validation"
//
//	// 1. Attach rules to columns (column rules are not lifecycle-bound).
//	nameRules := validation.For(UserSch.Name, validation.Required(), validation.Len(2, 50))
//	ageRules  := validation.For(UserSch.Age,  validation.Min(18), validation.Max(120))
//	roleRules := validation.For(UserSch.Role, validation.OneOf("admin", "user"))
//
//	// 2. Explicitly trigger validation for a set of column/value pairs.
//	errs := validation.CheckAll(
//	    nameRules.With("Alice"),
//	    ageRules.With(25),
//	    roleRules.With("admin"),
//	)
//	if errs != nil {
//	    // errs is ValidationErrors — a []ValidationError suitable for JSON.
//	}
//
//	// 3. Or validate a single column:
//	errs = nameRules.Check("A")   // returns errors for the Name column only
//
// # Business-layer Hooks
//
// Implement BusinessValidator[T] for persistence-event validation.
// This is intentionally kept separate from column rules:
//
//	type UserValidator struct{}
//	func (v UserValidator) BeforeInsert(u User) error { /* business logic */ return nil }
//	func (v UserValidator) BeforeUpdate(u User) error { /* business logic */ return nil }
//	func (v UserValidator) BeforeDelete(u User) error { /* business logic */ return nil }
package validation
