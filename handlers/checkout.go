package handlers

import (
	"auth-demo/database"
	"auth-demo/mail"
	"auth-demo/models"
	"auth-demo/mpesa"
	"auth-demo/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	mpesasdk "github.com/jwambugu/mpesa-golang-sdk"
)

// GuestCheckoutRequest is the body for POST /checkout/guest
type GuestCheckoutRequest struct {
	Items           []GuestCheckoutItem  `json:"items"`
	Email           string               `json:"email"`
	GuestName       string               `json:"guest_name"`
	ShippingAddress GuestShippingAddress `json:"shipping_address"`
}

type GuestCheckoutItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type GuestShippingAddress struct {
	FullName      string `json:"full_name"`
	StreetAddress string `json:"street_address"`
	City          string `json:"city"`
	State         string `json:"state"`
	PostalCode    string `json:"postal_code"`
	Country       string `json:"country"`
	Phone         string `json:"phone"`
}

// Simple IP rate limiter for guest checkout (in-memory, 10 per minute per IP)
var (
	guestCheckoutLimiter   = make(map[string][]time.Time)
	guestCheckoutLimiterMu sync.Mutex
	guestCheckoutLimit     = 10
	guestCheckoutWindow    = time.Minute
)

func allowGuestCheckoutByIP(ip string) bool {
	guestCheckoutLimiterMu.Lock()
	defer guestCheckoutLimiterMu.Unlock()
	now := time.Now()
	cutoff := now.Add(-guestCheckoutWindow)
	// prune old entries
	if guestCheckoutLimiter[ip] != nil {
		valid := guestCheckoutLimiter[ip][:0]
		for _, t := range guestCheckoutLimiter[ip] {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		guestCheckoutLimiter[ip] = valid
	}
	if len(guestCheckoutLimiter[ip]) >= guestCheckoutLimit {
		return false
	}
	guestCheckoutLimiter[ip] = append(guestCheckoutLimiter[ip], now)
	return true
}

func getClientIP(r *http.Request) string {
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		parts := strings.Split(x, ",")
		for _, part := range parts {
			ip := strings.TrimSpace(part)
			if ip != "" {
				return ip
			}
		}
	}
	if x := r.Header.Get("X-Real-IP"); x != "" {
		if ip := strings.TrimSpace(x); ip != "" {
			return ip
		}
	}
	remoteAddr := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return strings.TrimSpace(remoteAddr[:idx])
	}
	return remoteAddr
}

