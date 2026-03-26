package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"gess-backend/database"
	"gess-backend/models"
	"gess-backend/utils"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// CreateProductRequest represents the request body for creating a product
type CreateProductRequest struct {
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	Price           float64    `json:"price"`
	StockQuantity   int        `json:"stock_quantity"`
	CategoryID      *uuid.UUID `json:"category_id"`
	BrandID         *uuid.UUID `json:"brand_id"`
	SKU             string     `json:"sku"`
	ProductLine     string     `json:"product_line"`
	SizeValue       float64    `json:"size_value"`
	SizeUnit        string     `json:"size_unit"`
	Scent           string     `json:"scent"`
	SkinType        []string   `json:"skin_type"`
	Ingredients     string     `json:"ingredients"`
	KeyIngredients  []string   `json:"key_ingredients"`
	ApplicationArea string     `json:"application_area"`
	IsOrganic       bool       `json:"is_organic"`
	IsVegan         bool       `json:"is_vegan"`
	IsCrueltyFree   bool       `json:"is_cruelty_free"`
	IsParabenFree   bool       `json:"is_paraben_free"`
	IsFeatured      bool       `json:"is_featured"`
	ImageURL        string     `json:"image_url"`
	ImageURLs       []string   `json:"image_urls"`
}

// UpdateProductRequest represents the request body for updating a product
type UpdateProductRequest struct {
	Name            *string    `json:"name,omitempty"`
	Description     *string    `json:"description,omitempty"`
	Price           *float64   `json:"price,omitempty"`
	StockQuantity   *int       `json:"stock_quantity,omitempty"`
	CategoryID      *uuid.UUID `json:"category_id,omitempty"`
	BrandID         *uuid.UUID `json:"brand_id,omitempty"`
	SKU             *string    `json:"sku,omitempty"`
	ProductLine     *string    `json:"product_line,omitempty"`
	SizeValue       *float64   `json:"size_value,omitempty"`
	SizeUnit        *string    `json:"size_unit,omitempty"`
	Scent           *string    `json:"scent,omitempty"`
	SkinType        []string   `json:"skin_type,omitempty"`
	Ingredients     *string    `json:"ingredients,omitempty"`
	KeyIngredients  []string   `json:"key_ingredients,omitempty"`
	ApplicationArea *string    `json:"application_area,omitempty"`
	IsOrganic       *bool      `json:"is_organic,omitempty"`
	IsVegan         *bool      `json:"is_vegan,omitempty"`
	IsCrueltyFree   *bool      `json:"is_cruelty_free,omitempty"`
	IsParabenFree   *bool      `json:"is_paraben_free,omitempty"`
	IsFeatured      *bool      `json:"is_featured,omitempty"`
	ImageURL        *string    `json:"image_url,omitempty"`
	ImageURLs       *[]string  `json:"image_urls,omitempty"`
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
	brandID := r.URL.Query().Get("brand")
	minPrice := utils.ParseFloatParam(r, "min_price")
	maxPrice := utils.ParseFloatParam(r, "max_price")
	search := r.URL.Query().Get("search")
	scent := r.URL.Query().Get("scent")
	skinType := r.URL.Query().Get("skin_type")
	applicationArea := r.URL.Query().Get("application_area")

	// Boolean filters
	isOrganic := r.URL.Query().Get("is_organic") == "true"
	isVegan := r.URL.Query().Get("is_vegan") == "true"
	isCrueltyFree := r.URL.Query().Get("is_cruelty_free") == "true"
	isParabenFree := r.URL.Query().Get("is_paraben_free") == "true"
	isFeatured := r.URL.Query().Get("is_featured") == "true"

	// Validate price range
	if err := utils.ValidatePriceRange(minPrice, maxPrice); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Build query
	query := `
		SELECT p.id, p.name, COALESCE(p.description, ''), p.selling_price, p.stock_quantity, 
		       p.category_id, p.brand_id, COALESCE(p.sku, ''), COALESCE(p.product_line, ''),
		       p.size_value, COALESCE(p.size_unit, ''), COALESCE(p.scent, ''), COALESCE(p.skin_type, '{}'::text[]),
		       COALESCE(p.ingredients, ''), COALESCE(p.key_ingredients, '{}'::text[]), COALESCE(p.application_area, ''),
		       p.is_organic, p.is_vegan, p.is_cruelty_free, p.is_paraben_free, p.is_featured,
		       p.rating_average, p.review_count, COALESCE(p.image_url, ''),
		       COALESCE(p.image_urls, '{}'::text[]),
		       p.created_at, p.updated_at,
		       COALESCE(c.name, '') as category_name,
		       COALESCE(b.name, '') as brand_name
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		LEFT JOIN brands b ON p.brand_id = b.id
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
		query += fmt.Sprintf(" AND p.category_id = $%d", argCount)
		args = append(args, catUUID)
		argCount++
	}

	if brandID != "" {
		brandUUID, err := utils.ValidateUUID(brandID)
		if err != nil {
			utils.RespondError(w, http.StatusBadRequest, "invalid_brand", err.Error())
			return
		}
		query += fmt.Sprintf(" AND p.brand_id = $%d", argCount)
		args = append(args, brandUUID)
		argCount++
	}

	if minPrice != nil {
		query += fmt.Sprintf(" AND p.selling_price >= $%d", argCount)
		args = append(args, *minPrice)
		argCount++
	}

	if maxPrice != nil {
		query += fmt.Sprintf(" AND p.selling_price <= $%d", argCount)
		args = append(args, *maxPrice)
		argCount++
	}

	if search != "" {
		query += fmt.Sprintf(" AND (p.name ILIKE $%d OR p.description ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}

	if scent != "" {
		query += fmt.Sprintf(" AND p.scent ILIKE $%d", argCount)
		args = append(args, "%"+scent+"%")
		argCount++
	}

	if skinType != "" {
		query += fmt.Sprintf(" AND $%d = ANY(p.skin_type)", argCount)
		args = append(args, skinType)
		argCount++
	}

	if applicationArea != "" {
		query += fmt.Sprintf(" AND p.application_area = $%d", argCount)
		args = append(args, applicationArea)
		argCount++
	}

	if r.URL.Query().Get("is_organic") != "" {
		query += fmt.Sprintf(" AND p.is_organic = $%d", argCount)
		args = append(args, isOrganic)
		argCount++
	}

	if r.URL.Query().Get("is_vegan") != "" {
		query += fmt.Sprintf(" AND p.is_vegan = $%d", argCount)
		args = append(args, isVegan)
		argCount++
	}

	if r.URL.Query().Get("is_cruelty_free") != "" {
		query += fmt.Sprintf(" AND p.is_cruelty_free = $%d", argCount)
		args = append(args, isCrueltyFree)
		argCount++
	}

	if r.URL.Query().Get("is_paraben_free") != "" {
		query += fmt.Sprintf(" AND p.is_paraben_free = $%d", argCount)
		args = append(args, isParabenFree)
		argCount++
	}

	if r.URL.Query().Get("is_featured") != "" {
		query += fmt.Sprintf(" AND p.is_featured = $%d", argCount)
		args = append(args, isFeatured)
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
	query += fmt.Sprintf(" ORDER BY p.is_featured DESC, p.created_at LIMIT $%d OFFSET $%d", argCount, argCount+1)
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
			&p.CategoryID, &p.BrandID, &p.SKU, &p.ProductLine,
			&p.SizeValue, &p.SizeUnit, &p.Scent, &p.SkinType,
			&p.Ingredients, &p.KeyIngredients, &p.ApplicationArea,
			&p.IsOrganic, &p.IsVegan, &p.IsCrueltyFree, &p.IsParabenFree, &p.IsFeatured,
			&p.RatingAverage, &p.ReviewCount, &p.ImageURL,
			&p.ImageURLs,
			&p.CreatedAt, &p.UpdatedAt,
			&p.CategoryName, &p.BrandName,
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

	// Query product (COALESCE nullable arrays so Scan succeeds)
	query := `
		SELECT p.id, p.name, COALESCE(p.description, ''), p.selling_price, p.stock_quantity,
		       p.category_id, p.brand_id, COALESCE(p.sku, ''), COALESCE(p.product_line, ''),
		       p.size_value, COALESCE(p.size_unit, ''), COALESCE(p.scent, ''), COALESCE(p.skin_type, '{}'::text[]),
		       COALESCE(p.ingredients, ''), COALESCE(p.key_ingredients, '{}'::text[]), COALESCE(p.application_area, ''),
		       p.is_organic, p.is_vegan, p.is_cruelty_free, p.is_paraben_free, p.is_featured,
		       p.rating_average, p.review_count, COALESCE(p.image_url, ''),
		       COALESCE(p.image_urls, '{}'::text[]),
		       p.created_at, p.updated_at,
		       COALESCE(c.name, '') as category_name,
		       COALESCE(b.name, '') as brand_name
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		LEFT JOIN brands b ON p.brand_id = b.id
		WHERE p.id = $1
	`

	var product models.ProductWithCategory
	err = database.DB.QueryRow(query, id).Scan(
		&product.ID, &product.Name, &product.Description, &product.Price,
		&product.StockQuantity, &product.CategoryID, &product.BrandID,
		&product.SKU, &product.ProductLine, &product.SizeValue, &product.SizeUnit,
		&product.Scent, &product.SkinType, &product.Ingredients, &product.KeyIngredients,
		&product.ApplicationArea, &product.IsOrganic, &product.IsVegan,
		&product.IsCrueltyFree, &product.IsParabenFree, &product.IsFeatured,
		&product.RatingAverage, &product.ReviewCount, &product.ImageURL,
		&product.ImageURLs,
		&product.CreatedAt, &product.UpdatedAt,
		&product.CategoryName, &product.BrandName,
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

const maxBatchProductIDs = 50

// BatchProductsHandler handles GET /products/batch?ids=uuid,uuid,uuid - Fetch products by IDs (no auth, for guest cart).
func BatchProductsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}
	idsParam := r.URL.Query().Get("ids")
	if idsParam == "" {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "ids query parameter is required")
		return
	}
	parts := strings.Split(idsParam, ",")
	if len(parts) > maxBatchProductIDs {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", fmt.Sprintf("maximum %d ids allowed", maxBatchProductIDs))
		return
	}
	ids := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := utils.ValidateUUID(p)
		if err != nil {
			utils.RespondError(w, http.StatusBadRequest, "invalid_id", fmt.Sprintf("invalid uuid %q: %v", p, err))
			return
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		utils.RespondJSON(w, http.StatusOK, ProductListResponse{Products: []models.ProductWithCategory{}, Pagination: utils.PaginationMeta{Limit: 0, Offset: 0, Total: 0}})
		return
	}

	query := `
		SELECT p.id, p.name, COALESCE(p.description, ''), p.selling_price, p.stock_quantity,
		       p.category_id, p.brand_id, COALESCE(p.sku, ''), COALESCE(p.product_line, ''),
		       p.size_value, COALESCE(p.size_unit, ''), COALESCE(p.scent, ''), COALESCE(p.skin_type, '{}'::text[]),
		       COALESCE(p.ingredients, ''), COALESCE(p.key_ingredients, '{}'::text[]), COALESCE(p.application_area, ''),
		       p.is_organic, p.is_vegan, p.is_cruelty_free, p.is_paraben_free, p.is_featured,
		       p.rating_average, p.review_count, COALESCE(p.image_url, ''),
		       COALESCE(p.image_urls, '{}'::text[]),
		       p.created_at, p.updated_at,
		       COALESCE(c.name, '') as category_name,
		       COALESCE(b.name, '') as brand_name
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		LEFT JOIN brands b ON p.brand_id = b.id
		WHERE p.id = ANY($1)
	`
	rows, err := database.DB.Query(query, pq.Array(ids))
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
			&product.CreatedAt, &product.UpdatedAt,
			&product.CategoryName, &product.BrandName,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to scan product")
			return
		}
		products = append(products, product)
	}

	utils.RespondJSON(w, http.StatusOK, ProductListResponse{
		Products:   products,
		Pagination: utils.PaginationMeta{Limit: len(products), Offset: 0, Total: len(products)},
	})
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

	// Validate cosmetics-specific fields
	if err := utils.ValidateSKU(req.SKU); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	if err := utils.ValidateSizeUnit(req.SizeUnit); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	if err := utils.ValidateSkinTypes(req.SkinType); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	if err := utils.ValidateApplicationArea(req.ApplicationArea); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Primary image: first of image_urls if non-empty, else image_url
	imageURLs := req.ImageURLs
	if imageURLs == nil {
		imageURLs = []string{}
	}
	primaryImage := req.ImageURL
	if len(imageURLs) > 0 && imageURLs[0] != "" {
		primaryImage = imageURLs[0]
	}

	// Store NULL for empty SKU so multiple products can have no SKU (unique constraint allows multiple NULLs)
	skuArg := interface{}(req.SKU)
	if req.SKU == "" {
		skuArg = nil
	}

	// Insert product
	query := `
		INSERT INTO products (
			name, description, buying_price, selling_price, stock_quantity,
			category_id, brand_id, sku, product_line,
			size_value, size_unit, scent, skin_type,
			ingredients, key_ingredients, application_area,
			is_organic, is_vegan, is_cruelty_free, is_paraben_free, is_featured,
			image_url, image_urls
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
		RETURNING id, created_at, updated_at, rating_average, review_count
	`

	var product models.Product
	err := database.DB.QueryRow(
		query,
		req.Name, req.Description, 0, req.Price, req.StockQuantity,
		req.CategoryID, req.BrandID, skuArg, req.ProductLine,
		req.SizeValue, req.SizeUnit, req.Scent, pq.Array(req.SkinType),
		req.Ingredients, pq.Array(req.KeyIngredients), req.ApplicationArea,
		req.IsOrganic, req.IsVegan, req.IsCrueltyFree, req.IsParabenFree, req.IsFeatured,
		primaryImage, pq.Array(imageURLs),
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt, &product.RatingAverage, &product.ReviewCount)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			utils.RespondError(w, http.StatusConflict, "duplicate_sku", "SKU already exists")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to create product: "+err.Error())
		return
	}

	// Fill in the rest of the product data
	product.Name = req.Name
	product.Description = req.Description
	product.Price = req.Price
	product.StockQuantity = req.StockQuantity
	product.CategoryID = req.CategoryID
	product.BrandID = req.BrandID
	product.SKU = req.SKU
	product.ProductLine = req.ProductLine
	product.SizeValue = req.SizeValue
	product.SizeUnit = req.SizeUnit
	product.Scent = req.Scent
	product.SkinType = req.SkinType
	product.Ingredients = req.Ingredients
	product.KeyIngredients = req.KeyIngredients
	product.ApplicationArea = req.ApplicationArea
	product.IsOrganic = req.IsOrganic
	product.IsVegan = req.IsVegan
	product.IsCrueltyFree = req.IsCrueltyFree
	product.IsParabenFree = req.IsParabenFree
	product.IsFeatured = req.IsFeatured
	product.ImageURL = primaryImage
	product.ImageURLs = imageURLs

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
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to verify product")
		return
	}
	if !exists {
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
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argCount))
		args = append(args, *req.Name)
		argCount++
	}

	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argCount))
		args = append(args, *req.Description)
		argCount++
	}

	if req.Price != nil {
		if err := utils.ValidatePositiveFloat(*req.Price, "price"); err != nil {
			utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("selling_price = $%d", argCount))
		args = append(args, *req.Price)
		argCount++
	}

	if req.StockQuantity != nil {
		if err := utils.ValidateNonNegativeInt(*req.StockQuantity, "stock_quantity"); err != nil {
			utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("stock_quantity = $%d", argCount))
		args = append(args, *req.StockQuantity)
		argCount++
	}

	if req.CategoryID != nil {
		setClauses = append(setClauses, fmt.Sprintf("category_id = $%d", argCount))
		args = append(args, *req.CategoryID)
		argCount++
	}

	if req.BrandID != nil {
		setClauses = append(setClauses, fmt.Sprintf("brand_id = $%d", argCount))
		args = append(args, *req.BrandID)
		argCount++
	}

	if req.SKU != nil {
		if err := utils.ValidateSKU(*req.SKU); err != nil {
			utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("sku = $%d", argCount))
		skuVal := interface{}(*req.SKU)
		if *req.SKU == "" {
			skuVal = nil
		}
		args = append(args, skuVal)
		argCount++
	}

	if req.ProductLine != nil {
		setClauses = append(setClauses, fmt.Sprintf("product_line = $%d", argCount))
		args = append(args, *req.ProductLine)
		argCount++
	}

	if req.SizeValue != nil {
		setClauses = append(setClauses, fmt.Sprintf("size_value = $%d", argCount))
		args = append(args, *req.SizeValue)
		argCount++
	}

	if req.SizeUnit != nil {
		if err := utils.ValidateSizeUnit(*req.SizeUnit); err != nil {
			utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("size_unit = $%d", argCount))
		args = append(args, *req.SizeUnit)
		argCount++
	}

	if req.Scent != nil {
		setClauses = append(setClauses, fmt.Sprintf("scent = $%d", argCount))
		args = append(args, *req.Scent)
		argCount++
	}

	if req.SkinType != nil {
		if err := utils.ValidateSkinTypes(req.SkinType); err != nil {
			utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("skin_type = $%d", argCount))
		args = append(args, pq.Array(req.SkinType))
		argCount++
	}

	if req.Ingredients != nil {
		setClauses = append(setClauses, fmt.Sprintf("ingredients = $%d", argCount))
		args = append(args, *req.Ingredients)
		argCount++
	}

	if req.KeyIngredients != nil {
		setClauses = append(setClauses, fmt.Sprintf("key_ingredients = $%d", argCount))
		args = append(args, pq.Array(req.KeyIngredients))
		argCount++
	}

	if req.ApplicationArea != nil {
		if err := utils.ValidateApplicationArea(*req.ApplicationArea); err != nil {
			utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("application_area = $%d", argCount))
		args = append(args, *req.ApplicationArea)
		argCount++
	}

	if req.IsOrganic != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_organic = $%d", argCount))
		args = append(args, *req.IsOrganic)
		argCount++
	}

	if req.IsVegan != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_vegan = $%d", argCount))
		args = append(args, *req.IsVegan)
		argCount++
	}

	if req.IsCrueltyFree != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_cruelty_free = $%d", argCount))
		args = append(args, *req.IsCrueltyFree)
		argCount++
	}

	if req.IsParabenFree != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_paraben_free = $%d", argCount))
		args = append(args, *req.IsParabenFree)
		argCount++
	}

	if req.IsFeatured != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_featured = $%d", argCount))
		args = append(args, *req.IsFeatured)
		argCount++
	}

	if req.ImageURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("image_url = $%d", argCount))
		args = append(args, *req.ImageURL)
		argCount++
	}

	if req.ImageURLs != nil {
		setClauses = append(setClauses, fmt.Sprintf("image_urls = $%d", argCount))
		args = append(args, pq.Array(*req.ImageURLs))
		argCount++
		// Sync primary image from first URL when image_urls is updated
		if len(*req.ImageURLs) > 0 && (*req.ImageURLs)[0] != "" {
			setClauses = append(setClauses, fmt.Sprintf("image_url = $%d", argCount))
			args = append(args, (*req.ImageURLs)[0])
			argCount++
		}
	}

	if len(setClauses) == 0 {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "No fields to update")
		return
	}

	// Add updated_at
	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")

	// Build and execute query
	query := "UPDATE products SET " + strings.Join(setClauses, ", ") + fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)

	_, err = database.DB.Exec(query, args...)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update product")
		return
	}

	// Fetch updated product (COALESCE nullable columns so Scan into string/float64 never sees NULL)
	var product models.Product
	err = database.DB.QueryRow(`
		SELECT id, name, COALESCE(description, ''), selling_price, stock_quantity,
		       category_id, brand_id, COALESCE(sku, ''), COALESCE(product_line, ''),
		       COALESCE(size_value, 0), COALESCE(size_unit, ''), COALESCE(scent, ''),
		       COALESCE(skin_type, '{}'::text[]), COALESCE(ingredients, ''),
		       COALESCE(key_ingredients, '{}'::text[]), COALESCE(application_area, ''),
		       is_organic, is_vegan, is_cruelty_free, is_paraben_free, is_featured,
		       rating_average, review_count, COALESCE(image_url, ''),
		       COALESCE(image_urls, '{}'::text[]),
		       created_at, updated_at
		FROM products WHERE id = $1
	`, id).Scan(
		&product.ID, &product.Name, &product.Description, &product.Price,
		&product.StockQuantity, &product.CategoryID, &product.BrandID,
		&product.SKU, &product.ProductLine, &product.SizeValue, &product.SizeUnit,
		&product.Scent, &product.SkinType, &product.Ingredients, &product.KeyIngredients,
		&product.ApplicationArea, &product.IsOrganic, &product.IsVegan,
		&product.IsCrueltyFree, &product.IsParabenFree, &product.IsFeatured,
		&product.RatingAverage, &product.ReviewCount, &product.ImageURL,
		&product.ImageURLs,
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
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to verify product")
		return
	}
	if !exists {
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
