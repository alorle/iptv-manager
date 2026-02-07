package rewriter

import (
	"sort"
	"strings"

	"github.com/alorle/iptv-manager/domain"
)

// SortStreamsByName sorts streams alphabetically by display name (case-insensitive).
// Header lines (lines without URLs) are kept at the top in their original order.
// Stream entries (metadata + URL pairs) are sorted by the display name extracted from EXTINF.
func SortStreamsByName(content []byte) []byte {
	lines := strings.Split(string(content), "\n")
	var headers []Stream
	var streams []Stream

	// Parse M3U into header lines and stream entries
	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Look for EXTINF metadata lines
		if strings.HasPrefix(line, "#EXTINF:") {
			// Next line should be the URL
			if i+1 < len(lines) {
				metadata := line
				url := lines[i+1]
				i++ // Skip the URL line in the next iteration

				streams = append(streams, Stream{
					Metadata: metadata,
					URL:      url,
					AceID:    "", // Not needed for sorting
				})
			}
		} else if line != "" {
			// Preserve header lines (like #EXTM3U) and other non-URL lines
			headers = append(headers, Stream{
				Metadata: line,
				URL:      "",
				AceID:    "",
			})
		}
	}

	// Sort streams by group-title first, then by display name (case-insensitive)
	// Channels without group-title are placed at the end
	sort.SliceStable(streams, func(i, j int) bool {
		groupI := strings.ToLower(domain.ExtractGroupTitle(streams[i].Metadata))
		groupJ := strings.ToLower(domain.ExtractGroupTitle(streams[j].Metadata))
		if groupI != groupJ {
			// Empty group-title goes to the end
			if groupI == "" {
				return false
			}
			if groupJ == "" {
				return true
			}
			return groupI < groupJ
		}
		nameI := strings.ToLower(domain.ExtractDisplayName(streams[i].Metadata))
		nameJ := strings.ToLower(domain.ExtractDisplayName(streams[j].Metadata))
		return nameI < nameJ
	})

	// Rebuild M3U content with headers first, then sorted streams
	var result strings.Builder

	// Write headers first
	for i, header := range headers {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(header.Metadata)
	}

	// Write sorted streams
	for _, stream := range streams {
		if len(headers) > 0 || result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(stream.Metadata)
		result.WriteString("\n")
		result.WriteString(stream.URL)
	}

	return []byte(result.String())
}
