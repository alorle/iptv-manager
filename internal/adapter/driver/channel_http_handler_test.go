package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alorle/iptv-manager/internal/application"
	"github.com/alorle/iptv-manager/internal/channel"
	"github.com/alorle/iptv-manager/internal/stream"
)

// mockChannelRepository is a mock implementation for testing.
type mockChannelRepository struct {
	saveFunc       func(ctx context.Context, ch channel.Channel) error
	findByNameFunc func(ctx context.Context, name string) (channel.Channel, error)
	findAllFunc    func(ctx context.Context) ([]channel.Channel, error)
	deleteFunc     func(ctx context.Context, name string) error
}

func (m *mockChannelRepository) Save(ctx context.Context, ch channel.Channel) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, ch)
	}
	return nil
}

func (m *mockChannelRepository) FindByName(ctx context.Context, name string) (channel.Channel, error) {
	if m.findByNameFunc != nil {
		return m.findByNameFunc(ctx, name)
	}
	return channel.Channel{}, channel.ErrChannelNotFound
}

func (m *mockChannelRepository) FindAll(ctx context.Context) ([]channel.Channel, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(ctx)
	}
	return []channel.Channel{}, nil
}

func (m *mockChannelRepository) Delete(ctx context.Context, name string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, name)
	}
	return nil
}

// mockStreamRepository is a mock implementation for testing.
type mockStreamRepository struct {
	saveFunc                func(ctx context.Context, s stream.Stream) error
	findByInfoHashFunc      func(ctx context.Context, infoHash string) (stream.Stream, error)
	findAllFunc             func(ctx context.Context) ([]stream.Stream, error)
	findByChannelNameFunc   func(ctx context.Context, channelName string) ([]stream.Stream, error)
	deleteFunc              func(ctx context.Context, infoHash string) error
	deleteByChannelNameFunc func(ctx context.Context, channelName string) error
}

func (m *mockStreamRepository) Save(ctx context.Context, s stream.Stream) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, s)
	}
	return nil
}

func (m *mockStreamRepository) FindByInfoHash(ctx context.Context, infoHash string) (stream.Stream, error) {
	if m.findByInfoHashFunc != nil {
		return m.findByInfoHashFunc(ctx, infoHash)
	}
	return stream.Stream{}, stream.ErrStreamNotFound
}

func (m *mockStreamRepository) FindAll(ctx context.Context) ([]stream.Stream, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(ctx)
	}
	return []stream.Stream{}, nil
}

func (m *mockStreamRepository) FindByChannelName(ctx context.Context, channelName string) ([]stream.Stream, error) {
	if m.findByChannelNameFunc != nil {
		return m.findByChannelNameFunc(ctx, channelName)
	}
	return []stream.Stream{}, nil
}

func (m *mockStreamRepository) Delete(ctx context.Context, infoHash string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, infoHash)
	}
	return nil
}

func (m *mockStreamRepository) DeleteByChannelName(ctx context.Context, channelName string) error {
	if m.deleteByChannelNameFunc != nil {
		return m.deleteByChannelNameFunc(ctx, channelName)
	}
	return nil
}

