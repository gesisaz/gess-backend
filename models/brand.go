package models

import (
	"time"

	"github.com/google/uuid"
)

type Brand struct {
	ID              uuid.UUID `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	Description     string    `json:"description" db:"description"`
	LogoURL         string    `json:"logo_url" db:"logo_url"`
	WebsiteURL      string    `json:"website_url" db:"website_url"`
	CountryOfOrigin string    `json:"country_of_origin" db:"country_of_origin"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// BrandWithProductCount extends Brand with product count
type BrandWithProductCount struct {
	Brand
	ProductCount int `json:"product_count" db:"product_count"`
}
