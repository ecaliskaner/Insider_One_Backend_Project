package handlers

import (
	"net/http"

	"github.com/ecaliskaner/Insider_One_Backend_Project/database"
)

// HealthHandler serves platform health probes.
type HealthHandler struct {
	db *database.DB
}

func NewHealthHandler(db *database.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Healthz godoc
// @Summary      Health check
// @Description  Returns process liveness status.
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /health [get]
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	}, nil)
}

func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	if h.db == nil || h.db.Conn == nil {
		WriteProblem(w, r, http.StatusServiceUnavailable, "Service Not Ready", "Database connection is not configured.", "https://api.insiderfootball.com/errors/not-ready")
		return
	}

	if err := h.db.Conn.PingContext(r.Context()); err != nil {
		WriteProblem(w, r, http.StatusServiceUnavailable, "Service Not Ready", "Database connection check failed.", "https://api.insiderfootball.com/errors/not-ready")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	}, nil)
}
