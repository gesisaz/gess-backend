package models

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Name          string     `json:"name" db:"name"`
	Description   string     `json:"description" db:"description"`
	Price         float64    `json:"price" db:"price"`
	StockQuantity int        `json:"stock_quantity" db:"stock_quantity"`
	CategoryID    *uuid.UUID `json:"category_id" db:"category_id"`
	ImageURL      string     `json:"image_url" db:"image_url"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// ProductWithCategory extends Product with category information
type ProductWithCategory struct {
	Product
	CategoryName string `json:"category_name" db:"category_name"`
}

