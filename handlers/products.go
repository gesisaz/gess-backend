package handlers

import (
	"auth-demo/database"
	"auth-demo/models"
	"auth-demo/utils"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// CreateProductRequest represents the request body for creating a product
type CreateProductRequest struct {
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	Price         float64    `json:"price"`
	StockQuantity int        `json:"stock_quantity"`
	CategoryID    *uuid.UUID `json:"category_id"`
	ImageURL      string     `json:"image_url"`
}

// UpdateProductRequest represents the request body for updating a product
type UpdateProductRequest struct {
	Name          *string     `json:"name,omitempty"`
	Description   *string     `json:"description,omitempty"`
	Price         *float64    `json:"price,omitempty"`
	StockQuantity *int        `json:"stock_quantity,omitempty"`
	CategoryID    *uuid.UUID  `json:"category_id,omitempty"`
	ImageURL      *string     `json:"image_url,omitempty"`
}

// ProductListResponse represents the response for listing products
type ProductListResponse struct {
	Products   []models.ProductWithCategory `json:"products"`
	Pagination utils.PaginationMeta         `json:"pagination"`
}

// ListProductsHandler handles GET /products - List products with filters
func ListProductsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse pagination
	pagination := utils.ParsePagination(r)

	// Parse filters
	categoryID := r.URL.Query().Get("category")
	minPrice := utils.ParseFloatParam(r, "min_price")
	maxPrice := utils.ParseFloatParam(r, "max_price")
	search := r.URL.Query().Get("search")

	// Validate price range
	if err := utils.ValidatePriceRange(minPrice, maxPrice); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Build query
	query := `
		SELECT p.id, p.name, p.description, p.price, p.stock_quantity, 
		       p.category_id, p.image_url, p.created_at, p.updated_at,
		       COALESCE(c.name, '') as category_name
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	// Apply filters
	if categoryID != "" {
		catUUID, err := utils.ValidateUUID(categoryID)
		if err != nil {
			utils.RespondError(w, http.StatusBadRequest, "invalid_category", err.Error())
			return
		}
		query += ` AND p.category_id = $` + string(rune(argCount+'0'))
		args = append(args, catUUID)
		argCount++
	}

	if minPrice != nil {
		query += ` AND p.price >= $` + string(rune(argCount+'0'))
		args = append(args, *minPrice)
		argCount++
	}

	if maxPrice != nil {
		query += ` AND p.price <= $` + string(rune(argCount+'0'))
		args = append(args, *maxPrice)
		argCount++
	}

	if search != "" {
		query += ` AND p.name ILIKE $` + string(rune(argCount+'0'))
		args = append(args, "%"+search+"%")
		argCount++
	}

	// Count total (without pagination)
	countQuery := `SELECT COUNT(*) FROM (` + query + `) as filtered`
	var total int
	err := database.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to count products")
		return
	}

	// Add ordering and pagination
	query += ` ORDER BY p.created_at DESC LIMIT $` + string(rune(argCount+'0')) + ` OFFSET $` + string(rune(argCount+'1'))
	args = append(args, pagination.Limit, pagination.Offset)

	// Execute query
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch products")
		return
	}
	defer rows.Close()

	// Scan results
	products := []models.ProductWithCategory{}
	for rows.Next() {
		var p models.ProductWithCategory
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Price, &p.StockQuantity,
			&p.CategoryID, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt,
			&p.CategoryName,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to scan products")
			return
		}
		products = append(products, p)
	}

	// Build response
	response := ProductListResponse{
		Products:   products,
		Pagination: utils.CreatePaginationMeta(total, pagination.Limit, pagination.Offset),
	}

	utils.RespondJSON(w, http.StatusOK, response)
}

// GetProductHandler handles GET /products/:id - Get single product
func GetProductHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Product ID is required")
		return
	}
	productID := pathParts[len(pathParts)-1]

	// Validate UUID
	id, err := utils.ValidateUUID(productID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Query product
	query := `
		SELECT p.id, p.name, p.description, p.price, p.stock_quantity, 
		       p.category_id, p.image_url, p.created_at, p.updated_at,
		       COALESCE(c.name, '') as category_name
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE p.id = $1
	`

	var product models.ProductWithCategory
	err = database.DB.QueryRow(query, id).Scan(
		&product.ID, &product.Name, &product.Description, &product.Price,
		&product.StockQuantity, &product.CategoryID, &product.ImageURL,
		&product.CreatedAt, &product.UpdatedAt, &product.CategoryName,
	)

	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Product not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch product")
		return
	}

	utils.RespondJSON(w, http.StatusOK, product)
}

// CreateProductHandler handles POST /admin/products - Create product (Admin only)
func CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Validate required fields
	if err := utils.ValidateRequired(req.Name, "name"); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Validate price
	if err := utils.ValidatePositiveFloat(req.Price, "price"); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Validate stock quantity
	if err := utils.ValidateNonNegativeInt(req.StockQuantity, "stock_quantity"); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Insert product
	query := `
		INSERT INTO products (name, description, price, stock_quantity, category_id, image_url)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	var product models.Product
	product.Name = req.Name
	product.Description = req.Description
	product.Price = req.Price
	product.StockQuantity = req.StockQuantity
	product.CategoryID = req.CategoryID
	product.ImageURL = req.ImageURL

	err := database.DB.QueryRow(
		query,
		req.Name, req.Description, req.Price, req.StockQuantity, req.CategoryID, req.ImageURL,
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)

	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to create product")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, product)
}

