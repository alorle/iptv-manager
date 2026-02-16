package probe

import (
	"math"
	"testing"
	"time"
)

func TestComputeQualityScore_PerfectStream(t *testing.T) {
	now := time.Now()
	results := []Result{
		ReconstructResult("abc123", now, true, 500*time.Millisecond, 50, 1000000, "dl", ""),
		ReconstructResult("abc123", now.Add(-30*time.Minute), true, 500*time.Millisecond, 50, 1000000, "dl", ""),
	}

	m, err := NewMetrics("abc123", results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	score := ComputeQualityScore(m, 1000000, 50)

	// Perfect stream: uptime=1.0, speed=1.0, peers=1.0, stability=1.0 (stddev=0), latency=0.95
	// = 0.50 + 0.20 + 0.15 + 0.10 + 0.05*0.95 = 0.9975
	if math.Abs(score-0.9975) > 0.001 {
		t.Errorf("score = %f, want ~0.9975", score)
	}
}

func TestComputeQualityScore_DeadStream(t *testing.T) {
	now := time.Now()
	results := []Result{
		ReconstructResult("abc123", now, false, 0, 0, 0, "", "timeout"),
		ReconstructResult("abc123", now.Add(-30*time.Minute), false, 0, 0, 0, "", "timeout"),
	}

	m, err := NewMetrics("abc123", results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	score := ComputeQualityScore(m, 1000000, 50)
	if score != 0.0 {
		t.Errorf("score = %f, want 0.0", score)
	}
}

func TestComputeQualityScore_ZeroMaxValues(t *testing.T) {
	now := time.Now()
	results := []Result{
		ReconstructResult("abc123", now, true, time.Second, 10, 100000, "dl", ""),
	}

	m, err := NewMetrics("abc123", results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// When maxSpeed and maxPeers are 0, normalized values are 0
	score := ComputeQualityScore(m, 0, 0)

	// uptime=0.50, speed=0, peers=0, stability=0.10 (stddev=0, avg>0 so stability=1.0), latency=0.05*0.9=0.045
	expected := 0.50 + 0.10 + 0.045
	if math.Abs(score-expected) > 0.001 {
		t.Errorf("score = %f, want %f", score, expected)
	}
}

func TestComputeQualityScore_HighLatency(t *testing.T) {
	now := time.Now()
	results := []Result{
		ReconstructResult("abc123", now, true, 10*time.Second, 10, 100000, "dl", ""),
	}

	m, err := NewMetrics("abc123", results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	score := ComputeQualityScore(m, 100000, 10)

	// latency = 10000ms, maxLatency = 10000 → latencyScore = 1.0 - 1.0 = 0.0
	// uptime=0.50, speed=0.20, peers=0.15, stability=0.10, latency=0.0
	expected := 0.50 + 0.20 + 0.15 + 0.10
	if math.Abs(score-expected) > 0.001 {
		t.Errorf("score = %f, want %f", score, expected)
	}
}

func TestComputeQualityScore_PartialUptime(t *testing.T) {
	now := time.Now()
	results := []Result{
		ReconstructResult("abc123", now, true, time.Second, 10, 200000, "dl", ""),
		ReconstructResult("abc123", now.Add(-30*time.Minute), false, 0, 0, 0, "", "timeout"),
	}

	m, err := NewMetrics("abc123", results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	score := ComputeQualityScore(m, 200000, 10)

	// uptime=0.50*0.5=0.25
	if score < 0.25 || score > 0.75 {
		t.Errorf("score = %f, expected between 0.25 and 0.75", score)
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		name string
		v    float64
		lo   float64
		hi   float64
		want float64
	}{
		{"within range", 0.5, 0, 1, 0.5},
		{"below min", -0.5, 0, 1, 0},
		{"above max", 1.5, 0, 1, 1},
		{"at min", 0, 0, 1, 0},
		{"at max", 1, 0, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clamp(tt.v, tt.lo, tt.hi)
			if got != tt.want {
				t.Errorf("clamp(%f, %f, %f) = %f, want %f", tt.v, tt.lo, tt.hi, got, tt.want)
			}
		})
	}
}
