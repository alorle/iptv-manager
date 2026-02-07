package api

import (
	"github.com/alorle/iptv-manager/fetcher"
	"github.com/alorle/iptv-manager/overrides"
)

// Stream represents a single stream within a channel
type Stream struct {
	AcestreamID string `json:"acestream_id"`
	Name        string `json:"name"`
	TvgName     string `json:"tvg_name"`
	Source      string `json:"source"` // "elcano" or "newera"
	Enabled     bool   `json:"enabled"`
	HasOverride bool   `json:"has_override"`
}

// Channel represents a channel with its metadata and array of streams
type Channel struct {
	Name       string   `json:"name"`
	TvgID      string   `json:"tvg_id"`
	TvgLogo    string   `json:"tvg_logo"`
	GroupTitle string   `json:"group_title"`
	Streams    []Stream `json:"streams"`
}

// ChannelsHandler handles the GET /api/channels endpoint
type ChannelsHandler struct {
	fetcher      fetcher.Interface
	overridesMgr overrides.Interface
	playlistURLs []string
}

// NewChannelsHandler creates a new handler for the channels API
func NewChannelsHandler(fetch fetcher.Interface, overridesMgr overrides.Interface, playlistURLs ...string) *ChannelsHandler {
	return &ChannelsHandler{
		fetcher:      fetch,
		overridesMgr: overridesMgr,
		playlistURLs: playlistURLs,
	}
}

// UpdateChannelRequest represents the request body for updating a channel's metadata
type UpdateChannelRequest struct {
	Enabled    *bool   `json:"enabled,omitempty"`
	TvgID      *string `json:"tvg_id,omitempty"`
	TvgName    *string `json:"tvg_name,omitempty"`
	TvgLogo    *string `json:"tvg_logo,omitempty"`
	GroupTitle *string `json:"group_title,omitempty"`
}
