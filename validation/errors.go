package validation

import (
	"fmt"
	"strings"
)

// ValidationError is a structured validation failure suitable for frontend display.
type ValidationError struct {
	Field   string `json:"field,omitempty"`
	Rule    string `json:"rule,omitempty"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	if e.Field != "" && e.Rule != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Field, e.Rule, e.Message)
	}
	if e.Field != "" {
		return fmt.Sprintf("[%s] %s", e.Field, e.Message)
	}
	return e.Message
}

// ValidationErrors aggregates multiple ValidationError values.
// It implements the error interface for easy propagation.
type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return "no validation errors"
	}
	msgs := make([]string, len(v))
	for i, err := range v {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "; ")
}
