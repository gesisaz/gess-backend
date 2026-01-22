package utils

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// SuccessResponse represents a success response with data
type SuccessResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// RespondJSON sends a JSON response
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// RespondError sends an error response
func RespondError(w http.ResponseWriter, status int, errorType string, message string) {
	RespondJSON(w, status, ErrorResponse{
		Error:   errorType,
		Message: message,
	})
}

// RespondSuccess sends a success response
func RespondSuccess(w http.ResponseWriter, status int, data interface{}, message string) {
	response := SuccessResponse{
		Data:    data,
		Message: message,
	}
	RespondJSON(w, status, response)
}
