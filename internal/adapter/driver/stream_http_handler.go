package driver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/alorle/iptv-manager/internal/application"
	"github.com/alorle/iptv-manager/internal/channel"
	"github.com/alorle/iptv-manager/internal/stream"
)

// StreamHTTPHandler handles HTTP requests for stream management.
type StreamHTTPHandler struct {
	service *application.StreamService
}

// NewStreamHTTPHandler creates a new HTTP handler for streams.
func NewStreamHTTPHandler(service *application.StreamService) *StreamHTTPHandler {
	return &StreamHTTPHandler{service: service}
}

// streamRequest represents the JSON body for creating a stream.
type streamRequest struct {
	InfoHash    string `json:"info_hash"`
	ChannelName string `json:"channel_name"`
}

// streamResponse represents a stream in JSON format.
type streamResponse struct {
	InfoHash    string `json:"info_hash"`
	ChannelName string `json:"channel_name"`
}

// ServeHTTP routes the request to the appropriate handler based on method and path.
func (h *StreamHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/streams")

	// POST /streams - create a new stream
	if r.Method == http.MethodPost && path == "" {
		h.handleCreate(w, r)
		return
	}

	// GET /streams - list all streams
	if r.Method == http.MethodGet && path == "" {
		h.handleList(w, r)
		return
	}

	// GET /streams/{infoHash} - get a specific stream
	if r.Method == http.MethodGet && path != "" {
		infoHash := strings.TrimPrefix(path, "/")
		h.handleGet(w, r, infoHash)
		return
	}

	// DELETE /streams/{infoHash} - delete a stream
	if r.Method == http.MethodDelete && path != "" {
		infoHash := strings.TrimPrefix(path, "/")
		h.handleDelete(w, r, infoHash)
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

// handleCreate handles POST /streams
func (h *StreamHTTPHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req streamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	st, err := h.service.CreateStream(r.Context(), req.InfoHash, req.ChannelName)
	if err != nil {
		if errors.Is(err, stream.ErrEmptyInfoHash) || errors.Is(err, stream.ErrEmptyChannelName) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, channel.ErrChannelNotFound) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, stream.ErrStreamAlreadyExists) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, streamResponse{
		InfoHash:    st.InfoHash(),
		ChannelName: st.ChannelName(),
	})
}

// handleList handles GET /streams
func (h *StreamHTTPHandler) handleList(w http.ResponseWriter, r *http.Request) {
	streams, err := h.service.ListStreams(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response := make([]streamResponse, len(streams))
	for i, st := range streams {
		response[i] = streamResponse{
			InfoHash:    st.InfoHash(),
			ChannelName: st.ChannelName(),
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// handleGet handles GET /streams/{infoHash}
func (h *StreamHTTPHandler) handleGet(w http.ResponseWriter, r *http.Request, infoHash string) {
	st, err := h.service.GetStream(r.Context(), infoHash)
	if err != nil {
		if errors.Is(err, stream.ErrStreamNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, streamResponse{
		InfoHash:    st.InfoHash(),
		ChannelName: st.ChannelName(),
	})
}

// handleDelete handles DELETE /streams/{infoHash}
func (h *StreamHTTPHandler) handleDelete(w http.ResponseWriter, r *http.Request, infoHash string) {
	err := h.service.DeleteStream(r.Context(), infoHash)
	if err != nil {
		if errors.Is(err, stream.ErrStreamNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
