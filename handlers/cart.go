package handlers

import (
	"auth-demo/database"
	"auth-demo/middleware"
	"auth-demo/models"
	"auth-demo/utils"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// AddCartItemRequest represents the request body for adding an item to cart
type AddCartItemRequest struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

// UpdateCartItemRequest represents the request body for updating a cart item
type UpdateCartItemRequest struct {
	Quantity int `json:"quantity"`
}

// GetCartHandler handles GET /cart - Get user's cart with items and total
func GetCartHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Get or create cart
	var cart models.Cart
	query := `SELECT id, user_id, created_at, updated_at FROM carts WHERE user_id = $1`
	err := database.DB.QueryRow(query, user.ID).Scan(
		&cart.ID, &cart.UserID, &cart.CreatedAt, &cart.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Cart doesn't exist, return empty cart
		cart.UserID = user.ID
		response := models.CartResponse{
			Cart:  cart,
			Items: []models.CartItemWithProduct{},
			Total: 0,
		}
		utils.RespondJSON(w, http.StatusOK, response)
		return
	}

	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch cart")
		return
	}

	// Get cart items with product information
	itemsQuery := `
		SELECT 
			ci.id, ci.cart_id, ci.product_id, ci.quantity, ci.created_at, ci.updated_at,
			p.name as product_name, p.price as product_price, p.image_url as product_image
		FROM cart_items ci
		JOIN products p ON ci.product_id = p.id
		WHERE ci.cart_id = $1
		ORDER BY ci.created_at DESC
	`

	rows, err := database.DB.Query(itemsQuery, cart.ID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch cart items")
		return
	}
	defer rows.Close()

	items := []models.CartItemWithProduct{}
	var total float64

	for rows.Next() {
		var item models.CartItemWithProduct
		err := rows.Scan(
			&item.ID, &item.CartID, &item.ProductID, &item.Quantity,
			&item.CreatedAt, &item.UpdatedAt,
			&item.ProductName, &item.ProductPrice, &item.ProductImage,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to scan cart items")
			return
		}
		items = append(items, item)
		total += item.ProductPrice * float64(item.Quantity)
	}

	response := models.CartResponse{
		Cart:  cart,
		Items: items,
		Total: total,
	}

	utils.RespondJSON(w, http.StatusOK, response)
}

// AddCartItemHandler handles POST /cart/items - Add product to cart
func AddCartItemHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	var req AddCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Validate quantity
	if req.Quantity <= 0 {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "Quantity must be greater than 0")
		return
	}

	// Check if product exists and has stock
	var productPrice float64
	var stockQuantity int
	productQuery := `SELECT price, stock_quantity FROM products WHERE id = $1`
	err := database.DB.QueryRow(productQuery, req.ProductID).Scan(&productPrice, &stockQuantity)
	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Product not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to check product")
		return
	}

	// Check stock availability
	if stockQuantity < req.Quantity {
		utils.RespondError(w, http.StatusBadRequest, "insufficient_stock", "Insufficient stock available")
		return
	}

	// Get or create cart
	var cartID uuid.UUID
	cartQuery := `SELECT id FROM carts WHERE user_id = $1`
	err = database.DB.QueryRow(cartQuery, user.ID).Scan(&cartID)

	if err == sql.ErrNoRows {
		// Create cart
		createCartQuery := `INSERT INTO carts (user_id) VALUES ($1) RETURNING id`
		err = database.DB.QueryRow(createCartQuery, user.ID).Scan(&cartID)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to create cart")
			return
		}
	} else if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch cart")
		return
	}

	// Check if item already exists in cart
	var existingItemID uuid.UUID
	var existingQuantity int
	checkQuery := `SELECT id, quantity FROM cart_items WHERE cart_id = $1 AND product_id = $2`
	err = database.DB.QueryRow(checkQuery, cartID, req.ProductID).Scan(&existingItemID, &existingQuantity)

	if err == nil {
		// Item exists, update quantity
		newQuantity := existingQuantity + req.Quantity
		if newQuantity > stockQuantity {
			utils.RespondError(w, http.StatusBadRequest, "insufficient_stock", "Insufficient stock available")
			return
		}

		updateQuery := `UPDATE cart_items SET quantity = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 RETURNING id, cart_id, product_id, quantity, created_at, updated_at`
		var item models.CartItem
		err = database.DB.QueryRow(updateQuery, newQuantity, existingItemID).Scan(
			&item.ID, &item.CartID, &item.ProductID, &item.Quantity, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update cart item")
			return
		}

		utils.RespondJSON(w, http.StatusOK, item)
		return
	}

	// Item doesn't exist, insert new item
	insertQuery := `INSERT INTO cart_items (cart_id, product_id, quantity) VALUES ($1, $2, $3) RETURNING id, cart_id, product_id, quantity, created_at, updated_at`
	var item models.CartItem
	err = database.DB.QueryRow(insertQuery, cartID, req.ProductID, req.Quantity).Scan(
		&item.ID, &item.CartID, &item.ProductID, &item.Quantity, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to add item to cart")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, item)
}