func TestChannelHTTPHandler_Create(t *testing.T) {
	t.Run("POST /channels creates channel successfully", func(t *testing.T) {
		channelRepo := &mockChannelRepository{
			saveFunc: func(ctx context.Context, ch channel.Channel) error {
				return nil
			},
		}
		streamRepo := &mockStreamRepository{}
		service := application.NewChannelService(channelRepo, streamRepo)
		handler := NewChannelHTTPHandler(service)

		reqBody := bytes.NewBufferString(`{"name":"TestChannel"}`)
		req := httptest.NewRequest(http.MethodPost, "/channels", reqBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", rec.Code)
		}

		var resp channelResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Name != "TestChannel" {
			t.Errorf("expected channel name 'TestChannel', got %q", resp.Name)
		}
	})

	t.Run("POST /channels returns 400 for invalid JSON", func(t *testing.T) {
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{}
		service := application.NewChannelService(channelRepo, streamRepo)
		handler := NewChannelHTTPHandler(service)

		reqBody := bytes.NewBufferString(`invalid json`)
		req := httptest.NewRequest(http.MethodPost, "/channels", reqBody)
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

	t.Run("POST /channels returns 400 for empty name", func(t *testing.T) {
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{}
		service := application.NewChannelService(channelRepo, streamRepo)
		handler := NewChannelHTTPHandler(service)

		reqBody := bytes.NewBufferString(`{"name":""}`)
		req := httptest.NewRequest(http.MethodPost, "/channels", reqBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != channel.ErrEmptyName.Error() {
			t.Errorf("expected error %q, got %q", channel.ErrEmptyName.Error(), resp.Error)
		}
	})

	t.Run("POST /channels returns 409 for duplicate channel", func(t *testing.T) {
		channelRepo := &mockChannelRepository{
			saveFunc: func(ctx context.Context, ch channel.Channel) error {
				return channel.ErrChannelAlreadyExists
			},
		}
		streamRepo := &mockStreamRepository{}
		service := application.NewChannelService(channelRepo, streamRepo)
		handler := NewChannelHTTPHandler(service)

		reqBody := bytes.NewBufferString(`{"name":"TestChannel"}`)
		req := httptest.NewRequest(http.MethodPost, "/channels", reqBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("expected status 409, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != channel.ErrChannelAlreadyExists.Error() {
			t.Errorf("expected error %q, got %q", channel.ErrChannelAlreadyExists.Error(), resp.Error)
		}
	})
}

func TestChannelHTTPHandler_List(t *testing.T) {
	t.Run("GET /channels returns all channels", func(t *testing.T) {
		ch1, _ := channel.NewChannel("Channel1")
		ch2, _ := channel.NewChannel("Channel2")
		channelRepo := &mockChannelRepository{
			findAllFunc: func(ctx context.Context) ([]channel.Channel, error) {
				return []channel.Channel{ch1, ch2}, nil
			},
		}
		streamRepo := &mockStreamRepository{}
		service := application.NewChannelService(channelRepo, streamRepo)
		handler := NewChannelHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/channels", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp []channelResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 2 {
			t.Fatalf("expected 2 channels, got %d", len(resp))
		}
		if resp[0].Name != "Channel1" || resp[1].Name != "Channel2" {
			t.Errorf("unexpected channel names: %q, %q", resp[0].Name, resp[1].Name)
		}
	})

	t.Run("GET /channels returns empty array when no channels exist", func(t *testing.T) {
		channelRepo := &mockChannelRepository{
			findAllFunc: func(ctx context.Context) ([]channel.Channel, error) {
				return []channel.Channel{}, nil
			},
		}
		streamRepo := &mockStreamRepository{}
		service := application.NewChannelService(channelRepo, streamRepo)
		handler := NewChannelHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/channels", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp []channelResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 0 {
			t.Errorf("expected empty array, got %d channels", len(resp))
		}
	})
}

func TestChannelHTTPHandler_Get(t *testing.T) {
	t.Run("GET /channels/{name} returns channel", func(t *testing.T) {
		ch, _ := channel.NewChannel("TestChannel")
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				if name == "TestChannel" {
					return ch, nil
				}
				return channel.Channel{}, channel.ErrChannelNotFound
			},
		}
		streamRepo := &mockStreamRepository{}
		service := application.NewChannelService(channelRepo, streamRepo)
		handler := NewChannelHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/channels/TestChannel", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp channelResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Name != "TestChannel" {
			t.Errorf("expected channel name 'TestChannel', got %q", resp.Name)
		}
	})

	t.Run("GET /channels/{name} returns 404 for non-existent channel", func(t *testing.T) {
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				return channel.Channel{}, channel.ErrChannelNotFound
			},
		}
		streamRepo := &mockStreamRepository{}
		service := application.NewChannelService(channelRepo, streamRepo)
		handler := NewChannelHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/channels/NonExistent", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != channel.ErrChannelNotFound.Error() {
			t.Errorf("expected error %q, got %q", channel.ErrChannelNotFound.Error(), resp.Error)
		}
	})
}

func TestChannelHTTPHandler_Delete(t *testing.T) {
	t.Run("DELETE /channels/{name} deletes channel successfully", func(t *testing.T) {
		ch, _ := channel.NewChannel("TestChannel")
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				if name == "TestChannel" {
					return ch, nil
				}
				return channel.Channel{}, channel.ErrChannelNotFound
			},
			deleteFunc: func(ctx context.Context, name string) error {
				return nil
			},
		}
		streamRepo := &mockStreamRepository{
			deleteByChannelNameFunc: func(ctx context.Context, channelName string) error {
				return nil
			},
		}
		service := application.NewChannelService(channelRepo, streamRepo)
		handler := NewChannelHTTPHandler(service)

		req := httptest.NewRequest(http.MethodDelete, "/channels/TestChannel", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", rec.Code)
		}
	})

	t.Run("DELETE /channels/{name} returns 404 for non-existent channel", func(t *testing.T) {
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				return channel.Channel{}, channel.ErrChannelNotFound
			},
		}
		streamRepo := &mockStreamRepository{}
		service := application.NewChannelService(channelRepo, streamRepo)
		handler := NewChannelHTTPHandler(service)

		req := httptest.NewRequest(http.MethodDelete, "/channels/NonExistent", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != channel.ErrChannelNotFound.Error() {
			t.Errorf("expected error %q, got %q", channel.ErrChannelNotFound.Error(), resp.Error)
		}
	})

	t.Run("DELETE /channels/{name} returns 500 when stream deletion fails", func(t *testing.T) {
		ch, _ := channel.NewChannel("TestChannel")
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				return ch, nil
			},
		}
		streamRepo := &mockStreamRepository{
			deleteByChannelNameFunc: func(ctx context.Context, channelName string) error {
				return errors.New("stream deletion failed")
			},
		}
		service := application.NewChannelService(channelRepo, streamRepo)
		handler := NewChannelHTTPHandler(service)

		req := httptest.NewRequest(http.MethodDelete, "/channels/TestChannel", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != "internal server error" {
			t.Errorf("expected error 'internal server error', got %q", resp.Error)
		}
	})
}

func TestChannelHTTPHandler_MethodNotAllowed(t *testing.T) {
	t.Run("returns 405 for unsupported methods", func(t *testing.T) {
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{}
		service := application.NewChannelService(channelRepo, streamRepo)
		handler := NewChannelHTTPHandler(service)

		methods := []string{http.MethodPut, http.MethodPatch, http.MethodHead, http.MethodOptions}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/channels", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("method %s: expected status 405, got %d", method, rec.Code)
			}
		}
	})
}
