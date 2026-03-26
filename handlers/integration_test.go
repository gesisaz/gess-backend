//go:build integration

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gess-backend/database"
	"gess-backend/handlers"
	"gess-backend/internal/jwtutil"
	"gess-backend/internal/logger"
	"gess-backend/middleware"
	"gess-backend/models"

	"github.com/google/uuid"
)

func TestMain(m *testing.M) {
	logger.Init()
	mpesa.Init()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		slog.Error("TEST_DATABASE_URL is required for integration tests")
		os.Exit(1)
	}
	_ = os.Setenv("DATABASE_URL", url)
	if err := database.ConnectDB(); err != nil {
		slog.Error("database connect failed", "err", err)
		os.Exit(1)
	}
	jwtutil.Init([]byte("01234567890123456789012345678901"))
	code := m.Run()
	_ = database.Close()
	os.Exit(code)
}

func insertTestProduct(t *testing.T, name string, stock int) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := database.DB.QueryRow(`
		INSERT INTO products (name, buying_price, selling_price, stock_quantity)
		VALUES ($1, 1, 10, $2) RETURNING id
	`, name, stock).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = database.DB.Exec(`DELETE FROM order_items WHERE product_id = $1`, id)
		_, _ = database.DB.Exec(`DELETE FROM cart_items WHERE product_id = $1`, id)
		_, _ = database.DB.Exec(`DELETE FROM products WHERE id = $1`, id)
	})
	return id
}

func insertTestUser(t *testing.T, username, email string) uuid.UUID {
	t.Helper()
	hash, err := models.HashPassword("testpass123")
	if err != nil {
		t.Fatal(err)
	}
	var id uuid.UUID
	err = database.DB.QueryRow(`
		INSERT INTO users (username, email, password_hash, role)
		VALUES ($1, $2, $3, 'user') RETURNING id
	`, username, email, hash).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = database.DB.Exec(`DELETE FROM cart_items WHERE cart_id IN (SELECT id FROM carts WHERE user_id = $1)`, id)
		_, _ = database.DB.Exec(`DELETE FROM carts WHERE user_id = $1`, id)
		_, _ = database.DB.Exec(`DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

func TestGuestCheckout_insufficientStock(t *testing.T) {
	pid := insertTestProduct(t, "itest_guest_stock", 1)
	body := map[string]interface{}{
		"email": "guest@test.example",
		"items": []map[string]interface{}{{"product_id": pid.String(), "quantity": 5}},
		"shipping_address": map[string]interface{}{
			"full_name": "A", "street_address": "1 St", "city": "C",
			"postal_code": "12345", "country": "KE",
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/checkout/guest", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "198.51.100.10")
	rec := httptest.NewRecorder()
	handlers.GuestCheckoutHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "insufficient_stock") {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestGuestCheckout_success(t *testing.T) {
	pid := insertTestProduct(t, "itest_guest_ok", 10)
	body := map[string]interface{}{
		"email": "guest2@test.example",
		"items": []map[string]interface{}{{"product_id": pid.String(), "quantity": 2}},
		"shipping_address": map[string]interface{}{
			"full_name": "A", "street_address": "1 St", "city": "C",
			"postal_code": "12345", "country": "KE",
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/checkout/guest", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "198.51.100.11")
	rec := httptest.NewRecorder()
	handlers.GuestCheckoutHandler(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var stock int
	if err := database.DB.QueryRow(`SELECT stock_quantity FROM products WHERE id = $1`, pid).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	if stock != 8 {
		t.Fatalf("stock want 8 got %d", stock)
	}
	t.Cleanup(func() {
		_, _ = database.DB.Exec(`DELETE FROM order_items WHERE order_id IN (SELECT id FROM orders WHERE guest_email = $1)`, "guest2@test.example")
		_, _ = database.DB.Exec(`DELETE FROM orders WHERE guest_email = $1`, "guest2@test.example")
	})
}

func TestAddCartItem_insufficientStock(t *testing.T) {
	uid := insertTestUser(t, "itest_cart_u", "itcart@test.example")
	pid := insertTestProduct(t, "itest_cart_p", 3)
	u := models.User{ID: uid, Username: "itest_cart_u", Email: "itcart@test.example", Role: models.UserRoleUser}
	ctx := context.WithValue(context.Background(), middleware.UserContextKey, &u)
	body := map[string]interface{}{"product_id": pid.String(), "quantity": 10}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/cart/items", bytes.NewReader(b))
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handlers.AddCartItemHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestMpesaSTKCallback_success(t *testing.T) {
	const checkoutID = "ws_CO_itest_mpesa_success"
	const merchantID = "29115-itest-1"
	_, err := database.DB.Exec(`
		INSERT INTO orders (user_id, total_amount, status, guest_email, mpesa_checkout_request_id, mpesa_merchant_request_id)
		VALUES (NULL, 10.00, 'pending', 'mpesa_itest@test.example', $1, $2)
	`, checkoutID, merchantID)
	if err != nil {
		t.Fatal(err)
	}
	var orderID uuid.UUID
	if err := database.DB.QueryRow(
		`SELECT id FROM orders WHERE mpesa_checkout_request_id = $1`, checkoutID,
	).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = database.DB.Exec(`DELETE FROM orders WHERE id = $1`, orderID)
	})

	payload := `{
		"Body": {
			"stkCallback": {
				"MerchantRequestID": "` + merchantID + `",
				"CheckoutRequestID": "` + checkoutID + `",
				"ResultCode": 0,
				"ResultDesc": "The service request is processed successfully.",
				"CallbackMetadata": {
					"Item": [
						{"Name": "MpesaReceiptNumber", "Value": "RTEST123"}
					]
				}
			}
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/mpesa/stk", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handlers.MpesaSTKCallbackHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var status string
	var receipt string
	if err := database.DB.QueryRow(`SELECT status::text, COALESCE(mpesa_receipt_number,'') FROM orders WHERE id = $1`, orderID).Scan(&status, &receipt); err != nil {
		t.Fatal(err)
	}
	if status != "processing" {
		t.Fatalf("status want processing got %q", status)
	}
	if receipt != "RTEST123" {
		t.Fatalf("receipt %q", receipt)
	}
}
