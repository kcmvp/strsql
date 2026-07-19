package validation

import "github.com/kcmvp/strsql"

// BusinessValidator defines business-layer lifecycle hooks for an entity.
// These hooks (BeforeInsert / BeforeUpdate / BeforeDelete) are intentionally
// decoupled from column-level rules: column rules are explicit/manual triggered
// by the caller, while business hooks are tied to persistence operations and
// must be implemented by the application.
type BusinessValidator[T strsql.Entity] interface {
	BeforeInsert(entity T) error
	BeforeUpdate(entity T) error
	BeforeDelete(entity T) error
}
