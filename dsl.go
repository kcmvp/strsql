package strsql

import (
	"fmt"
	"reflect"
	"strings"
)

// ============================================================================
// Core Types & Interfaces (Public)
// ============================================================================

// Of creates a new Attribute closure with the given metadata.
// Used internally by the generated schema code.
func Of[T Entity](name, column string, typ reflect.Type) Attribute[T] {
	return func() Mapping[T] {
		return mapping[T]{
			name:   name,
			column: column,
			typ:    typ,
		}
	}
}

// Entity represents a database entity. Any struct mapped to a database table
// must implement this interface to provide its corresponding table name.
type Entity interface {
	TableName() string
}

// Mapping provides metadata about a database column mapping.
// It exposes the field name, column name, and its reflect.Type.
type Mapping[T Entity] interface {
	Name() string
	Column() string
	Type() reflect.Type
	mark() seal
}

// Attribute is a closure returning the mapping metadata.
// It provides a dot-chaining experience for users (e.g., OrderSch.Id).
type Attribute[T Entity] func() Mapping[T]

// SelectCol is an interface representing a column to be selected.
// It can be a simple Attribute or an Aggregate function.
type SelectCol interface {
	SelectSQL() string
}

// Ensure Attribute implements SelectCol
func (a Attribute[T]) SelectSQL() string {
	return a().Column()
}

// AnyAttribute represents any typed attribute, used for Join condition.
type AnyAttribute interface {
	SelectCol
	Column() string
	TableName() string
	FullColumn() string
}

func (a Attribute[T]) TableName() string {
	var t T
	return t.TableName()
}

func (a Attribute[T]) FullColumn() string {
	var t T
	return t.TableName() + "." + a().Column()
}

// SQLFragment holds the generated SQL query string and its corresponding arguments.
type SQLFragment struct {
	Query string
	Args  []any
}

// Predicate is a closure that evaluates to a SQL condition fragment.
type AnyPredicate interface{ SQLFragment() SQLFragment }

type Predicate[T Entity] interface{ AnyPredicate }

type predicateImpl[T Entity] struct{ fn func() SQLFragment }

func (p predicateImpl[T]) SQLFragment() SQLFragment { return p.fn() }

// Assignment is a closure that evaluates to a SQL assignment fragment (e.g., for UPDATE).
type Assignment[T Entity] func() SQLFragment

// OrderDir defines the sorting direction.
type OrderDir string

const (
	Asc  OrderDir = "ASC"
	Desc OrderDir = "DESC"
)

// Validator is a function type used to perform fail-fast checks before generating SQL.
type Validator func(meta Mapping[Entity], vals ...any)

// ============================================================================
// Builder Traits (Public)
// ============================================================================

// Builder is the final stage of SQL construction, generating the query and args.
type Builder interface {
	Build() (string, []any)
}

// LimitTrait represents the pagination stage of query building.
type LimitTrait[T Entity] interface {
	// Limit sets the maximum number of records to return.
	// An optional offset can be provided as the second argument.
	Limit(limit int, offset ...int) Builder
	Builder
}

// OrderByTrait represents the sorting stage of query building.
type OrderByTrait[T Entity] interface {
	OrderBy(attr Attribute[T], dir OrderDir) OrderByTrait[T]
	LimitTrait[T]
}

// SelectBuilder is the entry stage of building a SELECT query.
type SelectBuilder[T Entity] interface {
	Where(preds ...AnyPredicate) OrderByTrait[T]
	OrderByTrait[T]
}

// UpdateWhereTrait represents the condition stage of an UPDATE query.
type UpdateWhereTrait[T Entity] interface {
	Where(preds ...AnyPredicate) Builder
	Builder // Allows building without Where (update all rows)
}

// UpdateBuilder is the entry stage of building an UPDATE query, forcing the SET clause.
type UpdateBuilder[T Entity] interface {
	Set(assignments ...Assignment[T]) UpdateWhereTrait[T]
}

// DeleteBuilder is the entry stage of building a DELETE query.
type DeleteBuilder[T Entity] interface {
	Where(preds ...AnyPredicate) Builder
	Builder // Allows building without Where (delete all rows)
}

// ============================================================================
// CRUD Entry Points (Public)
// ============================================================================

// Select initializes a SELECT query builder for the given Entity.
// If no columns are provided, it defaults to SELECT *.
func Select[T Entity](cols ...SelectCol) SelectBuilder[T] {
	return &selectStatement[T]{cols: cols}
}

// Update initializes an UPDATE query builder for the given Entity.
func Update[T Entity]() UpdateBuilder[T] {
	return &updateStatement[T]{}
}

