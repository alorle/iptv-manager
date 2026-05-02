package driver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/alorle/iptv-manager/internal/application"
	"github.com/alorle/iptv-manager/internal/streaming"
)

// StreamProxy defines the streaming operations needed by the handler.
type StreamProxy interface {
	StreamToClient(ctx context.Context, infoHash string, dst io.Writer) error
}

// AceStreamHTTPHandler handles HTTP requests for AceStream proxy.
type AceStreamHTTPHandler struct {
	proxyService StreamProxy
	logger       *slog.Logger
}

// NewAceStreamHTTPHandler creates a new HTTP handler for AceStream proxy.
func NewAceStreamHTTPHandler(proxyService StreamProxy, logger *slog.Logger) *AceStreamHTTPHandler {
	return &AceStreamHTTPHandler{
		proxyService: proxyService,
		logger:       logger,
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
		h.logger.Warn("validation error", "error", "missing infohash", "remote_addr", r.RemoteAddr)
		writeError(w, http.StatusBadRequest, "missing 'id' query parameter")
		return
	}

	userAgent := r.Header.Get("User-Agent")
	h.logger.Info("stream request received", "remote_addr", r.RemoteAddr, "infohash", infoHash, "user_agent", userAgent)

	startTime := time.Now()

	// Set appropriate headers for streaming
	w.Header().Set("Content-Type", "video/mpeg")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Stream to client
	err := h.proxyService.StreamToClient(r.Context(), infoHash, w)
	duration := time.Since(startTime)

	if err != nil {
		// Log error but don't write response as streaming may have started
		if errors.Is(err, application.ErrInvalidInfoHash) {
			h.logger.Warn("validation error", "error", "invalid infohash", "remote_addr", r.RemoteAddr, "infohash", infoHash)
			// Only write error if we haven't started streaming
			writeError(w, http.StatusBadRequest, err.Error())
			h.logger.Info("request completed", "remote_addr", r.RemoteAddr, "infohash", infoHash, "duration", duration, "reason", "validation_error")
			return
		}
		if errors.Is(err, application.ErrEngineUnavailable) {
			h.logger.Error("service error", "error", "engine unavailable", "remote_addr", r.RemoteAddr, "infohash", infoHash)
			writeError(w, http.StatusServiceUnavailable, "acestream engine unavailable")
			h.logger.Info("request completed", "remote_addr", r.RemoteAddr, "infohash", infoHash, "duration", duration, "reason", "engine_unavailable")
			return
		}
		if streaming.IsClientDisconnectError(err) {
			h.logger.Info("request completed", "remote_addr", r.RemoteAddr, "infohash", infoHash, "duration", duration, "reason", "client_disconnected")
			return
		}
		h.logger.Error("service error", "error", "stream failed", "remote_addr", r.RemoteAddr, "infohash", infoHash, "details", err)
		h.logger.Info("request completed", "remote_addr", r.RemoteAddr, "infohash", infoHash, "duration", duration, "reason", "stream_error")
		return
	}

	h.logger.Info("request completed", "remote_addr", r.RemoteAddr, "infohash", infoHash, "duration", duration, "reason", "success")
}
