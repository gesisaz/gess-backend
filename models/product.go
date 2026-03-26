package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Product struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Name          string     `json:"name" db:"name"`
	Description   string     `json:"description" db:"description"`
	Price         float64    `json:"price" db:"price"`
	StockQuantity int        `json:"stock_quantity" db:"stock_quantity"`
	CategoryID    *uuid.UUID `json:"category_id" db:"category_id"`
	BrandID       *uuid.UUID `json:"brand_id" db:"brand_id"`

	// Product identification
	SKU         string `json:"sku" db:"sku"`
	ProductLine string `json:"product_line" db:"product_line"`

	// Size information
	SizeValue float64 `json:"size_value" db:"size_value"`
	SizeUnit  string  `json:"size_unit" db:"size_unit"` // ml, oz, g, kg

	// Cosmetics-specific attributes
	Scent           string         `json:"scent" db:"scent"`
	SkinType        pq.StringArray `json:"skin_type" db:"skin_type"`           // Array: dry, oily, sensitive, normal, combination, all
	Ingredients     string         `json:"ingredients" db:"ingredients"`       // Full ingredients list
	KeyIngredients  pq.StringArray `json:"key_ingredients" db:"key_ingredients"` // Array of highlighted ingredients
	ApplicationArea string         `json:"application_area" db:"application_area"` // body, face, hands, feet, hair, nails

	// Certifications/Features
	IsOrganic     bool `json:"is_organic" db:"is_organic"`
	IsVegan       bool `json:"is_vegan" db:"is_vegan"`
	IsCrueltyFree bool `json:"is_cruelty_free" db:"is_cruelty_free"`
	IsParabenFree bool `json:"is_paraben_free" db:"is_paraben_free"`
	IsFeatured    bool `json:"is_featured" db:"is_featured"`

	// Reviews
	RatingAverage float64 `json:"rating_average" db:"rating_average"`
	ReviewCount   int     `json:"review_count" db:"review_count"`

	// Media
	ImageURL  string         `json:"image_url" db:"image_url"`
	ImageURLs pq.StringArray `json:"image_urls" db:"image_urls"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ProductWithDetails extends Product with brand and category information
type ProductWithDetails struct {
	Product
	BrandName    string `json:"brand_name" db:"brand_name"`
	CategoryName string `json:"category_name" db:"category_name"`
}

// ProductWithCategory extends Product with category information (for backward compatibility)
type ProductWithCategory struct {
	Product
	CategoryName string `json:"category_name" db:"category_name"`
	BrandName    string `json:"brand_name" db:"brand_name"`
}
