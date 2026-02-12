package models

import (
	"time"

	"github.com/google/uuid"
)

type Cart struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CartItem struct {
	ID        uuid.UUID `json:"id" db:"id"`
	CartID    uuid.UUID `json:"cart_id" db:"cart_id"`
	ProductID uuid.UUID `json:"product_id" db:"product_id"`
	Quantity  int       `json:"quantity" db:"quantity"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CartItemWithProduct extends CartItem with product information
type CartItemWithProduct struct {
	CartItem
	ProductName  string  `json:"product_name" db:"product_name"`
	ProductPrice float64 `json:"product_price" db:"product_price"`
	ProductImage string  `json:"product_image" db:"product_image"`
}

// CartResponse represents a cart with its items for API responses
type CartResponse struct {
	Cart  Cart                  `json:"cart"`
	Items []CartItemWithProduct `json:"items"`
	Total float64               `json:"total"`
}

