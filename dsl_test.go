package strsql_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kcmvp/strsql"
	. "github.com/kcmvp/strsql/testdata/models"
)

func loadSQL(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", name+".sql")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read sql file %s: %v", path, err)
	}
	return strings.TrimSpace(string(b))
}

func TestSelectBuilder(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		builder      strsql.Builder
		expectedArgs []any
	}{
		{
			name: "Select_Simple",
			builder: strsql.Select[Order]().
				Where(strsql.Eq(OrderSch.ID, "ORD-123")),
			expectedArgs: []any{"ORD-123"},
		},
		{
			name: "Select_WithoutWhere",
			builder: strsql.Select[Order]().
				Limit(50),
			expectedArgs: nil,
		},
		{
			name: "Select_SpecificColumns",
			builder: strsql.Select[Product](ProductSch.ID, ProductSch.Name).
				Where(strsql.Eq(ProductSch.Price, 99.99)),
			expectedArgs: []any{99.99},
		},
		{
			name: "Select_AggregateFunctions",
			builder: strsql.Select[Order](
				strsql.Count[Order](),
				strsql.Count(OrderSch.ID),
				strsql.Sum(OrderSch.Status),
				strsql.Max(OrderSch.CreatedAt),
			).Where(strsql.Eq(OrderSch.IsPaid, true)),
			expectedArgs: []any{true},
		},
		{
			name: "Select_OrderByWithoutWhere",
			builder: strsql.Select[Order]().
				OrderBy(OrderSch.Status, strsql.Asc).
				OrderBy(OrderSch.CreatedAt, strsql.Desc),
			expectedArgs: nil,
		},
		{
			name: "Select_LimitWithoutWhereOrOrderBy",
			builder: strsql.Select[Order]().
				Limit(5),
			expectedArgs: nil,
		},
		{
			name: "Select_LimitAndOffset",
			builder: strsql.Select[Order]().
				Limit(10, 20),
			expectedArgs: nil,
		},
		{
			name:         "Select_All",
			builder:      strsql.Select[Order](),
			expectedArgs: nil,
		},
		{
			name: "Select_ComplexWithLogicCombinators",
			builder: strsql.Select[Order]().
				Where(
					strsql.And(
						strsql.Eq(OrderSch.Status, 1),
						strsql.Or(
							strsql.Eq(OrderSch.IsPaid, true),
							strsql.In(OrderSch.CustomerID, "C-1", "C-2"),
						),
						strsql.NotEq(OrderSch.Status, 0),
						strsql.Gt(OrderSch.Status, -1),
						strsql.Gte(OrderSch.Status, 1),
						strsql.Lte(OrderSch.Status, 99),
						strsql.Like(OrderSch.CustomerID, "C-%"),
					),
					strsql.Lt(OrderSch.CreatedAt, now),
				).
				OrderBy(OrderSch.CreatedAt, strsql.Desc).
				Limit(10),
			expectedArgs: []any{1, true, "C-1", "C-2", 0, -1, 1, 99, "C-%", now},
		},
		{
			name: "Select_WithBetweenNotInNotLike",
			builder: strsql.Select[Order]().
				Where(
					strsql.Between(OrderSch.Status, 1, 5),
					strsql.NotIn(OrderSch.CustomerID, "C-99", "C-100"),
					strsql.NotLike(OrderSch.CustomerID, "TEST-%"),
				),
			expectedArgs: []any{1, 5, "C-99", "C-100", "TEST-%"},
		},
		{
			name: "Select_WithIsNullAndIsNotNull",
			builder: strsql.Select[Order]().
				Where(
					strsql.IsNull(OrderSch.CustomerID),
					strsql.IsNotNull(OrderSch.Status),
				),
			expectedArgs: nil,
		},
		{
			name: "Select_WithEmptyAndOr",
			builder: strsql.Select[Order]().
				Where(
					strsql.And(),
					strsql.Or(),
					strsql.And(strsql.Eq(OrderSch.ID, "ORD-123")),
					strsql.Or(strsql.Eq(OrderSch.Status, 1)),
				),
			expectedArgs: []any{"ORD-123", 1},
		},
		{
			name: "Select_TableAliasOnly",
			builder: func() strsql.Builder {
				o := strsql.Alias[Order]("o")
				return strsql.Select[Order]().
					As(o.Name()).
					Where(strsql.Eq(o.Col(OrderSch.ID), "ORD-123"))
			}(),
			expectedArgs: []any{"ORD-123"},
		},
		{
			name: "Select_ColumnAlias",
			builder: strsql.Select[Product](
				strsql.AsCol(ProductSch.ID, "product_id"),
				strsql.AsCol(ProductSch.Name, "product_name"),
			).Where(strsql.Eq(ProductSch.ID, "P-1")),
			expectedArgs: []any{"P-1"},
		},
		{
			name:         "Select_AggregateAlias",
			builder:      strsql.Select[Order](strsql.AsCol(strsql.Count[Order](), "cnt")),
			expectedArgs: nil,
		},
		{
			name: "Select_Join_InnerSimple",
			builder: func() strsql.Builder {
				o := strsql.Alias[Order]("o")
				oi := strsql.Alias[OrderItem]("oi")
				return strsql.Select[Order]().
					As(o.Name()).
					InnerJoin(
						oi.Ref(),
						strsql.ColEq(o.Col(OrderSch.ID), oi.Col(OrderItemSch.OrderID)),
					).
					Where(strsql.Eq(oi.Col(OrderItemSch.ProductID), "P-1"))
			}(),
			expectedArgs: []any{"P-1"},
		},
		{
			name: "Select_Join_MultiJoinWithColumns",
			builder: func() strsql.Builder {
				o := strsql.Alias[Order]("o")
				oi := strsql.Alias[OrderItem]("oi")
				p := strsql.Alias[Product]("p")
				return strsql.Select[Order](
					o.Col(OrderSch.ID),
					p.Col(ProductSch.Name),
				).
					As(o.Name()).
					InnerJoin(
						oi.Ref(),
						strsql.ColEq(o.Col(OrderSch.ID), oi.Col(OrderItemSch.OrderID)),
					).
					InnerJoin(
						p.Ref(),
						strsql.ColEq(oi.Col(OrderItemSch.ProductID), p.Col(ProductSch.ID)),
					).
					Where(strsql.Gt(p.Col(ProductSch.Price), 10.0)).
					OrderBy(o.Col(OrderSch.CreatedAt), strsql.Desc).
					Limit(10)
			}(),
			expectedArgs: []any{10.0},
		},
		{
			name: "Select_Join_WithOr",
			builder: func() strsql.Builder {
				o := strsql.Alias[Order]("o")
				oi := strsql.Alias[OrderItem]("oi")
				return strsql.Select[Order]().
					As(o.Name()).
					InnerJoin(
						oi.Ref(),
						strsql.ColEq(o.Col(OrderSch.ID), oi.Col(OrderItemSch.OrderID)),
					).
					Where(
						strsql.Or(
							strsql.Eq(oi.Col(OrderItemSch.ProductID), "P-1"),
							strsql.Eq(oi.Col(OrderItemSch.ProductID), "P-2"),
						),
					).
					Limit(1)
			}(),
			expectedArgs: []any{"P-1", "P-2"},
		},
		{
			name: "Select_Join_RightJoin",
			builder: func() strsql.Builder {
				o := strsql.Alias[Order]("o")
				oi := strsql.Alias[OrderItem]("oi")
				return strsql.Select[Order]().
					As(o.Name()).
					RightJoin(
						oi.Ref(),
						strsql.ColEq(o.Col(OrderSch.ID), oi.Col(OrderItemSch.OrderID)),
					).
					Where(strsql.Eq(oi.Col(OrderItemSch.ProductID), "P-1")).
					Limit(1)
			}(),
			expectedArgs: []any{"P-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedSQL := loadSQL(t, tt.name)
			sql, args := tt.builder.Build()
			if sql != expectedSQL {
				t.Errorf("\nExpected SQL: %s\nGot SQL     : %s", expectedSQL, sql)
			}
			if len(args) == 0 && len(tt.expectedArgs) == 0 {
				// Both empty, it's fine
			} else if !reflect.DeepEqual(args, tt.expectedArgs) {
				t.Errorf("\nExpected args: %v\nGot args     : %v", tt.expectedArgs, args)
			}
		})
	}
}

