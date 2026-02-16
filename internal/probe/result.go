package probe

import (
	"strings"
	"time"
)

// Result represents a single health-check probe of an AceStream.
// It is an immutable value object.
type Result struct {
	infoHash       string
	timestamp      time.Time
	available      bool
	startupLatency time.Duration
	peerCount      int
	downloadSpeed  int64
	status         string
	errorMessage   string
}

// NewResult creates a new probe result with validation.
func NewResult(
	infoHash string,
	timestamp time.Time,
	available bool,
	startupLatency time.Duration,
	peerCount int,
	downloadSpeed int64,
	status string,
	errorMessage string,
) (Result, error) {
	infoHash = strings.TrimSpace(infoHash)
	if infoHash == "" {
		return Result{}, ErrEmptyInfoHash
	}
	if timestamp.IsZero() {
		return Result{}, ErrInvalidTimestamp
	}
	return Result{
		infoHash:       infoHash,
		timestamp:      timestamp,
		available:      available,
		startupLatency: startupLatency,
		peerCount:      peerCount,
		downloadSpeed:  downloadSpeed,
		status:         status,
		errorMessage:   errorMessage,
	}, nil
}

// ReconstructResult rebuilds a Result from persisted state.
// Intended for repository adapters only — bypasses validation.
func ReconstructResult(
	infoHash string,
	timestamp time.Time,
	available bool,
	startupLatency time.Duration,
	peerCount int,
	downloadSpeed int64,
	status string,
	errorMessage string,
) Result {
	return Result{
		infoHash:       infoHash,
		timestamp:      timestamp,
		available:      available,
		startupLatency: startupLatency,
		peerCount:      peerCount,
		downloadSpeed:  downloadSpeed,
		status:         status,
		errorMessage:   errorMessage,
	}
}

func (r Result) InfoHash() string              { return r.infoHash }
func (r Result) Timestamp() time.Time          { return r.timestamp }
func (r Result) Available() bool               { return r.available }
func (r Result) StartupLatency() time.Duration { return r.startupLatency }
func (r Result) PeerCount() int                { return r.peerCount }
func (r Result) DownloadSpeed() int64          { return r.downloadSpeed }
func (r Result) Status() string                { return r.status }
func (r Result) ErrorMessage() string          { return r.errorMessage }