// Delete initializes a DELETE query builder for the given Entity.
func Delete[T Entity]() DeleteBuilder[T] {
	return &deleteStatement[T]{}
}

// ============================================================================
// Aggregate Functions (Public)
// ============================================================================

type aggregateFunc struct {
	expr string
}

func (a aggregateFunc) SelectSQL() string {
	return a.expr
}

// Count constructs a COUNT(column) aggregate.
// If no attribute is provided, it defaults to COUNT(*).
func Count[T Entity](attrs ...Attribute[T]) SelectCol {
	if len(attrs) == 0 {
		return aggregateFunc{expr: "COUNT(*)"}
	}
	return aggregateFunc{expr: fmt.Sprintf("COUNT(%s)", attrs[0]().Column())}
}

// Sum constructs a SUM(column) aggregate.
// It will fail-fast if the column is not numeric.
func Sum[T Entity](attr Attribute[T]) SelectCol {
	validateNumeric(attr(), nil) // val is nil because we only check the column type
	return aggregateFunc{expr: fmt.Sprintf("SUM(%s)", attr().Column())}
}

// Max constructs a MAX(column) aggregate.
func Max[T Entity](attr Attribute[T]) SelectCol {
	return aggregateFunc{expr: fmt.Sprintf("MAX(%s)", attr().Column())}
}

// Min constructs a MIN(column) aggregate.
func Min[T Entity](attr Attribute[T]) SelectCol {
	return aggregateFunc{expr: fmt.Sprintf("MIN(%s)", attr().Column())}
}

// Avg constructs an AVG(column) aggregate.
// It will fail-fast if the column is not numeric.
func Avg[T Entity](attr Attribute[T]) SelectCol {
	validateNumeric(attr(), nil)
	return aggregateFunc{expr: fmt.Sprintf("AVG(%s)", attr().Column())}
}

// ============================================================================
// Assignments (Public)
// ============================================================================

// Set constructs an assignment expression: column = ?
func Set[T Entity](attr Attribute[T], val any) Assignment[T] {
	return wrapAssignment(attr, func(col string) string {
		return fmt.Sprintf("%s = ?", col)
	}, []any{val}, validateType)
}

// IncrNum constructs an increment assignment expression: column = column + ?
func IncrNum[T Entity](attr Attribute[T], val any) Assignment[T] {
	return wrapAssignment(attr, func(col string) string {
		return fmt.Sprintf("%s = %s + ?", col, col)
	}, []any{val}, validateType, validateNumeric)
}

// DecrNum constructs a decrement assignment expression: column = column - ?
func DecrNum[T Entity](attr Attribute[T], val any) Assignment[T] {
	return wrapAssignment(attr, func(col string) string {
		return fmt.Sprintf("%s = %s - ?", col, col)
	}, []any{val}, validateType, validateNumeric)
}

// ============================================================================
// Predicates & Logic Combinators (Public)
// ============================================================================

// Eq constructs an equality condition: column = ?
func Eq[T Entity](attr Attribute[T], val any) Predicate[T] {
	return wrapOp(attr, func(col string) string {
		return fmt.Sprintf("%s = ?", col)
	}, []any{val}, validateType)
}

// NotEq constructs an inequality condition: column <> ?
func NotEq[T Entity](attr Attribute[T], val any) Predicate[T] {
	return wrapOp(attr, func(col string) string {
		return fmt.Sprintf("%s <> ?", col)
	}, []any{val}, validateType)
}

// Gt constructs a greater-than condition: column > ?
func Gt[T Entity](attr Attribute[T], val any) Predicate[T] {
	return wrapOp(attr, func(col string) string {
		return fmt.Sprintf("%s > ?", col)
	}, []any{val}, validateType)
}

// Gte constructs a greater-than-or-equal condition: column >= ?
func Gte[T Entity](attr Attribute[T], val any) Predicate[T] {
	return wrapOp(attr, func(col string) string {
		return fmt.Sprintf("%s >= ?", col)
	}, []any{val}, validateType)
}

// Lt constructs a less-than condition: column < ?
func Lt[T Entity](attr Attribute[T], val any) Predicate[T] {
	return wrapOp(attr, func(col string) string {
		return fmt.Sprintf("%s < ?", col)
	}, []any{val}, validateType)
}

// Lte constructs a less-than-or-equal condition: column <= ?
func Lte[T Entity](attr Attribute[T], val any) Predicate[T] {
	return wrapOp(attr, func(col string) string {
		return fmt.Sprintf("%s <= ?", col)
	}, []any{val}, validateType)
}

