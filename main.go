package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"gess-backend/database"
	"gess-backend/handlers"
	"gess-backend/internal/cookieopts"
	"gess-backend/internal/jwtutil"
	"gess-backend/internal/logger"
	"gess-backend/internal/metrics"
	"gess-backend/internal/secrets"
	"gess-backend/internal/tracing"
	"gess-backend/middleware"
	"gess-backend/models"
	"gess-backend/mpesa"
	"gess-backend/utils"

	"github.com/google/uuid"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var cookieSettings cookieopts.Settings

func methodNotAllowed(w http.ResponseWriter) {
	utils.RespondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	var user models.User
	query := `SELECT id, username, email, password_hash, role, created_at, updated_at 
	          FROM users WHERE username = $1 OR email = $1`
	err := database.DB.QueryRow(query, creds.Username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "Invalid username or password")
		return
	}

	if !user.CheckPassword(creds.Password) {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "Invalid username or password")
		return
	}

	tokenString, expirationTime, err := jwtutil.SignToken(user, 24*time.Hour)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "internal_error", "Could not issue token")
		return
	}

	c := cookieopts.SessionCookie(cookieSettings, "token", tokenString, expirationTime)
	http.SetCookie(w, &c)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
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
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
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
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Welcome to admin dashboard, %s!", user.Username),
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	c := cookieopts.SessionCookie(cookieSettings, "token", "", time.Now().Add(-time.Hour))
	http.SetCookie(w, &c)
	utils.RespondJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "Username and password are required")
		return
	}

	if err := utils.ValidateEmail(req.Email); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	hashedPassword, err := models.HashPassword(req.Password)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "internal_error", "Could not process password")
		return
	}

	role := models.UserRoleUser
	var userID uuid.UUID
	query := `INSERT INTO users (username, email, password_hash, role) 
	          VALUES ($1, $2, $3, $4) RETURNING id`
	err = database.DB.QueryRow(query, req.Username, req.Email, hashedPassword, role).Scan(&userID)
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

