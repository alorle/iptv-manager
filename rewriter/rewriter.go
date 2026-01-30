package rewriter

import (
	"fmt"
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

			// Rewrite to player-compatible format
			rewrittenURL := fmt.Sprintf("%s?id=%s", r.playerBaseURL, streamID)
			result.WriteString(rewrittenURL)
		} else {
			// Preserve all other lines unchanged
			result.WriteString(line)
		}
	}

	return []byte(result.String())
}
