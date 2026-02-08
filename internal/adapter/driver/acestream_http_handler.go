package driver

import (
	"errors"
	"net/http"

	"github.com/alorle/iptv-manager/internal/application"
)

// AceStreamHTTPHandler handles HTTP requests for AceStream proxy.
type AceStreamHTTPHandler struct {
	proxyService *application.AceStreamProxyService
}

// NewAceStreamHTTPHandler creates a new HTTP handler for AceStream proxy.
func NewAceStreamHTTPHandler(proxyService *application.AceStreamProxyService) *AceStreamHTTPHandler {
	return &AceStreamHTTPHandler{
		proxyService: proxyService,
	}
}

// ServeHTTP handles GET /ace/getstream?id={infoHash}
func (h *AceStreamHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract infohash from query parameter
	infoHash := r.URL.Query().Get("id")
	if infoHash == "" {
		writeError(w, http.StatusBadRequest, "missing 'id' query parameter")
		return
	}

	// Set appropriate headers for streaming
	w.Header().Set("Content-Type", "video/mpeg")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Stream to client
	err := h.proxyService.StreamToClient(r.Context(), infoHash, w)
	if err != nil {
		// Log error but don't write response as streaming may have started
		if errors.Is(err, application.ErrInvalidInfoHash) {
			// Only write error if we haven't started streaming
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, application.ErrEngineUnavailable) {
			writeError(w, http.StatusServiceUnavailable, "acestream engine unavailable")
			return
		}
		// For other errors during streaming, the connection will be closed
		// which is appropriate for streaming failures
		return
	}
}

// activeStreamsResponse represents active stream information.
type activeStreamsResponse struct {
	Streams []streamInfo `json:"streams"`
}

type streamInfo struct {
	InfoHash    string   `json:"info_hash"`
	ClientCount int      `json:"client_count"`
	PIDs        []string `json:"pids"`
}

// HandleActiveStreams handles GET /ace/streams - returns active stream information.
func (h *AceStreamHTTPHandler) HandleActiveStreams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	activeStreams := h.proxyService.GetActiveStreams()

	response := activeStreamsResponse{
		Streams: make([]streamInfo, len(activeStreams)),
	}

	for i, info := range activeStreams {
		response.Streams[i] = streamInfo{
			InfoHash:    info.InfoHash,
			ClientCount: info.ClientCount,
			PIDs:        info.PIDs,
		}
	}

	writeJSON(w, http.StatusOK, response)
}
