package strsql_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kcmvp/strsql"
	"github.com/kcmvp/strsql/testdata/models"
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
			builder: strsql.Select[models.Order]().
				Where(strsql.Eq(models.OrderSch.ID, "ORD-123")),
			expectedArgs: []any{"ORD-123"},
		},
		{
			name: "Select_WithoutWhere",
			builder: strsql.Select[models.Order]().
				Limit(50),
			expectedArgs: nil,
		},
		{
			name: "Select_ComplexWithLogicCombinators",
			builder: strsql.Select[models.Order]().
				Where(
					strsql.And(
						strsql.Eq(models.OrderSch.Status, 1),
						strsql.Or(
							strsql.Eq(models.OrderSch.IsPaid, true),
							strsql.In(models.OrderSch.CustomerID, "C-1", "C-2"),
						),
					),
					strsql.Lt(models.OrderSch.CreatedAt, now),
				).
				OrderBy(models.OrderSch.CreatedAt, strsql.Desc).
				Limit(10),
			expectedArgs: []any{1, true, "C-1", "C-2", now},
		},
		{
			name: "Select_WithIsNullAndIsNotNull",
			builder: strsql.Select[models.Order]().
				Where(
					strsql.IsNull(models.OrderSch.CustomerID),
					strsql.IsNotNull(models.OrderSch.Status),
				),
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

func TestUpdateBuilder(t *testing.T) {
	tests := []struct {
		name         string
		builder      strsql.Builder
		expectedArgs []any
	}{
		{
			name: "Update_Simple",
			builder: strsql.Update[models.Product]().
				Set(strsql.Set(models.ProductSch.Price, 99.99)).
				Where(strsql.Eq(models.ProductSch.ID, "P-123")),
			expectedArgs: []any{99.99, "P-123"},
		},
		{
			name: "Update_WithIncrNumAndDecrNum",
			builder: strsql.Update[models.Product]().
				Set(
					strsql.Set(models.ProductSch.Price, 100.00),
					strsql.IncrNum(models.ProductSch.Stock, 10),
					strsql.DecrNum(models.ProductSch.Stock, 2),
				).
				Where(strsql.Eq(models.ProductSch.ID, "P-123")),
			expectedArgs: []any{100.00, 10, 2, "P-123"},
		},
		{
			name: "Update_WithoutWhere",
			builder: strsql.Update[models.Product]().
				Set(strsql.Set(models.ProductSch.Stock, 0)),
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
			builder: strsql.Delete[models.OrderItem]().
				Where(strsql.Eq(models.OrderItemSch.OrderID, "ORD-123")),
			expectedArgs: []any{"ORD-123"},
		},
		{
			name:         "Delete_WithoutWhere",
			builder:      strsql.Delete[models.OrderItem](),
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
				strsql.Eq(models.OrderItemSch.Quantity, "not_an_int")
			},
		},
		{
			name: "InvalidMathOp_IncrNum_on_String_column",
			panicAction: func() {
				strsql.IncrNum(models.ProductSch.ID, 1)
			},
		},
		{
			name: "EmptyInClause_No_variadic_arguments",
			panicAction: func() {
				strsql.In(models.OrderSch.ID)
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
