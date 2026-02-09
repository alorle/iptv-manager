package driver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/alorle/iptv-manager/internal/application"
	"github.com/alorle/iptv-manager/internal/subscription"
)

// SubscriptionHTTPHandler handles HTTP requests for subscription management.
type SubscriptionHTTPHandler struct {
	service *application.SubscriptionService
}

// NewSubscriptionHTTPHandler creates a new HTTP handler for subscriptions.
func NewSubscriptionHTTPHandler(service *application.SubscriptionService) *SubscriptionHTTPHandler {
	return &SubscriptionHTTPHandler{service: service}
}

// subscribeRequest represents the JSON body for subscribing to a channel.
type subscribeRequest struct {
	EPGChannelID string `json:"epg_channel_id"`
}

// subscriptionResponse represents a subscription in JSON format.
type subscriptionResponse struct {
	EPGChannelID   string `json:"epg_channel_id"`
	Enabled        bool   `json:"enabled"`
	ManualOverride bool   `json:"manual_override"`
}

// ServeHTTP routes the request to the appropriate handler based on method and path.
func (h *SubscriptionHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/subscriptions")

	// POST /api/subscriptions - subscribe to EPG channel
	if r.Method == http.MethodPost && path == "" {
		h.handleSubscribe(w, r)
		return
	}

	// GET /api/subscriptions - list user's subscriptions
	if r.Method == http.MethodGet && path == "" {
		h.handleList(w, r)
		return
	}

	// DELETE /api/subscriptions/{id} - unsubscribe
	if r.Method == http.MethodDelete && path != "" {
		id := strings.TrimPrefix(path, "/")
		h.handleUnsubscribe(w, r, id)
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

// handleSubscribe handles POST /api/subscriptions
func (h *SubscriptionHTTPHandler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	var req subscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.service.Subscribe(r.Context(), req.EPGChannelID)
	if err != nil {
		if errors.Is(err, subscription.ErrEmptyEPGChannelID) {
			writeError(w, http.StatusBadRequest, subscription.ErrEmptyEPGChannelID.Error())
			return
		}
		if errors.Is(err, subscription.ErrSubscriptionAlreadyExists) {
			writeError(w, http.StatusConflict, subscription.ErrSubscriptionAlreadyExists.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Get the created subscription to return it
	sub, err := h.service.GetSubscription(r.Context(), req.EPGChannelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, subscriptionResponse{
		EPGChannelID:   sub.EPGChannelID(),
		Enabled:        sub.IsEnabled(),
		ManualOverride: sub.HasManualOverride(),
	})
}

// handleList handles GET /api/subscriptions
func (h *SubscriptionHTTPHandler) handleList(w http.ResponseWriter, r *http.Request) {
	subscriptions, err := h.service.ListSubscriptions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response := make([]subscriptionResponse, len(subscriptions))
	for i, sub := range subscriptions {
		response[i] = subscriptionResponse{
			EPGChannelID:   sub.EPGChannelID(),
			Enabled:        sub.IsEnabled(),
			ManualOverride: sub.HasManualOverride(),
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// handleUnsubscribe handles DELETE /api/subscriptions/{id}
func (h *SubscriptionHTTPHandler) handleUnsubscribe(w http.ResponseWriter, r *http.Request, id string) {
	err := h.service.Unsubscribe(r.Context(), id)
	if err != nil {
		if errors.Is(err, subscription.ErrSubscriptionNotFound) {
			writeError(w, http.StatusNotFound, subscription.ErrSubscriptionNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
