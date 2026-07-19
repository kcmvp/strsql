package validation

import (
	"testing"
)

// UserRulesPlugin is an example of a plugin that registers rules for User.
type UserRulesPlugin struct{}

func (p *UserRulesPlugin) Name() string {
	return "UserRulesPlugin"
}

func (p *UserRulesPlugin) Register(r *Registry) error {
	Validate[User](r).
		Col(UserSch.Name).Required().Len(2, 50).
		Col(UserSch.Age).Min(18).Max(120).
		Col(UserSch.Role).OneOf([]any{"admin", "user"}).
		Entity().Rule("adult_guard", func(u User) error {
		if u.Age < 18 {
			return ValidationError{Message: "minor cannot be verified"}
		}
		return nil
	}, EventSave)

	return nil
}

func TestPluginRegistration(t *testing.T) {
	r := NewRegistry()
	plugin := &UserRulesPlugin{}

	err := r.AddPlugin(plugin)
	if err != nil {
		t.Fatalf("failed to add plugin: %v", err)
	}

	eng := NewEngine(r, false)
	u1 := User{
		ID:   1,
		Name: "A",
		Age:  16,
		Role: "guest",
	}

	err = eng.BeforeInsert(u1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errs := err.(ValidationErrors)
	if len(errs) != 4 {
		t.Fatalf("expected 4 errors, got %d", len(errs))
	}
}
