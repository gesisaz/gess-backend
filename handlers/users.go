package handlers

import (
	"encoding/json"
	"net/http"

	"gess-backend/database"
	"gess-backend/middleware"
	"gess-backend/models"
	"gess-backend/utils"

	"github.com/google/uuid"
)

// CreateAdminUserRequest is the body for POST /admin/users (admin-only).
type CreateAdminUserRequest struct {
	Username string          `json:"username"`
	Email    string          `json:"email"`
	Password string          `json:"password"`
	Role     models.UserRole `json:"role"`
}

// CreateAdminUserHandler creates a user with an explicit role (admin API only).
func CreateAdminUserHandler(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	var req CreateAdminUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "username and password are required")
		return
	}

	if err := utils.ValidateEmail(req.Email); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	role := req.Role
	switch role {
	case models.UserRoleUser, models.UserRoleAdmin:
	case models.UserRoleSuperAdmin:
		if !caller.IsSuperAdmin() {
			utils.RespondError(w, http.StatusForbidden, "forbidden", "Only super_admin can assign the super_admin role")
			return
		}
	default:
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "role must be user, admin, or super_admin")
		return
	}

	hashedPassword, err := models.HashPassword(req.Password)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "internal_error", "Could not process password")
		return
	}

	var userID uuid.UUID
	q := `INSERT INTO users (username, email, password_hash, role) VALUES ($1, $2, $3, $4) RETURNING id`
	err = database.DB.QueryRow(q, req.Username, req.Email, hashedPassword, role).Scan(&userID)
	if err != nil {
		utils.RespondError(w, http.StatusConflict, "conflict", "Username or email already exists")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "user created successfully",
		"user": map[string]interface{}{
			"id":       userID.String(),
			"username": req.Username,
			"email":    req.Email,
			"role":     role,
		},
	})
}
