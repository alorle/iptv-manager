package driver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alorle/iptv-manager/internal/application"
	"github.com/alorle/iptv-manager/internal/stream"
)

func TestPlaylistHTTPHandler_ServeHTTP(t *testing.T) {
	t.Run("GET /playlist.m3u returns M3U playlist with streams", func(t *testing.T) {
		st1, _ := stream.NewStream("abc123", "Channel1")
		st2, _ := stream.NewStream("def456", "Channel2")
		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return []stream.Stream{st1, st2}, nil
			},
		}
		service := application.NewPlaylistService(streamRepo)
		handler := NewPlaylistHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/playlist.m3u", nil)
		req.Host = "localhost:8080"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "audio/mpegurl" {
			t.Errorf("expected Content-Type 'audio/mpegurl', got %q", contentType)
		}

		body := rec.Body.String()

		// Check header
		if !strings.HasPrefix(body, "#EXTM3U\n") {
			t.Error("M3U playlist should start with #EXTM3U header")
		}

		// Check first stream
		if !strings.Contains(body, `#EXTINF:-1 tvg-id="Channel1",Channel1 - abc123`) {
			t.Error("M3U playlist should contain first stream metadata")
		}
		if !strings.Contains(body, "http://localhost:8080/ace/getstream?id=abc123") {
			t.Error("M3U playlist should contain first stream URL")
		}

		// Check second stream
		if !strings.Contains(body, `#EXTINF:-1 tvg-id="Channel2",Channel2 - def456`) {
			t.Error("M3U playlist should contain second stream metadata")
		}
		if !strings.Contains(body, "http://localhost:8080/ace/getstream?id=def456") {
			t.Error("M3U playlist should contain second stream URL")
		}
	})

	t.Run("GET /playlist.m3u returns empty playlist when no streams exist", func(t *testing.T) {
		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return []stream.Stream{}, nil
			},
		}
		service := application.NewPlaylistService(streamRepo)
		handler := NewPlaylistHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/playlist.m3u", nil)
		req.Host = "localhost:8080"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		body := rec.Body.String()
		if body != "#EXTM3U\n" {
			t.Errorf("expected only #EXTM3U header, got %q", body)
		}
	})

	t.Run("GET /playlist.m3u returns 500 on internal error", func(t *testing.T) {
		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return nil, errors.New("repository error")
			},
		}
		service := application.NewPlaylistService(streamRepo)
		handler := NewPlaylistHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/playlist.m3u", nil)
		req.Host = "localhost:8080"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rec.Code)
		}
	})

	t.Run("GET /playlist.m3u uses request Host header in URLs", func(t *testing.T) {
		st1, _ := stream.NewStream("xyz789", "TestChannel")
		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return []stream.Stream{st1}, nil
			},
		}
		service := application.NewPlaylistService(streamRepo)
		handler := NewPlaylistHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/playlist.m3u", nil)
		req.Host = "example.com:9000"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "http://example.com:9000/ace/getstream?id=xyz789") {
			t.Error("M3U playlist should use the request Host header in stream URLs")
		}
	})

	t.Run("POST /playlist.m3u returns 405 method not allowed", func(t *testing.T) {
		streamRepo := &mockStreamRepository{}
		service := application.NewPlaylistService(streamRepo)
		handler := NewPlaylistHTTPHandler(service)

		req := httptest.NewRequest(http.MethodPost, "/playlist.m3u", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", rec.Code)
		}
	})

	t.Run("PUT /playlist.m3u returns 405 method not allowed", func(t *testing.T) {
		streamRepo := &mockStreamRepository{}
		service := application.NewPlaylistService(streamRepo)
		handler := NewPlaylistHTTPHandler(service)

		req := httptest.NewRequest(http.MethodPut, "/playlist.m3u", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", rec.Code)
		}
	})

	t.Run("DELETE /playlist.m3u returns 405 method not allowed", func(t *testing.T) {
		streamRepo := &mockStreamRepository{}
		service := application.NewPlaylistService(streamRepo)
		handler := NewPlaylistHTTPHandler(service)

		req := httptest.NewRequest(http.MethodDelete, "/playlist.m3u", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", rec.Code)
		}
	})
}
