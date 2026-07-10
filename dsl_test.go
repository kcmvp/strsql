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
			name: "Select_OrderByWithoutWhere",
			builder: strsql.Select[Order]().
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
					strsql.And[Order](),
					strsql.Or[Order](),
					strsql.And(strsql.Eq(OrderSch.ID, "ORD-123")),
					strsql.Or(strsql.Eq(OrderSch.Status, 1)),
				),
			expectedArgs: []any{"ORD-123", 1},
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
