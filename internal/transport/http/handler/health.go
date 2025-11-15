package handler

import (
	"net/http"
)

// HealthHandler обрабатывает health check
type HealthHandler struct{}

// NewHealthHandler создает новый handler для health check
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Check обрабатывает GET /health
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
