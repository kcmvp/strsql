package validation

import (
	"reflect"
	"testing"

	"github.com/kcmvp/strsql"
)

// Dummy entity for testing
type User struct {
	ID    int
	Name  string
	Age   int
	Role  string
	Email string
}

func (u User) TableName() string { return "users" }

var UserSch = struct {
	ID    strsql.Attribute[User]
	Name  strsql.Attribute[User]
	Age   strsql.Attribute[User]
	Role  strsql.Attribute[User]
	Email strsql.Attribute[User]
}{
	ID:    strsql.Of[User]("ID", "id", reflect.TypeOf(0)),
	Name:  strsql.Of[User]("Name", "name", reflect.TypeOf("")),
	Age:   strsql.Of[User]("Age", "age", reflect.TypeOf(0)),
	Role:  strsql.Of[User]("Role", "role", reflect.TypeOf("")),
	Email: strsql.Of[User]("Email", "email", reflect.TypeOf("")),
}

func TestValidation(t *testing.T) {
	r := NewRegistry()

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

	eng := NewEngine(r, false)

	// Test 1: invalid user
	u1 := User{
		ID:   1,
		Name: "A",
		Age:  16,
		Role: "guest",
	}

	err := eng.Validate(u1, EventSave)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errs := err.(ValidationErrors)
	if len(errs) != 4 { // Name Len, Age Min, Role OneOf, Entity adult_guard
		t.Fatalf("expected 4 errors, got %d: %v", len(errs), errs)
	}

	// Test 2: valid user
	u2 := User{
		ID:   2,
		Name: "Alice",
		Age:  25,
		Role: "admin",
	}

	err = eng.Validate(u2, EventSave)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	
	// Test 3: partial validation
	u3 := User{
		ID:   3,
		Name: "Bob",
		Age:  15, // invalid age, but we only validate Name
		Role: "guest", // invalid role, but we only validate Name
	}
	err = eng.Validate(u3, EventUpdate, "Name")
	if err != nil {
		t.Fatalf("expected no error on partial validate, got %v", err)
	}
	
	u4 := User{
		Name: "A", // invalid len
	}
	err = eng.Validate(u4, EventUpdate, "Name")
	if err == nil {
		t.Fatalf("expected error on partial validate for Name")
	}
}
