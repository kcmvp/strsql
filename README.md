<p align="center">
  A highly opinionated, strictly-typed, and functional SQL DSL generator for Go.
  <br/>
  <br/>
  <a href="https://github.com/kcmvp/strsql/blob/main/LICENSE">
    <img alt="GitHub" src="https://img.shields.io/github/license/kcmvp/strsql"/>
  </a>
  <a href="https://pkg.go.dev/github.com/kcmvp/strsql">
    <img src="https://pkg.go.dev/badge/github.com/kcmvp/strsql.svg" alt="Go Reference"/>
  </a>
  <a href="https://goreportcard.com/report/github.com/kcmvp/strsql">
    <img src="https://goreportcard.com/badge/github.com/kcmvp/strsql" alt="report"/>
  </a>
  <a href="https://github.com/kcmvp/strsql/blob/main/.github/workflows/ci.yml" rel="nofollow">
     <img src="https://img.shields.io/github/actions/workflow/status/kcmvp/strsql/ci.yml?branch=main" alt="Build" />
  </a>
  <a href="https://app.codecov.io/gh/kcmvp/strsql" ref="nofollow">
    <img src ="https://img.shields.io/codecov/c/github/kcmvp/strsql" alt="coverage"/>
  </a>
</p>

# strsql

`strsql` is a highly opinionated, strictly-typed, and functional SQL DSL generator for Go.

It completely eliminates magic strings and runtime type errors by leveraging AST code generation and a Type-Safe Builder (Trait pattern), providing a Java-like dot-chaining autocompletion experience right inside your Go IDE.

**Core Philosophy: Constructing correct and safe SQL.**
This library focuses _only_ on safely generating SQL queries and arguments. It is entirely agnostic to your underlying database driver (whether you use `database/sql`, `sqlx`, or any other executor).

## Features

- **Zero Magic Strings**: Code generation extracts your struct `db` tags to create safe mapping Singletons (e.g., `OrderSch.Id`).
- **Fail-Fast Type Checking**: Pass an `int` to a `string` column? It panics _before_ the SQL is even generated.
- **Type-Safe Builder (Traits)**: Compile-time constraints ensure your SQL is syntactically valid (e.g., an `UPDATE` must have a `SET` clause).
- **Pure Functional Logic**: Say goodbye to ambiguous `A.And(B).Or(C)` chains. We use functional combinators like `strsql.And(A, B)` for crystal clear precedence.
- **No Driver Dependencies**: It outputs standard SQL strings and `[]any` arguments, ready to be fed into any database executor.

## Installation

```bash
# Install the core library
go get github.com/kcmvp/strsql

# Install the code generator CLI globally
go install github.com/kcmvp/strsql/cmd/strsql@latest
```

## Quick Start

### 1. Define your Entities

Your structs must implement the `strsql.Entity` interface (providing a `TableName() string` method). Use struct tags (default is `db`) to specify column names.

```go
package models

import (
	"time"
	"github.com/kcmvp/strsql"
)

type Order struct {
	Id      string    `db:"order_id"`
	OrdDate time.Time `db:"creation_date"`
}

// Implement the strsql.Entity interface
func (Order) TableName() string {
	return "orders"
}

// Ensure interface compliance
var _ strsql.Entity = Order{}
```

### 2. Generate Schema Mappings

Use the built-in CLI tool to parse your structs and generate the type-safe mapping code.

```bash
# Generate schema for the ./models directory (using global installation)
strsql gen ./models

# Or using go run directly from the module
go run github.com/kcmvp/strsql/cmd/strsql gen ./models

# You can also specify custom struct tags (e.g., gorm)
go run github.com/kcmvp/strsql/cmd/strsql gen -t gorm ./models
```

This generates a `schema_gen.go` file containing Singletons like `OrderSch`, which expose your columns as closures.

### 3. Build Type-Safe SQL

Enjoy the flawless IDE autocompletion and compile-time/runtime safety.

#### SELECT

```go
import "github.com/kcmvp/strsql"

// Default: SELECT *
sql, args := strsql.Select[Order]().
    Where(
        strsql.Eq(OrderSch.Id, "ORD-12345"),
        strsql.In(OrderSch.Id, "O1", "O2"), // Variadic arguments are automatically treated as AND
    ).
    OrderBy(OrderSch.OrdDate, strsql.Desc).
    Limit(10).
    Build()

// sql:  SELECT * FROM orders WHERE order_id = ? AND order_id IN (?, ?) ORDER BY creation_date DESC LIMIT 10
// args: [ORD-12345 O1 O2]

// Specific Columns & Aggregate Functions
sql, args = strsql.Select[Order](
    OrderSch.Id,
    strsql.Count(OrderSch.Id),
    strsql.Sum(OrderSch.Status),
).Where(strsql.Eq(OrderSch.IsPaid, true)).Build()
```

