package handlers

import (
	"auth-demo/database"
	"auth-demo/middleware"
	"auth-demo/models"
	"auth-demo/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// CreateAddressRequest represents the request body for creating an address
type CreateAddressRequest struct {
	FullName      string `json:"full_name"`
	StreetAddress string `json:"street_address"`
	City          string `json:"city"`
	State         string `json:"state"`
	PostalCode    string `json:"postal_code"`
	Country       string `json:"country"`
	Phone         string `json:"phone"`
	IsDefault     bool   `json:"is_default"`
}

// UpdateAddressRequest represents the request body for updating an address
type UpdateAddressRequest struct {
	FullName      *string `json:"full_name,omitempty"`
	StreetAddress *string `json:"street_address,omitempty"`
	City          *string `json:"city,omitempty"`
	State         *string `json:"state,omitempty"`
	PostalCode    *string `json:"postal_code,omitempty"`
	Country       *string `json:"country,omitempty"`
	Phone         *string `json:"phone,omitempty"`
	IsDefault     *bool   `json:"is_default,omitempty"`
}

// ListAddressesHandler handles GET /addresses - List user's addresses
func ListAddressesHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	query := `
		SELECT id, user_id, full_name, street_address, city, state, postal_code, country, phone, is_default, created_at, updated_at
		FROM addresses
		WHERE user_id = $1
		ORDER BY is_default DESC, created_at DESC
	`

	rows, err := database.DB.Query(query, user.ID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch addresses")
		return
	}
	defer rows.Close()

	addresses := []models.Address{}
	for rows.Next() {
		var addr models.Address
		err := rows.Scan(
			&addr.ID, &addr.UserID, &addr.FullName, &addr.StreetAddress,
			&addr.City, &addr.State, &addr.PostalCode, &addr.Country,
			&addr.Phone, &addr.IsDefault, &addr.CreatedAt, &addr.UpdatedAt,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to scan addresses")
			return
		}
		addresses = append(addresses, addr)
	}

	utils.RespondJSON(w, http.StatusOK, addresses)
}

// CreateAddressHandler handles POST /addresses - Add new address
func CreateAddressHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	var req CreateAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Validate required fields
	if err := utils.ValidateRequired(req.FullName, "full_name"); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := utils.ValidateRequired(req.StreetAddress, "street_address"); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := utils.ValidateRequired(req.City, "city"); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := utils.ValidateRequired(req.PostalCode, "postal_code"); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := utils.ValidateRequired(req.Country, "country"); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// If setting as default, unset other defaults
	if req.IsDefault {
		tx, err := database.DB.Begin()
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to start transaction")
			return
		}
		defer tx.Rollback()

		// Unset all other default addresses
		_, err = tx.Exec(`UPDATE addresses SET is_default = false WHERE user_id = $1`, user.ID)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update addresses")
			return
		}

		// Insert new address
		insertQuery := `
			INSERT INTO addresses (user_id, full_name, street_address, city, state, postal_code, country, phone, is_default)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id, created_at, updated_at
		`
		var addr models.Address
		err = tx.QueryRow(
			insertQuery, user.ID, req.FullName, req.StreetAddress,
			req.City, req.State, req.PostalCode, req.Country, req.Phone, req.IsDefault,
		).Scan(&addr.ID, &addr.CreatedAt, &addr.UpdatedAt)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to create address")
			return
		}

		if err = tx.Commit(); err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to commit transaction")
			return
		}

		addr.UserID = user.ID
		addr.FullName = req.FullName
		addr.StreetAddress = req.StreetAddress
		addr.City = req.City
		addr.State = req.State
		addr.PostalCode = req.PostalCode
		addr.Country = req.Country
		addr.Phone = req.Phone
		addr.IsDefault = req.IsDefault

		utils.RespondJSON(w, http.StatusCreated, addr)
		return
	}

	// Not setting as default, simple insert
	insertQuery := `
		INSERT INTO addresses (user_id, full_name, street_address, city, state, postal_code, country, phone, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`
	var addr models.Address
	err := database.DB.QueryRow(
		insertQuery, user.ID, req.FullName, req.StreetAddress,
		req.City, req.State, req.PostalCode, req.Country, req.Phone, req.IsDefault,
	).Scan(&addr.ID, &addr.CreatedAt, &addr.UpdatedAt)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to create address")
		return
	}

	addr.UserID = user.ID
	addr.FullName = req.FullName
	addr.StreetAddress = req.StreetAddress
	addr.City = req.City
	addr.State = req.State
	addr.PostalCode = req.PostalCode
	addr.Country = req.Country
	addr.Phone = req.Phone
	addr.IsDefault = req.IsDefault

	utils.RespondJSON(w, http.StatusCreated, addr)
}

