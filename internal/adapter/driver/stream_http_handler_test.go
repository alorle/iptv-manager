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

func TestStreamHTTPHandler_Create(t *testing.T) {
	t.Run("POST /streams creates stream successfully", func(t *testing.T) {
		ch, _ := channel.NewChannel("TestChannel")
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				if name == "TestChannel" {
					return ch, nil
				}
				return channel.Channel{}, channel.ErrChannelNotFound
			},
		}
		streamRepo := &mockStreamRepository{
			saveFunc: func(ctx context.Context, s stream.Stream) error {
				return nil
			},
		}
		service := application.NewStreamService(streamRepo, channelRepo)
		handler := NewStreamHTTPHandler(service)

		reqBody := bytes.NewBufferString(`{"info_hash":"abc123","channel_name":"TestChannel"}`)
		req := httptest.NewRequest(http.MethodPost, "/streams", reqBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", rec.Code)
		}

		var resp streamResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.InfoHash != "abc123" {
			t.Errorf("expected infohash 'abc123', got %q", resp.InfoHash)
		}
		if resp.ChannelName != "TestChannel" {
			t.Errorf("expected channel name 'TestChannel', got %q", resp.ChannelName)
		}
	})

	t.Run("POST /streams returns 400 for invalid JSON", func(t *testing.T) {
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{}
		service := application.NewStreamService(streamRepo, channelRepo)
		handler := NewStreamHTTPHandler(service)

		reqBody := bytes.NewBufferString(`invalid json`)
		req := httptest.NewRequest(http.MethodPost, "/streams", reqBody)
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

	t.Run("POST /streams returns 400 for empty infohash", func(t *testing.T) {
		ch, _ := channel.NewChannel("TestChannel")
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				return ch, nil
			},
		}
		streamRepo := &mockStreamRepository{}
		service := application.NewStreamService(streamRepo, channelRepo)
		handler := NewStreamHTTPHandler(service)

		reqBody := bytes.NewBufferString(`{"info_hash":"","channel_name":"TestChannel"}`)
		req := httptest.NewRequest(http.MethodPost, "/streams", reqBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != stream.ErrEmptyInfoHash.Error() {
			t.Errorf("expected error %q, got %q", stream.ErrEmptyInfoHash.Error(), resp.Error)
		}
	})

	t.Run("POST /streams returns 400 for empty channel name", func(t *testing.T) {
		ch, _ := channel.NewChannel("TestChannel")
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				return ch, nil
			},
		}
		streamRepo := &mockStreamRepository{}
		service := application.NewStreamService(streamRepo, channelRepo)
		handler := NewStreamHTTPHandler(service)

		reqBody := bytes.NewBufferString(`{"info_hash":"abc123","channel_name":""}`)
		req := httptest.NewRequest(http.MethodPost, "/streams", reqBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != stream.ErrEmptyChannelName.Error() {
			t.Errorf("expected error %q, got %q", stream.ErrEmptyChannelName.Error(), resp.Error)
		}
	})

	t.Run("POST /streams returns 400 when channel not found", func(t *testing.T) {
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				return channel.Channel{}, channel.ErrChannelNotFound
			},
		}
		streamRepo := &mockStreamRepository{}
		service := application.NewStreamService(streamRepo, channelRepo)
		handler := NewStreamHTTPHandler(service)

		reqBody := bytes.NewBufferString(`{"info_hash":"abc123","channel_name":"NonExistent"}`)
		req := httptest.NewRequest(http.MethodPost, "/streams", reqBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != channel.ErrChannelNotFound.Error() {
			t.Errorf("expected error %q, got %q", channel.ErrChannelNotFound.Error(), resp.Error)
		}
	})

	t.Run("POST /streams returns 409 for duplicate stream", func(t *testing.T) {
		ch, _ := channel.NewChannel("TestChannel")
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				return ch, nil
			},
		}
		streamRepo := &mockStreamRepository{
			saveFunc: func(ctx context.Context, s stream.Stream) error {
				return stream.ErrStreamAlreadyExists
			},
		}
		service := application.NewStreamService(streamRepo, channelRepo)
		handler := NewStreamHTTPHandler(service)

		reqBody := bytes.NewBufferString(`{"info_hash":"abc123","channel_name":"TestChannel"}`)
		req := httptest.NewRequest(http.MethodPost, "/streams", reqBody)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("expected status 409, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != stream.ErrStreamAlreadyExists.Error() {
			t.Errorf("expected error %q, got %q", stream.ErrStreamAlreadyExists.Error(), resp.Error)
		}
	})
}

