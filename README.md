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
                strsql.Lt(OrderSch.OrdDate, time.Now()),
                strsql.IsNotNull(OrderSch.OrdDate),
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
- `In` (IN)
- `IsNull` (IS NULL)
- `IsNotNull` (IS NOT NULL)

**Assignments (SET clauses):**

- `Set` (column = ?)
- `IncrNum` (column = column + ?)
- `DecrNum` (column = column - ?)

**Combinators:**

- `And`
- `Or`