// Like constructs a LIKE condition: column LIKE ?
func Like[T Entity](attr Attribute[T], val any) Predicate[T] {
	return wrapOp(attr, func(col string) string {
		return fmt.Sprintf("%s LIKE ?", col)
	}, []any{val}, validateType)
}

// NotLike constructs a NOT LIKE condition: column NOT LIKE ?
func NotLike[T Entity](attr Attribute[T], val any) Predicate[T] {
	return wrapOp(attr, func(col string) string {
		return fmt.Sprintf("%s NOT LIKE ?", col)
	}, []any{val}, validateType)
}

// IsNull constructs an IS NULL condition: column IS NULL
func IsNull[T Entity](attr Attribute[T]) Predicate[T] {
	return wrapOp(attr, func(col string) string {
		return fmt.Sprintf("%s IS NULL", col)
	}, nil) // IS NULL does not take any arguments
}

// IsNotNull constructs an IS NOT NULL condition: column IS NOT NULL
func IsNotNull[T Entity](attr Attribute[T]) Predicate[T] {
	return wrapOp(attr, func(col string) string {
		return fmt.Sprintf("%s IS NOT NULL", col)
	}, nil) // IS NOT NULL does not take any arguments
}

// In constructs an IN condition: column IN (?, ?, ?)
func In[T Entity](attr Attribute[T], vals ...any) Predicate[T] {
	return wrapOp(attr, func(col string) string {
		placeholders := make([]string, len(vals))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		return fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", "))
	}, vals, validateNotEmpty, validateType)
}

// NotIn constructs a NOT IN condition: column NOT IN (?, ?, ?)
func NotIn[T Entity](attr Attribute[T], vals ...any) Predicate[T] {
	return wrapOp(attr, func(col string) string {
		placeholders := make([]string, len(vals))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		return fmt.Sprintf("%s NOT IN (%s)", col, strings.Join(placeholders, ", "))
	}, vals, validateNotEmpty, validateType)
}

// Between constructs a BETWEEN condition: column BETWEEN ? AND ?
func Between[T Entity](attr Attribute[T], start, end any) Predicate[T] {
	return wrapOp(attr, func(col string) string {
		return fmt.Sprintf("%s BETWEEN ? AND ?", col)
	}, []any{start, end}, validateType)
}

// And combines multiple predicates with logical AND: (cond1 AND cond2 AND ...)
func And(preds ...AnyPredicate) AnyPredicate {
	return predicateImpl[Entity]{fn: func() SQLFragment {
		if len(preds) == 0 {
			return SQLFragment{}
		}

		var queries []string
		var args []any

		for _, p := range preds {
			frag := p.SQLFragment()
			if frag.Query != "" {
				queries = append(queries, frag.Query)
				args = append(args, frag.Args...)
			}
		}

		if len(queries) == 1 {
			return SQLFragment{Query: queries[0], Args: args}
		}

		return SQLFragment{
			Query: fmt.Sprintf("(%s)", strings.Join(queries, " AND ")),
			Args:  args,
		}
	}}
}

// Or combines multiple predicates with logical OR: (cond1 OR cond2 OR ...)
func Or(preds ...AnyPredicate) AnyPredicate {
	return predicateImpl[Entity]{fn: func() SQLFragment {
		if len(preds) == 0 {
			return SQLFragment{}
		}

		var queries []string
		var args []any

		for _, p := range preds {
			frag := p.SQLFragment()
			if frag.Query != "" {
				queries = append(queries, frag.Query)
				args = append(args, frag.Args...)
			}
		}

		if len(queries) == 1 {
			return SQLFragment{Query: queries[0], Args: args}
		}

		return SQLFragment{
			Query: fmt.Sprintf("(%s)", strings.Join(queries, " OR ")),
			Args:  args,
		}
	}}
}

// JoinType defines the type of SQL JOIN.
type JoinType string

const (
	InnerJoin JoinType = "INNER JOIN"
	LeftJoin  JoinType = "LEFT JOIN"
	RightJoin JoinType = "RIGHT JOIN"
)

// JoinContext provides a context for building queries with joined tables.
type JoinContext[T Entity] interface {
	Join(joinType JoinType, leftAttr AnyAttribute, rightAttr AnyAttribute) JoinContext[T]
	Select(cols ...SelectCol) SelectBuilder[T]
}

func WithJoin[S, T Entity](sAttr Attribute[S], tAttr Attribute[T]) JoinContext[S] {
	ctx := &joinContext[S]{}
	return ctx.Join(InnerJoin, sAttr, tAttr)
}

func WithLeftJoin[S, T Entity](sAttr Attribute[S], tAttr Attribute[T]) JoinContext[S] {
	ctx := &joinContext[S]{}
	return ctx.Join(LeftJoin, sAttr, tAttr)
}

