package handlers

import (
	"context"
	"net/http"
	"time"

	"gess-backend/database"
	"gess-backend/utils"
)

const dbPingTimeout = 2 * time.Second

// HealthHandler verifies the process and database (for load balancers that only probe /health).
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	healthReady(w, r, "healthy")
}

// ReadyHandler verifies database connectivity (for Kubernetes-style readiness probes).
func ReadyHandler(w http.ResponseWriter, r *http.Request) {
	healthReady(w, r, "ready")
}

func healthReady(w http.ResponseWriter, r *http.Request, status string) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), dbPingTimeout)
	defer cancel()
	if err := database.Ping(ctx); err != nil {
		utils.RespondError(w, http.StatusServiceUnavailable, "not_ready", "database unavailable")
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]string{"status": status})
}