// UpdateProductHandler handles PUT /admin/products/:id - Update product (Admin only)
func UpdateProductHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Product ID is required")
		return
	}
	productID := pathParts[len(pathParts)-1]

	// Validate UUID
	id, err := utils.ValidateUUID(productID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Decode request
	var req UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Check if product exists
	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)", id).Scan(&exists)
	if err != nil || !exists {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Product not found")
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

	if req.Price != nil {
		if err := utils.ValidatePositiveFloat(*req.Price, "price"); err != nil {
			utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		setClauses = append(setClauses, "price = $"+string(rune(argCount+'0')))
		args = append(args, *req.Price)
		argCount++
	}

	if req.StockQuantity != nil {
		if err := utils.ValidateNonNegativeInt(*req.StockQuantity, "stock_quantity"); err != nil {
			utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		setClauses = append(setClauses, "stock_quantity = $"+string(rune(argCount+'0')))
		args = append(args, *req.StockQuantity)
		argCount++
	}

	if req.CategoryID != nil {
		setClauses = append(setClauses, "category_id = $"+string(rune(argCount+'0')))
		args = append(args, *req.CategoryID)
		argCount++
	}

	if req.ImageURL != nil {
		setClauses = append(setClauses, "image_url = $"+string(rune(argCount+'0')))
		args = append(args, *req.ImageURL)
		argCount++
	}

	if len(setClauses) == 0 {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "No fields to update")
		return
	}

	// Add updated_at
	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")

	// Build and execute query
	query := "UPDATE products SET " + strings.Join(setClauses, ", ") + " WHERE id = $" + string(rune(argCount+'0'))
	args = append(args, id)

	_, err = database.DB.Exec(query, args...)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update product")
		return
	}

	// Fetch updated product
	var product models.Product
	err = database.DB.QueryRow(`
		SELECT id, name, description, price, stock_quantity, category_id, image_url, created_at, updated_at
		FROM products WHERE id = $1
	`, id).Scan(
		&product.ID, &product.Name, &product.Description, &product.Price,
		&product.StockQuantity, &product.CategoryID, &product.ImageURL,
		&product.CreatedAt, &product.UpdatedAt,
	)

	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch updated product")
		return
	}

	utils.RespondJSON(w, http.StatusOK, product)
}

// DeleteProductHandler handles DELETE /admin/products/:id - Delete product (Admin only)
func DeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Product ID is required")
		return
	}
	productID := pathParts[len(pathParts)-1]

	// Validate UUID
	id, err := utils.ValidateUUID(productID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Check if product exists
	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)", id).Scan(&exists)
	if err != nil || !exists {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Product not found")
		return
	}

	// Delete product
	_, err = database.DB.Exec("DELETE FROM products WHERE id = $1", id)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to delete product")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, nil, "Product deleted successfully")
}
