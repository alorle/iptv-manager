package rewriter

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Rewriter handles URL rewriting for M3U playlists
type Rewriter struct {
}

// New creates a new Rewriter with the specified stream base URL
func New() *Rewriter {
	return &Rewriter{}
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

// ExtractDisplayName extracts the display name from an EXTINF line.
// Display name is the text after the comma in "#EXTINF:-1 tvg-id="..." tvg-name="...",Channel Name"
// Returns empty string if the line is not an EXTINF line or has no comma.
func ExtractDisplayName(extinf string) string {
	if !strings.HasPrefix(extinf, "#EXTINF:") {
		return ""
	}

	// Find the last comma, as that separates metadata from display name
	commaIdx := strings.LastIndex(extinf, ",")
	if commaIdx == -1 {
		return ""
	}

	// Extract everything after the comma and trim whitespace
	displayName := strings.TrimSpace(extinf[commaIdx+1:])
	return displayName
}

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

	// Sort streams by display name (case-insensitive)
	sort.SliceStable(streams, func(i, j int) bool {
		nameI := strings.ToLower(ExtractDisplayName(streams[i].Metadata))
		nameJ := strings.ToLower(ExtractDisplayName(streams[j].Metadata))
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
// to internal server URLs in the format /stream?id={content_id}
// Preserves transcode_audio parameter if present in original URL
func (r *Rewriter) RewriteM3U(content []byte, baseUrl string) []byte {
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

			// Build the rewritten URL
			var rewrittenURL string
			if baseUrl == "" {
				// Use relative URL if no base URL provided
				rewrittenURL = fmt.Sprintf("/stream?id=%s", streamID)
			} else {
				// Use absolute URL with base URL
				rewrittenURL = fmt.Sprintf("%s/stream?id=%s", baseUrl, streamID)
			}
			result.WriteString(rewrittenURL)
		} else if strings.Contains(line, "?id=") && (strings.Contains(line, "/stream") || strings.Contains(line, "/ace/getstream")) {
			// This is already a rewritten URL, check for transcode_audio parameter
			if strings.Contains(line, "transcode_audio=") {
				// Extract the content ID and transcode_audio parameter
				var contentID, transcodeAudio string

				// Find the id parameter
				idIdx := strings.Index(line, "?id=")
				if idIdx != -1 {
					afterID := line[idIdx+4:]
					// Extract content ID (40 hex characters)
					endIdx := 0
					for endIdx < len(afterID) && endIdx < 40 {
						c := afterID[endIdx]
						if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
							break
						}
						endIdx++
					}
					contentID = afterID[:endIdx]

					// Find transcode_audio parameter
					transcodeIdx := strings.Index(line, "transcode_audio=")
					if transcodeIdx != -1 {
						afterTranscode := line[transcodeIdx+16:]
						transcodeEndIdx := strings.IndexAny(afterTranscode, "&\n\r")
						if transcodeEndIdx == -1 {
							transcodeAudio = afterTranscode
						} else {
							transcodeAudio = afterTranscode[:transcodeEndIdx]
						}
					}
				}

				// Build the new URL with transcode_audio preserved
				var rewrittenURL string
				if baseUrl == "" {
					rewrittenURL = fmt.Sprintf("/stream?id=%s&transcode_audio=%s", contentID, transcodeAudio)
				} else {
					rewrittenURL = fmt.Sprintf("%s/stream?id=%s&transcode_audio=%s", baseUrl, contentID, transcodeAudio)
				}
				result.WriteString(rewrittenURL)
			} else {
				// No transcode_audio, just rewrite the URL format
				// Extract content ID
				var contentID string
				idIdx := strings.Index(line, "?id=")
				if idIdx != -1 {
					afterID := line[idIdx+4:]
					endIdx := 0
					for endIdx < len(afterID) && endIdx < 40 {
						c := afterID[endIdx]
						if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
							break
						}
						endIdx++
					}
					contentID = afterID[:endIdx]
				}

				var rewrittenURL string
				if baseUrl == "" {
					rewrittenURL = fmt.Sprintf("/stream?id=%s", contentID)
				} else {
					rewrittenURL = fmt.Sprintf("%s/stream?id=%s", baseUrl, contentID)
				}
				result.WriteString(rewrittenURL)
			}
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