func TestUpdateBuilder(t *testing.T) {
	tests := []struct {
		name         string
		builder      strsql.Builder
		expectedArgs []any
	}{
		{
			name: "Update_Simple",
			builder: strsql.Update[Product]().
				Set(strsql.Set(ProductSch.Price, 99.99)).
				Where(strsql.Eq(ProductSch.ID, "P-123")),
			expectedArgs: []any{99.99, "P-123"},
		},
		{
			name: "Update_WithIncrNumAndDecrNum",
			builder: strsql.Update[Product]().
				Set(
					strsql.Set(ProductSch.Price, 100.00),
					strsql.IncrNum(ProductSch.Stock, 10),
					strsql.DecrNum(ProductSch.Stock, 2),
				).
				Where(strsql.Eq(ProductSch.ID, "P-123")),
			expectedArgs: []any{100.00, 10, 2, "P-123"},
		},
		{
			name: "Update_WithoutWhere",
			builder: strsql.Update[Product]().
				Set(strsql.Set(ProductSch.Stock, 0)),
			expectedArgs: []any{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedSQL := loadSQL(t, tt.name)
			sql, args := tt.builder.Build()
			if sql != expectedSQL {
				t.Errorf("\nExpected SQL: %s\nGot SQL     : %s", expectedSQL, sql)
			}
			if len(args) == 0 && len(tt.expectedArgs) == 0 {
				// Both empty, it's fine
			} else if !reflect.DeepEqual(args, tt.expectedArgs) {
				t.Errorf("\nExpected args: %v\nGot args     : %v", tt.expectedArgs, args)
			}
		})
	}
}

