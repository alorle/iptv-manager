package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alorle/iptv-manager/internal/application"
	"github.com/alorle/iptv-manager/internal/epg"
	"github.com/alorle/iptv-manager/internal/subscription"
)

// mockSubscriptionRepository is a mock implementation for testing.
type mockSubscriptionRepository struct {
	saveFunc        func(ctx context.Context, sub subscription.Subscription) error
	findByEPGIDFunc func(ctx context.Context, epgChannelID string) (subscription.Subscription, error)
	findAllFunc     func(ctx context.Context) ([]subscription.Subscription, error)
	deleteFunc      func(ctx context.Context, epgChannelID string) error
}

func (m *mockSubscriptionRepository) Save(ctx context.Context, sub subscription.Subscription) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, sub)
	}
	return nil
}

func (m *mockSubscriptionRepository) FindByEPGID(ctx context.Context, epgChannelID string) (subscription.Subscription, error) {
	if m.findByEPGIDFunc != nil {
		return m.findByEPGIDFunc(ctx, epgChannelID)
	}
	return subscription.Subscription{}, subscription.ErrSubscriptionNotFound
}

func (m *mockSubscriptionRepository) FindAll(ctx context.Context) ([]subscription.Subscription, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(ctx)
	}
	return []subscription.Subscription{}, nil
}

func (m *mockSubscriptionRepository) Delete(ctx context.Context, epgChannelID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, epgChannelID)
	}
	return nil
}

// mockEPGFetcher is a mock implementation for testing.
type mockEPGFetcher struct {
	fetchEPGFunc func(ctx context.Context) ([]epg.Channel, error)
}

func (m *mockEPGFetcher) FetchEPG(ctx context.Context) ([]epg.Channel, error) {
	if m.fetchEPGFunc != nil {
		return m.fetchEPGFunc(ctx)
	}
	return []epg.Channel{}, nil
}

func TestSubscriptionHTTPHandler_Subscribe(t *testing.T) {
	t.Run("POST /api/subscriptions creates subscription successfully", func(t *testing.T) {
		var savedSub subscription.Subscription
		subRepo := &mockSubscriptionRepository{
			saveFunc: func(ctx context.Context, sub subscription.Subscription) error {
				savedSub = sub
				return nil
			},
			findByEPGIDFunc: func(ctx context.Context, epgChannelID string) (subscription.Subscription, error) {
				if epgChannelID == savedSub.EPGChannelID() {
					return savedSub, nil
				}
				return subscription.Subscription{}, subscription.ErrSubscriptionNotFound
			},
		}
		epgFetcher := &mockEPGFetcher{}
		service := application.NewSubscriptionService(subRepo, epgFetcher)
		handler := NewSubscriptionHTTPHandler(service)

		reqBody := bytes.NewBufferString(`{"epg_channel_id":"epg123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", reqBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", rec.Code)
		}

		var resp subscriptionResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.EPGChannelID != "epg123" {
			t.Errorf("expected epg_channel_id 'epg123', got %q", resp.EPGChannelID)
		}
		if !resp.Enabled {
			t.Errorf("expected subscription to be enabled")
		}
		if resp.ManualOverride {
			t.Errorf("expected manual_override to be false")
		}
	})

	t.Run("POST /api/subscriptions returns 400 for invalid JSON", func(t *testing.T) {
		subRepo := &mockSubscriptionRepository{}
		epgFetcher := &mockEPGFetcher{}
		service := application.NewSubscriptionService(subRepo, epgFetcher)
		handler := NewSubscriptionHTTPHandler(service)

		reqBody := bytes.NewBufferString(`invalid json`)
		req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", reqBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != "invalid request body" {
			t.Errorf("expected error 'invalid request body', got %q", resp.Error)
		}
	})

	t.Run("POST /api/subscriptions returns 400 for empty epg_channel_id", func(t *testing.T) {
		subRepo := &mockSubscriptionRepository{}
		epgFetcher := &mockEPGFetcher{}
		service := application.NewSubscriptionService(subRepo, epgFetcher)
		handler := NewSubscriptionHTTPHandler(service)

		reqBody := bytes.NewBufferString(`{"epg_channel_id":""}`)
		req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", reqBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != subscription.ErrEmptyEPGChannelID.Error() {
			t.Errorf("expected error %q, got %q", subscription.ErrEmptyEPGChannelID.Error(), resp.Error)
		}
	})

	t.Run("POST /api/subscriptions returns 409 for duplicate subscription", func(t *testing.T) {
		subRepo := &mockSubscriptionRepository{
			saveFunc: func(ctx context.Context, sub subscription.Subscription) error {
				return subscription.ErrSubscriptionAlreadyExists
			},
		}
		epgFetcher := &mockEPGFetcher{}
		service := application.NewSubscriptionService(subRepo, epgFetcher)
		handler := NewSubscriptionHTTPHandler(service)

		reqBody := bytes.NewBufferString(`{"epg_channel_id":"epg123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", reqBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("expected status 409, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != subscription.ErrSubscriptionAlreadyExists.Error() {
			t.Errorf("expected error %q, got %q", subscription.ErrSubscriptionAlreadyExists.Error(), resp.Error)
		}
	})
}

func TestSubscriptionHTTPHandler_List(t *testing.T) {
	t.Run("GET /api/subscriptions returns all subscriptions", func(t *testing.T) {
		sub1, _ := subscription.NewSubscription("epg1")
		sub2, _ := subscription.NewSubscription("epg2")
		subRepo := &mockSubscriptionRepository{
			findAllFunc: func(ctx context.Context) ([]subscription.Subscription, error) {
				return []subscription.Subscription{sub1, sub2}, nil
			},
		}
		epgFetcher := &mockEPGFetcher{}
		service := application.NewSubscriptionService(subRepo, epgFetcher)
		handler := NewSubscriptionHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/api/subscriptions", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp []subscriptionResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 2 {
			t.Fatalf("expected 2 subscriptions, got %d", len(resp))
		}
		if resp[0].EPGChannelID != "epg1" || resp[1].EPGChannelID != "epg2" {
			t.Errorf("unexpected epg_channel_ids: %q, %q", resp[0].EPGChannelID, resp[1].EPGChannelID)
		}
	})

	t.Run("GET /api/subscriptions returns empty array when no subscriptions exist", func(t *testing.T) {
		subRepo := &mockSubscriptionRepository{
			findAllFunc: func(ctx context.Context) ([]subscription.Subscription, error) {
				return []subscription.Subscription{}, nil
			},
		}
		epgFetcher := &mockEPGFetcher{}
		service := application.NewSubscriptionService(subRepo, epgFetcher)
		handler := NewSubscriptionHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/api/subscriptions", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp []subscriptionResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 0 {
			t.Errorf("expected empty array, got %d subscriptions", len(resp))
		}
	})
}

