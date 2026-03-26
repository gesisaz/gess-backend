package handlers

import (
	"gess-backend/database"
	"gess-backend/mail"
	"gess-backend/middleware"
	"gess-backend/models"
	"gess-backend/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CreateOrderRequest represents the request body for creating an order
type CreateOrderRequest struct {
	ShippingAddressID uuid.UUID `json:"shipping_address_id"`
}

// UpdateOrderStatusRequest represents the request body for updating order status
type UpdateOrderStatusRequest struct {
	Status models.OrderStatus `json:"status"`
}

// OrderListResponse represents the response for listing orders
type OrderListResponse struct {
	Orders     []models.Order       `json:"orders"`
	Pagination utils.PaginationMeta `json:"pagination"`
}

// CreateOrderHandler handles POST /orders - Create order from cart (checkout)
func CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Validate shipping address belongs to user
	var address models.Address
	addressQuery := `SELECT id, user_id, full_name, street_address, city, state, postal_code, country, phone, is_default, created_at, updated_at FROM addresses WHERE id = $1 AND user_id = $2`
	err := database.DB.QueryRow(addressQuery, req.ShippingAddressID, user.ID).Scan(
		&address.ID, &address.UserID, &address.FullName, &address.StreetAddress,
		&address.City, &address.State, &address.PostalCode, &address.Country,
		&address.Phone, &address.IsDefault, &address.CreatedAt, &address.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Shipping address not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to verify shipping address")
		return
	}

	// Get user's cart
	var cartID uuid.UUID
	cartQuery := `SELECT id FROM carts WHERE user_id = $1`
	err = database.DB.QueryRow(cartQuery, user.ID).Scan(&cartID)
	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusBadRequest, "empty_cart", "Cart is empty")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch cart")
		return
	}

	// Get cart items with product information
	cartItemsQuery := `
		SELECT ci.id, ci.product_id, ci.quantity, p.selling_price, p.stock_quantity, p.name
		FROM cart_items ci
		JOIN products p ON ci.product_id = p.id
		WHERE ci.cart_id = $1
	`
	rows, err := database.DB.Query(cartItemsQuery, cartID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch cart items")
		return
	}
	defer rows.Close()

	type CartItemInfo struct {
		ID            uuid.UUID
		ProductID     uuid.UUID
		Quantity      int
		Price         float64
		StockQuantity int
		ProductName   string
	}

	cartItems := []CartItemInfo{}
	for rows.Next() {
		var item CartItemInfo
		err := rows.Scan(&item.ID, &item.ProductID, &item.Quantity, &item.Price, &item.StockQuantity, &item.ProductName)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to scan cart items")
			return
		}
		cartItems = append(cartItems, item)
	}
	if err := rows.Err(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to iterate cart items")
		return
	}

	if len(cartItems) == 0 {
		utils.RespondError(w, http.StatusBadRequest, "empty_cart", "Cart is empty")
		return
	}

	// Validate stock availability
	for _, item := range cartItems {
		if item.StockQuantity < item.Quantity {
			utils.RespondError(w, http.StatusBadRequest, "insufficient_stock",
				fmt.Sprintf("Insufficient stock for product: %s (available: %d, requested: %d)", item.ProductName, item.StockQuantity, item.Quantity))
			return
		}
	}

	// Calculate total
	var totalAmount float64
	for _, item := range cartItems {
		totalAmount += item.Price * float64(item.Quantity)
	}

	// Start transaction for checkout
	tx, err := database.DB.Begin()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Create order
	var order models.Order
	orderQuery := `
		INSERT INTO orders (user_id, total_amount, status, shipping_address_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, total_amount, status, shipping_address_id, created_at, updated_at
	`
	err = tx.QueryRow(orderQuery, user.ID, totalAmount, models.OrderStatusPending, req.ShippingAddressID).Scan(
		&order.ID, &order.UserID, &order.TotalAmount, &order.Status, &order.ShippingAddressID,
		&order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to create order")
		return
	}

	// Create order items and update product stock
	orderItems := []models.OrderItem{}
	for _, cartItem := range cartItems {
		// Insert order item
		var orderItem models.OrderItem
		orderItemQuery := `
			INSERT INTO order_items (order_id, product_id, quantity, price_at_purchase)
			VALUES ($1, $2, $3, $4)
			RETURNING id, order_id, product_id, quantity, price_at_purchase, created_at
		`
		err = tx.QueryRow(orderItemQuery, order.ID, cartItem.ProductID, cartItem.Quantity, cartItem.Price).Scan(
			&orderItem.ID, &orderItem.OrderID, &orderItem.ProductID, &orderItem.Quantity,
			&orderItem.PriceAtPurchase, &orderItem.CreatedAt,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to create order item")
			return
		}
		orderItems = append(orderItems, orderItem)

		// Update product stock
		updateStockQuery := `UPDATE products SET stock_quantity = stock_quantity - $1 WHERE id = $2`
		_, err = tx.Exec(updateStockQuery, cartItem.Quantity, cartItem.ProductID)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update product stock")
			return
		}
	}

	// Clear cart
	deleteCartItemsQuery := `DELETE FROM cart_items WHERE cart_id = $1`
	_, err = tx.Exec(deleteCartItemsQuery, cartID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to clear cart")
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to commit transaction")
		return
	}

	// Send order confirmation email (do not fail request on error)
	var summaryParts []string
	for _, it := range cartItems {
		summaryParts = append(summaryParts, fmt.Sprintf("%s x %d", it.ProductName, it.Quantity))
	}
	summary := strings.Join(summaryParts, "\n")
	if err := mail.SendOrderConfirmationEmail(user.Email, order.ID.String(), totalAmount, summary); err != nil {
		log.Printf("order confirmation email failed: %v", err)
	}

	// Build response
	response := models.OrderWithItems{
		Order:           order,
		Items:           orderItems,
		ShippingAddress: address,
	}

	utils.RespondJSON(w, http.StatusCreated, response)
}

