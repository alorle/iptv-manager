package rewriter

import (
	"strings"
)

// extractContentID extracts the acestream content ID from a URL query parameter.
// Returns the first 40 hex characters found after "?id=".
func extractContentID(line string) string {
	idIdx := strings.Index(line, "?id=")
	if idIdx == -1 {
		return ""
	}

	afterID := line[idIdx+4:]
	endIdx := 0
	for endIdx < len(afterID) && endIdx < 40 {
		c := afterID[endIdx]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			break
		}
		endIdx++
	}
	return afterID[:endIdx]
}

// extractTranscodeAudio extracts the transcode_audio parameter value from a URL.
// Returns empty string if the parameter is not found.
func extractTranscodeAudio(line string) string {
	transcodeIdx := strings.Index(line, "transcode_audio=")
	if transcodeIdx == -1 {
		return ""
	}

	afterTranscode := line[transcodeIdx+16:]
	transcodeEndIdx := strings.IndexAny(afterTranscode, "&\n\r")
	if transcodeEndIdx == -1 {
		return afterTranscode
	}
	return afterTranscode[:transcodeEndIdx]
}

// buildStreamURL constructs a stream URL with the given content ID, optional transcode parameter, and base URL.
func buildStreamURL(baseURL, contentID, transcodeAudio string) string {
	path := "/stream?id=" + contentID
	if transcodeAudio != "" {
		path += "&transcode_audio=" + transcodeAudio
	}

	if baseURL == "" {
		return path
	}
	return baseURL + path
}

// rewriteAcestreamURL converts an acestream:// URL to the internal /stream format.
func rewriteAcestreamURL(line, baseURL string) string {
	streamID := strings.TrimPrefix(line, "acestream://")
	streamID = strings.TrimSpace(streamID)
	return buildStreamURL(baseURL, streamID, "")
}

// rewriteAlreadyRewrittenURL normalizes an already-rewritten stream URL to the standard format.
// Preserves transcode_audio parameter if present.
func rewriteAlreadyRewrittenURL(line, baseURL string) string {
	contentID := extractContentID(line)
	transcodeAudio := extractTranscodeAudio(line)
	return buildStreamURL(baseURL, contentID, transcodeAudio)
}

// isRewrittenStreamURL checks if a line is an already-rewritten stream URL.
func isRewrittenStreamURL(line string) bool {
	return strings.Contains(line, "?id=") &&
		(strings.Contains(line, "/stream") || strings.Contains(line, "/ace/getstream"))
}

// RewriteM3U processes M3U content line by line and rewrites acestream:// URLs
// to internal server URLs in the format /stream?id={content_id}
// Preserves transcode_audio parameter if present in original URL
func (r *Rewriter) RewriteM3U(content []byte, baseURL string) []byte {
	lines := strings.Split(string(content), "\n")
	var result strings.Builder

	for i, line := range lines {
		// Add newline for all lines except the first
		if i > 0 {
			result.WriteString("\n")
		}

		// Route line processing based on type
		switch {
		case strings.HasPrefix(line, "acestream://"):
			result.WriteString(rewriteAcestreamURL(line, baseURL))
		case isRewrittenStreamURL(line):
			result.WriteString(rewriteAlreadyRewrittenURL(line, baseURL))
		case strings.HasPrefix(line, "#EXTINF:"):
			result.WriteString(RemoveLogoMetadata(line))
		default:
			result.WriteString(line)
		}
	}

	return []byte(result.String())
}
