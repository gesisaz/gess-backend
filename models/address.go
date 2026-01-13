package models

import (
	"time"

	"github.com/google/uuid"
)

type Address struct {
	ID            uuid.UUID `json:"id" db:"id"`
	UserID        uuid.UUID `json:"user_id" db:"user_id"`
	FullName      string    `json:"full_name" db:"full_name"`
	StreetAddress string    `json:"street_address" db:"street_address"`
	City          string    `json:"city" db:"city"`
	State         string    `json:"state" db:"state"`
	PostalCode    string    `json:"postal_code" db:"postal_code"`
	Country       string    `json:"country" db:"country"`
	Phone         string    `json:"phone" db:"phone"`
	IsDefault     bool      `json:"is_default" db:"is_default"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

