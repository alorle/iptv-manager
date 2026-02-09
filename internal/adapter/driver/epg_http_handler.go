package driver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alorle/iptv-manager/internal/application"
)

// EPGHTTPHandler handles HTTP requests for EPG operations.
type EPGHTTPHandler struct {
	epgSyncService  *application.EPGSyncService
	subscriptionSvc *application.SubscriptionService
	channelService  *application.ChannelService
}

// NewEPGHTTPHandler creates a new HTTP handler for EPG operations.
func NewEPGHTTPHandler(
	epgSyncService *application.EPGSyncService,
	subscriptionSvc *application.SubscriptionService,
	channelService *application.ChannelService,
) *EPGHTTPHandler {
	return &EPGHTTPHandler{
		epgSyncService:  epgSyncService,
		subscriptionSvc: subscriptionSvc,
		channelService:  channelService,
	}
}

// epgChannelResponse represents an EPG channel in JSON format.
type epgChannelResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Logo     string `json:"logo"`
	Category string `json:"category"`
	Language string `json:"language"`
	EPGID    string `json:"epg_id"`
}

// mappingResponse represents a channel's EPG mapping in JSON format.
type mappingResponse struct {
	ChannelName string `json:"channel_name"`
	EPGID       string `json:"epg_id"`
	Source      string `json:"source"`
	LastSynced  string `json:"last_synced"`
}

// updateMappingRequest represents the JSON body for updating a manual mapping.
type updateMappingRequest struct {
	EPGID string `json:"epg_id"`
}

// ServeHTTP routes the request to the appropriate handler based on method and path.
func (h *EPGHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/epg")

	// POST /api/epg/import - trigger initial import
	if r.Method == http.MethodPost && path == "/import" {
		h.handleImport(w, r)
		return
	}

	// GET /api/epg/channels - list available EPG channels with filters
	if r.Method == http.MethodGet && path == "/channels" {
		h.handleListChannels(w, r)
		return
	}

	// GET /api/epg/mappings - list mapping status for all channels
	if r.Method == http.MethodGet && path == "/mappings" {
		h.handleListMappings(w, r)
		return
	}

	// PUT /api/epg/mappings/{channelName} - update manual mapping
	if r.Method == http.MethodPut && strings.HasPrefix(path, "/mappings/") {
		channelName := strings.TrimPrefix(path, "/mappings/")
		h.handleUpdateMapping(w, r, channelName)
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

// handleImport handles POST /api/epg/import
func (h *EPGHTTPHandler) handleImport(w http.ResponseWriter, r *http.Request) {
	err := h.epgSyncService.SyncChannels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to import EPG data")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "EPG import started successfully",
	})
}

// handleListChannels handles GET /api/epg/channels with optional filters
func (h *EPGHTTPHandler) handleListChannels(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters for filtering
	category := r.URL.Query().Get("category")
	search := r.URL.Query().Get("search")

	filter := application.ChannelFilter{
		Category:   category,
		SearchTerm: search,
	}

	channels, err := h.subscriptionSvc.ListAvailableEPGChannels(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch EPG channels")
		return
	}

	response := make([]epgChannelResponse, len(channels))
	for i, ch := range channels {
		response[i] = epgChannelResponse{
			ID:       ch.ID(),
			Name:     ch.Name(),
			Logo:     ch.Logo(),
			Category: ch.Category(),
			Language: ch.Language(),
			EPGID:    ch.EPGID(),
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// handleListMappings handles GET /api/epg/mappings
func (h *EPGHTTPHandler) handleListMappings(w http.ResponseWriter, r *http.Request) {
	channels, err := h.channelService.ListChannels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch channels")
		return
	}

	response := make([]mappingResponse, 0, len(channels))
	for _, ch := range channels {
		mapping := ch.EPGMapping()
		if mapping != nil {
			response = append(response, mappingResponse{
				ChannelName: ch.Name(),
				EPGID:       mapping.EPGID(),
				Source:      string(mapping.Source()),
				LastSynced:  mapping.LastSynced().Format("2006-01-02T15:04:05Z07:00"),
			})
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// handleUpdateMapping handles PUT /api/epg/mappings/{channelName}
func (h *EPGHTTPHandler) handleUpdateMapping(w http.ResponseWriter, r *http.Request, channelName string) {
	var req updateMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update the EPG mapping to the new manual mapping
	err := h.channelService.UpdateEPGMapping(r.Context(), channelName, req.EPGID)
	if err != nil {
		writeError(w, http.StatusNotFound, "channel not found")
		return
	}

	// Get the updated channel to return the mapping
	ch, err := h.channelService.GetChannel(r.Context(), channelName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	mapping := ch.EPGMapping()
	if mapping == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, mappingResponse{
		ChannelName: ch.Name(),
		EPGID:       mapping.EPGID(),
		Source:      string(mapping.Source()),
		LastSynced:  mapping.LastSynced().Format("2006-01-02T15:04:05Z07:00"),
	})
}
