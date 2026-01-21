package main

import (
	"auth-demo/database"
	"auth-demo/middleware"
	"auth-demo/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/cors"
)

var jwtKey []byte

func init() {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtKey = []byte("supersecretkey") // fallback for demo
	} else {
		jwtKey = []byte(jwtSecret)
	}
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Query user from database (including role)
	var user models.User
	query := `SELECT id, username, email, password_hash, role, created_at, updated_at 
	          FROM users WHERE username = $1 OR email = $1`
	err := database.DB.QueryRow(query, creds.Username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, 
		&user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Check password
	if !user.CheckPassword(creds.Password) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Expires:  expirationTime,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
		Path:     "/",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": tokenString,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": fmt.Sprintf("Welcome, %s!", user.Username),
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

func adminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": fmt.Sprintf("Welcome to admin dashboard, %s!", user.Username),
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
		Path:     "/",
	})
	json.NewEncoder(w).Encode(map[string]string{"message": "logged out"})
}

type RegisterRequest struct {
	Username string          `json:"username"`
	Email    string          `json:"email"`
	Password string          `json:"password"`
	Role     models.UserRole `json:"role,omitempty"` // Optional, only admins can set this
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password required", http.StatusBadRequest)
		return
	}

	// Set default email if not provided
	if req.Email == "" {
		req.Email = req.Username + "@example.com"
	}

	// Default role is 'user', unless specified (would need admin check for that)
	if req.Role == "" {
		req.Role = models.UserRoleUser
	} else {
		// TODO: Only allow admins to set custom roles
		// For now, force role to 'user' for security
		req.Role = models.UserRoleUser
	}

	// Hash password
	hashedPassword, err := models.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Insert user into database
	var userID uuid.UUID
	query := `INSERT INTO users (username, email, password_hash, role) 
	          VALUES ($1, $2, $3, $4) RETURNING id`
	err = database.DB.QueryRow(query, req.Username, req.Email, hashedPassword, req.Role).Scan(&userID)
	if err != nil {
		http.Error(w, "username or email already exists", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "user created successfully",
		"user": map[string]interface{}{
			"id":       userID.String(),
			"username": req.Username,
			"email":    req.Email,
			"role":     req.Role,
		},
	})
}

func main() {
	// Initialize database connection
	if err := database.ConnectDB(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	mux := http.NewServeMux()
	
	// Public routes
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/register", registerHandler)
	mux.HandleFunc("/logout", logoutHandler)
	
	// User routes (authenticated)
	mux.HandleFunc("/protected", middleware.AuthMiddleware(protectedHandler))
	
	// Admin routes (authenticated + admin role)
	mux.HandleFunc("/admin/dashboard", middleware.AdminMiddleware(adminDashboardHandler))

	// Configure CORS
	allowedOrigin := os.Getenv("BASE_URL")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:3000"
	}

	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{allowedOrigin},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
	}).Handler(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on :%s\n", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
