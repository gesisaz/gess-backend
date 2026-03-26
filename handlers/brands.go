package handlers

import (
	"gess-backend/database"
	"gess-backend/models"
	"gess-backend/utils"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

// CreateBrandRequest represents the request body for creating a brand
type CreateBrandRequest struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	LogoURL         string `json:"logo_url"`
	WebsiteURL      string `json:"website_url"`
	CountryOfOrigin string `json:"country_of_origin"`
}

// UpdateBrandRequest represents the request body for updating a brand
type UpdateBrandRequest struct {
	Name            *string `json:"name,omitempty"`
	Description     *string `json:"description,omitempty"`
	LogoURL         *string `json:"logo_url,omitempty"`
	WebsiteURL      *string `json:"website_url,omitempty"`
	CountryOfOrigin *string `json:"country_of_origin,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
}

// ListBrandsHandler handles GET /brands - List all active brands
func ListBrandsHandler(w http.ResponseWriter, r *http.Request) {
	// Query parameter to include inactive brands (admin only)
	includeInactive := r.URL.Query().Get("include_inactive") == "true"

	query := `
		SELECT b.id, b.name, 
		       COALESCE(b.description, '') as description,
		       COALESCE(b.logo_url, '') as logo_url,
		       COALESCE(b.website_url, '') as website_url, 
		       COALESCE(b.country_of_origin, '') as country_of_origin,
		       b.is_active, b.created_at, b.updated_at,
		       COUNT(p.id) as product_count
		FROM brands b
		LEFT JOIN products p ON b.id = p.brand_id
	`

	if !includeInactive {
		query += ` WHERE b.is_active = true`
	}

	query += `
		GROUP BY b.id, b.name, b.description, b.logo_url, b.website_url, 
		         b.country_of_origin, b.is_active, b.created_at, b.updated_at
		ORDER BY b.name
	`

	rows, err := database.DB.Query(query)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch brands")
		return
	}
	defer rows.Close()

	brands := []models.BrandWithProductCount{}
	for rows.Next() {
		var brand models.BrandWithProductCount
		err := rows.Scan(
			&brand.ID, &brand.Name, &brand.Description, &brand.LogoURL,
			&brand.WebsiteURL, &brand.CountryOfOrigin, &brand.IsActive,
			&brand.CreatedAt, &brand.UpdatedAt, &brand.ProductCount,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to scan brands")
			return
		}
		brands = append(brands, brand)
	}

	utils.RespondJSON(w, http.StatusOK, brands)
}

// GetBrandHandler handles GET /brands/:id - Get brand with its products
func GetBrandHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Brand ID is required")
		return
	}
	brandID := pathParts[len(pathParts)-1]

	// Validate UUID
	id, err := utils.ValidateUUID(brandID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Query brand
	var brand models.Brand
	err = database.DB.QueryRow(`
		SELECT id, name, description, logo_url, website_url, 
		       country_of_origin, is_active, created_at, updated_at
		FROM brands
		WHERE id = $1
	`, id).Scan(
		&brand.ID, &brand.Name, &brand.Description, &brand.LogoURL,
		&brand.WebsiteURL, &brand.CountryOfOrigin, &brand.IsActive,
		&brand.CreatedAt, &brand.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Brand not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch brand")
		return
	}

	// Query products for this brand
	productsQuery := `
		SELECT p.id, p.name, p.description, p.price, p.stock_quantity,
		       p.category_id, p.brand_id, p.sku, p.product_line,
		       p.size_value, p.size_unit, p.scent, p.skin_type,
		       p.ingredients, p.key_ingredients, p.application_area,
		       p.is_organic, p.is_vegan, p.is_cruelty_free, p.is_paraben_free, p.is_featured,
		       p.rating_average, p.review_count, p.image_url,
		       COALESCE(p.image_urls, '{}'::text[]),
		       p.created_at, p.updated_at,
		       COALESCE(c.name, '') as category_name
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE p.brand_id = $1
		ORDER BY p.created_at DESC
	`

	rows, err := database.DB.Query(productsQuery, id)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch products")
		return
	}
	defer rows.Close()

	products := []models.ProductWithCategory{}
	for rows.Next() {
		var product models.ProductWithCategory
		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.Price,
			&product.StockQuantity, &product.CategoryID, &product.BrandID,
			&product.SKU, &product.ProductLine, &product.SizeValue, &product.SizeUnit,
			&product.Scent, &product.SkinType, &product.Ingredients, &product.KeyIngredients,
			&product.ApplicationArea, &product.IsOrganic, &product.IsVegan,
			&product.IsCrueltyFree, &product.IsParabenFree, &product.IsFeatured,
			&product.RatingAverage, &product.ReviewCount, &product.ImageURL,
			&product.ImageURLs,
			&product.CreatedAt, &product.UpdatedAt, &product.CategoryName,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to scan products")
			return
		}
		products = append(products, product)
	}

	// Build response
	response := map[string]interface{}{
		"brand":    brand,
		"products": products,
	}

	utils.RespondJSON(w, http.StatusOK, response)
}

// CreateBrandHandler handles POST /admin/brands - Create brand (Admin only)
func CreateBrandHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateBrandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Validate required fields
	if err := utils.ValidateRequired(req.Name, "name"); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Insert brand
	query := `
		INSERT INTO brands (name, description, logo_url, website_url, country_of_origin)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at, is_active
	`

	var brand models.Brand
	brand.Name = req.Name
	brand.Description = req.Description
	brand.LogoURL = req.LogoURL
	brand.WebsiteURL = req.WebsiteURL
	brand.CountryOfOrigin = req.CountryOfOrigin

	err := database.DB.QueryRow(
		query,
		req.Name, req.Description, req.LogoURL, req.WebsiteURL, req.CountryOfOrigin,
	).Scan(&brand.ID, &brand.CreatedAt, &brand.UpdatedAt, &brand.IsActive)

	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			utils.RespondError(w, http.StatusConflict, "duplicate_name", "Brand name already exists")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to create brand")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, brand)
}

// UpdateBrandHandler handles PUT /admin/brands/:id - Update brand (Admin only)
func UpdateBrandHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Brand ID is required")
		return
	}
	brandID := pathParts[len(pathParts)-1]

	// Validate UUID
	id, err := utils.ValidateUUID(brandID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Decode request
	var req UpdateBrandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Check if brand exists
	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM brands WHERE id = $1)", id).Scan(&exists)
	if err != nil || !exists {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Brand not found")
		return
	}

	// Build dynamic update query
	setClauses := []string{}
	args := []interface{}{}
	argCount := 1

	if req.Name != nil {
		if err := utils.ValidateRequired(*req.Name, "name"); err != nil {
			utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		setClauses = append(setClauses, "name = $"+string(rune(argCount+'0')))
		args = append(args, *req.Name)
		argCount++
	}

	if req.Description != nil {
		setClauses = append(setClauses, "description = $"+string(rune(argCount+'0')))
		args = append(args, *req.Description)
		argCount++
	}

	if req.LogoURL != nil {
		setClauses = append(setClauses, "logo_url = $"+string(rune(argCount+'0')))
		args = append(args, *req.LogoURL)
		argCount++
	}

	if req.WebsiteURL != nil {
		setClauses = append(setClauses, "website_url = $"+string(rune(argCount+'0')))
		args = append(args, *req.WebsiteURL)
		argCount++
	}

	if req.CountryOfOrigin != nil {
		setClauses = append(setClauses, "country_of_origin = $"+string(rune(argCount+'0')))
		args = append(args, *req.CountryOfOrigin)
		argCount++
	}

	if req.IsActive != nil {
		setClauses = append(setClauses, "is_active = $"+string(rune(argCount+'0')))
		args = append(args, *req.IsActive)
		argCount++
	}

	if len(setClauses) == 0 {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "No fields to update")
		return
	}

	// Add updated_at
	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")

	// Build and execute query
	query := "UPDATE brands SET " + strings.Join(setClauses, ", ") + " WHERE id = $" + string(rune(argCount+'0'))
	args = append(args, id)

	_, err = database.DB.Exec(query, args...)
	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			utils.RespondError(w, http.StatusConflict, "duplicate_name", "Brand name already exists")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update brand")
		return
	}

	// Fetch updated brand
	var brand models.Brand
	err = database.DB.QueryRow(`
		SELECT id, name, description, logo_url, website_url, 
		       country_of_origin, is_active, created_at, updated_at
		FROM brands WHERE id = $1
	`, id).Scan(
		&brand.ID, &brand.Name, &brand.Description, &brand.LogoURL,
		&brand.WebsiteURL, &brand.CountryOfOrigin, &brand.IsActive,
		&brand.CreatedAt, &brand.UpdatedAt,
	)

	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch updated brand")
		return
	}

	utils.RespondJSON(w, http.StatusOK, brand)
}

// DeleteBrandHandler handles DELETE /admin/brands/:id - Deactivate brand (Admin only)
func DeleteBrandHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Brand ID is required")
		return
	}
	brandID := pathParts[len(pathParts)-1]

	// Validate UUID
	id, err := utils.ValidateUUID(brandID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Check if brand exists
	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM brands WHERE id = $1)", id).Scan(&exists)
	if err != nil || !exists {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Brand not found")
		return
	}

	// Soft delete: set is_active to false
	_, err = database.DB.Exec("UPDATE brands SET is_active = false, updated_at = CURRENT_TIMESTAMP WHERE id = $1", id)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to deactivate brand")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, nil, "Brand deactivated successfully")
}
