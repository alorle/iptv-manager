package probe

import (
	"errors"
	"testing"
	"time"
)

func TestNewResult(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		infoHash       string
		timestamp      time.Time
		available      bool
		startupLatency time.Duration
		peerCount      int
		downloadSpeed  int64
		status         string
		errorMessage   string
		wantError      error
	}{
		{
			name:           "valid successful probe",
			infoHash:       "abc123",
			timestamp:      now,
			available:      true,
			startupLatency: 2 * time.Second,
			peerCount:      10,
			downloadSpeed:  500000,
			status:         "dl",
			wantError:      nil,
		},
		{
			name:         "valid failed probe",
			infoHash:     "abc123",
			timestamp:    now,
			available:    false,
			errorMessage: "connection refused",
			wantError:    nil,
		},
		{
			name:      "empty infohash",
			infoHash:  "",
			timestamp: now,
			wantError: ErrEmptyInfoHash,
		},
		{
			name:      "whitespace-only infohash",
			infoHash:  "   ",
			timestamp: now,
			wantError: ErrEmptyInfoHash,
		},
		{
			name:      "zero timestamp",
			infoHash:  "abc123",
			timestamp: time.Time{},
			wantError: ErrInvalidTimestamp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewResult(
				tt.infoHash, tt.timestamp, tt.available,
				tt.startupLatency, tt.peerCount, tt.downloadSpeed,
				tt.status, tt.errorMessage,
			)

			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Errorf("expected error %v, got %v", tt.wantError, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.InfoHash() != tt.infoHash {
				t.Errorf("InfoHash() = %q, want %q", result.InfoHash(), tt.infoHash)
			}
			if !result.Timestamp().Equal(tt.timestamp) {
				t.Errorf("Timestamp() = %v, want %v", result.Timestamp(), tt.timestamp)
			}
			if result.Available() != tt.available {
				t.Errorf("Available() = %v, want %v", result.Available(), tt.available)
			}
			if result.StartupLatency() != tt.startupLatency {
				t.Errorf("StartupLatency() = %v, want %v", result.StartupLatency(), tt.startupLatency)
			}
			if result.PeerCount() != tt.peerCount {
				t.Errorf("PeerCount() = %d, want %d", result.PeerCount(), tt.peerCount)
			}
			if result.DownloadSpeed() != tt.downloadSpeed {
				t.Errorf("DownloadSpeed() = %d, want %d", result.DownloadSpeed(), tt.downloadSpeed)
			}
			if result.Status() != tt.status {
				t.Errorf("Status() = %q, want %q", result.Status(), tt.status)
			}
			if result.ErrorMessage() != tt.errorMessage {
				t.Errorf("ErrorMessage() = %q, want %q", result.ErrorMessage(), tt.errorMessage)
			}
		})
	}
}

func TestNewResult_TrimsWhitespace(t *testing.T) {
	result, err := NewResult("  abc123  ", time.Now(), true, 0, 0, 0, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.InfoHash() != "abc123" {
		t.Errorf("InfoHash() = %q, want %q", result.InfoHash(), "abc123")
	}
}

func TestReconstructResult(t *testing.T) {
	now := time.Now()
	result := ReconstructResult("abc123", now, true, 2*time.Second, 10, 500000, "dl", "")

	if result.InfoHash() != "abc123" {
		t.Errorf("InfoHash() = %q, want %q", result.InfoHash(), "abc123")
	}
	if !result.Timestamp().Equal(now) {
		t.Errorf("Timestamp() = %v, want %v", result.Timestamp(), now)
	}
	if !result.Available() {
		t.Error("Available() = false, want true")
	}
	if result.StartupLatency() != 2*time.Second {
		t.Errorf("StartupLatency() = %v, want %v", result.StartupLatency(), 2*time.Second)
	}
	if result.PeerCount() != 10 {
		t.Errorf("PeerCount() = %d, want %d", result.PeerCount(), 10)
	}
	if result.DownloadSpeed() != 500000 {
		t.Errorf("DownloadSpeed() = %d, want %d", result.DownloadSpeed(), 500000)
	}
	if result.Status() != "dl" {
		t.Errorf("Status() = %q, want %q", result.Status(), "dl")
	}
}

func TestDomainErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "ErrEmptyInfoHash",
			err:  ErrEmptyInfoHash,
			msg:  "probe infohash cannot be empty",
		},
		{
			name: "ErrInvalidTimestamp",
			err:  ErrInvalidTimestamp,
			msg:  "probe timestamp must not be zero",
		},
		{
			name: "ErrNoProbeData",
			err:  ErrNoProbeData,
			msg:  "no probe data available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.msg)
			}
		})
	}
}
