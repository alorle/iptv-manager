package driver

import (
	"net/http"

	"github.com/alorle/iptv-manager/internal/application"
)

// PlaylistHTTPHandler handles HTTP requests for playlist generation.
type PlaylistHTTPHandler struct {
	service *application.PlaylistService
}

// NewPlaylistHTTPHandler creates a new HTTP handler for playlists.
func NewPlaylistHTTPHandler(service *application.PlaylistService) *PlaylistHTTPHandler {
	return &PlaylistHTTPHandler{service: service}
}

// ServeHTTP handles GET /playlist.m3u
func (h *PlaylistHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only GET method is allowed
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Generate M3U playlist using the request's Host header
	m3u, err := h.service.GenerateM3U(r.Context(), r.Host)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Write M3U response with proper content type
	w.Header().Set("Content-Type", "audio/mpegurl")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(m3u))
}