func WithRightJoin[S, T Entity](sAttr Attribute[S], tAttr Attribute[T]) JoinContext[S] {
	ctx := &joinContext[S]{}
	return ctx.Join(RightJoin, sAttr, tAttr)
}

type joinContext[S Entity] struct {
	joins []string
}

func (c *joinContext[S]) Join(joinType JoinType, leftAttr AnyAttribute, rightAttr AnyAttribute) JoinContext[S] {
	c.joins = append(c.joins, string(joinType)+" "+rightAttr.TableName()+" ON "+leftAttr.FullColumn()+" = "+rightAttr.FullColumn())
	return c
}

func (c *joinContext[S]) Select(cols ...SelectCol) SelectBuilder[S] {
	return &selectStatement[S]{
		joins: c.joins,
		cols:  cols,
	}
}

// ============================================================================
// Private Types & Implementations
// ============================================================================

// seal is used to ensure the Mapping interface cannot be implemented outside this package.
type seal struct{}

// mapping is the internal implementation of the Mapping interface.
type mapping[T Entity] struct {
	name   string
	column string
	typ    reflect.Type
}

// Column implements [Mapping].
func (m mapping[T]) Column() string {
	return m.column
}

// Name implements [Mapping].
func (m mapping[T]) Name() string {
	return m.name
}

// Type implements [Mapping].
func (m mapping[T]) Type() reflect.Type {
	return m.typ
}

// mark implements [Mapping].
func (m mapping[T]) mark() seal {
	return seal{}
}

// selectStatement implements the SelectBuilder and its related traits.
type selectStatement[T Entity] struct {
	joins  []string
	cols   []SelectCol
	wheres []SQLFragment
	orders []string
	limit  int
	offset int
}

func (s *selectStatement[T]) Where(preds ...AnyPredicate) OrderByTrait[T] {
	for _, p := range preds {
		frag := p.SQLFragment()
		if frag.Query != "" {
			s.wheres = append(s.wheres, frag)
		}
	}
	return s
}

func (s *selectStatement[T]) OrderBy(attr Attribute[T], dir OrderDir) OrderByTrait[T] {
	meta := attr()
	s.orders = append(s.orders, fmt.Sprintf("%s %s", meta.Column(), dir))
	return s
}

func (s *selectStatement[T]) Limit(limit int, offset ...int) Builder {
	s.limit = limit
	if len(offset) > 0 {
		s.offset = offset[0]
	}
	return s
}

func (s *selectStatement[T]) Build() (string, []any) {
	var queryBuilder strings.Builder
	var finalArgs []any

	model := *new(T)

	if len(s.cols) == 0 {
		if len(s.joins) > 0 {
			fmt.Fprintf(&queryBuilder, "SELECT %s.* FROM %s", model.TableName(), model.TableName())
		} else {
			fmt.Fprintf(&queryBuilder, "SELECT * FROM %s", model.TableName())
		}
	} else {
		var colNames []string
		for _, col := range s.cols {
			colNames = append(colNames, col.SelectSQL())
		}
		fmt.Fprintf(&queryBuilder, "SELECT %s FROM %s", strings.Join(colNames, ", "), model.TableName())
	}

	if len(s.joins) > 0 {
		queryBuilder.WriteString(" ")
		queryBuilder.WriteString(strings.Join(s.joins, " "))
	}

	if len(s.wheres) > 0 {
		queryBuilder.WriteString(" WHERE ")
		var whereQueries []string
		for _, w := range s.wheres {
			whereQueries = append(whereQueries, w.Query)
			finalArgs = append(finalArgs, w.Args...)
		}
		queryBuilder.WriteString(strings.Join(whereQueries, " AND "))
	}

	if len(s.orders) > 0 {
		queryBuilder.WriteString(" ORDER BY ")
		queryBuilder.WriteString(strings.Join(s.orders, ", "))
	}

	if s.limit > 0 {
		fmt.Fprintf(&queryBuilder, " LIMIT %d", s.limit)
		if s.offset > 0 {
			fmt.Fprintf(&queryBuilder, " OFFSET %d", s.offset)
		}
	}

	return queryBuilder.String(), finalArgs
}

// updateStatement implements the UpdateBuilder and its related traits.
type updateStatement[T Entity] struct {
	sets   []SQLFragment
	wheres []SQLFragment
}

func (u *updateStatement[T]) Set(assignments ...Assignment[T]) UpdateWhereTrait[T] {
	for _, a := range assignments {
		u.sets = append(u.sets, a())
	}
	return u
}

