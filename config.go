package main

import (
	"encoding/json"
	"fmt"
	"os"

	domain "github.com/alorle/iptv-manager/internal"
	"github.com/google/uuid"
)

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
		// Parse channel ID or generate new one
		var channelID uuid.UUID
		if c.ID != "" {
			channelID, err = uuid.Parse(c.ID)
			if err != nil {
				return nil, fmt.Errorf("error parsing channel ID %s: %v", c.ID, err)
			}
		} else {
			channelID = uuid.New()
		}

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
				ChannelID:      channelID,
				AcestreamID:    s.AcestreamID,
				Quality:        s.Quality,
				Tags:           s.Tags,
				NetworkCaching: s.NetworkCaching,
			}
		}

		channels[i] = &domain.Channel{
			ID:         channelID,
			Title:      c.Title,
			GuideID:    c.GuideID,
			Logo:       c.Logo,
			GroupTitle: c.GroupTitle,
			Streams:    streams,
		}
	}

	return channels, nil
}
