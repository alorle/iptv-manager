package driver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alorle/iptv-manager/internal/application"
	"github.com/alorle/iptv-manager/internal/channel"
	"github.com/alorle/iptv-manager/internal/port/driven"
)

// mockChannelRepositoryForHealth is a mock implementation for health check testing.
type mockChannelRepositoryForHealth struct {
	pingFunc func(ctx context.Context) error
}

func (m *mockChannelRepositoryForHealth) Ping(ctx context.Context) error {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return nil
}

func (m *mockChannelRepositoryForHealth) Save(ctx context.Context, ch channel.Channel) error {
	return nil
}

func (m *mockChannelRepositoryForHealth) FindByName(ctx context.Context, name string) (channel.Channel, error) {
	return channel.Channel{}, channel.ErrChannelNotFound
}

func (m *mockChannelRepositoryForHealth) FindAll(ctx context.Context) ([]channel.Channel, error) {
	return []channel.Channel{}, nil
}

func (m *mockChannelRepositoryForHealth) Delete(ctx context.Context, name string) error {
	return nil
}

// mockAceStreamEngine is a mock implementation for health check testing.
type mockAceStreamEngine struct {
	pingFunc func(ctx context.Context) error
}

func (m *mockAceStreamEngine) Ping(ctx context.Context) error {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return nil
}

func (m *mockAceStreamEngine) StartStream(ctx context.Context, infoHash, pid string) (string, error) {
	return "", nil
}

func (m *mockAceStreamEngine) GetStats(ctx context.Context, pid string) (driven.StreamStats, error) {
	return driven.StreamStats{}, nil
}

func (m *mockAceStreamEngine) StopStream(ctx context.Context, pid string) error {
	return nil
}

func (m *mockAceStreamEngine) StreamContent(ctx context.Context, streamURL string, dst io.Writer) error {
	return nil
}

func TestHealthHTTPHandler_ServeHTTP(t *testing.T) {
	t.Run("GET /health returns 200 when all dependencies are healthy", func(t *testing.T) {
		// Arrange: Create mocks that return no errors
		dbRepo := &mockChannelRepositoryForHealth{
			pingFunc: func(ctx context.Context) error {
				return nil
			},
		}
		engine := &mockAceStreamEngine{
			pingFunc: func(ctx context.Context) error {
				return nil
			},
		}
		service := application.NewHealthService(dbRepo, engine)
		handler := NewHealthHTTPHandler(service)

		// Act: Make request
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Assert: Verify response
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp healthResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Status != "ok" {
			t.Errorf("expected status 'ok', got '%s'", resp.Status)
		}
		if resp.DB != "ok" {
			t.Errorf("expected db 'ok', got '%s'", resp.DB)
		}
		if resp.AceStreamEngine != "ok" {
			t.Errorf("expected acestream_engine 'ok', got '%s'", resp.AceStreamEngine)
		}
	})

	t.Run("GET /health returns 503 when database is unavailable", func(t *testing.T) {
		// Arrange: Database returns an error
		dbRepo := &mockChannelRepositoryForHealth{
			pingFunc: func(ctx context.Context) error {
				return errors.New("database connection failed")
			},
		}
		engine := &mockAceStreamEngine{
			pingFunc: func(ctx context.Context) error {
				return nil
			},
		}
		service := application.NewHealthService(dbRepo, engine)
		handler := NewHealthHTTPHandler(service)

		// Act: Make request
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Assert: Verify response
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", rec.Code)
		}

		var resp healthResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Status != "degraded" {
			t.Errorf("expected status 'degraded', got '%s'", resp.Status)
		}
		if resp.DB != "error" {
			t.Errorf("expected db 'error', got '%s'", resp.DB)
		}
		if resp.AceStreamEngine != "ok" {
			t.Errorf("expected acestream_engine 'ok', got '%s'", resp.AceStreamEngine)
		}
	})

	t.Run("GET /health returns 503 when AceStream engine is unavailable", func(t *testing.T) {
		// Arrange: Engine returns an error
		dbRepo := &mockChannelRepositoryForHealth{
			pingFunc: func(ctx context.Context) error {
				return nil
			},
		}
		engine := &mockAceStreamEngine{
			pingFunc: func(ctx context.Context) error {
				return errors.New("acestream engine not reachable")
			},
		}
		service := application.NewHealthService(dbRepo, engine)
		handler := NewHealthHTTPHandler(service)

		// Act: Make request
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Assert: Verify response
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", rec.Code)
		}

		var resp healthResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Status != "degraded" {
			t.Errorf("expected status 'degraded', got '%s'", resp.Status)
		}
		if resp.DB != "ok" {
			t.Errorf("expected db 'ok', got '%s'", resp.DB)
		}
		if resp.AceStreamEngine != "error" {
			t.Errorf("expected acestream_engine 'error', got '%s'", resp.AceStreamEngine)
		}
	})

	t.Run("GET /health returns 503 when both dependencies are unavailable", func(t *testing.T) {
		// Arrange: Both return errors
		dbRepo := &mockChannelRepositoryForHealth{
			pingFunc: func(ctx context.Context) error {
				return errors.New("database connection failed")
			},
		}
		engine := &mockAceStreamEngine{
			pingFunc: func(ctx context.Context) error {
				return errors.New("acestream engine not reachable")
			},
		}
		service := application.NewHealthService(dbRepo, engine)
		handler := NewHealthHTTPHandler(service)

		// Act: Make request
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Assert: Verify response
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", rec.Code)
		}

		var resp healthResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Status != "degraded" {
			t.Errorf("expected status 'degraded', got '%s'", resp.Status)
		}
		if resp.DB != "error" {
			t.Errorf("expected db 'error', got '%s'", resp.DB)
		}
		if resp.AceStreamEngine != "error" {
			t.Errorf("expected acestream_engine 'error', got '%s'", resp.AceStreamEngine)
		}
	})

	t.Run("POST /health returns 405 Method Not Allowed", func(t *testing.T) {
		// Arrange
		dbRepo := &mockChannelRepositoryForHealth{}
		engine := &mockAceStreamEngine{}
		service := application.NewHealthService(dbRepo, engine)
		handler := NewHealthHTTPHandler(service)

		// Act: Make POST request
		req := httptest.NewRequest(http.MethodPost, "/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Assert: Verify response
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", rec.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Error != "method not allowed" {
			t.Errorf("expected error 'method not allowed', got '%s'", resp.Error)
		}
	})

	t.Run("PUT /health returns 405 Method Not Allowed", func(t *testing.T) {
		// Arrange
		dbRepo := &mockChannelRepositoryForHealth{}
		engine := &mockAceStreamEngine{}
		service := application.NewHealthService(dbRepo, engine)
		handler := NewHealthHTTPHandler(service)

		// Act: Make PUT request
		req := httptest.NewRequest(http.MethodPut, "/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Assert: Verify response
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", rec.Code)
		}
	})

	t.Run("DELETE /health returns 405 Method Not Allowed", func(t *testing.T) {
		// Arrange
		dbRepo := &mockChannelRepositoryForHealth{}
		engine := &mockAceStreamEngine{}
		service := application.NewHealthService(dbRepo, engine)
		handler := NewHealthHTTPHandler(service)

		// Act: Make DELETE request
		req := httptest.NewRequest(http.MethodDelete, "/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Assert: Verify response
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", rec.Code)
		}
	})
}