func (u *updateStatement[T]) Where(preds ...AnyPredicate) Builder {
	for _, p := range preds {
		frag := p.SQLFragment()
		if frag.Query != "" {
			u.wheres = append(u.wheres, frag)
		}
	}
	return u
}

func (u *updateStatement[T]) Build() (string, []any) {
	var queryBuilder strings.Builder
	var finalArgs []any

	model := *new(T)
	fmt.Fprintf(&queryBuilder, "UPDATE %s SET ", model.TableName())

	var setQueries []string
	for _, s := range u.sets {
		setQueries = append(setQueries, s.Query)
		finalArgs = append(finalArgs, s.Args...)
	}
	queryBuilder.WriteString(strings.Join(setQueries, ", "))

	if len(u.wheres) > 0 {
		queryBuilder.WriteString(" WHERE ")
		var whereQueries []string
		for _, w := range u.wheres {
			whereQueries = append(whereQueries, w.Query)
			finalArgs = append(finalArgs, w.Args...)
		}
		queryBuilder.WriteString(strings.Join(whereQueries, " AND "))
	}

	return queryBuilder.String(), finalArgs
}

// deleteStatement implements the DeleteBuilder and its related traits.
type deleteStatement[T Entity] struct {
	wheres []SQLFragment
}

func (d *deleteStatement[T]) Where(preds ...AnyPredicate) Builder {
	for _, p := range preds {
		frag := p.SQLFragment()
		if frag.Query != "" {
			d.wheres = append(d.wheres, frag)
		}
	}
	return d
}

func (d *deleteStatement[T]) Build() (string, []any) {
	var queryBuilder strings.Builder
	var finalArgs []any

	model := *new(T)
	fmt.Fprintf(&queryBuilder, "DELETE FROM %s", model.TableName())

	if len(d.wheres) > 0 {
		queryBuilder.WriteString(" WHERE ")
		var whereQueries []string
		for _, w := range d.wheres {
			whereQueries = append(whereQueries, w.Query)
			finalArgs = append(finalArgs, w.Args...)
		}
		queryBuilder.WriteString(strings.Join(whereQueries, " AND "))
	}

	return queryBuilder.String(), finalArgs
}

// ============================================================================
// Private Validation Helpers
// ============================================================================

// validateType ensures all provided values match the expected field type.
func validateType(meta Mapping[Entity], vals ...any) {
	for _, val := range vals {
		valType := reflect.TypeOf(val)
		if valType != meta.Type() {
			panic(fmt.Sprintf("Fail-Fast: Type mismatch for column '%s'. Expected %s, but got %T", meta.Column(), meta.Type(), val))
		}
	}
}

// validateNumeric ensures the field type is numeric (used for Incr/Decr operations).
func validateNumeric(meta Mapping[Entity], vals ...any) {
	typ := meta.Type()
	kind := typ.Kind()

	// Dereference pointer to get the underlying kind
	if kind == reflect.Pointer {
		kind = typ.Elem().Kind()
	}

	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return
	default:
		panic(fmt.Sprintf("Fail-Fast: Mathematical operations (IncrNum/DecrNum) are only allowed on numeric types. Column '%s' is of type %s", meta.Column(), typ))
	}
}

// validateNotEmpty ensures the variadic arguments are not empty (used for IN clauses).
func validateNotEmpty(meta Mapping[Entity], vals ...any) {
	if len(vals) == 0 {
		panic(fmt.Sprintf("Fail-Fast: condition for column '%s' requires at least one value", meta.Column()))
	}
}

// wrapOp is a higher-order function that extracts metadata, runs validators,
// and returns the SQLFragment closure.
func wrapAssignment[T Entity](
	attr Attribute[T],
	queryFn func(column string) string,
	vals []any,
	validators ...Validator,
) Assignment[T] {
	meta := attr()
	m := meta.(Mapping[Entity])

	for _, validate := range validators {
		validate(m, vals...)
	}

	return func() SQLFragment {
		return SQLFragment{
			Query: queryFn(m.Column()),
			Args:  vals,
		}
	}
}

func wrapOp[T Entity](
	attr Attribute[T],
	queryFn func(column string) string,
	vals []any,
	validators ...Validator,
) Predicate[T] {
	meta := attr()
	m := meta.(Mapping[Entity])

	// Run all fail-fast validations
	for _, validate := range validators {
		validate(m, vals...)
	}

	// Return the final closure
	return predicateImpl[T]{
		fn: func() SQLFragment {
			return SQLFragment{
				Query: queryFn(m.Column()),
				Args:  vals,
			}
		},
	}
}

func (a Attribute[T]) Column() string {
	return a().Column()
}
