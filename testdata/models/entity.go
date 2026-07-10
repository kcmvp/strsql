package models

import (
	"time"

	"github.com/kcmvp/strsql"
)

// Product represents a product entity
type Product struct {
	ID    string  `db:"id"`
	Name  string  `db:"name"`
	Price float64 `db:"price"`
	Stock int     `db:"stock"`
}

func (Product) TableName() string { return "products" }

// Order represents an order entity
type Order struct {
	ID         string    `db:"id"`
	CustomerID string    `db:"customer_id"`
	Status     int       `db:"status"`
	IsPaid     bool      `db:"is_paid"`
	CreatedAt  time.Time `db:"created_at"`
}

func (Order) TableName() string { return "orders" }

// OrderItem represents an order item entity (One-to-Many with Order)
type OrderItem struct {
	ID        int     `db:"id"`
	OrderID   string  `db:"order_id"`
	ProductID string  `db:"product_id"`
	Quantity  int     `db:"quantity"`
	UnitPrice float64 `db:"unit_price"`
}

func (OrderItem) TableName() string { return "order_items" }

// InvalidModel is a struct that DOES NOT implement strsql.Entity
// This is used to test that the generator correctly ignores it.
type InvalidModel struct {
	Name string `db:"name"`
}

// Verify interface compliance at compile time
var _ strsql.Entity = Product{}
var _ strsql.Entity = Order{}
var _ strsql.Entity = OrderItem{}