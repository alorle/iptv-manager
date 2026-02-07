package rewriter

import (
	"strings"
)

// DeduplicateStreams removes duplicate streams based on acestream ID.
// For acestream URLs, keeps the first occurrence of each unique ID.
// Non-acestream URLs are always preserved (no deduplication applied).
func DeduplicateStreams(content []byte) []byte {
	lines := strings.Split(string(content), "\n")
	var streams []Stream
	seen := make(map[string]bool)

	// Parse M3U into stream entries
	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Look for EXTINF metadata lines
		if strings.HasPrefix(line, "#EXTINF:") {
			// Next line should be the URL
			if i+1 < len(lines) {
				metadata := line
				url := lines[i+1]
				i++ // Skip the URL line in the next iteration

				// Extract acestream ID if this is an acestream URL
				var aceID string
				if strings.HasPrefix(url, "acestream://") {
					aceID = strings.TrimPrefix(url, "acestream://")
					aceID = strings.TrimSpace(aceID)
				}

				// For acestream URLs, deduplicate by ID
				// For non-acestream URLs, always keep
				if aceID != "" {
					if !seen[aceID] {
						seen[aceID] = true
						streams = append(streams, Stream{
							Metadata: metadata,
							URL:      url,
							AceID:    aceID,
						})
					}
					// Skip duplicate acestream entries
				} else {
					// Always preserve non-acestream URLs
					streams = append(streams, Stream{
						Metadata: metadata,
						URL:      url,
						AceID:    "",
					})
				}
			}
		} else if !strings.HasPrefix(line, "acestream://") && line != "" {
			// Preserve header lines (like #EXTM3U) and other non-URL lines
			streams = append(streams, Stream{
				Metadata: line,
				URL:      "",
				AceID:    "",
			})
		}
	}

	// Rebuild M3U content
	var result strings.Builder
	for i, stream := range streams {
		if i > 0 {
			result.WriteString("\n")
		}

		if stream.URL != "" {
			// This is a complete stream entry with metadata + URL
			result.WriteString(stream.Metadata)
			result.WriteString("\n")
			result.WriteString(stream.URL)
		} else {
			// This is a header or other single line
			result.WriteString(stream.Metadata)
		}
	}

	return []byte(result.String())
}

// ExtractAcestreamIDs extracts all unique acestream IDs from M3U content.
// This is useful for identifying which channels are present in the playlist
// before applying overrides or deduplication.
func ExtractAcestreamIDs(m3u []byte) []string {
	lines := strings.Split(string(m3u), "\n")
	seen := make(map[string]bool)
	var ids []string

	for _, line := range lines {
		// Look for acestream:// URLs
		if strings.HasPrefix(line, "acestream://") {
			aceID := strings.TrimPrefix(line, "acestream://")
			aceID = strings.TrimSpace(aceID)

			// Only add unique IDs
			if aceID != "" && !seen[aceID] {
				seen[aceID] = true
				ids = append(ids, aceID)
			}
		}
	}

	return ids
}