func TestStreamHTTPHandler_List(t *testing.T) {
	t.Run("GET /streams returns all streams", func(t *testing.T) {
		st1, _ := stream.NewStream("abc123", "Channel1")
		st2, _ := stream.NewStream("def456", "Channel2")
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return []stream.Stream{st1, st2}, nil
			},
		}
		service := application.NewStreamService(streamRepo, channelRepo)
		handler := NewStreamHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/streams", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp []streamResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 2 {
			t.Fatalf("expected 2 streams, got %d", len(resp))
		}
		if resp[0].InfoHash != "abc123" || resp[1].InfoHash != "def456" {
			t.Errorf("unexpected stream infohashes: %q, %q", resp[0].InfoHash, resp[1].InfoHash)
		}
	})

	t.Run("GET /streams returns empty array when no streams exist", func(t *testing.T) {
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return []stream.Stream{}, nil
			},
		}
		service := application.NewStreamService(streamRepo, channelRepo)
		handler := NewStreamHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/streams", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp []streamResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 0 {
			t.Errorf("expected empty array, got %d streams", len(resp))
		}
	})
}

func TestStreamHTTPHandler_Get(t *testing.T) {
	t.Run("GET /streams/{infoHash} returns stream", func(t *testing.T) {
		st, _ := stream.NewStream("abc123", "TestChannel")
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{
			findByInfoHashFunc: func(ctx context.Context, infoHash string) (stream.Stream, error) {
				if infoHash == "abc123" {
					return st, nil
				}
				return stream.Stream{}, stream.ErrStreamNotFound
			},
		}
		service := application.NewStreamService(streamRepo, channelRepo)
		handler := NewStreamHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/streams/abc123", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp streamResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.InfoHash != "abc123" {
			t.Errorf("expected infohash 'abc123', got %q", resp.InfoHash)
		}
		if resp.ChannelName != "TestChannel" {
			t.Errorf("expected channel name 'TestChannel', got %q", resp.ChannelName)
		}
	})

	t.Run("GET /streams/{infoHash} returns 404 for non-existent stream", func(t *testing.T) {
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{
			findByInfoHashFunc: func(ctx context.Context, infoHash string) (stream.Stream, error) {
				return stream.Stream{}, stream.ErrStreamNotFound
			},
		}
		service := application.NewStreamService(streamRepo, channelRepo)
		handler := NewStreamHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/streams/nonexistent", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != stream.ErrStreamNotFound.Error() {
			t.Errorf("expected error %q, got %q", stream.ErrStreamNotFound.Error(), resp.Error)
		}
	})
}

func TestStreamHTTPHandler_Delete(t *testing.T) {
	t.Run("DELETE /streams/{infoHash} deletes stream successfully", func(t *testing.T) {
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{
			deleteFunc: func(ctx context.Context, infoHash string) error {
				if infoHash == "abc123" {
					return nil
				}
				return stream.ErrStreamNotFound
			},
		}
		service := application.NewStreamService(streamRepo, channelRepo)
		handler := NewStreamHTTPHandler(service)

		req := httptest.NewRequest(http.MethodDelete, "/streams/abc123", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", rec.Code)
		}
	})

	t.Run("DELETE /streams/{infoHash} returns 404 for non-existent stream", func(t *testing.T) {
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{
			deleteFunc: func(ctx context.Context, infoHash string) error {
				return stream.ErrStreamNotFound
			},
		}
		service := application.NewStreamService(streamRepo, channelRepo)
		handler := NewStreamHTTPHandler(service)

		req := httptest.NewRequest(http.MethodDelete, "/streams/nonexistent", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if resp.Error != stream.ErrStreamNotFound.Error() {
			t.Errorf("expected error %q, got %q", stream.ErrStreamNotFound.Error(), resp.Error)
		}
	})

	t.Run("DELETE /streams/{infoHash} returns 500 on internal error", func(t *testing.T) {
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{
			deleteFunc: func(ctx context.Context, infoHash string) error {
				return errors.New("internal error")
			},
		}
		service := application.NewStreamService(streamRepo, channelRepo)
		handler := NewStreamHTTPHandler(service)

		req := httptest.NewRequest(http.MethodDelete, "/streams/abc123", nil)
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

func TestStreamHTTPHandler_MethodNotAllowed(t *testing.T) {
	t.Run("returns 405 for unsupported methods", func(t *testing.T) {
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{}
		service := application.NewStreamService(streamRepo, channelRepo)
		handler := NewStreamHTTPHandler(service)

		methods := []string{http.MethodPut, http.MethodPatch, http.MethodHead, http.MethodOptions}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/streams", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("method %s: expected status 405, got %d", method, rec.Code)
			}
		}
	})
}