// ListOrdersHandler handles GET /orders - List user's orders
func ListOrdersHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Parse pagination
	pagination := utils.ParsePagination(r)

	// Count total
	countQuery := `SELECT COUNT(*) FROM orders WHERE user_id = $1`
	var total int
	err := database.DB.QueryRow(countQuery, user.ID).Scan(&total)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to count orders")
		return
	}

	// Get orders (COALESCE nullable M-PESA strings so Scan into string succeeds)
	query := `
		SELECT id, user_id, total_amount, status, shipping_address_id,
		       COALESCE(mpesa_checkout_request_id, ''), COALESCE(mpesa_merchant_request_id, ''), COALESCE(mpesa_receipt_number, ''),
		       created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := database.DB.Query(query, user.ID, pagination.Limit, pagination.Offset)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch orders")
		return
	}
	defer rows.Close()

	orders := []models.Order{}
	for rows.Next() {
		var order models.Order
		err := rows.Scan(
			&order.ID, &order.UserID, &order.TotalAmount, &order.Status,
			&order.ShippingAddressID,
			&order.MpesaCheckoutRequestID, &order.MpesaMerchantRequestID, &order.MpesaReceiptNumber,
			&order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to scan orders")
			return
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to iterate orders")
		return
	}

	response := OrderListResponse{
		Orders:     orders,
		Pagination: utils.CreatePaginationMeta(total, pagination.Limit, pagination.Offset),
	}

	utils.RespondJSON(w, http.StatusOK, response)
}

// GetOrderHandler handles GET /orders/:id - Get order details
func GetOrderHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Order ID is required")
		return
	}
	orderIDStr := pathParts[len(pathParts)-1]

	// Validate UUID
	orderID, err := utils.ValidateUUID(orderIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Get order (COALESCE nullable string columns so Scan into string succeeds)
	var order models.Order
	orderQuery := `
		SELECT id, user_id, total_amount, status, shipping_address_id,
		       COALESCE(guest_email, ''), COALESCE(guest_name, ''), COALESCE(shipping_full_name, ''), COALESCE(shipping_street_address, ''),
		       COALESCE(shipping_city, ''), COALESCE(shipping_state, ''), COALESCE(shipping_postal_code, ''), COALESCE(shipping_country, ''), COALESCE(shipping_phone, ''),
		       COALESCE(mpesa_checkout_request_id, ''), COALESCE(mpesa_merchant_request_id, ''), COALESCE(mpesa_receipt_number, ''),
		       created_at, updated_at
		FROM orders
		WHERE id = $1 AND user_id = $2
	`
	err = database.DB.QueryRow(orderQuery, orderID, user.ID).Scan(
		&order.ID, &order.UserID, &order.TotalAmount, &order.Status,
		&order.ShippingAddressID,
		&order.GuestEmail, &order.GuestName, &order.ShippingFullName, &order.ShippingStreetAddress,
		&order.ShippingCity, &order.ShippingState, &order.ShippingPostalCode, &order.ShippingCountry, &order.ShippingPhone,
		&order.MpesaCheckoutRequestID, &order.MpesaMerchantRequestID, &order.MpesaReceiptNumber,
		&order.CreatedAt, &order.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Order not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch order")
		return
	}

	// Get order items
	itemsQuery := `
		SELECT id, order_id, product_id, quantity, price_at_purchase, created_at
		FROM order_items
		WHERE order_id = $1
		ORDER BY created_at ASC
	`
	rows, err := database.DB.Query(itemsQuery, order.ID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch order items")
		return
	}
	defer rows.Close()

	items := []models.OrderItem{}
	for rows.Next() {
		var item models.OrderItem
		err := rows.Scan(
			&item.ID, &item.OrderID, &item.ProductID, &item.Quantity,
			&item.PriceAtPurchase, &item.CreatedAt,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to scan order items")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to iterate order items")
		return
	}

	// Get shipping address (from addresses table for user orders, or from inline fields for guest)
	var address models.Address
	if order.ShippingAddressID.Valid {
		addressQuery := `SELECT id, user_id, full_name, street_address, city, state, postal_code, country, phone, is_default, created_at, updated_at FROM addresses WHERE id = $1`
		err = database.DB.QueryRow(addressQuery, order.ShippingAddressID.UUID).Scan(
			&address.ID, &address.UserID, &address.FullName, &address.StreetAddress,
			&address.City, &address.State, &address.PostalCode, &address.Country,
			&address.Phone, &address.IsDefault, &address.CreatedAt, &address.UpdatedAt,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch shipping address")
			return
		}
	} else {
		// Guest order: build address from inline shipping fields
		address.FullName = order.ShippingFullName
		address.StreetAddress = order.ShippingStreetAddress
		address.City = order.ShippingCity
		address.State = order.ShippingState
		address.PostalCode = order.ShippingPostalCode
		address.Country = order.ShippingCountry
		address.Phone = order.ShippingPhone
	}

	response := models.OrderWithItems{
		Order:           order,
		Items:           items,
		ShippingAddress: address,
	}

	utils.RespondJSON(w, http.StatusOK, response)
}

// UpdateOrderStatusHandler handles PUT /admin/orders/:id/status - Update order status (Admin only)
func UpdateOrderStatusHandler(w http.ResponseWriter, r *http.Request) {
	var req UpdateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Validate status
	validStatuses := map[models.OrderStatus]bool{
		models.OrderStatusPending:    true,
		models.OrderStatusProcessing: true,
		models.OrderStatusShipped:    true,
		models.OrderStatusDelivered:  true,
		models.OrderStatusCancelled:  true,
		models.OrderStatusRefunded:   true,
	}
	if !validStatuses[req.Status] {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "Invalid order status")
		return
	}

	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Order ID is required")
		return
	}
	orderIDStr := pathParts[len(pathParts)-2] // Second to last (before "status")

	// Validate UUID
	orderID, err := utils.ValidateUUID(orderIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Verify order exists (COALESCE nullable M-PESA strings so Scan succeeds)
	var existingOrder models.Order
	verifyQuery := `SELECT id, user_id, total_amount, status, shipping_address_id, COALESCE(mpesa_checkout_request_id, ''), COALESCE(mpesa_merchant_request_id, ''), COALESCE(mpesa_receipt_number, ''), created_at, updated_at FROM orders WHERE id = $1`
	err = database.DB.QueryRow(verifyQuery, orderID).Scan(
		&existingOrder.ID, &existingOrder.UserID, &existingOrder.TotalAmount, &existingOrder.Status,
		&existingOrder.ShippingAddressID, &existingOrder.MpesaCheckoutRequestID, &existingOrder.MpesaMerchantRequestID, &existingOrder.MpesaReceiptNumber,
		&existingOrder.CreatedAt, &existingOrder.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Order not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to verify order")
		return
	}

	// Update order status (COALESCE nullable M-PESA strings in RETURNING so Scan succeeds)
	updateQuery := `
		UPDATE orders 
		SET status = $1, updated_at = CURRENT_TIMESTAMP 
		WHERE id = $2
		RETURNING id, user_id, total_amount, status, shipping_address_id, COALESCE(mpesa_checkout_request_id, ''), COALESCE(mpesa_merchant_request_id, ''), COALESCE(mpesa_receipt_number, ''), created_at, updated_at
	`
	var order models.Order
	err = database.DB.QueryRow(updateQuery, req.Status, orderID).Scan(
		&order.ID, &order.UserID, &order.TotalAmount, &order.Status,
		&order.ShippingAddressID, &order.MpesaCheckoutRequestID, &order.MpesaMerchantRequestID, &order.MpesaReceiptNumber,
		&order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update order status")
		return
	}

	utils.RespondJSON(w, http.StatusOK, order)
}

// ListAllOrdersHandler handles GET /admin/orders - List all orders with filters (Admin only)
func ListAllOrdersHandler(w http.ResponseWriter, r *http.Request) {
	// Parse pagination
	pagination := utils.ParsePagination(r)

	// Parse filters
	statusFilter := r.URL.Query().Get("status")
	userIDFilter := r.URL.Query().Get("user_id")
	startDateFilter := r.URL.Query().Get("start_date")
	endDateFilter := r.URL.Query().Get("end_date")

	// Build query (COALESCE nullable M-PESA strings so Scan into string succeeds)
	query := `SELECT id, user_id, total_amount, status, shipping_address_id, COALESCE(mpesa_checkout_request_id, '') as mpesa_checkout_request_id, COALESCE(mpesa_merchant_request_id, '') as mpesa_merchant_request_id, COALESCE(mpesa_receipt_number, '') as mpesa_receipt_number, created_at, updated_at FROM orders WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM orders WHERE 1=1`
	args := []interface{}{}
	countArgs := []interface{}{}
	argCount := 1

	if statusFilter != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, statusFilter)
		countArgs = append(countArgs, statusFilter)
		argCount++
	}

	if userIDFilter != "" {
		userID, err := utils.ValidateUUID(userIDFilter)
		if err == nil {
			query += fmt.Sprintf(" AND user_id = $%d", argCount)
			countQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
			args = append(args, userID)
			countArgs = append(countArgs, userID)
			argCount++
		}
	}

	if startDateFilter != "" {
		_, err := time.Parse("2006-01-02", startDateFilter)
		if err == nil {
			query += fmt.Sprintf(" AND created_at >= $%d", argCount)
			countQuery += fmt.Sprintf(" AND created_at >= $%d", argCount)
			args = append(args, startDateFilter)
			countArgs = append(countArgs, startDateFilter)
			argCount++
		}
	}

	if endDateFilter != "" {
		_, err := time.Parse("2006-01-02", endDateFilter)
		if err == nil {
			query += fmt.Sprintf(" AND created_at <= $%d", argCount)
			countQuery += fmt.Sprintf(" AND created_at <= $%d", argCount)
			args = append(args, endDateFilter+" 23:59:59")
			countArgs = append(countArgs, endDateFilter+" 23:59:59")
			argCount++
		}
	}

	// Count total
	var total int
	err := database.DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to count orders")
		return
	}

	// Add ordering and pagination
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, pagination.Limit, pagination.Offset)

	// Execute query
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch orders")
		return
	}
	defer rows.Close()

	orders := []models.Order{}
	for rows.Next() {
		var order models.Order
		err := rows.Scan(
			&order.ID, &order.UserID, &order.TotalAmount, &order.Status,
			&order.ShippingAddressID,
			&order.MpesaCheckoutRequestID, &order.MpesaMerchantRequestID, &order.MpesaReceiptNumber,
			&order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to scan orders")
			return
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to iterate orders")
		return
	}

	response := OrderListResponse{
		Orders:     orders,
		Pagination: utils.CreatePaginationMeta(total, pagination.Limit, pagination.Offset),
	}

	utils.RespondJSON(w, http.StatusOK, response)
}