func main() {
	logger.Init()

	ctx := context.Background()
	traceShutdown := tracing.Init(ctx)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := traceShutdown(shutdownCtx); err != nil {
			slog.Error("tracing shutdown error", "err", err)
		}
	}()

	if err := database.ConnectDB(); err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer database.Close()

	mpesa.Init()

	secret, err := secrets.LoadJWTSecret()
	if err != nil {
		slog.Error("JWT configuration", "err", err)
		os.Exit(1)
	}
	jwtutil.Init(secret)

	cookieSettings, err = cookieopts.Load()
	if err != nil {
		slog.Error("cookie configuration", "err", err)
		os.Exit(1)
	}

	metrics.RegisterIntegrationGauges()
	metrics.StartDBPingLoop(ctx, 20*time.Second)

	mux := http.NewServeMux()

	// Prometheus metrics; restrict exposure at the network layer in production.
	mux.Handle("GET /metrics", metrics.MetricsHandler())

	mux.HandleFunc("/health", handlers.HealthHandler)
	mux.HandleFunc("/ready", handlers.ReadyHandler)

	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/register", registerHandler)
	mux.HandleFunc("/logout", logoutHandler)

	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.ListProductsHandler(w, r)
		} else {
			methodNotAllowed(w)
		}
	})
	mux.HandleFunc("/products/batch", handlers.BatchProductsHandler)
	mux.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) >= 3 && pathParts[len(pathParts)-1] == "reviews" {
			switch r.Method {
			case http.MethodGet:
				handlers.ListProductReviewsHandler(w, r)
			case http.MethodPost:
				middleware.AuthMiddleware(handlers.CreateReviewHandler)(w, r)
			default:
				methodNotAllowed(w)
			}
		} else {
			handlers.GetProductHandler(w, r)
		}
	})
	mux.HandleFunc("/categories", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.ListCategoriesHandler(w, r)
		} else {
			methodNotAllowed(w)
		}
	})
	mux.HandleFunc("/categories/", handlers.GetCategoryHandler)
	mux.HandleFunc("/brands", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.ListBrandsHandler(w, r)
		} else {
			methodNotAllowed(w)
		}
	})
	mux.HandleFunc("/brands/", handlers.GetBrandHandler)

	mux.HandleFunc("/me", middleware.AuthMiddleware(meHandler))

	mux.HandleFunc("/cart", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetCartHandler(w, r)
		case http.MethodDelete:
			handlers.ClearCartHandler(w, r)
		default:
			methodNotAllowed(w)
		}
	}))
	mux.HandleFunc("/cart/items", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.AddCartItemHandler(w, r)
		} else {
			methodNotAllowed(w)
		}
	}))
	mux.HandleFunc("/cart/items/", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			handlers.UpdateCartItemHandler(w, r)
		case http.MethodDelete:
			handlers.DeleteCartItemHandler(w, r)
		default:
			methodNotAllowed(w)
		}
	}))

	mux.HandleFunc("/addresses", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.ListAddressesHandler(w, r)
		case http.MethodPost:
			handlers.CreateAddressHandler(w, r)
		default:
			methodNotAllowed(w)
		}
	}))
	mux.HandleFunc("/addresses/", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) >= 3 && pathParts[len(pathParts)-1] == "default" {
			if r.Method == http.MethodPost {
				handlers.SetDefaultAddressHandler(w, r)
			} else {
				methodNotAllowed(w)
			}
		} else {
			switch r.Method {
			case http.MethodPut:
				handlers.UpdateAddressHandler(w, r)
			case http.MethodDelete:
				handlers.DeleteAddressHandler(w, r)
			default:
				methodNotAllowed(w)
			}
		}
	}))

	mux.HandleFunc("/webhooks/mpesa/stk", handlers.MpesaSTKCallbackHandler)
	mux.HandleFunc("/checkout/guest", handlers.GuestCheckoutHandler)

	mux.HandleFunc("/orders", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.ListOrdersHandler(w, r)
		case http.MethodPost:
			handlers.CreateOrderHandler(w, r)
		default:
			methodNotAllowed(w)
		}
	}))
	mux.HandleFunc("/orders/checkout-mpesa", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.CheckoutMpesaHandler(w, r)
		} else {
			methodNotAllowed(w)
		}
	}))
	mux.HandleFunc("/orders/", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.GetOrderHandler(w, r)
		} else {
			methodNotAllowed(w)
		}
	}))

	mux.HandleFunc("/reviews/me", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.ListUserReviewsHandler(w, r)
		} else {
			methodNotAllowed(w)
		}
	}))
	mux.HandleFunc("/reviews/", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			handlers.UpdateReviewHandler(w, r)
		case http.MethodDelete:
			handlers.DeleteReviewHandler(w, r)
		default:
			methodNotAllowed(w)
		}
	}))

	mux.HandleFunc("/admin/dashboard", middleware.AdminMiddleware(adminDashboardHandler))

	mux.HandleFunc("/admin/users", middleware.AdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.CreateAdminUserHandler(w, r)
		} else {
			methodNotAllowed(w)
		}
	}))

	mux.HandleFunc("/admin/products", middleware.AdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.CreateProductHandler(w, r)
		case http.MethodGet:
			handlers.ListProductsHandler(w, r)
		default:
			methodNotAllowed(w)
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
			methodNotAllowed(w)
		}
	}))

	mux.HandleFunc("/admin/categories", middleware.AdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.CreateCategoryHandler(w, r)
		case http.MethodGet:
			handlers.ListCategoriesHandler(w, r)
		default:
			methodNotAllowed(w)
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
			methodNotAllowed(w)
		}
	}))

	mux.HandleFunc("/admin/brands", middleware.AdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.CreateBrandHandler(w, r)
		case http.MethodGet:
			handlers.ListBrandsHandler(w, r)
		default:
			methodNotAllowed(w)
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
			methodNotAllowed(w)
		}
	}))

	mux.HandleFunc("/admin/orders", middleware.AdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.ListAllOrdersHandler(w, r)
		} else {
			methodNotAllowed(w)
		}
	}))
	mux.HandleFunc("/admin/orders/", middleware.AdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) >= 4 && pathParts[len(pathParts)-1] == "status" {
			if r.Method == http.MethodPut {
				handlers.UpdateOrderStatusHandler(w, r)
			} else {
				methodNotAllowed(w)
			}
		} else {
			methodNotAllowed(w)
		}
	}))

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

	core := middleware.RequestIDMiddleware(middleware.AccessLogMiddleware(mux))
	traced := otelhttp.NewHandler(core, "gess-backend",
		otelhttp.WithFilter(func(r *http.Request) bool {
			return r.URL.Path != "/metrics"
		}),
	)

	handler := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Request-ID"},
	}).Handler(traced)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("server listening", "addr", ":"+port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