// UpdateCartItemHandler handles PUT /cart/items/:id - Update cart item quantity
func UpdateCartItemHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Cart item ID is required")
		return
	}
	itemIDStr := pathParts[len(pathParts)-1]

	// Validate UUID
	itemID, err := utils.ValidateUUID(itemIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req UpdateCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Validate quantity
	if req.Quantity <= 0 {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "Quantity must be greater than 0")
		return
	}

	// Verify cart item belongs to user and get product info
	var cartID uuid.UUID
	var productID uuid.UUID
	var stockQuantity int
	verifyQuery := `
		SELECT ci.cart_id, ci.product_id, p.stock_quantity
		FROM cart_items ci
		JOIN carts c ON ci.cart_id = c.id
		JOIN products p ON ci.product_id = p.id
		WHERE ci.id = $1 AND c.user_id = $2
	`
	err = database.DB.QueryRow(verifyQuery, itemID, user.ID).Scan(&cartID, &productID, &stockQuantity)
	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Cart item not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to verify cart item")
		return
	}

	// Check stock availability
	if stockQuantity < req.Quantity {
		utils.RespondError(w, http.StatusBadRequest, "insufficient_stock", "Insufficient stock available")
		return
	}

	// Update cart item
	updateQuery := `UPDATE cart_items SET quantity = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 RETURNING id, cart_id, product_id, quantity, created_at, updated_at`
	var item models.CartItem
	err = database.DB.QueryRow(updateQuery, req.Quantity, itemID).Scan(
		&item.ID, &item.CartID, &item.ProductID, &item.Quantity, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update cart item")
		return
	}

	utils.RespondJSON(w, http.StatusOK, item)
}

// DeleteCartItemHandler handles DELETE /cart/items/:id - Remove item from cart
func DeleteCartItemHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Cart item ID is required")
		return
	}
	itemIDStr := pathParts[len(pathParts)-1]

	// Validate UUID
	itemID, err := utils.ValidateUUID(itemIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Verify cart item belongs to user
	verifyQuery := `
		SELECT ci.id
		FROM cart_items ci
		JOIN carts c ON ci.cart_id = c.id
		WHERE ci.id = $1 AND c.user_id = $2
	`
	var foundID uuid.UUID
	err = database.DB.QueryRow(verifyQuery, itemID, user.ID).Scan(&foundID)
	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Cart item not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to verify cart item")
		return
	}

	// Delete cart item
	deleteQuery := `DELETE FROM cart_items WHERE id = $1`
	result, err := database.DB.Exec(deleteQuery, itemID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to delete cart item")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to delete cart item")
		return
	}

	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Cart item not found")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, nil, "Cart item deleted successfully")
}

// ClearCartHandler handles DELETE /cart - Clear entire cart
func ClearCartHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Get cart ID
	var cartID uuid.UUID
	cartQuery := `SELECT id FROM carts WHERE user_id = $1`
	err := database.DB.QueryRow(cartQuery, user.ID).Scan(&cartID)
	if err == sql.ErrNoRows {
		utils.RespondSuccess(w, http.StatusOK, nil, "Cart is already empty")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch cart")
		return
	}

	// Delete all cart items
	deleteQuery := `DELETE FROM cart_items WHERE cart_id = $1`
	_, err = database.DB.Exec(deleteQuery, cartID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to clear cart")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, nil, "Cart cleared successfully")
}
