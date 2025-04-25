package main

import (
	"encoding/json"
	"fmt"
	"os"

	domain "github.com/alorle/iptv-manager/internal"
	"github.com/google/uuid"
)

type jsonChannel struct {
	Title          string   `json:"title"`
	GuideID        string   `json:"guideId"`
	GroupTitle     string   `json:"groupTitle"`
	Quality        string   `json:"quality"`
	Tags           []string `json:"tags"`
	StreamID       string   `json:"acestream_id"`
	NetworkCaching uint64   `json:"networkCaching"`
}

type jsonConfig struct {
	Channels []jsonChannel `json:"streams"`
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
		channels[i] = &domain.Channel{
			ID:             uuid.New(),
			Title:          c.Title,
			GuideID:        c.GuideID,
			GroupTitle:     c.GroupTitle,
			Quality:        c.Quality,
			Tags:           c.Tags,
			StreamID:       c.StreamID,
			NetworkCaching: c.NetworkCaching,
		}
	}

	return channels, nil
}