func TestDeleteBuilder(t *testing.T) {
	tests := []struct {
		name         string
		builder      strsql.Builder
		expectedArgs []any
	}{
		{
			name: "Delete_Simple",
			builder: strsql.Delete[OrderItem]().
				Where(strsql.Eq(OrderItemSch.OrderID, "ORD-123")),
			expectedArgs: []any{"ORD-123"},
		},
		{
			name:         "Delete_WithoutWhere",
			builder:      strsql.Delete[OrderItem](),
			expectedArgs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedSQL := loadSQL(t, tt.name)
			sql, args := tt.builder.Build()
			if sql != expectedSQL {
				t.Errorf("\nExpected SQL: %s\nGot SQL     : %s", expectedSQL, sql)
			}
			if len(args) == 0 && len(tt.expectedArgs) == 0 {
				// Both empty, it's fine
			} else if !reflect.DeepEqual(args, tt.expectedArgs) {
				t.Errorf("\nExpected args: %v\nGot args     : %v", tt.expectedArgs, args)
			}
		})
	}
}

func TestFailFastPanics(t *testing.T) {
	tests := []struct {
		name        string
		panicAction func()
	}{
		{
			name: "TypeMismatch_String_passed_to_Int_column",
			panicAction: func() {
				strsql.Eq(OrderItemSch.Quantity, "not_an_int")
			},
		},
		{
			name: "InvalidMathOp_IncrNum_on_String_column",
			panicAction: func() {
				// We need to bypass the `validateType` check first to hit the `default`
				// branch in `validateNumeric`. The easiest way is to pass a string to a string column.
				// Product.ID is a string column, and we pass a string value.
				// validateType will pass, but validateNumeric will panic because it's a string column.
				strsql.IncrNum(ProductSch.ID, "1")
			},
		},
		{
			name: "InvalidAggregate_Sum_on_String_column",
			panicAction: func() {
				strsql.Sum(ProductSch.ID)
			},
		},
		{
			name: "InvalidAggregate_Avg_on_String_column",
			panicAction: func() {
				strsql.Avg(ProductSch.Name)
			},
		},
		{
			name: "InvalidSelect_MixedEntityColumns",
			panicAction: func() {
				strsql.Select[Product](OrderSch.ID).Build()
			},
		},
		{
			name: "InvalidAggregate_Count_with_multiple_columns",
			panicAction: func() {
				strsql.Count(OrderSch.ID, OrderSch.CustomerID)
			},
		},
		{
			name: "EmptyInClause_No_variadic_arguments",
			panicAction: func() {
				strsql.In(OrderSch.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic in test '%s', but execution continued normally", tt.name)
				}
			}()
			tt.panicAction()
		})
	}
}
