package main

import (
	"encoding/json"
	"fmt"
	"os"

	domain "github.com/alorle/iptv-manager/internal"
	"github.com/google/uuid"
)

type jsonStream struct {
	AcestreamID    string   `json:"acestream_id"`
	Quality        string   `json:"quality"`
	Tags           []string `json:"tags"`
	NetworkCaching uint64   `json:"networkCaching"`
}

type jsonChannel struct {
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
		channelID := uuid.New()

		streams := make([]*domain.Stream, len(c.Streams))
		for j, s := range c.Streams {
			streams[j] = &domain.Stream{
				ID:             uuid.New(),
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
