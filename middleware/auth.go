package middleware

import (
	"auth-demo/database"
	"auth-demo/models"
	"context"
	"encoding/json"
	"net/http"
	"os"

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

// AuthMiddleware validates JWT token and adds user to context
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "unauthorized: missing token", http.StatusUnauthorized)
			return
		}

		claims := &Claims{}
		tkn, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
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

		// Fetch user from database to get current role
		var user models.User
		query := `SELECT id, username, email, role, created_at, updated_at FROM users WHERE id = $1`
		err = database.DB.QueryRow(query, userID).Scan(
			&user.ID, &user.Username, &user.Email, &user.Role,
			&user.CreatedAt, &user.UpdatedAt,
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
