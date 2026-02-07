package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/config"
	"github.com/alorle/iptv-manager/domain"
	"github.com/alorle/iptv-manager/logging"
	"github.com/alorle/iptv-manager/multiplexer"
	"github.com/alorle/iptv-manager/pidmanager"
)

const testRemoteAddr = "192.168.1.100:12345"

// Helper to create test dependencies with a no-op logger
func createTestDeps() StreamDependencies {
	return StreamDependencies{
		Logger:      logging.NewWithWriter(logging.INFO, "test", io.Discard),
		Multiplexer: multiplexer.New(multiplexer.DefaultConfig()),
		PidMgr:      pidmanager.NewManager(),
	}
}

func TestIsValidContentID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected bool
	}{
		{
			name:     "valid 40 hex chars",
			id:       "0123456789abcdef0123456789abcdef01234567",
			expected: true,
		},
		{
			name:     "valid uppercase hex",
			id:       "0123456789ABCDEF0123456789ABCDEF01234567",
			expected: true,
		},
		{
			name:     "valid mixed case",
			id:       "0123456789AbCdEf0123456789aBcDeF01234567",
			expected: true,
		},
		{
			name:     "too short",
			id:       "0123456789abcdef",
			expected: false,
		},
		{
			name:     "too long",
			id:       "0123456789abcdef0123456789abcdef012345678",
			expected: false,
		},
		{
			name:     "invalid chars",
			id:       "0123456789abcdef0123456789abcdef0123456g",
			expected: false,
		},
		{
			name:     "spaces",
			id:       "0123456789abcdef 123456789abcdef01234567",
			expected: false,
		},
		{
			name:     "empty",
			id:       "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domain.IsValidContentID(tt.id)
			if result != tt.expected {
				t.Errorf("isValidContentID(%s) = %v, expected %v", tt.id, result, tt.expected)
			}
		})
	}
}

func TestStreamHandler_MissingID(t *testing.T) {
	cfg := &config.Config{}
	cfg.Acestream.EngineURL = "http://localhost:6878"
	deps := createTestDeps()
	handler := CreateStreamHandler(cfg, deps)

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Missing id parameter") {
		t.Errorf("Expected 'Missing id parameter' error, got: %s", w.Body.String())
	}
}

func TestStreamHandler_InvalidID(t *testing.T) {
	cfg := &config.Config{}
	cfg.Acestream.EngineURL = "http://localhost:6878"
	deps := createTestDeps()
	handler := CreateStreamHandler(cfg, deps)

	tests := []struct {
		name string
		id   string
	}{
		{"too short", "abc123"},
		{"too long", "0123456789abcdef0123456789abcdef012345678"},
		{"invalid chars", "0123456789abcdef0123456789abcdef0123456g"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/stream?id="+tt.id, nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}

			if !strings.Contains(w.Body.String(), "Invalid id format") {
				t.Errorf("Expected 'Invalid id format' error, got: %s", w.Body.String())
			}
		})
	}
}

func TestStreamHandler_MethodNotAllowed(t *testing.T) {
	cfg := &config.Config{}
	cfg.Acestream.EngineURL = "http://localhost:6878"
	deps := createTestDeps()
	handler := CreateStreamHandler(cfg, deps)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/stream?id=0123456789abcdef0123456789abcdef01234567", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

func TestStreamHandler_WithMockEngine(t *testing.T) {
	// Create mock Ace Stream Engine
	mockEngine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify PID is passed
		pid := r.URL.Query().Get("pid")
		if pid == "" {
			t.Error("PID not passed to engine")
		}

		// Verify content ID is passed
		id := r.URL.Query().Get("id")
		if id != "0123456789abcdef0123456789abcdef01234567" {
			t.Errorf("Expected content ID in query, got: %s", id)
		}

		// Simulate streaming response
		w.Header().Set("Content-Type", "video/mp2t")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mock stream data"))
	}))
	defer mockEngine.Close()

	cfg := &config.Config{}
	cfg.Acestream.EngineURL = mockEngine.URL
	deps := createTestDeps()
	handler := CreateStreamHandler(cfg, deps)

	req := httptest.NewRequest(http.MethodGet, "/stream?id=0123456789abcdef0123456789abcdef01234567", nil)
	req.RemoteAddr = testRemoteAddr
	req.Header.Set("User-Agent", "VLC/3.0.18")

	w := httptest.NewRecorder()

	// Use a context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req = req.WithContext(ctx)

	handler(w, req)

	// Note: The handler may not complete fully in test mode due to streaming nature
	// We're mainly testing that it doesn't error out
}

func TestStreamHandler_EngineConnectionError(t *testing.T) {
	// Use invalid engine URL
	cfg := &config.Config{}
	cfg.Acestream.EngineURL = "http://localhost:99999"
	deps := createTestDeps()
	handler := CreateStreamHandler(cfg, deps)

	req := httptest.NewRequest(http.MethodGet, "/stream?id=0123456789abcdef0123456789abcdef01234567", nil)
	req.RemoteAddr = testRemoteAddr

	w := httptest.NewRecorder()

	// Use short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req = req.WithContext(ctx)

	handler(w, req)

	// Should get Bad Gateway error
	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected status 502, got %d", w.Code)
	}
}

func TestStreamHandler_TranscodeParameters(t *testing.T) {
	// Mock engine that verifies transcode parameters
	mockEngine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify transcode_audio parameter
		transcodeAudio := r.URL.Query().Get("transcode_audio")
		if transcodeAudio != "mp3" {
			t.Errorf("Expected transcode_audio=mp3, got: %s", transcodeAudio)
		}

		w.Header().Set("Content-Type", "video/mp2t")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mock data"))
	}))
	defer mockEngine.Close()

	cfg := &config.Config{}
	cfg.Acestream.EngineURL = mockEngine.URL
	deps := createTestDeps()
	handler := CreateStreamHandler(cfg, deps)

	req := httptest.NewRequest(http.MethodGet, "/stream?id=0123456789abcdef0123456789abcdef01234567&transcode_audio=mp3", nil)
	req.RemoteAddr = testRemoteAddr

	w := httptest.NewRecorder()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	req = req.WithContext(ctx)

	handler(w, req)
}

func TestStreamHandler_PIDReuse(t *testing.T) {
	// This test verifies PID manager reuses PIDs for reconnecting clients
	pidMgr := pidmanager.NewManager()

	contentID := "0123456789abcdef0123456789abcdef01234567"
	clientID := pidmanager.ClientIdentifier{
		IP:        "192.168.1.100:12345",
		UserAgent: "VLC/3.0.18",
	}

	// First connection
	firstPID := pidMgr.GetOrCreatePID(contentID, clientID)

	// Disconnect (release PID but don't cleanup yet)
	_ = pidMgr.ReleasePID(firstPID)

	// Reconnection from same client - should reuse PID
	secondPID := pidMgr.GetOrCreatePID(contentID, clientID)

	if firstPID != secondPID {
		t.Errorf("Expected PID reuse for reconnecting client, got first=%d, second=%d", firstPID, secondPID)
	}

	// After cleanup, new PID should be generated
	_ = pidMgr.ReleasePID(secondPID)
	pidMgr.CleanupDisconnected()

	thirdPID := pidMgr.GetOrCreatePID(contentID, clientID)
	if thirdPID == firstPID {
		t.Error("After cleanup, new connection should get a new PID")
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	healthHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}

	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got: %s", w.Body.String())
	}
}
