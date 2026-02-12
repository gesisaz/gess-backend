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

// CreateReviewRequest represents the request body for creating a review
type CreateReviewRequest struct {
	Rating  int    `json:"rating"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
}

// UpdateReviewRequest represents the request body for updating a review
type UpdateReviewRequest struct {
	Rating  *int    `json:"rating,omitempty"`
	Title   *string `json:"title,omitempty"`
	Comment *string `json:"comment,omitempty"`
}

// ReviewListResponse represents the response for listing reviews
type ReviewListResponse struct {
	Reviews    []models.ReviewWithUser `json:"reviews"`
	Pagination utils.PaginationMeta    `json:"pagination"`
}

// CreateReviewHandler handles POST /products/:id/reviews - Create review
func CreateReviewHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract product ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Product ID is required")
		return
	}
	productIDStr := pathParts[len(pathParts)-2] // Second to last (before "reviews")

	// Validate UUID
	productID, err := utils.ValidateUUID(productIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Validate rating
	if req.Rating < 1 || req.Rating > 5 {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "Rating must be between 1 and 5")
		return
	}

	// Verify product exists
	var productPrice float64
	var currentRatingAvg float64
	var currentReviewCount int
	productQuery := `SELECT price, rating_average, review_count FROM products WHERE id = $1`
	err = database.DB.QueryRow(productQuery, productID).Scan(&productPrice, &currentRatingAvg, &currentReviewCount)
	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Product not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to verify product")
		return
	}

	// Check if user already has a review for this product
	var existingReviewID uuid.UUID
	checkQuery := `SELECT id FROM reviews WHERE product_id = $1 AND user_id = $2`
	err = database.DB.QueryRow(checkQuery, productID, user.ID).Scan(&existingReviewID)
	if err == nil {
		utils.RespondError(w, http.StatusConflict, "duplicate_review", "You have already reviewed this product")
		return
	}
	if err != sql.ErrNoRows {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to check existing review")
		return
	}

	// Calculate new rating average
	newReviewCount := currentReviewCount + 1
	newRatingAvg := (currentRatingAvg*float64(currentReviewCount) + float64(req.Rating)) / float64(newReviewCount)

	// Start transaction
	tx, err := database.DB.Begin()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Create review
	var review models.Review
	reviewQuery := `
		INSERT INTO reviews (product_id, user_id, rating, title, comment)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, product_id, user_id, rating, title, comment, created_at, updated_at
	`
	err = tx.QueryRow(reviewQuery, productID, user.ID, req.Rating, req.Title, req.Comment).Scan(
		&review.ID, &review.ProductID, &review.UserID, &review.Rating,
		&review.Title, &review.Comment, &review.CreatedAt, &review.UpdatedAt,
	)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to create review")
		return
	}

	// Update product rating and review count
	updateProductQuery := `UPDATE products SET rating_average = $1, review_count = $2 WHERE id = $3`
	_, err = tx.Exec(updateProductQuery, newRatingAvg, newReviewCount, productID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update product rating")
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to commit transaction")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, review)
}

// ListProductReviewsHandler handles GET /products/:id/reviews - List product reviews
func ListProductReviewsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract product ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Product ID is required")
		return
	}
	productIDStr := pathParts[len(pathParts)-2] // Second to last (before "reviews")

	// Validate UUID
	productID, err := utils.ValidateUUID(productIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Verify product exists
	var productIDCheck uuid.UUID
	productQuery := `SELECT id FROM products WHERE id = $1`
	err = database.DB.QueryRow(productQuery, productID).Scan(&productIDCheck)
	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Product not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to verify product")
		return
	}

	// Parse pagination
	pagination := utils.ParsePagination(r)

	// Count total
	countQuery := `SELECT COUNT(*) FROM reviews WHERE product_id = $1`
	var total int
	err = database.DB.QueryRow(countQuery, productID).Scan(&total)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to count reviews")
		return
	}

	// Get reviews with user information
	query := `
		SELECT r.id, r.product_id, r.user_id, r.rating, r.title, r.comment, r.created_at, r.updated_at,
		       u.username
		FROM reviews r
		JOIN users u ON r.user_id = u.id
		WHERE r.product_id = $1
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := database.DB.Query(query, productID, pagination.Limit, pagination.Offset)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch reviews")
		return
	}
	defer rows.Close()

	reviews := []models.ReviewWithUser{}
	for rows.Next() {
		var review models.ReviewWithUser
		err := rows.Scan(
			&review.ID, &review.ProductID, &review.UserID, &review.Rating,
			&review.Title, &review.Comment, &review.CreatedAt, &review.UpdatedAt,
			&review.Username,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to scan reviews")
			return
		}
		reviews = append(reviews, review)
	}

	response := ReviewListResponse{
		Reviews:    reviews,
		Pagination: utils.CreatePaginationMeta(total, pagination.Limit, pagination.Offset),
	}

	utils.RespondJSON(w, http.StatusOK, response)
}

