package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NullUUID represents a UUID that may be null for DB/JSON.
type NullUUID struct {
	UUID  uuid.UUID
	Valid bool
}

// Scan implements sql.Scanner for NullUUID.
func (n *NullUUID) Scan(value interface{}) error {
	if value == nil {
		n.Valid = false
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if err := n.UUID.UnmarshalBinary(v); err != nil {
			return err
		}
	case string:
		if err := n.UUID.UnmarshalText([]byte(v)); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot scan %T into NullUUID", value)
	}
	n.Valid = true
	return nil
}

// Value implements driver.Valuer for NullUUID.
func (n NullUUID) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.UUID.String(), nil
}

// MarshalJSON implements json.Marshaler.
func (n NullUUID) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.UUID.String())
}

// UnmarshalJSON implements json.Unmarshaler.
func (n *NullUUID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Valid = false
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if err := n.UUID.UnmarshalText([]byte(s)); err != nil {
		return err
	}
	n.Valid = true
	return nil
}

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusRefunded   OrderStatus = "refunded"
)

type Order struct {
	ID                     uuid.UUID   `json:"id" db:"id"`
	UserID                 NullUUID    `json:"user_id" db:"user_id"`
	TotalAmount            float64     `json:"total_amount" db:"total_amount"`
	Status                 OrderStatus `json:"status" db:"status"`
	ShippingAddressID      NullUUID    `json:"shipping_address_id" db:"shipping_address_id"`
	GuestEmail             string      `json:"guest_email,omitempty" db:"guest_email"`
	GuestName              string      `json:"guest_name,omitempty" db:"guest_name"`
	ShippingFullName       string      `json:"shipping_full_name,omitempty" db:"shipping_full_name"`
	ShippingStreetAddress  string      `json:"shipping_street_address,omitempty" db:"shipping_street_address"`
	ShippingCity           string      `json:"shipping_city,omitempty" db:"shipping_city"`
	ShippingState          string      `json:"shipping_state,omitempty" db:"shipping_state"`
	ShippingPostalCode     string      `json:"shipping_postal_code,omitempty" db:"shipping_postal_code"`
	ShippingCountry        string      `json:"shipping_country,omitempty" db:"shipping_country"`
	ShippingPhone          string      `json:"shipping_phone,omitempty" db:"shipping_phone"`
	MpesaCheckoutRequestID string      `json:"mpesa_checkout_request_id,omitempty" db:"mpesa_checkout_request_id"`
	MpesaMerchantRequestID string      `json:"mpesa_merchant_request_id,omitempty" db:"mpesa_merchant_request_id"`
	MpesaReceiptNumber     string      `json:"mpesa_receipt_number,omitempty" db:"mpesa_receipt_number"`
	CreatedAt              time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at" db:"updated_at"`
}

type OrderItem struct {
	ID              uuid.UUID `json:"id" db:"id"`
	OrderID         uuid.UUID `json:"order_id" db:"order_id"`
	ProductID       uuid.UUID `json:"product_id" db:"product_id"`
	Quantity        int       `json:"quantity" db:"quantity"`
	PriceAtPurchase float64   `json:"price_at_purchase" db:"price_at_purchase"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// OrderWithItems represents an order with its items for API responses
type OrderWithItems struct {
	Order           Order       `json:"order"`
	Items           []OrderItem `json:"items"`
	ShippingAddress Address     `json:"shipping_address"`
}

