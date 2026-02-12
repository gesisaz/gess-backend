package middleware

import (
	"auth-demo/database"
	"auth-demo/models"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte(os.Getenv("JWT_SECRET"))

func init() {
	if len(jwtKey) == 0 {
		jwtKey = []byte("supersecretkey") // fallback for demo
	}
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type contextKey string

const UserContextKey contextKey = "user"

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
			http.Error(w, "unauthorized: missing token", http.StatusUnauthorized)
			return
		}

		claims := &Claims{}
		tkn, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !tkn.Valid {
			http.Error(w, "unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		// Get user ID from claims
		userID := claims.Subject
		if userID == "" {
			http.Error(w, "unauthorized: invalid token claims", http.StatusUnauthorized)
			return
		}

		// Fetch user from database to get current role and verification status
		var user models.User
		query := `SELECT id, username, email, role, email_verified_at, created_at, updated_at FROM users WHERE id = $1`
		err = database.DB.QueryRow(query, userID).Scan(
			&user.ID, &user.Username, &user.Email, &user.Role,
			&user.EmailVerifiedAt, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			http.Error(w, "unauthorized: user not found", http.StatusUnauthorized)
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), UserContextKey, &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// AdminMiddleware ensures the user has admin role
func AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(UserContextKey).(*models.User)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if !user.IsAdmin() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "forbidden: admin access required",
			})
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
