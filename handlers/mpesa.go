package handlers

import (
	"auth-demo/database"
	"auth-demo/mail"
	"auth-demo/middleware"
	"auth-demo/models"
	"auth-demo/mpesa"
	"auth-demo/utils"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	mpesasdk "github.com/jwambugu/mpesa-golang-sdk"
)

// CheckoutMpesaRequest is the request body for POST /orders/checkout-mpesa
type CheckoutMpesaRequest struct {
	ShippingAddressID uuid.UUID `json:"shipping_address_id"`
	PhoneNumber       string    `json:"phone_number"`
}

// CheckoutMpesaHandler handles POST /orders/checkout-mpesa - Create order and initiate M-PESA STK Push
func CheckoutMpesaHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	if !mpesa.Enabled() {
		utils.RespondError(w, http.StatusServiceUnavailable, "payment_unavailable", "M-PESA checkout is not configured")
		return
	}

	var req CheckoutMpesaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	phone := utils.NormalizeMpesaPhone(req.PhoneNumber)
	if phone == "" {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "Invalid phone number; use 254XXXXXXXXX or 07XXXXXXXX")
		return
	}

	var err error
	// Validate shipping address
	var address models.Address
	addressQuery := `SELECT id, user_id, full_name, street_address, city, state, postal_code, country, phone, is_default, created_at, updated_at FROM addresses WHERE id = $1 AND user_id = $2`
	err = database.DB.QueryRow(addressQuery, req.ShippingAddressID, user.ID).Scan(
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

	// Get cart and items (same as CreateOrderHandler)
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

	// Start transaction before stock validation/reservation.
	tx, err := database.DB.BeginTx(r.Context(), nil)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	cartItemsQuery := `
		SELECT ci.id, ci.product_id, ci.quantity, p.selling_price, p.stock_quantity, p.name
		FROM cart_items ci
		JOIN products p ON ci.product_id = p.id
		WHERE ci.cart_id = $1
		FOR UPDATE OF p
	`
	rows, err := tx.Query(cartItemsQuery, cartID)
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
	var cartItems []CartItemInfo
	for rows.Next() {
		var item CartItemInfo
		if err := rows.Scan(&item.ID, &item.ProductID, &item.Quantity, &item.Price, &item.StockQuantity, &item.ProductName); err != nil {
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

	for _, item := range cartItems {
		result, err := tx.Exec(
			`UPDATE products SET stock_quantity = stock_quantity - $1 WHERE id = $2 AND stock_quantity >= $1`,
			item.Quantity,
			item.ProductID,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to reserve stock")
			return
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to verify stock reservation")
			return
		}
		if rowsAffected == 0 {
			utils.RespondError(w, http.StatusBadRequest, "insufficient_stock", "Insufficient stock for "+item.ProductName)
			return
		}
	}

	var totalAmount float64
	for _, item := range cartItems {
		totalAmount += item.Price * float64(item.Quantity)
	}
	amountKES := int(math.Round(totalAmount))
	if amountKES < 1 {
		amountKES = 1
	}

	shortcodeStr := strings.TrimSpace(strings.TrimPrefix(getEnv("MPESA_SHORTCODE", "174379"), "0"))
	shortcode, err := strconv.ParseUint(shortcodeStr, 10, 64)
	if err != nil || shortcode == 0 {
		utils.RespondError(w, http.StatusInternalServerError, "config_error", "Invalid MPESA_SHORTCODE")
		return
	}
	phoneUint, err := strconv.ParseUint(phone, 10, 64)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "Invalid phone number")
		return
	}

	callbackBase := getEnv("MPESA_CALLBACK_BASE_URL", "")
	if callbackBase == "" {
		utils.RespondError(w, http.StatusInternalServerError, "config_error", "MPESA_CALLBACK_BASE_URL not set")
		return
	}
	callbackURL := strings.TrimSuffix(callbackBase, "/") + "/webhooks/mpesa/stk"
	passkey := getEnv("MPESA_PASSKEY", "")

	// Create order (COALESCE nullable M-PESA strings in RETURNING so Scan succeeds)
	var order models.Order
	orderQuery := `
		INSERT INTO orders (user_id, total_amount, status, shipping_address_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, total_amount, status, shipping_address_id, COALESCE(mpesa_checkout_request_id, ''), COALESCE(mpesa_merchant_request_id, ''), COALESCE(mpesa_receipt_number, ''), created_at, updated_at
	`
	err = tx.QueryRow(orderQuery, user.ID, totalAmount, models.OrderStatusPending, req.ShippingAddressID).Scan(
		&order.ID, &order.UserID, &order.TotalAmount, &order.Status, &order.ShippingAddressID,
		&order.MpesaCheckoutRequestID, &order.MpesaMerchantRequestID, &order.MpesaReceiptNumber,
		&order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to create order")
		return
	}

	for _, cartItem := range cartItems {
		_, err = tx.Exec(`INSERT INTO order_items (order_id, product_id, quantity, price_at_purchase) VALUES ($1, $2, $3, $4)`,
			order.ID, cartItem.ProductID, cartItem.Quantity, cartItem.Price)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to create order item")
			return
		}
	}
	_, err = tx.Exec(`DELETE FROM cart_items WHERE cart_id = $1`, cartID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to clear cart")
		return
	}

	accountRef := order.ID.String()
	if len(accountRef) > 12 {
		accountRef = accountRef[:12]
	}
	txnDesc := "Order payment"
	if len(txnDesc) > 13 {
		txnDesc = txnDesc[:13]
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	stkResp, err := mpesa.Client().STKPush(ctx, passkey, mpesasdk.STKPushRequest{
		BusinessShortCode: uint(shortcode),
		TransactionType:   mpesasdk.CustomerPayBillOnlineTransactionType,
		Amount:            uint(amountKES),
		PartyA:            uint(phoneUint),
		PartyB:            uint(shortcode),
		PhoneNumber:       phoneUint,
		CallBackURL:       callbackURL,
		AccountReference:  accountRef,
		TransactionDesc:   txnDesc,
	})
	if err != nil {
		utils.RespondError(w, http.StatusBadGateway, "payment_provider_error", "Failed to initiate M-PESA payment: "+err.Error())
		return
	}
	if stkResp.ErrorCode != "" || stkResp.ResponseCode != "0" {
		utils.RespondError(w, http.StatusBadGateway, "payment_provider_error", stkResp.ErrorMessage+" ("+stkResp.ErrorCode+")")
		return
	}

	_, err = tx.Exec(`UPDATE orders SET mpesa_checkout_request_id = $1, mpesa_merchant_request_id = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`,
		stkResp.CheckoutRequestID, stkResp.MerchantRequestID, order.ID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to save payment request")
		return
	}

	if err = tx.Commit(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to commit transaction")
		return
	}

	order.MpesaCheckoutRequestID = stkResp.CheckoutRequestID
	order.MpesaMerchantRequestID = stkResp.MerchantRequestID

	utils.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"message":             "Complete payment on your phone",
		"order":               order,
		"checkout_request_id": stkResp.CheckoutRequestID,
	})
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// MpesaSTKCallbackHandler handles POST /webhooks/mpesa/stk - Safaricom STK Push callback
func MpesaSTKCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	callback, err := mpesasdk.UnmarshalSTKPushCallback(bytes.NewReader(body))
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if callback == nil {
		respondMpesaCallback(w, 0, "Accepted")
		return
	}

	stk := callback.Body.STKCallback
	checkoutID := stk.CheckoutRequestID
	merchantID := stk.MerchantRequestID
	if checkoutID == "" && merchantID == "" {
		respondMpesaCallback(w, 0, "Accepted")
		return
	}

	var orderID uuid.UUID
	var status string
	err = database.DB.QueryRow(`
		SELECT id, status FROM orders 
		WHERE mpesa_checkout_request_id = $1 OR mpesa_merchant_request_id = $2
		LIMIT 1
	`, checkoutID, merchantID).Scan(&orderID, &status)
	if err == sql.ErrNoRows {
		respondMpesaCallback(w, 0, "Accepted")
		return
	}
	if err != nil {
		respondMpesaCallback(w, 0, "Accepted")
		return
	}

	if status != string(models.OrderStatusPending) {
		respondMpesaCallback(w, 0, "Accepted")
		return
	}

	if stk.ResultCode == 0 {
		receipt := ""
		for _, item := range stk.CallbackMetadata.Item {
			if item.Name == "MpesaReceiptNumber" {
				switch v := item.Value.(type) {
				case string:
					receipt = v
				case float64:
					receipt = strconv.FormatInt(int64(v), 10)
				}
				break
			}
		}
		_, err = database.DB.Exec(`
			UPDATE orders SET status = $1, mpesa_receipt_number = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3
		`, models.OrderStatusProcessing, receipt, orderID)
		if err != nil {
			respondMpesaCallback(w, 0, "Accepted")
			return
		}
		// Send order confirmation email (do not fail callback)
		var orderUserID models.NullUUID
		var guestEmail string
		var totalAmount float64
		err = database.DB.QueryRow(`SELECT user_id, guest_email, total_amount FROM orders WHERE id = $1`, orderID).Scan(&orderUserID, &guestEmail, &totalAmount)
		if err == nil {
			var to string
			if orderUserID.Valid {
				_ = database.DB.QueryRow(`SELECT email FROM users WHERE id = $1`, orderUserID.UUID).Scan(&to)
			} else {
				to = guestEmail
			}
			if to != "" {
				var summaryParts []string
				rows, qerr := database.DB.Query(`SELECT p.name, oi.quantity FROM order_items oi JOIN products p ON p.id = oi.product_id WHERE oi.order_id = $1`, orderID)
				if qerr == nil {
					for rows.Next() {
						var name string
						var qty int
						if rows.Scan(&name, &qty) == nil {
							summaryParts = append(summaryParts, fmt.Sprintf("%s x %d", name, qty))
						}
					}
					rows.Close()
				}
				summary := strings.Join(summaryParts, "\n")
				if sendErr := mail.SendOrderConfirmationEmail(to, orderID.String(), totalAmount, summary); sendErr != nil {
					log.Printf("mpesa callback order confirmation email failed: %v", sendErr)
				}
			}
		}
	} else {
		_, err = database.DB.Exec(`UPDATE orders SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`, models.OrderStatusCancelled, orderID)
		if err != nil {
			respondMpesaCallback(w, 0, "Accepted")
			return
		}
		// Restore stock
		rows, err := database.DB.Query(`SELECT product_id, quantity FROM order_items WHERE order_id = $1`, orderID)
		if err == nil {
			for rows.Next() {
				var productID uuid.UUID
				var qty int
				if rows.Scan(&productID, &qty) == nil {
					database.DB.Exec(`UPDATE products SET stock_quantity = stock_quantity + $1 WHERE id = $2`, qty, productID)
				}
			}
			rows.Close()
		}
	}

	respondMpesaCallback(w, 0, "Accepted")
}

func respondMpesaCallback(w http.ResponseWriter, resultCode int, resultDesc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ResultCode": resultCode,
		"ResultDesc": resultDesc,
	})
}
