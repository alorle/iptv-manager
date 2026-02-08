package driven

import (
	"context"
	"io"
)

// AceStreamEngine defines the interface for interacting with the AceStream Engine HTTP API.
// This is a driven port that will be implemented by concrete adapters (e.g., HTTP client).
type AceStreamEngine interface {
	// StartStream initiates a stream for the given infohash with a unique PID.
	// Returns the stream URL endpoint and any error encountered.
	StartStream(ctx context.Context, infoHash, pid string) (streamURL string, err error)

	// GetStats retrieves statistics for an active stream identified by its PID.
	// Returns stream statistics and any error encountered.
	GetStats(ctx context.Context, pid string) (stats StreamStats, err error)

	// StopStream terminates the stream identified by its PID.
	// Returns any error encountered during the stop operation.
	StopStream(ctx context.Context, pid string) error

	// StreamContent establishes a streaming connection and copies the stream data
	// to the provided writer. This method blocks until the stream ends or an error occurs.
	StreamContent(ctx context.Context, streamURL string, dst io.Writer) error
}

// StreamStats contains statistics about an active AceStream.
type StreamStats struct {
	PID        string
	InfoHash   string
	Status     string
	Peers      int
	SpeedDown  int64 // bytes per second
	SpeedUp    int64 // bytes per second
	Downloaded int64 // total bytes downloaded
	Uploaded   int64 // total bytes uploaded
}
