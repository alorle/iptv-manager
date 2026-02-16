package driver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/internal/application"
	"github.com/alorle/iptv-manager/internal/port/driven"
	"github.com/alorle/iptv-manager/internal/probe"
	"github.com/alorle/iptv-manager/internal/stream"
)

// mockProbeRepository implements driven.ProbeRepository for testing.
type mockProbeRepository struct {
	saveFunc                func(ctx context.Context, r probe.Result) error
	findByInfoHashFunc      func(ctx context.Context, infoHash string) ([]probe.Result, error)
	findByInfoHashSinceFunc func(ctx context.Context, infoHash string, since time.Time) ([]probe.Result, error)
	deleteBeforeFunc        func(ctx context.Context, before time.Time) error
}

func (m *mockProbeRepository) Save(ctx context.Context, r probe.Result) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, r)
	}
	return nil
}

func (m *mockProbeRepository) FindByInfoHash(ctx context.Context, infoHash string) ([]probe.Result, error) {
	if m.findByInfoHashFunc != nil {
		return m.findByInfoHashFunc(ctx, infoHash)
	}
	return []probe.Result{}, nil
}

func (m *mockProbeRepository) FindByInfoHashSince(ctx context.Context, infoHash string, since time.Time) ([]probe.Result, error) {
	if m.findByInfoHashSinceFunc != nil {
		return m.findByInfoHashSinceFunc(ctx, infoHash, since)
	}
	return []probe.Result{}, nil
}

func (m *mockProbeRepository) DeleteBefore(ctx context.Context, before time.Time) error {
	if m.deleteBeforeFunc != nil {
		return m.deleteBeforeFunc(ctx, before)
	}
	return nil
}

// mockAceStreamEngineForProbe is a minimal mock for constructing ProbeService in handler tests.
type mockAceStreamEngineForProbe struct{}

func (m *mockAceStreamEngineForProbe) StartStream(ctx context.Context, infoHash, pid string) (string, error) {
	return "", nil
}
func (m *mockAceStreamEngineForProbe) GetStats(ctx context.Context, pid string) (driven.StreamStats, error) {
	return driven.StreamStats{}, nil
}
func (m *mockAceStreamEngineForProbe) StopStream(ctx context.Context, pid string) error { return nil }
func (m *mockAceStreamEngineForProbe) StreamContent(ctx context.Context, streamURL string, dst io.Writer, infoHash, pid string, writeTimeout time.Duration) error {
	return nil
}
func (m *mockAceStreamEngineForProbe) Ping(ctx context.Context) error { return nil }

func newProbeTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newProbeTestService(probeRepo *mockProbeRepository, streamRepo *mockStreamRepository) *application.ProbeService {
	return application.NewProbeService(
		probeRepo, streamRepo, &mockAceStreamEngineForProbe{}, newProbeTestLogger(),
		30*time.Second, 24*time.Hour,
	)
}

func TestProbeHTTPHandler_History(t *testing.T) {
	t.Run("GET /probes/{infoHash} returns probe history", func(t *testing.T) {
		now := time.Now()
		probeRepo := &mockProbeRepository{
			findByInfoHashSinceFunc: func(ctx context.Context, infoHash string, since time.Time) ([]probe.Result, error) {
				return []probe.Result{
					probe.ReconstructResult("abc123", now, true, 2*time.Second, 10, 100000, "dl", ""),
					probe.ReconstructResult("abc123", now.Add(-30*time.Minute), false, 0, 0, 0, "", "timeout"),
				}, nil
			},
		}

		service := newProbeTestService(probeRepo, &mockStreamRepository{})
		handler := NewProbeHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/probes/abc123", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp []probeResultResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 2 {
			t.Fatalf("expected 2 results, got %d", len(resp))
		}
		if resp[0].Available != true {
			t.Error("first result should be available")
		}
		if resp[1].Available != false {
			t.Error("second result should not be available")
		}
	})

	t.Run("GET /probes/{infoHash} returns empty array for unknown hash", func(t *testing.T) {
		probeRepo := &mockProbeRepository{}
		service := newProbeTestService(probeRepo, &mockStreamRepository{})
		handler := NewProbeHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/probes/unknown", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp []probeResultResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 0 {
			t.Errorf("expected 0 results, got %d", len(resp))
		}
	})
}

