package domain

import (
	"regexp"
	"strings"
)

var (
	tvgIDRegex      = regexp.MustCompile(`tvg-id="([^"]*)"`)
	tvgLogoRegex    = regexp.MustCompile(`tvg-logo="([^"]*)"`)
	groupTitleRegex = regexp.MustCompile(`group-title="([^"]*)"`)
)

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

// ExtractGroupTitle extracts the group-title attribute from an EXTINF line.
// Returns empty string if the line is not an EXTINF line or has no group-title.
func ExtractGroupTitle(extinf string) string {
	if !strings.HasPrefix(extinf, "#EXTINF:") {
		return ""
	}

	matches := groupTitleRegex.FindStringSubmatch(extinf)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// ExtractTvgID extracts the tvg-id attribute from an EXTINF line.
// Returns empty string if the line is not an EXTINF line or has no tvg-id.
func ExtractTvgID(extinf string) string {
	if !strings.HasPrefix(extinf, "#EXTINF:") {
		return ""
	}

	matches := tvgIDRegex.FindStringSubmatch(extinf)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// ExtractTvgLogo extracts the tvg-logo attribute from an EXTINF line.
// Returns empty string if the line is not an EXTINF line or has no tvg-logo.
func ExtractTvgLogo(extinf string) string {
	if !strings.HasPrefix(extinf, "#EXTINF:") {
		return ""
	}

	matches := tvgLogoRegex.FindStringSubmatch(extinf)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// ExtractMetadata extracts all metadata attributes from an EXTINF line
// Returns tvg-id, tvg-logo, and group-title
func ExtractMetadata(extinf string) (tvgID, tvgLogo, groupTitle string) {
	tvgID = ExtractTvgID(extinf)
	tvgLogo = ExtractTvgLogo(extinf)
	groupTitle = ExtractGroupTitle(extinf)
	return
}
