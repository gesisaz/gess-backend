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

// CreateCategoryRequest represents the request body for creating a category
type CreateCategoryRequest struct {
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	ParentCategoryID *uuid.UUID `json:"parent_category_id"`
	DisplayOrder     int        `json:"display_order"`
}

// UpdateCategoryRequest represents the request body for updating a category
type UpdateCategoryRequest struct {
	Name             *string    `json:"name,omitempty"`
	Description      *string    `json:"description,omitempty"`
	ParentCategoryID *uuid.UUID `json:"parent_category_id,omitempty"`
	DisplayOrder     *int       `json:"display_order,omitempty"`
}

// CategoryWithProductCount extends Category with product count
type CategoryWithProductCount struct {
	models.Category
	ProductCount int `json:"product_count"`
}

// ListCategoriesHandler handles GET /categories - List all categories with hierarchy
func ListCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	// Check if hierarchical view is requested
	hierarchical := r.URL.Query().Get("hierarchical") == "true"
	parentOnly := r.URL.Query().Get("parent_only") == "true"

	var query string
	if parentOnly {
		// Only return parent categories
		query = `
			SELECT c.id, c.name, c.description, c.parent_category_id, c.display_order, c.created_at, 
			       COUNT(p.id) as product_count
			FROM categories c
			LEFT JOIN products p ON c.id = p.category_id
			WHERE c.parent_category_id IS NULL
			GROUP BY c.id
			ORDER BY c.display_order, c.name
		`
	} else {
		// Return all categories with parent info
		query = `
			SELECT c.id, c.name, c.description, c.parent_category_id, c.display_order, c.created_at,
			       COUNT(p.id) as product_count,
			       COALESCE(parent.name, '') as parent_category_name
			FROM categories c
			LEFT JOIN categories parent ON c.parent_category_id = parent.id
			LEFT JOIN products p ON c.id = p.category_id
			GROUP BY c.id, parent.name
			ORDER BY c.display_order, c.name
		`
	}

	rows, err := database.DB.Query(query)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch categories")
		return
	}
	defer rows.Close()

	type CategoryWithCount struct {
		models.Category
		ProductCount       int    `json:"product_count"`
		ParentCategoryName string `json:"parent_category_name,omitempty"`
	}

	categories := []CategoryWithCount{}
	for rows.Next() {
		var cat CategoryWithCount
		var parentName sql.NullString

		if parentOnly {
			err := rows.Scan(
				&cat.ID, &cat.Name, &cat.Description, &cat.ParentCategoryID,
				&cat.DisplayOrder, &cat.CreatedAt, &cat.ProductCount,
			)
			if err != nil {
				utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to scan categories")
				return
			}
		} else {
			err := rows.Scan(
				&cat.ID, &cat.Name, &cat.Description, &cat.ParentCategoryID,
				&cat.DisplayOrder, &cat.CreatedAt, &cat.ProductCount, &parentName,
			)
			if err != nil {
				utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to scan categories")
				return
			}
			if parentName.Valid {
				cat.ParentCategoryName = parentName.String
			}
		}

		categories = append(categories, cat)
	}

	// If hierarchical view is requested, build tree structure
	if hierarchical {
		// Convert CategoryWithCount to CategoryWithProductCount
		categoriesWithCount := make([]CategoryWithProductCount, len(categories))
		for i, cat := range categories {
			categoriesWithCount[i] = CategoryWithProductCount{
				Category:     cat.Category,
				ProductCount: cat.ProductCount,
			}
		}
		tree := buildCategoryTree(categoriesWithCount)
		utils.RespondJSON(w, http.StatusOK, tree)
		return
	}

	utils.RespondJSON(w, http.StatusOK, categories)
}

// buildCategoryTree builds a hierarchical tree from flat category list
func buildCategoryTree(categories []CategoryWithProductCount) []map[string]interface{} {
	type CategoryNode struct {
		models.Category
		ProductCount int                      `json:"product_count"`
		Children     []map[string]interface{} `json:"children,omitempty"`
	}

	categoryMap := make(map[uuid.UUID]*CategoryNode)
	var roots []*CategoryNode

	// First pass: create all nodes
	for _, cat := range categories {
		node := &CategoryNode{
			Category:     cat.Category,
			ProductCount: cat.ProductCount,
			Children:     []map[string]interface{}{},
		}
		categoryMap[cat.ID] = node
	}

	// Second pass: build tree
	for _, cat := range categories {
		node := categoryMap[cat.ID]
		if cat.ParentCategoryID == nil {
			roots = append(roots, node)
		} else {
			if parent, exists := categoryMap[*cat.ParentCategoryID]; exists {
				parent.Children = append(parent.Children, map[string]interface{}{
					"id":            node.ID,
					"name":          node.Name,
					"description":   node.Description,
					"display_order": node.DisplayOrder,
					"product_count": node.ProductCount,
					"created_at":    node.CreatedAt,
				})
			}
		}
	}

	// Convert to interface for JSON
	result := []map[string]interface{}{}
	for _, root := range roots {
		result = append(result, map[string]interface{}{
			"id":            root.ID,
			"name":          root.Name,
			"description":   root.Description,
			"display_order": root.DisplayOrder,
			"product_count": root.ProductCount,
			"created_at":    root.CreatedAt,
			"children":      root.Children,
		})
	}

	return result
}

