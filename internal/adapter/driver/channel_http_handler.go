package driver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/alorle/iptv-manager/internal/application"
	"github.com/alorle/iptv-manager/internal/channel"
)

// ChannelHTTPHandler handles HTTP requests for channel management.
type ChannelHTTPHandler struct {
	service *application.ChannelService
}

// NewChannelHTTPHandler creates a new HTTP handler for channels.
func NewChannelHTTPHandler(service *application.ChannelService) *ChannelHTTPHandler {
	return &ChannelHTTPHandler{service: service}
}

// errorResponse represents a JSON error response.
type errorResponse struct {
	Error string `json:"error"`
}

// channelRequest represents the JSON body for creating a channel.
type channelRequest struct {
	Name string `json:"name"`
}

// epgMappingResponse represents an EPG mapping in JSON format.
type epgMappingResponse struct {
	EPGID      string `json:"epg_id"`
	Source     string `json:"source"`
	LastSynced string `json:"last_synced"`
}

// channelResponse represents a channel in JSON format.
type channelResponse struct {
	Name       string              `json:"name"`
	Status     string              `json:"status"`
	EPGMapping *epgMappingResponse `json:"epg_mapping,omitempty"`
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

// ServeHTTP routes the request to the appropriate handler based on method and path.
func (h *ChannelHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/channels")

	// POST /channels - create a new channel
	if r.Method == http.MethodPost && path == "" {
		h.handleCreate(w, r)
		return
	}

	// GET /channels - list all channels
	if r.Method == http.MethodGet && path == "" {
		h.handleList(w, r)
		return
	}

	// GET /channels/{name} - get a specific channel
	if r.Method == http.MethodGet && path != "" {
		name := strings.TrimPrefix(path, "/")
		h.handleGet(w, r, name)
		return
	}

	// DELETE /channels/{name} - delete a channel
	if r.Method == http.MethodDelete && path != "" {
		name := strings.TrimPrefix(path, "/")
		h.handleDelete(w, r, name)
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

// toChannelResponse converts a channel domain object to an API response.
func toChannelResponse(ch channel.Channel) channelResponse {
	resp := channelResponse{
		Name:   ch.Name(),
		Status: string(ch.Status()),
	}

	if mapping := ch.EPGMapping(); mapping != nil {
		resp.EPGMapping = &epgMappingResponse{
			EPGID:      mapping.EPGID(),
			Source:     string(mapping.Source()),
			LastSynced: mapping.LastSynced().Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return resp
}

// handleCreate handles POST /channels
func (h *ChannelHTTPHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req channelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ch, err := h.service.CreateChannel(r.Context(), req.Name)
	if err != nil {
		if errors.Is(err, channel.ErrEmptyName) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, channel.ErrChannelAlreadyExists) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, toChannelResponse(ch))
}

// handleList handles GET /channels
func (h *ChannelHTTPHandler) handleList(w http.ResponseWriter, r *http.Request) {
	channels, err := h.service.ListChannels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response := make([]channelResponse, len(channels))
	for i, ch := range channels {
		response[i] = toChannelResponse(ch)
	}

	writeJSON(w, http.StatusOK, response)
}

// handleGet handles GET /channels/{name}
func (h *ChannelHTTPHandler) handleGet(w http.ResponseWriter, r *http.Request, name string) {
	ch, err := h.service.GetChannel(r.Context(), name)
	if err != nil {
		if errors.Is(err, channel.ErrChannelNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, toChannelResponse(ch))
}

// handleDelete handles DELETE /channels/{name}
func (h *ChannelHTTPHandler) handleDelete(w http.ResponseWriter, r *http.Request, name string) {
	err := h.service.DeleteChannel(r.Context(), name)
	if err != nil {
		if errors.Is(err, channel.ErrChannelNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