// UpdateAddressHandler handles PUT /addresses/:id - Update address
func UpdateAddressHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Address ID is required")
		return
	}
	addressIDStr := pathParts[len(pathParts)-1]

	// Validate UUID
	addressID, err := utils.ValidateUUID(addressIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req UpdateAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Verify address belongs to user
	var existingAddr models.Address
	verifyQuery := `SELECT id, user_id, full_name, street_address, city, state, postal_code, country, phone, is_default, created_at, updated_at FROM addresses WHERE id = $1 AND user_id = $2`
	err = database.DB.QueryRow(verifyQuery, addressID, user.ID).Scan(
		&existingAddr.ID, &existingAddr.UserID, &existingAddr.FullName, &existingAddr.StreetAddress,
		&existingAddr.City, &existingAddr.State, &existingAddr.PostalCode, &existingAddr.Country,
		&existingAddr.Phone, &existingAddr.IsDefault, &existingAddr.CreatedAt, &existingAddr.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Address not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to verify address")
		return
	}

	// If setting as default, need transaction
	if req.IsDefault != nil && *req.IsDefault {
		tx, err := database.DB.Begin()
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to start transaction")
			return
		}
		defer tx.Rollback()

		// Unset all other default addresses
		_, err = tx.Exec(`UPDATE addresses SET is_default = false WHERE user_id = $1 AND id != $2`, user.ID, addressID)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update addresses")
			return
		}

		// Build update query dynamically
		updateFields := []string{}
		args := []interface{}{}
		argCount := 1

		if req.FullName != nil {
			updateFields = append(updateFields, fmt.Sprintf("full_name = $%d", argCount))
			args = append(args, *req.FullName)
			argCount++
		}
		if req.StreetAddress != nil {
			updateFields = append(updateFields, fmt.Sprintf("street_address = $%d", argCount))
			args = append(args, *req.StreetAddress)
			argCount++
		}
		if req.City != nil {
			updateFields = append(updateFields, fmt.Sprintf("city = $%d", argCount))
			args = append(args, *req.City)
			argCount++
		}
		if req.State != nil {
			updateFields = append(updateFields, fmt.Sprintf("state = $%d", argCount))
			args = append(args, *req.State)
			argCount++
		}
		if req.PostalCode != nil {
			updateFields = append(updateFields, fmt.Sprintf("postal_code = $%d", argCount))
			args = append(args, *req.PostalCode)
			argCount++
		}
		if req.Country != nil {
			updateFields = append(updateFields, fmt.Sprintf("country = $%d", argCount))
			args = append(args, *req.Country)
			argCount++
		}
		if req.Phone != nil {
			updateFields = append(updateFields, fmt.Sprintf("phone = $%d", argCount))
			args = append(args, *req.Phone)
			argCount++
		}
		if req.IsDefault != nil {
			updateFields = append(updateFields, fmt.Sprintf("is_default = $%d", argCount))
			args = append(args, *req.IsDefault)
			argCount++
		}

		if len(updateFields) == 0 {
			utils.RespondError(w, http.StatusBadRequest, "validation_error", "No fields to update")
			return
		}

		updateFields = append(updateFields, "updated_at = CURRENT_TIMESTAMP")
		args = append(args, addressID)

		updateQuery := `UPDATE addresses SET ` + strings.Join(updateFields, ", ") + fmt.Sprintf(" WHERE id = $%d", argCount) + ` RETURNING id, user_id, full_name, street_address, city, state, postal_code, country, phone, is_default, created_at, updated_at`

		var addr models.Address
		err = tx.QueryRow(updateQuery, args...).Scan(
			&addr.ID, &addr.UserID, &addr.FullName, &addr.StreetAddress,
			&addr.City, &addr.State, &addr.PostalCode, &addr.Country,
			&addr.Phone, &addr.IsDefault, &addr.CreatedAt, &addr.UpdatedAt,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update address")
			return
		}

		if err = tx.Commit(); err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to commit transaction")
			return
		}

		utils.RespondJSON(w, http.StatusOK, addr)
		return
	}

	// Not setting as default, simple update
	updateFields := []string{}
	args := []interface{}{}
	argCount := 1

	if req.FullName != nil {
		updateFields = append(updateFields, fmt.Sprintf("full_name = $%d", argCount))
		args = append(args, *req.FullName)
		argCount++
	}
	if req.StreetAddress != nil {
		updateFields = append(updateFields, fmt.Sprintf("street_address = $%d", argCount))
		args = append(args, *req.StreetAddress)
		argCount++
	}
	if req.City != nil {
		updateFields = append(updateFields, fmt.Sprintf("city = $%d", argCount))
		args = append(args, *req.City)
		argCount++
	}
	if req.State != nil {
		updateFields = append(updateFields, fmt.Sprintf("state = $%d", argCount))
		args = append(args, *req.State)
		argCount++
	}
	if req.PostalCode != nil {
		updateFields = append(updateFields, fmt.Sprintf("postal_code = $%d", argCount))
		args = append(args, *req.PostalCode)
		argCount++
	}
	if req.Country != nil {
		updateFields = append(updateFields, fmt.Sprintf("country = $%d", argCount))
		args = append(args, *req.Country)
		argCount++
	}
	if req.Phone != nil {
		updateFields = append(updateFields, fmt.Sprintf("phone = $%d", argCount))
		args = append(args, *req.Phone)
		argCount++
	}
	if req.IsDefault != nil {
		updateFields = append(updateFields, fmt.Sprintf("is_default = $%d", argCount))
		args = append(args, *req.IsDefault)
		argCount++
	}

	if len(updateFields) == 0 {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "No fields to update")
		return
	}

	updateFields = append(updateFields, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, addressID)

	updateQuery := `UPDATE addresses SET ` + strings.Join(updateFields, ", ") + fmt.Sprintf(" WHERE id = $%d", argCount) + ` RETURNING id, user_id, full_name, street_address, city, state, postal_code, country, phone, is_default, created_at, updated_at`

	var addr models.Address
	err = database.DB.QueryRow(updateQuery, args...).Scan(
		&addr.ID, &addr.UserID, &addr.FullName, &addr.StreetAddress,
		&addr.City, &addr.State, &addr.PostalCode, &addr.Country,
		&addr.Phone, &addr.IsDefault, &addr.CreatedAt, &addr.UpdatedAt,
	)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update address")
		return
	}

	utils.RespondJSON(w, http.StatusOK, addr)
}

