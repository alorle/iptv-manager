package driven

import "context"

// AcestreamSource defines the interface for fetching Acestream hash lists from external sources.
// This is a driven port that will be implemented by concrete adapters (e.g., HTTP client, file reader).
type AcestreamSource interface {
	// FetchHashes retrieves Acestream hashes from an external source.
	// The source parameter identifies the source to fetch from (e.g., 'new-era', 'elcano').
	// Returns a map of channel names to their corresponding Acestream hashes.
	// Multiple hashes per channel are supported for redundancy.
	FetchHashes(ctx context.Context, source string) (map[string][]string, error)
}
