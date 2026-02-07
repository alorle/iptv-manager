package playlist

import (
	"log"
	"strings"

	"github.com/alorle/iptv-manager/overrides"
	"github.com/alorle/iptv-manager/rewriter"
)

// Source represents a fetched playlist source
type Source struct {
	Name      string
	Content   []byte
	Err       error
	FromCache bool
	Stale     bool
}

// logCacheStatus logs the cache status for a fetched source
func logCacheStatus(sourceName string, fromCache, stale bool) {
	if !fromCache {
		log.Printf("Using fresh content for %s in unified playlist", sourceName)
		return
	}

	if stale {
		log.Printf("Using stale cache for %s in unified playlist", sourceName)
	} else {
		log.Printf("Using fresh cache for %s in unified playlist", sourceName)
	}
}

// stripM3UHeader removes the #EXTM3U header from playlist content if present
func stripM3UHeader(content []byte) string {
	str := string(content)
	if strings.HasPrefix(str, "#EXTM3U") {
		str = strings.TrimPrefix(str, "#EXTM3U")
		str = strings.TrimLeft(str, "\n")
	}
	return str
}

// MergeSources merges multiple playlist sources into a single M3U
func MergeSources(sources []Source) string {
	var merged strings.Builder
	merged.WriteString("#EXTM3U\n")

	for _, source := range sources {
		if source.Err != nil {
			log.Printf("Skipping %s source in unified playlist: %v", source.Name, source.Err)
			continue
		}

		logCacheStatus(source.Name, source.FromCache, source.Stale)

		// Add newline separator if we're appending to existing content
		if merged.Len() > len("#EXTM3U\n") {
			merged.WriteString("\n")
		}
		merged.WriteString(stripM3UHeader(source.Content))
	}

	return merged.String()
}

// CleanOrphanedOverrides removes overrides for channels no longer in the playlists
func CleanOrphanedOverrides(overridesMgr overrides.Interface, sources []Source) {
	// Only clean if we have fresh data from at least one source
	hasFreshData := false
	for _, source := range sources {
		if source.Err == nil && !source.Stale {
			hasFreshData = true
			break
		}
	}

	if !hasFreshData {
		log.Printf("Skipping orphan cleanup - using only stale cache data")
		return
	}

	// Collect all valid acestream IDs from successful fetches
	var validIDs []string
	for _, source := range sources {
		if source.Err == nil {
			ids := rewriter.ExtractAcestreamIDs(source.Content)
			validIDs = append(validIDs, ids...)
		}
	}

	// Clean orphaned overrides
	deletedCount, err := overridesMgr.CleanOrphans(validIDs)
	if err != nil {
		log.Printf("WARNING: Failed to clean orphaned overrides: %v", err)
	} else if deletedCount > 0 {
		log.Printf("Cleaned up %d orphaned override(s)", deletedCount)
	}
}

// Process applies the full M3U processing pipeline: overrides, dedup, sort, rewrite
func Process(overridesMgr overrides.Interface, rw rewriter.Interface, content string, baseURL string) []byte {
	contentBytes := []byte(content)

	// Apply channel overrides BEFORE deduplication and sorting
	overridden := rewriter.ApplyOverrides(contentBytes, overridesMgr)

	// Apply deduplication by acestream ID
	deduplicated := rewriter.DeduplicateStreams(overridden)

	// Apply alphabetical sorting by display name
	sorted := rewriter.SortStreamsByName(deduplicated)

	// Rewrite acestream:// URLs and remove logos
	return rw.RewriteM3U(sorted, baseURL)
}
