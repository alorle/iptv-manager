package api

import (
	"strings"

	"github.com/alorle/iptv-manager/domain"
)

// streamData holds raw parsed stream data before grouping
type streamData struct {
	AcestreamID string
	Name        string
	TvgID       string
	TvgName     string
	TvgLogo     string
	GroupTitle  string
	Source      string
	Enabled     bool
	HasOverride bool
}

// parseM3UStreams parses M3U content and extracts stream information
func parseM3UStreams(content []byte, source string) []streamData {
	lines := strings.Split(string(content), "\n")
	var streams []streamData

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Look for EXTINF metadata lines
		if strings.HasPrefix(line, "#EXTINF:") {
			// Next line should be the URL
			if i+1 < len(lines) {
				metadata := line
				url := strings.TrimSpace(lines[i+1])
				i++ // Skip the URL line in the next iteration

				// Only process acestream URLs
				if strings.HasPrefix(url, "acestream://") {
					aceID := strings.TrimPrefix(url, "acestream://")
					aceID = strings.TrimSpace(aceID)

					// Validate acestream ID (40 hex characters)
					if domain.IsValidAcestreamID(aceID) {
						// Extract display name
						name := domain.ExtractDisplayName(metadata)

						// Extract metadata attributes
						tvgID, tvgLogo, groupTitle := domain.ExtractMetadata(metadata)

						streams = append(streams, streamData{
							AcestreamID: aceID,
							Name:        name,
							TvgID:       tvgID,
							TvgLogo:     tvgLogo,
							GroupTitle:  groupTitle,
							Source:      source,
							Enabled:     true, // Default, will be updated with overrides
							HasOverride: false,
						})
					}
				}
			}
		}
	}

	return streams
}
