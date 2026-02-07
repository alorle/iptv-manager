package api

import (
	"strings"

	"github.com/alorle/iptv-manager/overrides"
)

// isValidTvgID checks if a tvg-id is valid (not empty or whitespace-only)
func isValidTvgID(tvgID string) bool {
	return strings.TrimSpace(tvgID) != ""
}

// groupStreamsByTvgID groups streams by their tvg-id
// Streams with empty/whitespace tvg-id are returned as individual channels
func groupStreamsByTvgID(streams []streamData) []Channel {
	var channels []Channel

	// Group streams by valid tvg-id
	grouped := make(map[string][]streamData)
	var ungrouped []streamData

	for _, stream := range streams {
		if isValidTvgID(stream.TvgID) {
			grouped[stream.TvgID] = append(grouped[stream.TvgID], stream)
		} else {
			ungrouped = append(ungrouped, stream)
		}
	}

	// Create channels from grouped streams
	for tvgID, streamList := range grouped {
		// Use first stream's metadata for the channel
		first := streamList[0]

		var streamObjs []Stream
		for _, s := range streamList {
			streamObjs = append(streamObjs, Stream{
				ContentID:   s.ContentID,
				Name:        s.Name,
				TvgName:     s.TvgName,
				Source:      s.Source,
				Enabled:     s.Enabled,
				HasOverride: s.HasOverride,
			})
		}

		channels = append(channels, Channel{
			Name:       first.Name,
			TvgID:      tvgID,
			TvgLogo:    first.TvgLogo,
			GroupTitle: first.GroupTitle,
			Streams:    streamObjs,
		})
	}

	// Create individual channels for ungrouped streams
	for _, stream := range ungrouped {
		channels = append(channels, Channel{
			Name:       stream.Name,
			TvgID:      stream.TvgID,
			TvgLogo:    stream.TvgLogo,
			GroupTitle: stream.GroupTitle,
			Streams: []Stream{
				{
					ContentID:   stream.ContentID,
					Name:        stream.Name,
					TvgName:     stream.TvgName,
					Source:      stream.Source,
					Enabled:     stream.Enabled,
					HasOverride: stream.HasOverride,
				},
			},
		})
	}

	return channels
}

// applyOverrides applies override settings to channels and marks which ones have overrides
func applyOverrides(channels []Channel, overridesMgr overrides.Interface) []Channel {
	allOverrides := overridesMgr.List()

	for i := range channels {
		ch := &channels[i]

		// Apply overrides to each stream in the channel
		for j := range ch.Streams {
			stream := &ch.Streams[j]
			if override, exists := allOverrides[stream.ContentID]; exists {
				stream.HasOverride = true

				// Apply overrides if they are set (not nil)
				if override.Enabled != nil {
					stream.Enabled = *override.Enabled
				}
				if override.TvgName != nil {
					stream.TvgName = *override.TvgName
				}
			}
		}

		// Apply channel-level overrides from the first stream if it has overrides
		if len(ch.Streams) > 0 {
			firstStream := ch.Streams[0]
			if override, exists := allOverrides[firstStream.ContentID]; exists {
				if override.TvgID != nil {
					ch.TvgID = *override.TvgID
				}
				if override.TvgLogo != nil {
					ch.TvgLogo = *override.TvgLogo
				}
				if override.GroupTitle != nil {
					ch.GroupTitle = *override.GroupTitle
				}
			}
		}
	}

	return channels
}

// filterChannels filters channels by name and/or group (case-insensitive substring match)
func filterChannels(channels []Channel, nameFilter, groupFilter string) []Channel {
	var result []Channel

	nameLower := strings.ToLower(nameFilter)
	groupLower := strings.ToLower(groupFilter)

	for _, ch := range channels {
		nameMatches := nameFilter == "" || strings.Contains(strings.ToLower(ch.Name), nameLower)
		groupMatches := groupFilter == "" || strings.Contains(strings.ToLower(ch.GroupTitle), groupLower)

		if nameMatches && groupMatches {
			result = append(result, ch)
		}
	}

	return result
}
