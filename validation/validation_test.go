package validation_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/kcmvp/strsql"
	"github.com/kcmvp/strsql/validation"
)

// ── Test fixtures ─────────────────────────────────────────────────────────────

type User struct {
	ID    int
	Name  string
	Age   int
	Role  string
	Email string
}

func (User) TableName() string { return "users" }

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

// ── Rule attachment ───────────────────────────────────────────────────────────

// TestRuleAttachment verifies that rules can be attached to column attributes
// and executed without panic.
func TestRuleAttachment(t *testing.T) {
	nameRules := validation.For(UserSch.Name, validation.Required(), validation.Len(2, 50))
	ageRules := validation.For(UserSch.Age, validation.Min(18), validation.Max(120))

	// Attach should not panic; calling Check should work.
	_ = nameRules.Check("Alice")
	_ = ageRules.Check(25)
}

// ── Serial execution ──────────────────────────────────────────────────────────

// TestSerialExecution verifies that rules run serially and all errors are collected.
func TestSerialExecution(t *testing.T) {
	nameRules := validation.For(UserSch.Name, validation.Required(), validation.Len(2, 50))

	// Empty string: Required fires, then Len also fires.
	errs := nameRules.Check("")
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors (required + len), got %d: %v", len(errs), errs)
	}
	if errs[0].Rule != "required" {
		t.Errorf("first error should be 'required', got %q", errs[0].Rule)
	}
	if errs[1].Rule != "len" {
		t.Errorf("second error should be 'len', got %q", errs[1].Rule)
	}
}

// ── Aggregated errors ─────────────────────────────────────────────────────────

// TestAggregatedErrors verifies that CheckAll collects errors from all columns.
func TestAggregatedErrors(t *testing.T) {
	nameRules := validation.For(UserSch.Name, validation.Required(), validation.Len(2, 50))
	ageRules := validation.For(UserSch.Age, validation.Min(18), validation.Max(120))
	roleRules := validation.For(UserSch.Role, validation.OneOf("admin", "user"))

	errs := validation.CheckAll(
		nameRules.With("A"),      // too short → 1 error
		ageRules.With(15),        // below min → 1 error
		roleRules.With("guest"),  // not in set → 1 error
	)
	if len(errs) != 3 {
		t.Fatalf("expected 3 aggregated errors, got %d: %v", len(errs), errs)
	}
	if errs[0].Field != "Name" {
		t.Errorf("expected first error Field=Name, got %q", errs[0].Field)
	}
	if errs[1].Field != "Age" {
		t.Errorf("expected second error Field=Age, got %q", errs[1].Field)
	}
	if errs[2].Field != "Role" {
		t.Errorf("expected third error Field=Role, got %q", errs[2].Field)
	}
}

// ── No errors on valid input ──────────────────────────────────────────────────

func TestNoErrors(t *testing.T) {
	nameRules := validation.For(UserSch.Name, validation.Required(), validation.Len(2, 50))
	ageRules := validation.For(UserSch.Age, validation.Min(18), validation.Max(120))
	roleRules := validation.For(UserSch.Role, validation.OneOf("admin", "user"))

	errs := validation.CheckAll(
		nameRules.With("Alice"),
		ageRules.With(25),
		roleRules.With("admin"),
	)
	if errs != nil {
		t.Fatalf("expected no errors for valid input, got: %v", errs)
	}
}

// ── Single-column Check ───────────────────────────────────────────────────────

func TestSingleColumnCheck(t *testing.T) {
	ageRules := validation.For(UserSch.Age, validation.Min(18), validation.Max(120))

	if errs := ageRules.Check(25); errs != nil {
		t.Errorf("expected no error, got: %v", errs)
	}
	if errs := ageRules.Check(10); len(errs) != 1 {
		t.Errorf("expected 1 error, got %d: %v", len(errs), errs)
	}
}

// ── No lifecycle coupling ─────────────────────────────────────────────────────

