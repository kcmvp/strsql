package validation

import (
	"fmt"
	"strings"
)

// ValidationError represents a single validation failure.
type ValidationError struct {
	Entity  string
	Field   string
	Rule    string
	Code    string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("[%s.%s] %s: %s", e.Entity, e.Field, e.Rule, e.Message)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Entity, e.Rule, e.Message)
}

// ValidationErrors aggregates multiple ValidationError instances.
type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return "no validation errors"
	}
	var msgs []string
	for _, err := range v {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}