// UpdateReviewHandler handles PUT /reviews/:id - Update own review
func UpdateReviewHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Review ID is required")
		return
	}
	reviewIDStr := pathParts[len(pathParts)-1]

	// Validate UUID
	reviewID, err := utils.ValidateUUID(reviewIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req UpdateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Verify review belongs to user and get current review data
	var existingReview models.Review
	var productID uuid.UUID
	verifyQuery := `SELECT id, product_id, user_id, rating, title, comment, created_at, updated_at FROM reviews WHERE id = $1 AND user_id = $2`
	err = database.DB.QueryRow(verifyQuery, reviewID, user.ID).Scan(
		&existingReview.ID, &productID, &existingReview.UserID, &existingReview.Rating,
		&existingReview.Title, &existingReview.Comment, &existingReview.CreatedAt, &existingReview.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Review not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to verify review")
		return
	}

	// Validate rating if provided
	if req.Rating != nil {
		if *req.Rating < 1 || *req.Rating > 5 {
			utils.RespondError(w, http.StatusBadRequest, "validation_error", "Rating must be between 1 and 5")
			return
		}
	}

	// Get current product stats
	var currentRatingAvg float64
	var currentReviewCount int
	productQuery := `SELECT rating_average, review_count FROM products WHERE id = $1`
	err = database.DB.QueryRow(productQuery, productID).Scan(&currentRatingAvg, &currentReviewCount)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch product stats")
		return
	}

	// Calculate new rating if rating changed
	newRatingAvg := currentRatingAvg
	if req.Rating != nil && *req.Rating != existingReview.Rating {
		// Recalculate: remove old rating, add new rating
		totalRating := currentRatingAvg * float64(currentReviewCount)
		totalRating = totalRating - float64(existingReview.Rating) + float64(*req.Rating)
		newRatingAvg = totalRating / float64(currentReviewCount)
	}

	// Start transaction
	tx, err := database.DB.Begin()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Build update query
	updateFields := []string{}
	args := []interface{}{}
	argCount := 1

	if req.Rating != nil {
		updateFields = append(updateFields, fmt.Sprintf("rating = $%d", argCount))
		args = append(args, *req.Rating)
		argCount++
	}
	if req.Title != nil {
		updateFields = append(updateFields, fmt.Sprintf("title = $%d", argCount))
		args = append(args, *req.Title)
		argCount++
	}
	if req.Comment != nil {
		updateFields = append(updateFields, fmt.Sprintf("comment = $%d", argCount))
		args = append(args, *req.Comment)
		argCount++
	}

	if len(updateFields) == 0 {
		utils.RespondError(w, http.StatusBadRequest, "validation_error", "No fields to update")
		return
	}

	updateFields = append(updateFields, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, reviewID)

	updateQuery := `UPDATE reviews SET ` + strings.Join(updateFields, ", ") + fmt.Sprintf(" WHERE id = $%d", argCount) + ` RETURNING id, product_id, user_id, rating, title, comment, created_at, updated_at`

	var review models.Review
	err = tx.QueryRow(updateQuery, args...).Scan(
		&review.ID, &review.ProductID, &review.UserID, &review.Rating,
		&review.Title, &review.Comment, &review.CreatedAt, &review.UpdatedAt,
	)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update review")
		return
	}

	// Update product rating if rating changed
	if req.Rating != nil && *req.Rating != existingReview.Rating {
		updateProductQuery := `UPDATE products SET rating_average = $1 WHERE id = $2`
		_, err = tx.Exec(updateProductQuery, newRatingAvg, productID)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update product rating")
			return
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to commit transaction")
		return
	}

	utils.RespondJSON(w, http.StatusOK, review)
}