#### Pagination (Limit & Offset)

The `Limit` method accepts an optional second argument for the `OFFSET`.

```go
sql, args := strsql.Select[Order]().
    OrderBy(OrderSch.OrdDate, strsql.Desc).
    Limit(10, 20). // LIMIT 10 OFFSET 20
    Build()
```

#### Complex Logic Combinators (And / Or)

Functional combinators eliminate precedence ambiguity.

```go
sql, args := strsql.Select[Order]().
    Where(
        strsql.Or(
            strsql.Eq(OrderSch.Id, "O-A"),
            strsql.And(
                strsql.NotEq(OrderSch.Id, "O-B"),
                strsql.Like(OrderSch.Id, "%O-C%"),
                strsql.Between(OrderSch.Status, 1, 5),
                strsql.NotIn(OrderSch.Id, "O-X", "O-Y"),
            ),
        ),
    ).
    Limit(1).
    Build()
```

#### UPDATE (with strict lifecycle)

The compiler forces you to call `Set` before `Build`.

```go
sql, args := strsql.Update[Order]().
    Set(
        strsql.Set(OrderSch.OrdDate, time.Now()),
    ).
    Where(strsql.Eq(OrderSch.Id, "ORD-12345")).
    Build()
```

#### Math Operations (IncrNum / DecrNum)

Safely increment/decrement numeric columns. The library will fail-fast if you try this on a `string` column!

```go
sql, args := strsql.Update[OrderItem]().
    Set(
        strsql.IncrNum(OrderItemSch.Qty, 5),
        strsql.DecrNum(OrderItemSch.Qty, 2),
    ).
    Where(strsql.Eq(OrderItemSch.PrdId, "P-100")).
    Build()
```

#### DELETE

```go
sql, args := strsql.Delete[Order]().
    Where(strsql.Eq(OrderSch.Id, "ORD-12345")).
    Build()
```

## Supported Operators

**Predicates (WHERE clauses):**

- `Eq` (=)
- `NotEq` (<>)
- `Gt` (>)
- `Gte` (>=)
- `Lt` (<)
- `Lte` (<=)
- `Like` (LIKE)
- `NotLike` (NOT LIKE)
- `In` (IN)
- `NotIn` (NOT IN)
- `Between` (BETWEEN ? AND ?)
- `IsNull` (IS NULL)
- `IsNotNull` (IS NOT NULL)

**Aggregate Functions:**

- `Count` (COUNT(col) or COUNT(\*))
- `Sum` (SUM(col))
- `Max` (MAX(col))
- `Min` (MIN(col))
- `Avg` (AVG(col))

**Assignments (SET clauses):**

- `Set` (column = ?)
- `IncrNum` (column = column + ?)
- `DecrNum` (column = column - ?)

**Combinators:**

- `And`
- `Or`

## Column Validation (`validation` package)

The `validation` package provides a minimal, function-first rule carrier for column attributes.

> **Design note:** Column rules are **always explicitly/manually triggered** by the caller. They are never bound to any persistence lifecycle event. Business-layer events (`BeforeInsert` / `BeforeUpdate` / `BeforeDelete`) are kept entirely separate via the `BusinessValidator` interface.

### Attach rules to columns

```go
import "github.com/kcmvp/strsql/validation"

nameRules := validation.For(OrderSch.Id,   validation.Required(), validation.Len(1, 64))
priceRules := validation.For(ProductSch.Price, validation.Min(0))
roleRules  := validation.For(UserSch.Role, validation.OneOf("admin", "user"))
```

Rules are plain `func(value any) error` functions, executed serially in the order they are defined.

### Explicitly trigger validation (manual, never automatic)

```go
// Validate a single column:
errs := nameRules.Check(someValue)

// Validate multiple columns in one call (collect-all errors):
errs = validation.CheckAll(
    nameRules.With(req.Name),
    priceRules.With(req.Price),
    roleRules.With(req.Role),
)
if errs != nil {
    // errs is validation.ValidationErrors — a []ValidationError for frontend display.
}
```

### Built-in rules

| Rule | Description |
|------|-------------|
| `Required()` | Fails on nil or zero value |
| `Min(n)` | Numeric lower bound |
| `Max(n)` | Numeric upper bound |
| `Len(min, max)` | String / slice length range |
| `OneOf(vals...)` | Enum allowlist |

Custom rules are just functions matching `func(value any) error`.

### Business-layer hooks (separate from column rules)

Implement `validation.BusinessValidator[T]` to add persistence-event logic. This interface is **entirely decoupled** from column-rule execution:

```go
type OrderValidator struct{}

func (v OrderValidator) BeforeInsert(o Order) error { /* business logic */ return nil }
func (v OrderValidator) BeforeUpdate(o Order) error { /* business logic */ return nil }
func (v OrderValidator) BeforeDelete(o Order) error { /* business logic */ return nil }
```
