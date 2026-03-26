package middleware

import (
	"context"
	"net/http"
	"strings"

	"gess-backend/database"
	"gess-backend/internal/jwtutil"
	"gess-backend/models"
	"gess-backend/utils"
)

type contextKey string

const (
	UserContextKey      contextKey = "user"
	loggerContextKey    contextKey = "logger"
	requestIDContextKey contextKey = "request_id"
)

// getTokenFromRequest returns the JWT from Authorization: Bearer <token> or from cookie.
func getTokenFromRequest(r *http.Request) string {
	if ah := r.Header.Get("Authorization"); ah != "" {
		const prefix = "Bearer "
		if len(ah) > len(prefix) && strings.EqualFold(ah[:len(prefix)], prefix) {
			if token := strings.TrimSpace(ah[len(prefix):]); token != "" {
				return token
			}
		}
	}
	if c, err := r.Cookie("token"); err == nil && c.Value != "" {
		return c.Value
	}
	return ""
}

// AuthMiddleware validates JWT token and adds user to context
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := getTokenFromRequest(r)
		if tokenString == "" {
			utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "Missing token")
			return
		}

		claims, err := jwtutil.ParseToken(tokenString)
		if err != nil {
			utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "Invalid token")
			return
		}

		userID := claims.Subject
		if userID == "" {
			utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "Invalid token claims")
			return
		}

		var user models.User
		query := `SELECT id, username, email, role, email_verified_at, created_at, updated_at FROM users WHERE id = $1`
		err = database.DB.QueryRow(query, userID).Scan(
			&user.ID, &user.Username, &user.Email, &user.Role,
			&user.EmailVerifiedAt, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found")
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// AdminMiddleware ensures the user has admin role
func AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(UserContextKey).(*models.User)
		if !ok {
			utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
			return
		}

		if !user.IsAdmin() {
			utils.RespondError(w, http.StatusForbidden, "forbidden", "Admin access required")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext retrieves user from request context
func GetUserFromContext(r *http.Request) (*models.User, bool) {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	return user, ok
}
