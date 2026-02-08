package driver

import (
	"net/http"

	"github.com/alorle/iptv-manager/internal/application"
)

// HealthHTTPHandler handles HTTP requests for health checks.
type HealthHTTPHandler struct {
	service *application.HealthService
}

// NewHealthHTTPHandler creates a new HTTP handler for health checks.
func NewHealthHTTPHandler(service *application.HealthService) *HealthHTTPHandler {
	return &HealthHTTPHandler{service: service}
}

// healthResponse represents the JSON response for health check endpoint.
type healthResponse struct {
	Status          string `json:"status"`
	DB              string `json:"db"`
	AceStreamEngine string `json:"acestream_engine"`
}

// ServeHTTP handles GET /health
func (h *HealthHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only GET method is allowed
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Perform health check
	status := h.service.Check(r.Context())

	// Build response
	resp := healthResponse{
		Status:          status.Status,
		DB:              status.DB.Status,
		AceStreamEngine: status.AceStreamEngine.Status,
	}

	// Determine HTTP status code
	httpStatus := http.StatusOK
	if status.Status != "ok" {
		httpStatus = http.StatusServiceUnavailable
	}

	// Write JSON response
	writeJSON(w, httpStatus, resp)
}
