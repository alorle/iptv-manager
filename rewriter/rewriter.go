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
