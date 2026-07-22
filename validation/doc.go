// Package validation provides a DSL-based, pluginable validation framework
// for strsql. It supports both column-level and entity-level validations.
//
// Features:
// - DSL-first API for registering validation rules.
// - Support for both Fail-Fast and Collect-All error reporting modes.
// - Event-based validation lifecycle (e.g., EventSave, EventUpdate).
// - Plugin architecture for modular rule registration.
//
// Example Usage:
//
//	r := validation.NewRegistry()
//
//	// Register rules using the DSL builder
//	validation.Validate[User](r).
//		Col(UserSch.Name).Required().Len(2, 50).
//		Col(UserSch.Age).Min(18).Max(120).
//		Entity().Rule("adult_guard", func(u User) error {
//			if u.Age < 18 {
//				return validation.ValidationError{Message: "minor cannot be verified"}
//			}
//			return nil
//		}, validation.EventSave)
//
//	// Create an engine (failFast = false)
//	eng := validation.NewEngine(r, false)
//
//	// Execute validation before insert
//	err := eng.BeforeInsert(userInstance)
//	if err != nil {
//		// err is of type validation.ValidationErrors
//	}
//
// # Plugins
//
// You can group rules into plugins and register them easily:
//
//	type UserPlugin struct{}
//	func (p *UserPlugin) Name() string { return "UserPlugin" }
//	func (p *UserPlugin) Register(r *validation.Registry) error {
//		validation.Validate[User](r).Col(UserSch.Name).Required()
//		return nil
//	}
//
//	r.AddPlugin(&UserPlugin{})
package validation
