package rewriter

import (
	"strings"

	"github.com/alorle/iptv-manager/overrides"
)

// ApplyOverrides applies channel overrides to M3U content.
// It filters out disabled channels and replaces metadata according to configured overrides.
// This function should be called BEFORE deduplication and sorting in the pipeline.
func ApplyOverrides(m3u []byte, manager overrides.Interface) []byte {
	if manager == nil {
		return m3u
	}

	lines := strings.Split(string(m3u), "\n")
	var result strings.Builder

	// Process M3U line by line
	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Look for EXTINF metadata lines
		if strings.HasPrefix(line, "#EXTINF:") {
			// Next line should be the URL
			if i+1 < len(lines) {
				metadata := line
				url := lines[i+1]

				// Extract acestream ID from the URL
				var aceID string
				if strings.HasPrefix(url, "acestream://") {
					aceID = strings.TrimPrefix(url, "acestream://")
					aceID = strings.TrimSpace(aceID)
				}

				// If we have an acestream ID, check for overrides
				if aceID != "" {
					override := manager.Get(aceID)

					// Filter out disabled channels
					if override != nil && override.Enabled != nil && !*override.Enabled {
						// Skip this channel (both metadata and URL lines)
						i++
						continue
					}

					// Apply metadata overrides if they exist
					if override != nil {
						metadata = applyMetadataOverrides(metadata, override)
					}
				}

				// Write the (possibly modified) metadata and URL
				if result.Len() > 0 {
					result.WriteString("\n")
				}
				result.WriteString(metadata)
				result.WriteString("\n")
				result.WriteString(url)
				i++ // Skip the URL line in the next iteration
			}
		} else if line != "" {
			// Preserve header lines (like #EXTM3U) and other non-URL lines
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(line)
		}
	}

	return []byte(result.String())
}
