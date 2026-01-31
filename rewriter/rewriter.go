package rewriter

import (
	"fmt"
	"regexp"
	"strings"
)

// Rewriter handles URL rewriting for M3U playlists
type Rewriter struct {
	playerBaseURL string
}

// New creates a new Rewriter with the specified player base URL
func New(playerBaseURL string) *Rewriter {
	return &Rewriter{
		playerBaseURL: playerBaseURL,
	}
}

var logoRegex = regexp.MustCompile(`\s*tvg-logo="[^"]*"`)

// RemoveLogoMetadata removes tvg-logo attribute from EXTINF line while preserving other metadata
func RemoveLogoMetadata(line string) string {
	if !strings.HasPrefix(line, "#EXTINF:") {
		return line
	}

	// Remove tvg-logo="..." attribute
	result := logoRegex.ReplaceAllString(line, "")

	// Clean up any double spaces that might have been created
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}

	// Clean up space before comma (when logo was last attribute before display name)
	result = strings.ReplaceAll(result, " ,", ",")

	return result
}

// Stream represents a single M3U playlist entry
type Stream struct {
	Metadata string
	URL      string
	AceID    string
}

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

// RewriteM3U processes M3U content line by line and rewrites acestream:// URLs
// to player-compatible format
func (r *Rewriter) RewriteM3U(content []byte) []byte {
	lines := strings.Split(string(content), "\n")
	var result strings.Builder

	for i, line := range lines {
		// Add newline for all lines except the first
		if i > 0 {
			result.WriteString("\n")
		}

		// Check if line starts with acestream://
		if strings.HasPrefix(line, "acestream://") {
			// Extract stream ID from acestream:// URL
			streamID := strings.TrimPrefix(line, "acestream://")
			// Remove any trailing whitespace or carriage return
			streamID = strings.TrimSpace(streamID)

			// Rewrite to player-compatible format with network-caching parameter
			rewrittenURL := fmt.Sprintf("%s?id=%s&network-caching=1000", r.playerBaseURL, streamID)
			result.WriteString(rewrittenURL)
		} else if strings.HasPrefix(line, "#EXTINF:") {
			// Remove logo metadata from EXTINF lines
			result.WriteString(RemoveLogoMetadata(line))
		} else {
			// Preserve all other lines unchanged
			result.WriteString(line)
		}
	}

	return []byte(result.String())
}
