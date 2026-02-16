package driven

import (
	"context"
	"time"

	"github.com/alorle/iptv-manager/internal/probe"
)

// ProbeRepository defines the interface for probe result persistence.
// This is a driven port implemented by concrete adapters (e.g., BoltDB).
type ProbeRepository interface {
	// Save persists a probe result.
	Save(ctx context.Context, r probe.Result) error

	// FindByInfoHash retrieves all probe results for a given stream,
	// ordered by timestamp descending (most recent first).
	FindByInfoHash(ctx context.Context, infoHash string) ([]probe.Result, error)

	// FindByInfoHashSince retrieves probe results for a stream since the
	// given time, ordered by timestamp descending. This supports the
	// rolling window requirement.
	FindByInfoHashSince(ctx context.Context, infoHash string, since time.Time) ([]probe.Result, error)

	// DeleteBefore removes all probe results older than the given time.
	// This is used for retention/cleanup.
	DeleteBefore(ctx context.Context, before time.Time) error
}