// GuestCheckoutHandler handles POST /checkout/guest - Create order from guest cart (no auth).
func GuestCheckoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := getClientIP(r)
	if !allowGuestCheckoutByIP(ip) {
		utils.RespondError(w, http.StatusTooManyRequests, "rate_limit", "Too many checkout attempts. Please try again later.")
		return
	}

	var req GuestCheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	if len(req.Items) == 0 {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "Items are required")
		return
	}
	if req.Email == "" {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "Email is required")
		return
	}
	addr := &req.ShippingAddress
	if addr.FullName == "" || addr.StreetAddress == "" || addr.City == "" || addr.PostalCode == "" || addr.Country == "" {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "Shipping address must include full_name, street_address, city, postal_code, country")
		return
	}

	// Parse and validate item product IDs and quantities; load price and stock from DB
	type itemInfo struct {
		ProductID uuid.UUID
		Quantity  int
		Price     float64
		Name      string
		Stock     int
	}
	var items []itemInfo
	tx, err := database.DB.Begin()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	for _, line := range req.Items {
		if line.Quantity <= 0 {
			utils.RespondError(w, http.StatusBadRequest, "validation_error", "Quantity must be greater than 0")
			return
		}
		productID, err := utils.ValidateUUID(line.ProductID)
		if err != nil {
			utils.RespondError(w, http.StatusBadRequest, "invalid_id", "Invalid product_id: "+line.ProductID)
			return
		}
		var price float64
		var stock int
		var name string
		q := `SELECT selling_price, stock_quantity, name FROM products WHERE id = $1 FOR UPDATE`
		err = tx.QueryRow(q, productID).Scan(&price, &stock, &name)
		if err != nil {
			utils.RespondError(w, http.StatusNotFound, "not_found", "Product not found: "+line.ProductID)
			return
		}
		if stock < line.Quantity {
			utils.RespondError(w, http.StatusBadRequest, "insufficient_stock", "Insufficient stock for product: "+name)
			return
		}
		items = append(items, itemInfo{ProductID: productID, Quantity: line.Quantity, Price: price, Name: name, Stock: stock})
	}

	var totalAmount float64
	for _, it := range items {
		totalAmount += it.Price * float64(it.Quantity)
	}

	orderQuery := `
		INSERT INTO orders (
			user_id, shipping_address_id, total_amount, status,
			guest_email, guest_name,
			shipping_full_name, shipping_street_address, shipping_city, shipping_state,
			shipping_postal_code, shipping_country, shipping_phone
		) VALUES (NULL, NULL, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, total_amount, status, guest_email, guest_name,
			shipping_full_name, shipping_street_address, shipping_city, shipping_state,
			shipping_postal_code, shipping_country, shipping_phone, created_at, updated_at
	`
	var order models.Order
	err = tx.QueryRow(orderQuery,
		totalAmount, models.OrderStatusPending,
		req.Email, req.GuestName,
		addr.FullName, addr.StreetAddress, addr.City, addr.State,
		addr.PostalCode, addr.Country, addr.Phone,
	).Scan(
		&order.ID, &order.TotalAmount, &order.Status, &order.GuestEmail, &order.GuestName,
		&order.ShippingFullName, &order.ShippingStreetAddress, &order.ShippingCity, &order.ShippingState,
		&order.ShippingPostalCode, &order.ShippingCountry, &order.ShippingPhone, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to create order")
		return
	}

	orderItems := []models.OrderItem{}
	for _, it := range items {
		var orderItem models.OrderItem
		orderItemQuery := `
			INSERT INTO order_items (order_id, product_id, quantity, price_at_purchase)
			VALUES ($1, $2, $3, $4)
			RETURNING id, order_id, product_id, quantity, price_at_purchase, created_at
		`
		err = tx.QueryRow(orderItemQuery, order.ID, it.ProductID, it.Quantity, it.Price).Scan(
			&orderItem.ID, &orderItem.OrderID, &orderItem.ProductID, &orderItem.Quantity,
			&orderItem.PriceAtPurchase, &orderItem.CreatedAt,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to create order item")
			return
		}
		orderItems = append(orderItems, orderItem)

		result, err := tx.Exec(`UPDATE products SET stock_quantity = stock_quantity - $1 WHERE id = $2 AND stock_quantity >= $1`, it.Quantity, it.ProductID)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update stock")
			return
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to verify stock update")
			return
		}
		if rowsAffected == 0 {
			utils.RespondError(w, http.StatusBadRequest, "insufficient_stock", "Insufficient stock for product: "+it.Name)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to commit")
		return
	}

	stkSent := false
	msg := ""

	if mpesa.Enabled() && strings.TrimSpace(addr.Phone) != "" {
		phone := utils.NormalizeMpesaPhone(addr.Phone)
		if phone != "" {
			shortcodeStr := strings.TrimSpace(strings.TrimPrefix(getEnv("MPESA_SHORTCODE", "174379"), "0"))
			passkey := os.Getenv("MPESA_PASSKEY")
			callbackBase := os.Getenv("MPESA_CALLBACK_BASE_URL")
			if shortcodeStr != "" && passkey != "" && callbackBase != "" {
				shortcode, err := strconv.ParseUint(shortcodeStr, 10, 64)
				if err != nil || shortcode == 0 {
					utils.RespondError(w, http.StatusInternalServerError, "config_error", "Invalid MPESA_SHORTCODE")
					return
				}
				phoneUint, err := strconv.ParseUint(phone, 10, 64)
				if err == nil {
					amountKES := uint(math.Round(totalAmount))
					if amountKES < 1 {
						amountKES = 1
					}
					callbackURL := strings.TrimSuffix(callbackBase, "/") + "/webhooks/mpesa/stk"
					accountRef := order.ID.String()
					if len(accountRef) > 12 {
						accountRef = accountRef[:12]
					}
					txnDesc := "Order payment"
					if len(txnDesc) > 13 {
						txnDesc = txnDesc[:13]
					}
					ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
					stkResp, err := mpesa.Client().STKPush(ctx, passkey, mpesasdk.STKPushRequest{
						BusinessShortCode: uint(shortcode),
						TransactionType:   mpesasdk.CustomerPayBillOnlineTransactionType,
						Amount:            amountKES,
						PartyA:            uint(phoneUint),
						PartyB:            uint(shortcode),
						PhoneNumber:       phoneUint,
						CallBackURL:       callbackURL,
						AccountReference:  accountRef,
						TransactionDesc:   txnDesc,
					})
					cancel()
					if err == nil && stkResp.ErrorCode == "" && stkResp.ResponseCode == "0" {
						_, _ = database.DB.Exec(`UPDATE orders SET mpesa_checkout_request_id = $1, mpesa_merchant_request_id = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`,
							stkResp.CheckoutRequestID, stkResp.MerchantRequestID, order.ID)
						order.MpesaCheckoutRequestID = stkResp.CheckoutRequestID
						order.MpesaMerchantRequestID = stkResp.MerchantRequestID
						stkSent = true
						msg = "Complete payment on your phone"
					}
				}
			}
		}
	}

	// Send order confirmation email (do not fail request on error)
	var summaryParts []string
	for _, it := range items {
		summaryParts = append(summaryParts, fmt.Sprintf("%s x %d", it.Name, it.Quantity))
	}
	summary := strings.Join(summaryParts, "\n")
	if err := mail.SendOrderConfirmationEmail(req.Email, order.ID.String(), totalAmount, summary); err != nil {
		log.Printf("guest order confirmation email failed: %v", err)
	}

	address := models.Address{
		FullName:      order.ShippingFullName,
		StreetAddress: order.ShippingStreetAddress,
		City:          order.ShippingCity,
		State:         order.ShippingState,
		PostalCode:    order.ShippingPostalCode,
		Country:       order.ShippingCountry,
		Phone:         order.ShippingPhone,
	}
	response := struct {
		models.OrderWithItems
		StkSent bool   `json:"stk_sent"`
		Message string `json:"message,omitempty"`
	}{
		OrderWithItems: models.OrderWithItems{Order: order, Items: orderItems, ShippingAddress: address},
		StkSent:        stkSent,
		Message:        msg,
	}
	utils.RespondJSON(w, http.StatusCreated, response)
}
