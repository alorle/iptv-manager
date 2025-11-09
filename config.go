package main

import (
	"encoding/json"
	"fmt"
	"os"

	domain "github.com/alorle/iptv-manager/internal"
	"github.com/google/uuid"
)

// New format: flat streams array
type jsonStreamFlat struct {
	ID             string   `json:"id,omitempty"`
	GuideID        string   `json:"guideId"`
	AcestreamID    string   `json:"acestream_id"`
	Quality        string   `json:"quality"`
	Tags           []string `json:"tags"`
	NetworkCaching uint64   `json:"networkCaching"`
}

type jsonConfigFlat struct {
	Streams []jsonStreamFlat `json:"streams"`
}

// Old format: nested channels with streams (for backward compatibility)
type jsonStream struct {
	ID             string   `json:"id,omitempty"`
	AcestreamID    string   `json:"acestream_id"`
	Quality        string   `json:"quality"`
	Tags           []string `json:"tags"`
	NetworkCaching uint64   `json:"networkCaching"`
}

type jsonChannel struct {
	ID         string       `json:"id,omitempty"`
	Title      string       `json:"title"`
	GuideID    string       `json:"guideId"`
	Logo       string       `json:"logo"`
	GroupTitle string       `json:"groupTitle"`
	Streams    []jsonStream `json:"streams"`
}

type jsonConfig struct {
	Channels []jsonChannel `json:"channels"`
}

// loadStreams loads streams from JSON file (new flat format or old nested format)
func loadStreams(filePath string) ([]*domain.Stream, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %v", filePath, err)
	}

	// Try parsing as new flat format first
	var flatConfig jsonConfigFlat
	if err := json.Unmarshal(data, &flatConfig); err == nil && len(flatConfig.Streams) > 0 {
		// New format detected
		streams := make([]*domain.Stream, len(flatConfig.Streams))
		for i, s := range flatConfig.Streams {
			// Parse stream ID or generate new one
			var streamID uuid.UUID
			if s.ID != "" {
				streamID, err = uuid.Parse(s.ID)
				if err != nil {
					return nil, fmt.Errorf("error parsing stream ID %s: %v", s.ID, err)
				}
			} else {
				streamID = uuid.New()
			}

			streams[i] = &domain.Stream{
				ID:             streamID,
				GuideID:        s.GuideID,
				AcestreamID:    s.AcestreamID,
				Quality:        s.Quality,
				Tags:           s.Tags,
				NetworkCaching: s.NetworkCaching,
			}
		}
		return streams, nil
	}

	// Fall back to old nested format
	var nestedConfig jsonConfig
	if err := json.Unmarshal(data, &nestedConfig); err != nil {
		return nil, fmt.Errorf("error parsing %s (tried both formats): %v", filePath, err)
	}

	// Flatten nested format to streams
	var streams []*domain.Stream
	for _, c := range nestedConfig.Channels {
		for _, s := range c.Streams {
			// Parse stream ID or generate new one
			var streamID uuid.UUID
			if s.ID != "" {
				streamID, err = uuid.Parse(s.ID)
				if err != nil {
					return nil, fmt.Errorf("error parsing stream ID %s: %v", s.ID, err)
				}
			} else {
				streamID = uuid.New()
			}

			streams = append(streams, &domain.Stream{
				ID:             streamID,
				GuideID:        c.GuideID, // Use channel's GuideID
				AcestreamID:    s.AcestreamID,
				Quality:        s.Quality,
				Tags:           s.Tags,
				NetworkCaching: s.NetworkCaching,
			})
		}
	}

	return streams, nil
}

// DEPRECATED: Use loadStreams instead. Kept for backward compatibility during transition.
func loadChannels(filePath string) ([]*domain.Channel, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %v", filePath, err)
	}

	var config jsonConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing %s: %v", filePath, err)
	}

	channels := make([]*domain.Channel, len(config.Channels))
	for i, c := range config.Channels {
		streams := make([]*domain.Stream, len(c.Streams))
		for j, s := range c.Streams {
			// Parse stream ID or generate new one
			var streamID uuid.UUID
			if s.ID != "" {
				streamID, err = uuid.Parse(s.ID)
				if err != nil {
					return nil, fmt.Errorf("error parsing stream ID %s: %v", s.ID, err)
				}
			} else {
				streamID = uuid.New()
			}

			streams[j] = &domain.Stream{
				ID:             streamID,
				GuideID:        c.GuideID, // Use channel's GuideID
				AcestreamID:    s.AcestreamID,
				Quality:        s.Quality,
				Tags:           s.Tags,
				NetworkCaching: s.NetworkCaching,
			}
		}

		channels[i] = &domain.Channel{
			Title:      c.Title,
			GuideID:    c.GuideID,
			Logo:       c.Logo,
			GroupTitle: c.GroupTitle,
			Streams:    streams,
		}
	}

	return channels, nil
}
