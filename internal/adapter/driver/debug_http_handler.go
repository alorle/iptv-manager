package driver

import (
	"net/http"

	"github.com/alorle/iptv-manager/internal/application"
)

// DebugHTTPHandler exposes diagnostic information about the streaming subsystem.
type DebugHTTPHandler struct {
	proxyService *application.AceStreamProxyService
}

// NewDebugHTTPHandler creates a new debug handler.
func NewDebugHTTPHandler(proxyService *application.AceStreamProxyService) *DebugHTTPHandler {
	return &DebugHTTPHandler{proxyService: proxyService}
}

// ServeHTTP handles GET /debug/streams.
func (h *DebugHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	diag := h.proxyService.Diagnostics(r.Context())
	writeJSON(w, http.StatusOK, diag)
}