// DeleteReviewHandler handles DELETE /reviews/:id - Delete own review
func DeleteReviewHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		utils.RespondError(w, http.StatusBadRequest, "invalid_request", "Review ID is required")
		return
	}
	reviewIDStr := pathParts[len(pathParts)-1]

	// Validate UUID
	reviewID, err := utils.ValidateUUID(reviewIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	// Verify review belongs to user and get review data
	var productID uuid.UUID
	var rating int
	verifyQuery := `SELECT product_id, rating FROM reviews WHERE id = $1 AND user_id = $2`
	err = database.DB.QueryRow(verifyQuery, reviewID, user.ID).Scan(&productID, &rating)
	if err == sql.ErrNoRows {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Review not found")
		return
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to verify review")
		return
	}

	// Get current product stats
	var currentRatingAvg float64
	var currentReviewCount int
	productQuery := `SELECT rating_average, review_count FROM products WHERE id = $1`
	err = database.DB.QueryRow(productQuery, productID).Scan(&currentRatingAvg, &currentReviewCount)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch product stats")
		return
	}

	// Calculate new rating average
	newReviewCount := currentReviewCount - 1
	var newRatingAvg float64
	if newReviewCount > 0 {
		totalRating := currentRatingAvg * float64(currentReviewCount)
		newRatingAvg = (totalRating - float64(rating)) / float64(newReviewCount)
	} else {
		newRatingAvg = 0
	}

	// Start transaction
	tx, err := database.DB.Begin()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Delete review
	deleteQuery := `DELETE FROM reviews WHERE id = $1`
	result, err := tx.Exec(deleteQuery, reviewID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to delete review")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to delete review")
		return
	}

	if rowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "not_found", "Review not found")
		return
	}

	// Update product rating and review count
	updateProductQuery := `UPDATE products SET rating_average = $1, review_count = $2 WHERE id = $3`
	_, err = tx.Exec(updateProductQuery, newRatingAvg, newReviewCount, productID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to update product rating")
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to commit transaction")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, nil, "Review deleted successfully")
}

// ListUserReviewsHandler handles GET /reviews/me - Get current user's reviews
func ListUserReviewsHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Parse pagination
	pagination := utils.ParsePagination(r)

	// Count total
	countQuery := `SELECT COUNT(*) FROM reviews WHERE user_id = $1`
	var total int
	err := database.DB.QueryRow(countQuery, user.ID).Scan(&total)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to count reviews")
		return
	}

	// Get reviews
	query := `
		SELECT id, product_id, user_id, rating, title, comment, created_at, updated_at
		FROM reviews
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := database.DB.Query(query, user.ID, pagination.Limit, pagination.Offset)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "database_error", "Failed to fetch reviews")
		return
	}
	defer rows.Close()

	reviews := []models.Review{}
	for rows.Next() {
		var review models.Review
		err := rows.Scan(
			&review.ID, &review.ProductID, &review.UserID, &review.Rating,
			&review.Title, &review.Comment, &review.CreatedAt, &review.UpdatedAt,
		)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "scan_error", "Failed to scan reviews")
			return
		}
		reviews = append(reviews, review)
	}

	response := ReviewListResponse{
		Reviews:    []models.ReviewWithUser{}, // Empty since we don't need username for own reviews
		Pagination: utils.CreatePaginationMeta(total, pagination.Limit, pagination.Offset),
	}

	// Convert to ReviewWithUser for consistency (username will be empty)
	reviewList := []models.ReviewWithUser{}
	for _, r := range reviews {
		reviewList = append(reviewList, models.ReviewWithUser{
			Review:   r,
			Username: user.Username,
		})
	}
	response.Reviews = reviewList

	utils.RespondJSON(w, http.StatusOK, response)
}
