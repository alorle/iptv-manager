package rewriter

import "github.com/alorle/iptv-manager/overrides"

// Interface defines the contract for M3U playlist URL rewriting
type Interface interface {
	// RewriteM3U processes M3U content line by line and rewrites acestream:// URLs
	// to internal server URLs in the format /stream?id={content_id}
	RewriteM3U(content []byte, baseURL string) []byte
}

// PlaylistProcessor defines the contract for the full M3U processing pipeline
type PlaylistProcessor interface {
	// ApplyOverrides applies channel overrides to M3U content
	// Filters out disabled channels and replaces metadata according to configured overrides
	ApplyOverrides(m3u []byte, manager overrides.Interface) []byte

	// DeduplicateStreams removes duplicate streams based on acestream ID
	DeduplicateStreams(content []byte) []byte

	// SortStreamsByName sorts streams alphabetically by group-title then display name
	SortStreamsByName(content []byte) []byte

	// ExtractAcestreamIDs extracts all unique acestream IDs from M3U content
	ExtractAcestreamIDs(m3u []byte) []string

	// RemoveLogoMetadata removes tvg-logo attribute from EXTINF line while preserving other metadata
	RemoveLogoMetadata(line string) string
}
