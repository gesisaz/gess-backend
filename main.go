package main

import (
	"auth-demo/database"
	"auth-demo/handlers"
	"auth-demo/middleware"
	"auth-demo/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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

func meHandler(w http.ResponseWriter, r *http.Request) {
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
	
	// Public routes - Authentication
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/register", registerHandler)
	mux.HandleFunc("/logout", logoutHandler)
	
	// Public routes - Products & Categories (no auth required)
	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.ListProductsHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/products/batch", handlers.BatchProductsHandler)
	mux.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) >= 3 && pathParts[len(pathParts)-1] == "reviews" {
			// /products/:id/reviews
			switch r.Method {
			case http.MethodGet:
				handlers.ListProductReviewsHandler(w, r)
			case http.MethodPost:
				middleware.AuthMiddleware(handlers.CreateReviewHandler)(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else {
			// /products/:id (existing handler)
			handlers.GetProductHandler(w, r)
		}
	})
	mux.HandleFunc("/categories", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.ListCategoriesHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/categories/", handlers.GetCategoryHandler)
	mux.HandleFunc("/brands", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.ListBrandsHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/brands/", handlers.GetBrandHandler)
	
	// User routes (authenticated)
	mux.HandleFunc("/me", middleware.AuthMiddleware(meHandler))
	
	// Cart routes (authenticated)
	mux.HandleFunc("/cart", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetCartHandler(w, r)
		case http.MethodDelete:
			handlers.ClearCartHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/cart/items", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.AddCartItemHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/cart/items/", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			handlers.UpdateCartItemHandler(w, r)
		case http.MethodDelete:
			handlers.DeleteCartItemHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	
	// Address routes (authenticated)
	mux.HandleFunc("/addresses", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.ListAddressesHandler(w, r)
		case http.MethodPost:
			handlers.CreateAddressHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/addresses/", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) >= 3 && pathParts[len(pathParts)-1] == "default" {
			// POST /addresses/:id/default
			if r.Method == http.MethodPost {
				handlers.SetDefaultAddressHandler(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else {
			// PUT /addresses/:id or DELETE /addresses/:id
			switch r.Method {
			case http.MethodPut:
				handlers.UpdateAddressHandler(w, r)
			case http.MethodDelete:
				handlers.DeleteAddressHandler(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		}
	}))
	
	// M-PESA webhook (public, no auth - Safaricom calls this)
	mux.HandleFunc("/webhooks/mpesa/stk", handlers.MpesaSTKCallbackHandler)

	// Guest checkout (public, no auth)
	mux.HandleFunc("/checkout/guest", handlers.GuestCheckoutHandler)

	// Order routes (authenticated)
	mux.HandleFunc("/orders", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.ListOrdersHandler(w, r)
		case http.MethodPost:
			handlers.CreateOrderHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/orders/checkout-mpesa", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.CheckoutMpesaHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/orders/", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.GetOrderHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	
	// Review routes
	mux.HandleFunc("/reviews/me", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.ListUserReviewsHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/reviews/", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			handlers.UpdateReviewHandler(w, r)
		case http.MethodDelete:
			handlers.DeleteReviewHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	
	// Admin routes - Dashboard
	mux.HandleFunc("/admin/dashboard", middleware.AdminMiddleware(adminDashboardHandler))
	
	// Admin routes - Product Management
	mux.HandleFunc("/admin/products", middleware.AdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.CreateProductHandler(w, r)
		case http.MethodGet:
			handlers.ListProductsHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/admin/products/", middleware.AdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			handlers.UpdateProductHandler(w, r)
		case http.MethodDelete:
			handlers.DeleteProductHandler(w, r)
		case http.MethodGet:
			handlers.GetProductHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	
	// Admin routes - Category Management
	mux.HandleFunc("/admin/categories", middleware.AdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.CreateCategoryHandler(w, r)
		case http.MethodGet:
			handlers.ListCategoriesHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/admin/categories/", middleware.AdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			handlers.UpdateCategoryHandler(w, r)
		case http.MethodDelete:
			handlers.DeleteCategoryHandler(w, r)
		case http.MethodGet:
			handlers.GetCategoryHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	
	// Admin routes - Brand Management
	mux.HandleFunc("/admin/brands", middleware.AdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.CreateBrandHandler(w, r)
		case http.MethodGet:
			handlers.ListBrandsHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
		mux.HandleFunc("/admin/brands/", middleware.AdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			handlers.UpdateBrandHandler(w, r)
		case http.MethodDelete:
			handlers.DeleteBrandHandler(w, r)
		case http.MethodGet:
			handlers.GetBrandHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	
	// Admin routes - Order Management
	mux.HandleFunc("/admin/orders", middleware.AdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.ListAllOrdersHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/admin/orders/", middleware.AdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) >= 4 && pathParts[len(pathParts)-1] == "status" {
			// PUT /admin/orders/:id/status
			if r.Method == http.MethodPut {
				handlers.UpdateOrderStatusHandler(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	// Configure CORS - allow store frontend and admin UI origins
	baseURL := os.Getenv("BASE_URL")
	adminUIURL := os.Getenv("ADMIN_UI_URL")
	allowedOrigins := []string{baseURL}
	if allowedOrigins[0] == "" {
		allowedOrigins[0] = "http://localhost:3000"
	}
	if adminUIURL != "" {
		allowedOrigins = append(allowedOrigins, adminUIURL)
	} else {
		allowedOrigins = append(allowedOrigins, "http://localhost:4200")
	}

	handler := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
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
