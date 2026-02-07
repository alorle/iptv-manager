package rewriter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/alorle/iptv-manager/overrides"
)

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

// applyMetadataOverrides replaces metadata attributes in an EXTINF line according to the override configuration.
// It handles tvg-id, tvg-name, tvg-logo, group-title, and the display name.
func applyMetadataOverrides(extinf string, override *overrides.ChannelOverride) string {
	if override == nil {
		return extinf
	}

	// Start with the original line
	result := extinf

	// Override tvg-id
	if override.TvgID != nil {
		result = replaceOrAddAttribute(result, "tvg-id", *override.TvgID)
	}

	// Override tvg-name (also use it for display name if set)
	if override.TvgName != nil {
		result = replaceOrAddAttribute(result, "tvg-name", *override.TvgName)
		// Also replace the display name (text after last comma)
		result = replaceDisplayName(result, *override.TvgName)
	}

	// Override tvg-logo
	if override.TvgLogo != nil {
		result = replaceOrAddAttribute(result, "tvg-logo", *override.TvgLogo)
	}

	// Override group-title
	if override.GroupTitle != nil {
		result = replaceOrAddAttribute(result, "group-title", *override.GroupTitle)
	}

	return result
}

// replaceOrAddAttribute replaces or adds an attribute in an EXTINF line.
// If the attribute exists, it's replaced with the new value.
// If it doesn't exist, it's added before the display name (before the last comma).
func replaceOrAddAttribute(extinf, attrName, attrValue string) string {
	if !strings.HasPrefix(extinf, "#EXTINF:") {
		return extinf
	}

	attrPattern := regexp.MustCompile(fmt.Sprintf(`\s*%s="[^"]*"`, regexp.QuoteMeta(attrName)))

	// Check if attribute already exists
	if attrPattern.MatchString(extinf) {
		// Replace existing attribute
		if attrValue == "" {
			// Remove attribute if value is empty
			result := attrPattern.ReplaceAllString(extinf, "")
			// Clean up any double spaces
			for strings.Contains(result, "  ") {
				result = strings.ReplaceAll(result, "  ", " ")
			}
			return result
		}
		// Replace with new value
		replacement := fmt.Sprintf(` %s="%s"`, attrName, attrValue)
		return attrPattern.ReplaceAllString(extinf, replacement)
	}

	// Attribute doesn't exist, add it before the display name
	// Find the last comma, as that separates metadata from display name
	commaIdx := strings.LastIndex(extinf, ",")
	if commaIdx == -1 {
		// No comma found, can't add attribute safely
		return extinf
	}

	// Insert attribute before the comma
	if attrValue != "" {
		beforeComma := extinf[:commaIdx]
		afterComma := extinf[commaIdx:]
		return fmt.Sprintf(`%s %s="%s"%s`, beforeComma, attrName, attrValue, afterComma)
	}

	return extinf
}

// replaceDisplayName replaces the display name (text after last comma) in an EXTINF line.
func replaceDisplayName(extinf, newName string) string {
	if !strings.HasPrefix(extinf, "#EXTINF:") {
		return extinf
	}

	// Find the last comma, as that separates metadata from display name
	commaIdx := strings.LastIndex(extinf, ",")
	if commaIdx == -1 {
		// No comma found, can't replace
		return extinf
	}

	// Replace everything after the comma with the new name
	return extinf[:commaIdx+1] + newName
}
