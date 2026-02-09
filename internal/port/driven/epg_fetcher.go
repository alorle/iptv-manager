package driven

import (
	"context"

	"github.com/alorle/iptv-manager/internal/epg"
)

// EPGFetcher defines the interface for fetching EPG data from external sources.
// This is a driven port that will be implemented by concrete adapters (e.g., HTTP client, file reader).
type EPGFetcher interface {
	// FetchEPG retrieves EPG channel data from an external source.
	// Returns a slice of EPG channels or an error if the fetch operation fails.
	FetchEPG(ctx context.Context) ([]epg.Channel, error)
}
