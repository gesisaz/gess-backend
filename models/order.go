package models

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusRefunded   OrderStatus = "refunded"
)

type Order struct {
	ID                uuid.UUID   `json:"id" db:"id"`
	UserID            uuid.UUID   `json:"user_id" db:"user_id"`
	TotalAmount       float64     `json:"total_amount" db:"total_amount"`
	Status            OrderStatus `json:"status" db:"status"`
	ShippingAddressID uuid.UUID   `json:"shipping_address_id" db:"shipping_address_id"`
	CreatedAt         time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at" db:"updated_at"`
}

type OrderItem struct {
	ID              uuid.UUID `json:"id" db:"id"`
	OrderID         uuid.UUID `json:"order_id" db:"order_id"`
	ProductID       uuid.UUID `json:"product_id" db:"product_id"`
	Quantity        int       `json:"quantity" db:"quantity"`
	PriceAtPurchase float64   `json:"price_at_purchase" db:"price_at_purchase"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// OrderWithItems represents an order with its items for API responses
type OrderWithItems struct {
	Order           Order       `json:"order"`
	Items           []OrderItem `json:"items"`
	ShippingAddress Address     `json:"shipping_address"`
}