func TestSubscriptionHTTPHandler_Unsubscribe(t *testing.T) {
	t.Run("DELETE /api/subscriptions/{id} deletes subscription successfully", func(t *testing.T) {
		subRepo := &mockSubscriptionRepository{
			deleteFunc: func(ctx context.Context, epgChannelID string) error {
				if epgChannelID == "epg123" {
					return nil
				}
				return subscription.ErrSubscriptionNotFound
			},
		}
		epgFetcher := &mockEPGFetcher{}
		service := application.NewSubscriptionService(subRepo, epgFetcher)
		handler := NewSubscriptionHTTPHandler(service)

		req := httptest.NewRequest(http.MethodDelete, "/api/subscriptions/epg123", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", rec.Code)
		}
	})

	t.Run("DELETE /api/subscriptions/{id} returns 404 for non-existent subscription", func(t *testing.T) {
		subRepo := &mockSubscriptionRepository{
			deleteFunc: func(ctx context.Context, epgChannelID string) error {
				return subscription.ErrSubscriptionNotFound
			},
		}
		epgFetcher := &mockEPGFetcher{}
		service := application.NewSubscriptionService(subRepo, epgFetcher)
		handler := NewSubscriptionHTTPHandler(service)

		req := httptest.NewRequest(http.MethodDelete, "/api/subscriptions/nonexistent", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != subscription.ErrSubscriptionNotFound.Error() {
			t.Errorf("expected error %q, got %q", subscription.ErrSubscriptionNotFound.Error(), resp.Error)
		}
	})
}

func TestSubscriptionHTTPHandler_MethodNotAllowed(t *testing.T) {
	t.Run("returns 405 for unsupported methods", func(t *testing.T) {
		subRepo := &mockSubscriptionRepository{}
		epgFetcher := &mockEPGFetcher{}
		service := application.NewSubscriptionService(subRepo, epgFetcher)
		handler := NewSubscriptionHTTPHandler(service)

		methods := []string{http.MethodPut, http.MethodPatch, http.MethodHead, http.MethodOptions}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/api/subscriptions", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("method %s: expected status 405, got %d", method, rec.Code)
			}
		}
	})
}
