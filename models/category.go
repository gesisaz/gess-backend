package models

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	Name             string     `json:"name" db:"name"`
	Description      string     `json:"description" db:"description"`
	ParentCategoryID *uuid.UUID `json:"parent_category_id" db:"parent_category_id"`
	DisplayOrder     int        `json:"display_order" db:"display_order"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// CategoryWithParent includes parent category information
type CategoryWithParent struct {
	Category
	ParentCategoryName string `json:"parent_category_name,omitempty" db:"parent_category_name"`
}

// CategoryTree represents hierarchical category structure
type CategoryTree struct {
	Category
	Children []CategoryTree `json:"children,omitempty"`
}