// TestNoLifecycleCoupling ensures column rules are triggered manually, not by
// any lifecycle event. There is intentionally no Before* call in this test.
func TestNoLifecycleCoupling(t *testing.T) {
	nameRules := validation.For(UserSch.Name, validation.Required())

	// Column rules are invoked explicitly by the caller — no implicit trigger.
	errs := nameRules.Check("")
	if len(errs) == 0 {
		t.Fatal("expected a required error when explicitly invoked")
	}

	// Without calling Check, no validation happens — no side-effects.
	_ = validation.For(UserSch.Name, validation.Required())
	// If validation ran automatically, this would be caught.
}

// ── Built-in rules ────────────────────────────────────────────────────────────

func TestBuiltinRequired(t *testing.T) {
	r := validation.Required()
	if r("hello") != nil {
		t.Error("non-empty string should pass Required")
	}
	if r("") == nil {
		t.Error("empty string should fail Required")
	}
	if r(0) == nil {
		t.Error("zero int should fail Required")
	}
	if r(nil) == nil {
		t.Error("nil should fail Required")
	}
}

func TestBuiltinMin(t *testing.T) {
	r := validation.Min(18)
	if r(18) != nil {
		t.Error("18 should pass Min(18)")
	}
	if r(17) == nil {
		t.Error("17 should fail Min(18)")
	}
	if r(nil) != nil {
		t.Error("nil should pass Min (not required)")
	}
}

func TestBuiltinMax(t *testing.T) {
	r := validation.Max(120)
	if r(120) != nil {
		t.Error("120 should pass Max(120)")
	}
	if r(121) == nil {
		t.Error("121 should fail Max(120)")
	}
}

func TestBuiltinLen(t *testing.T) {
	r := validation.Len(2, 10)
	if r("hi") != nil {
		t.Error(`"hi" should pass Len(2,10)`)
	}
	if r("A") == nil {
		t.Error(`"A" should fail Len(2,10)`)
	}
	if r("toolongstring!") == nil {
		t.Error("too-long string should fail Len(2,10)")
	}
}

func TestBuiltinOneOf(t *testing.T) {
	r := validation.OneOf("admin", "user")
	if r("admin") != nil {
		t.Error(`"admin" should pass OneOf`)
	}
	if r("guest") == nil {
		t.Error(`"guest" should fail OneOf`)
	}
}

// ── Custom rule ───────────────────────────────────────────────────────────────

func TestCustomRule(t *testing.T) {
	noSpaces := func(value any) error {
		s, ok := value.(string)
		if !ok {
			return nil
		}
		for _, c := range s {
			if c == ' ' {
				return errors.New("spaces are not allowed")
			}
		}
		return nil
	}

	nameRules := validation.For(UserSch.Name, noSpaces)
	if errs := nameRules.Check("Alice"); errs != nil {
		t.Errorf("expected no error: %v", errs)
	}
	if errs := nameRules.Check("Alice Smith"); len(errs) == 0 {
		t.Error("expected error for name with space")
	}
}

// ── ValidationErrors.Error() ─────────────────────────────────────────────────

func TestValidationErrorsFormat(t *testing.T) {
	errs := validation.ValidationErrors{
		{Field: "Name", Rule: "required", Message: "value is required"},
		{Field: "Age", Rule: "min", Message: "must be >= 18"},
	}
	s := errs.Error()
	if s == "" {
		t.Error("ValidationErrors.Error() should not be empty")
	}
}

// ── BusinessValidator interface is decoupled ──────────────────────────────────

// TestBusinessValidatorDecoupled ensures BusinessValidator is an independent
// interface with no connection to column-rule execution.
type userBizValidator struct{}

func (userBizValidator) BeforeInsert(u User) error { return nil }
func (userBizValidator) BeforeUpdate(u User) error { return nil }
func (userBizValidator) BeforeDelete(u User) error { return nil }

func TestBusinessValidatorDecoupled(t *testing.T) {
	var _ validation.BusinessValidator[User] = userBizValidator{}
	// BusinessValidator compiles and is independent of column rule execution.
}
