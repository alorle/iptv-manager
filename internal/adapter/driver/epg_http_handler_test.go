package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/internal/application"
	"github.com/alorle/iptv-manager/internal/channel"
	"github.com/alorle/iptv-manager/internal/epg"
	"github.com/alorle/iptv-manager/internal/subscription"
)

// mockAcestreamSource is a mock implementation for testing.
type mockAcestreamSource struct {
	fetchHashesFunc func(ctx context.Context, source string) (map[string][]string, error)
}

func (m *mockAcestreamSource) FetchHashes(ctx context.Context, source string) (map[string][]string, error) {
	if m.fetchHashesFunc != nil {
		return m.fetchHashesFunc(ctx, source)
	}
	return map[string][]string{}, nil
}

func TestEPGHTTPHandler_Import(t *testing.T) {
	t.Run("POST /epg/import triggers EPG sync successfully", func(t *testing.T) {
		syncCalled := false
		epgFetcher := &mockEPGFetcher{
			fetchEPGFunc: func(ctx context.Context) ([]epg.Channel, error) {
				syncCalled = true
				return []epg.Channel{}, nil
			},
		}
		acestreamSrc := &mockAcestreamSource{
			fetchHashesFunc: func(ctx context.Context, source string) (map[string][]string, error) {
				return map[string][]string{}, nil
			},
		}
		channelRepo := &mockChannelRepository{
			findAllFunc: func(ctx context.Context) ([]channel.Channel, error) {
				return []channel.Channel{}, nil
			},
		}
		streamRepo := &mockStreamRepository{}
		subRepo := &mockSubscriptionRepository{
			findAllFunc: func(ctx context.Context) ([]subscription.Subscription, error) {
				return []subscription.Subscription{}, nil
			},
		}

		epgSyncService := application.NewEPGSyncService(epgFetcher, acestreamSrc, channelRepo, streamRepo, subRepo)
		subscriptionSvc := application.NewSubscriptionService(subRepo, epgFetcher)
		channelService := application.NewChannelService(channelRepo, streamRepo)
		handler := NewEPGHTTPHandler(epgSyncService, subscriptionSvc, channelService)

		req := httptest.NewRequest(http.MethodPost, "/epg/import", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		if !syncCalled {
			t.Errorf("expected EPG sync to be called")
		}

		var resp map[string]string
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["message"] != "EPG import started successfully" {
			t.Errorf("expected success message, got %q", resp["message"])
		}
	})

	t.Run("POST /epg/import returns 500 on sync error", func(t *testing.T) {
		epgFetcher := &mockEPGFetcher{
			fetchEPGFunc: func(ctx context.Context) ([]epg.Channel, error) {
				return nil, errors.New("fetch failed")
			},
		}
		acestreamSrc := &mockAcestreamSource{}
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{}
		subRepo := &mockSubscriptionRepository{}

		epgSyncService := application.NewEPGSyncService(epgFetcher, acestreamSrc, channelRepo, streamRepo, subRepo)
		subscriptionSvc := application.NewSubscriptionService(subRepo, epgFetcher)
		channelService := application.NewChannelService(channelRepo, streamRepo)
		handler := NewEPGHTTPHandler(epgSyncService, subscriptionSvc, channelService)

		req := httptest.NewRequest(http.MethodPost, "/epg/import", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != "failed to import EPG data" {
			t.Errorf("expected error 'failed to import EPG data', got %q", resp.Error)
		}
	})
}

func TestEPGHTTPHandler_ListChannels(t *testing.T) {
	t.Run("GET /epg/channels returns all channels", func(t *testing.T) {
		ch1, _ := epg.NewChannel("1", "Channel One", "logo1.png", "Sports", "en", "epg1")
		ch2, _ := epg.NewChannel("2", "Channel Two", "logo2.png", "News", "en", "epg2")

		epgFetcher := &mockEPGFetcher{
			fetchEPGFunc: func(ctx context.Context) ([]epg.Channel, error) {
				return []epg.Channel{ch1, ch2}, nil
			},
		}
		subRepo := &mockSubscriptionRepository{}
		acestreamSrc := &mockAcestreamSource{}
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{}

		subscriptionSvc := application.NewSubscriptionService(subRepo, epgFetcher)
		epgSyncService := application.NewEPGSyncService(epgFetcher, acestreamSrc, channelRepo, streamRepo, subRepo)
		channelService := application.NewChannelService(channelRepo, streamRepo)
		handler := NewEPGHTTPHandler(epgSyncService, subscriptionSvc, channelService)

		req := httptest.NewRequest(http.MethodGet, "/epg/channels", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp []epgChannelResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 2 {
			t.Fatalf("expected 2 channels, got %d", len(resp))
		}
		if resp[0].Name != "Channel One" || resp[1].Name != "Channel Two" {
			t.Errorf("unexpected channel names: %q, %q", resp[0].Name, resp[1].Name)
		}
	})

	t.Run("GET /epg/channels filters by category", func(t *testing.T) {
		ch1, _ := epg.NewChannel("1", "Sports Channel", "logo1.png", "Sports", "en", "epg1")
		ch2, _ := epg.NewChannel("2", "News Channel", "logo2.png", "News", "en", "epg2")

		epgFetcher := &mockEPGFetcher{
			fetchEPGFunc: func(ctx context.Context) ([]epg.Channel, error) {
				return []epg.Channel{ch1, ch2}, nil
			},
		}
		subRepo := &mockSubscriptionRepository{}
		acestreamSrc := &mockAcestreamSource{}
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{}

		subscriptionSvc := application.NewSubscriptionService(subRepo, epgFetcher)
		epgSyncService := application.NewEPGSyncService(epgFetcher, acestreamSrc, channelRepo, streamRepo, subRepo)
		channelService := application.NewChannelService(channelRepo, streamRepo)
		handler := NewEPGHTTPHandler(epgSyncService, subscriptionSvc, channelService)

		req := httptest.NewRequest(http.MethodGet, "/epg/channels?category=sports", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp []epgChannelResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 1 {
			t.Fatalf("expected 1 channel, got %d", len(resp))
		}
		if resp[0].Category != "Sports" {
			t.Errorf("expected category 'Sports', got %q", resp[0].Category)
		}
	})

	t.Run("GET /epg/channels filters by search term", func(t *testing.T) {
		ch1, _ := epg.NewChannel("1", "ESPN Sports", "logo1.png", "Sports", "en", "epg1")
		ch2, _ := epg.NewChannel("2", "BBC News", "logo2.png", "News", "en", "epg2")

		epgFetcher := &mockEPGFetcher{
			fetchEPGFunc: func(ctx context.Context) ([]epg.Channel, error) {
				return []epg.Channel{ch1, ch2}, nil
			},
		}
		subRepo := &mockSubscriptionRepository{}
		acestreamSrc := &mockAcestreamSource{}
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{}

		subscriptionSvc := application.NewSubscriptionService(subRepo, epgFetcher)
		epgSyncService := application.NewEPGSyncService(epgFetcher, acestreamSrc, channelRepo, streamRepo, subRepo)
		channelService := application.NewChannelService(channelRepo, streamRepo)
		handler := NewEPGHTTPHandler(epgSyncService, subscriptionSvc, channelService)

		req := httptest.NewRequest(http.MethodGet, "/epg/channels?search=bbc", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp []epgChannelResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 1 {
			t.Fatalf("expected 1 channel, got %d", len(resp))
		}
		if resp[0].Name != "BBC News" {
			t.Errorf("expected name 'BBC News', got %q", resp[0].Name)
		}
	})
}

func TestEPGHTTPHandler_ListMappings(t *testing.T) {
	t.Run("GET /epg/mappings returns all mappings", func(t *testing.T) {
		ch1, _ := channel.NewChannel("Channel1")
		mapping1, _ := channel.NewEPGMapping("epg1", channel.MappingAuto, time.Now())
		ch1.SetEPGMapping(mapping1)

		ch2, _ := channel.NewChannel("Channel2")
		mapping2, _ := channel.NewEPGMapping("epg2", channel.MappingManual, time.Now())
		ch2.SetEPGMapping(mapping2)

		channelRepo := &mockChannelRepository{
			findAllFunc: func(ctx context.Context) ([]channel.Channel, error) {
				return []channel.Channel{ch1, ch2}, nil
			},
		}
		streamRepo := &mockStreamRepository{}
		epgFetcher := &mockEPGFetcher{}
		subRepo := &mockSubscriptionRepository{}
		acestreamSrc := &mockAcestreamSource{}

		channelService := application.NewChannelService(channelRepo, streamRepo)
		subscriptionSvc := application.NewSubscriptionService(subRepo, epgFetcher)
		epgSyncService := application.NewEPGSyncService(epgFetcher, acestreamSrc, channelRepo, streamRepo, subRepo)
		handler := NewEPGHTTPHandler(epgSyncService, subscriptionSvc, channelService)

		req := httptest.NewRequest(http.MethodGet, "/epg/mappings", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp []mappingResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 2 {
			t.Fatalf("expected 2 mappings, got %d", len(resp))
		}
		if resp[0].ChannelName != "Channel1" || resp[0].EPGID != "epg1" {
			t.Errorf("unexpected first mapping: %q -> %q", resp[0].ChannelName, resp[0].EPGID)
		}
		if resp[1].Source != "manual" {
			t.Errorf("expected source 'manual', got %q", resp[1].Source)
		}
	})

	t.Run("GET /epg/mappings excludes channels without mappings", func(t *testing.T) {
		ch1, _ := channel.NewChannel("Channel1")
		mapping1, _ := channel.NewEPGMapping("epg1", channel.MappingAuto, time.Now())
		ch1.SetEPGMapping(mapping1)

		ch2, _ := channel.NewChannel("Channel2")
		// ch2 has no mapping

		channelRepo := &mockChannelRepository{
			findAllFunc: func(ctx context.Context) ([]channel.Channel, error) {
				return []channel.Channel{ch1, ch2}, nil
			},
		}
		streamRepo := &mockStreamRepository{}
		epgFetcher := &mockEPGFetcher{}
		subRepo := &mockSubscriptionRepository{}
		acestreamSrc := &mockAcestreamSource{}

		channelService := application.NewChannelService(channelRepo, streamRepo)
		subscriptionSvc := application.NewSubscriptionService(subRepo, epgFetcher)
		epgSyncService := application.NewEPGSyncService(epgFetcher, acestreamSrc, channelRepo, streamRepo, subRepo)
		handler := NewEPGHTTPHandler(epgSyncService, subscriptionSvc, channelService)

		req := httptest.NewRequest(http.MethodGet, "/epg/mappings", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp []mappingResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 1 {
			t.Fatalf("expected 1 mapping, got %d", len(resp))
		}
		if resp[0].ChannelName != "Channel1" {
			t.Errorf("expected channel name 'Channel1', got %q", resp[0].ChannelName)
		}
	})
}

func TestEPGHTTPHandler_UpdateMapping(t *testing.T) {
	t.Run("PUT /epg/mappings/{channelName} updates mapping successfully", func(t *testing.T) {
		ch, _ := channel.NewChannel("TestChannel")
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				if name == "TestChannel" {
					return ch, nil
				}
				return channel.Channel{}, channel.ErrChannelNotFound
			},
			updateFunc: func(ctx context.Context, savedCh channel.Channel) error {
				ch = savedCh
				return nil
			},
		}
		streamRepo := &mockStreamRepository{}
		epgFetcher := &mockEPGFetcher{}
		subRepo := &mockSubscriptionRepository{}
		acestreamSrc := &mockAcestreamSource{}

		channelService := application.NewChannelService(channelRepo, streamRepo)
		subscriptionSvc := application.NewSubscriptionService(subRepo, epgFetcher)
		epgSyncService := application.NewEPGSyncService(epgFetcher, acestreamSrc, channelRepo, streamRepo, subRepo)
		handler := NewEPGHTTPHandler(epgSyncService, subscriptionSvc, channelService)

		reqBody := bytes.NewBufferString(`{"epg_id":"new_epg_id"}`)
		req := httptest.NewRequest(http.MethodPut, "/epg/mappings/TestChannel", reqBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp mappingResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.EPGID != "new_epg_id" {
			t.Errorf("expected epg_id 'new_epg_id', got %q", resp.EPGID)
		}
		if resp.Source != "manual" {
			t.Errorf("expected source 'manual', got %q", resp.Source)
		}
	})

	t.Run("PUT /epg/mappings/{channelName} returns 404 for non-existent channel", func(t *testing.T) {
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				return channel.Channel{}, channel.ErrChannelNotFound
			},
		}
		streamRepo := &mockStreamRepository{}
		epgFetcher := &mockEPGFetcher{}
		subRepo := &mockSubscriptionRepository{}
		acestreamSrc := &mockAcestreamSource{}

		channelService := application.NewChannelService(channelRepo, streamRepo)
		subscriptionSvc := application.NewSubscriptionService(subRepo, epgFetcher)
		epgSyncService := application.NewEPGSyncService(epgFetcher, acestreamSrc, channelRepo, streamRepo, subRepo)
		handler := NewEPGHTTPHandler(epgSyncService, subscriptionSvc, channelService)

		reqBody := bytes.NewBufferString(`{"epg_id":"new_epg_id"}`)
		req := httptest.NewRequest(http.MethodPut, "/epg/mappings/NonExistent", reqBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != "channel not found" {
			t.Errorf("expected error 'channel not found', got %q", resp.Error)
		}
	})

	t.Run("PUT /epg/mappings/{channelName} returns 400 for invalid JSON", func(t *testing.T) {
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{}
		epgFetcher := &mockEPGFetcher{}
		subRepo := &mockSubscriptionRepository{}
		acestreamSrc := &mockAcestreamSource{}

		channelService := application.NewChannelService(channelRepo, streamRepo)
		subscriptionSvc := application.NewSubscriptionService(subRepo, epgFetcher)
		epgSyncService := application.NewEPGSyncService(epgFetcher, acestreamSrc, channelRepo, streamRepo, subRepo)
		handler := NewEPGHTTPHandler(epgSyncService, subscriptionSvc, channelService)

		reqBody := bytes.NewBufferString(`invalid json`)
		req := httptest.NewRequest(http.MethodPut, "/epg/mappings/TestChannel", reqBody)
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
}

func TestEPGHTTPHandler_MethodNotAllowed(t *testing.T) {
	t.Run("returns 405 for unsupported methods", func(t *testing.T) {
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{}
		epgFetcher := &mockEPGFetcher{}
		subRepo := &mockSubscriptionRepository{}
		acestreamSrc := &mockAcestreamSource{}

		channelService := application.NewChannelService(channelRepo, streamRepo)
		subscriptionSvc := application.NewSubscriptionService(subRepo, epgFetcher)
		epgSyncService := application.NewEPGSyncService(epgFetcher, acestreamSrc, channelRepo, streamRepo, subRepo)
		handler := NewEPGHTTPHandler(epgSyncService, subscriptionSvc, channelService)

		methods := []string{http.MethodPatch, http.MethodHead, http.MethodOptions}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/epg/channels", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("method %s: expected status 405, got %d", method, rec.Code)
			}
		}
	})
}