// GetCategoryHandler handles GET /categories/:id - Get category with its products
func GetCategoryHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Category ID is required")
		return
	}
	categoryID := pathParts[len(pathParts)-1]

	// Validate UUID
	id, err := utils.ValidateUUID(categoryID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Query category
	var category models.Category
	err = database.DB.QueryRow(`
		SELECT id, name, description, parent_category_id, display_order, created_at
		FROM categories
		WHERE id = $1
	`, id).Scan(&category.ID, &category.Name, &category.Description, &category.ParentCategoryID, &category.DisplayOrder, &category.CreatedAt)

	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Category not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch category")
		return
	}

	// Query products in this category
	productsQuery := `
		SELECT id, name, description, price, stock_quantity, category_id, image_url, created_at, updated_at
		FROM products
		WHERE category_id = $1
		ORDER BY created_at DESC
	`

	rows, err := database.DB.Query(productsQuery, id)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch products")
		return
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		var product models.Product
		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.Price,
			&product.StockQuantity, &product.CategoryID, &product.ImageURL,
			&product.CreatedAt, &product.UpdatedAt,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to scan products")
			return
		}
		products = append(products, product)
	}

	// Build response
	response := map[string]interface{}{
		"category": category,
		"products": products,
	}

	utils.RespondJSON(w, http.StatusOK, response)
}

// CreateCategoryHandler handles POST /admin/categories - Create category (Admin only)
func CreateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Validate required fields
	if err := utils.ValidateRequired(req.Name, "name"); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Insert category
	query := `
		INSERT INTO categories (name, description, parent_category_id, display_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	var category models.Category
	category.Name = req.Name
	category.Description = req.Description
	category.ParentCategoryID = req.ParentCategoryID
	category.DisplayOrder = req.DisplayOrder

	err := database.DB.QueryRow(query, req.Name, req.Description, req.ParentCategoryID, req.DisplayOrder).Scan(&category.ID, &category.CreatedAt)

	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			utils.RespondError(w, http.StatusConflict, "duplicate_name", "Category name already exists")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to create category")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, category)
}

// UpdateCategoryHandler handles PUT /admin/categories/:id - Update category (Admin only)
func UpdateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Category ID is required")
		return
	}
	categoryID := pathParts[len(pathParts)-1]

	// Validate UUID
	id, err := utils.ValidateUUID(categoryID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Decode request
	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Check if category exists
	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)", id).Scan(&exists)
	if err != nil || !exists {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Category not found")
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

	if req.ParentCategoryID != nil {
		setClauses = append(setClauses, "parent_category_id = $"+string(rune(argCount+'0')))
		args = append(args, *req.ParentCategoryID)
		argCount++
	}

	if req.DisplayOrder != nil {
		setClauses = append(setClauses, "display_order = $"+string(rune(argCount+'0')))
		args = append(args, *req.DisplayOrder)
		argCount++
	}

	if len(setClauses) == 0 {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "No fields to update")
		return
	}

	// Build and execute query
	query := "UPDATE categories SET " + strings.Join(setClauses, ", ") + " WHERE id = $" + string(rune(argCount+'0'))
	args = append(args, id)

	_, err = database.DB.Exec(query, args...)
	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			utils.RespondError(w, http.StatusConflict, "duplicate_name", "Category name already exists")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update category")
		return
	}

	// Fetch updated category
	var category models.Category
	err = database.DB.QueryRow(`
		SELECT id, name, description, parent_category_id, display_order, created_at
		FROM categories WHERE id = $1
	`, id).Scan(&category.ID, &category.Name, &category.Description, &category.ParentCategoryID, &category.DisplayOrder, &category.CreatedAt)

	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch updated category")
		return
	}

	utils.RespondJSON(w, http.StatusOK, category)
}

// DeleteCategoryHandler handles DELETE /admin/categories/:id - Delete category (Admin only)
func DeleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Category ID is required")
		return
	}
	categoryID := pathParts[len(pathParts)-1]

	// Validate UUID
	id, err := utils.ValidateUUID(categoryID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Check if category exists
	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)", id).Scan(&exists)
	if err != nil || !exists {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Category not found")
		return
	}

	// Delete category (products will have category_id set to NULL due to ON DELETE SET NULL)
	_, err = database.DB.Exec("DELETE FROM categories WHERE id = $1", id)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to delete category")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, nil, "Category deleted successfully")
}