func TestProbeHTTPHandler_Metrics(t *testing.T) {
	t.Run("GET /probes/{infoHash}/metrics returns metrics", func(t *testing.T) {
		now := time.Now()
		probeRepo := &mockProbeRepository{
			findByInfoHashSinceFunc: func(ctx context.Context, infoHash string, since time.Time) ([]probe.Result, error) {
				return []probe.Result{
					probe.ReconstructResult("abc123", now, true, time.Second, 10, 100000, "dl", ""),
					probe.ReconstructResult("abc123", now.Add(-30*time.Minute), true, 2*time.Second, 20, 200000, "dl", ""),
				}, nil
			},
		}

		service := newProbeTestService(probeRepo, &mockStreamRepository{})
		handler := NewProbeHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/probes/abc123/metrics", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp metricsResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.TotalProbes != 2 {
			t.Errorf("TotalProbes = %d, want 2", resp.TotalProbes)
		}
		if resp.UptimeRatio != 1.0 {
			t.Errorf("UptimeRatio = %f, want 1.0", resp.UptimeRatio)
		}
	})

	t.Run("GET /probes/{infoHash}/metrics returns 404 for no data", func(t *testing.T) {
		probeRepo := &mockProbeRepository{}
		service := newProbeTestService(probeRepo, &mockStreamRepository{})
		handler := NewProbeHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/probes/unknown/metrics", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})
}

func TestProbeHTTPHandler_Quality(t *testing.T) {
	t.Run("GET /quality/{channelName} returns sorted scores", func(t *testing.T) {
		now := time.Now()
		s1, _ := stream.NewStream("hash1", "Channel1")
		s2, _ := stream.NewStream("hash2", "Channel1")

		streamRepo := &mockStreamRepository{
			findByChannelNameFunc: func(ctx context.Context, channelName string) ([]stream.Stream, error) {
				return []stream.Stream{s1, s2}, nil
			},
		}

		probeRepo := &mockProbeRepository{
			findByInfoHashSinceFunc: func(ctx context.Context, infoHash string, since time.Time) ([]probe.Result, error) {
				if infoHash == "hash1" {
					return []probe.Result{
						probe.ReconstructResult(infoHash, now, true, time.Second, 20, 200000, "dl", ""),
					}, nil
				}
				return []probe.Result{
					probe.ReconstructResult(infoHash, now, true, 5*time.Second, 3, 30000, "dl", ""),
				}, nil
			},
		}

		service := newProbeTestService(probeRepo, streamRepo)
		handler := NewProbeHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/quality/Channel1", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp []qualityResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 2 {
			t.Fatalf("expected 2 quality scores, got %d", len(resp))
		}
		if resp[0].InfoHash != "hash1" {
			t.Errorf("expected hash1 first (best), got %q", resp[0].InfoHash)
		}
		if resp[0].Score <= resp[1].Score {
			t.Errorf("first score (%f) should be > second score (%f)", resp[0].Score, resp[1].Score)
		}
	})

	t.Run("GET /quality/{channelName} returns empty for unknown channel", func(t *testing.T) {
		streamRepo := &mockStreamRepository{}
		probeRepo := &mockProbeRepository{}

		service := newProbeTestService(probeRepo, streamRepo)
		handler := NewProbeHTTPHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/quality/Unknown", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp []qualityResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 0 {
			t.Errorf("expected 0 results, got %d", len(resp))
		}
	})
}

func TestProbeHTTPHandler_MethodNotAllowed(t *testing.T) {
	probeRepo := &mockProbeRepository{}
	service := newProbeTestService(probeRepo, &mockStreamRepository{})
	handler := NewProbeHTTPHandler(service)

	req := httptest.NewRequest(http.MethodDelete, "/probes/abc123", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
}