// DeleteAddressHandler handles DELETE /addresses/:id - Delete address
func DeleteAddressHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Address ID is required")
		return
	}
	addressIDStr := pathParts[len(pathParts)-1]

	// Validate UUID
	addressID, err := utils.ValidateUUID(addressIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Verify address belongs to user
	var foundID uuid.UUID
	verifyQuery := `SELECT id FROM addresses WHERE id = $1 AND user_id = $2`
	err = database.DB.QueryRow(verifyQuery, addressID, user.ID).Scan(&foundID)
	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Address not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to verify address")
		return
	}

	// Delete address
	deleteQuery := `DELETE FROM addresses WHERE id = $1`
	result, err := database.DB.Exec(deleteQuery, addressID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to delete address")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to delete address")
		return
	}

	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Address not found")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, nil, "Address deleted successfully")
}

// SetDefaultAddressHandler handles POST /addresses/:id/default - Set address as default
func SetDefaultAddressHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Address ID is required")
		return
	}
	addressIDStr := pathParts[len(pathParts)-2] // Second to last (before "default")

	// Validate UUID
	addressID, err := utils.ValidateUUID(addressIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Verify address belongs to user
	var foundID uuid.UUID
	verifyQuery := `SELECT id FROM addresses WHERE id = $1 AND user_id = $2`
	err = database.DB.QueryRow(verifyQuery, addressID, user.ID).Scan(&foundID)
	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Address not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to verify address")
		return
	}

	// Use transaction to unset other defaults and set this one
	tx, err := database.DB.Begin()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Unset all other default addresses
	_, err = tx.Exec(`UPDATE addresses SET is_default = false WHERE user_id = $1`, user.ID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update addresses")
		return
	}

	// Set this address as default
	updateQuery := `UPDATE addresses SET is_default = true, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING id, user_id, full_name, street_address, city, state, postal_code, country, phone, is_default, created_at, updated_at`
	var addr models.Address
	err = tx.QueryRow(updateQuery, addressID).Scan(
		&addr.ID, &addr.UserID, &addr.FullName, &addr.StreetAddress,
		&addr.City, &addr.State, &addr.PostalCode, &addr.Country,
		&addr.Phone, &addr.IsDefault, &addr.CreatedAt, &addr.UpdatedAt,
	)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update address")
		return
	}

	if err = tx.Commit(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to commit transaction")
		return
	}

	utils.RespondJSON(w, http.StatusOK, addr)
}
