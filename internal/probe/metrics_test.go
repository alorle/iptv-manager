package probe

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestNewMetrics_EmptyResults(t *testing.T) {
	_, err := NewMetrics("abc123", nil)
	if !errors.Is(err, ErrNoProbeData) {
		t.Errorf("expected ErrNoProbeData, got %v", err)
	}

	_, err = NewMetrics("abc123", []Result{})
	if !errors.Is(err, ErrNoProbeData) {
		t.Errorf("expected ErrNoProbeData, got %v", err)
	}
}

func TestNewMetrics_AllSuccessful(t *testing.T) {
	now := time.Now()
	results := []Result{
		ReconstructResult("abc123", now, true, 2*time.Second, 10, 100000, "dl", ""),
		ReconstructResult("abc123", now.Add(-30*time.Minute), true, 3*time.Second, 20, 200000, "dl", ""),
		ReconstructResult("abc123", now.Add(-60*time.Minute), true, 1*time.Second, 15, 150000, "dl", ""),
	}

	m, err := NewMetrics("abc123", results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.InfoHash() != "abc123" {
		t.Errorf("InfoHash() = %q, want %q", m.InfoHash(), "abc123")
	}
	if m.TotalProbes() != 3 {
		t.Errorf("TotalProbes() = %d, want %d", m.TotalProbes(), 3)
	}
	if m.SuccessfulProbes() != 3 {
		t.Errorf("SuccessfulProbes() = %d, want %d", m.SuccessfulProbes(), 3)
	}
	if m.UptimeRatio() != 1.0 {
		t.Errorf("UptimeRatio() = %f, want %f", m.UptimeRatio(), 1.0)
	}
	if m.FailureRate() != 0.0 {
		t.Errorf("FailureRate() = %f, want %f", m.FailureRate(), 0.0)
	}

	// AvgPeerCount: (10+20+15)/3 = 15.0
	if m.AvgPeerCount() != 15.0 {
		t.Errorf("AvgPeerCount() = %f, want %f", m.AvgPeerCount(), 15.0)
	}

	// AvgDownloadSpeed: (100000+200000+150000)/3 = 150000
	if m.AvgDownloadSpeed() != 150000.0 {
		t.Errorf("AvgDownloadSpeed() = %f, want %f", m.AvgDownloadSpeed(), 150000.0)
	}

	// AvgStartupLatency: (2000+3000+1000)/3 = 2000 ms
	if m.AvgStartupLatency() != 2000.0 {
		t.Errorf("AvgStartupLatency() = %f, want %f", m.AvgStartupLatency(), 2000.0)
	}

	// SpeedStdDev for [100000, 200000, 150000], mean=150000
	// diffs: -50000, 50000, 0 → squares: 2.5e9, 2.5e9, 0 → mean: 5e9/3 → sqrt ≈ 40824.83
	expectedStdDev := math.Sqrt(5e9 / 3)
	if math.Abs(m.SpeedStdDev()-expectedStdDev) > 0.01 {
		t.Errorf("SpeedStdDev() = %f, want %f", m.SpeedStdDev(), expectedStdDev)
	}
}

func TestNewMetrics_AllFailed(t *testing.T) {
	now := time.Now()
	results := []Result{
		ReconstructResult("abc123", now, false, 0, 0, 0, "", "timeout"),
		ReconstructResult("abc123", now.Add(-30*time.Minute), false, 0, 0, 0, "", "refused"),
	}

	m, err := NewMetrics("abc123", results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.TotalProbes() != 2 {
		t.Errorf("TotalProbes() = %d, want %d", m.TotalProbes(), 2)
	}
	if m.SuccessfulProbes() != 0 {
		t.Errorf("SuccessfulProbes() = %d, want %d", m.SuccessfulProbes(), 0)
	}
	if m.UptimeRatio() != 0.0 {
		t.Errorf("UptimeRatio() = %f, want %f", m.UptimeRatio(), 0.0)
	}
	if m.FailureRate() != 1.0 {
		t.Errorf("FailureRate() = %f, want %f", m.FailureRate(), 1.0)
	}
	if m.AvgPeerCount() != 0.0 {
		t.Errorf("AvgPeerCount() = %f, want %f", m.AvgPeerCount(), 0.0)
	}
	if m.AvgDownloadSpeed() != 0.0 {
		t.Errorf("AvgDownloadSpeed() = %f, want %f", m.AvgDownloadSpeed(), 0.0)
	}
	if m.SpeedStdDev() != 0.0 {
		t.Errorf("SpeedStdDev() = %f, want %f", m.SpeedStdDev(), 0.0)
	}
}

func TestNewMetrics_MixedResults(t *testing.T) {
	now := time.Now()
	results := []Result{
		ReconstructResult("abc123", now, true, 2*time.Second, 10, 100000, "dl", ""),
		ReconstructResult("abc123", now.Add(-30*time.Minute), false, 0, 0, 0, "", "timeout"),
		ReconstructResult("abc123", now.Add(-60*time.Minute), true, 4*time.Second, 20, 200000, "dl", ""),
		ReconstructResult("abc123", now.Add(-90*time.Minute), false, 0, 0, 0, "", "refused"),
	}

	m, err := NewMetrics("abc123", results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.TotalProbes() != 4 {
		t.Errorf("TotalProbes() = %d, want %d", m.TotalProbes(), 4)
	}
	if m.SuccessfulProbes() != 2 {
		t.Errorf("SuccessfulProbes() = %d, want %d", m.SuccessfulProbes(), 2)
	}
	if m.UptimeRatio() != 0.5 {
		t.Errorf("UptimeRatio() = %f, want %f", m.UptimeRatio(), 0.5)
	}
	if m.FailureRate() != 0.5 {
		t.Errorf("FailureRate() = %f, want %f", m.FailureRate(), 0.5)
	}

	// Only successful probes contribute to averages
	// AvgPeerCount: (10+20)/2 = 15
	if m.AvgPeerCount() != 15.0 {
		t.Errorf("AvgPeerCount() = %f, want %f", m.AvgPeerCount(), 15.0)
	}

	// AvgDownloadSpeed: (100000+200000)/2 = 150000
	if m.AvgDownloadSpeed() != 150000.0 {
		t.Errorf("AvgDownloadSpeed() = %f, want %f", m.AvgDownloadSpeed(), 150000.0)
	}

	// AvgStartupLatency: (2000+4000)/2 = 3000 ms
	if m.AvgStartupLatency() != 3000.0 {
		t.Errorf("AvgStartupLatency() = %f, want %f", m.AvgStartupLatency(), 3000.0)
	}
}
