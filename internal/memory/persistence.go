package memory

import (
	"encoding/json"
	"fmt"
	"os"

	domain "github.com/alorle/iptv-manager/internal"
)

// New flat format
type jsonStreamFlat struct {
	ID             string   `json:"id"`
	GuideID        string   `json:"guideId"`
	AcestreamID    string   `json:"acestream_id"`
	Quality        string   `json:"quality"`
	Tags           []string `json:"tags"`
	NetworkCaching uint64   `json:"networkCaching"`
}

type jsonConfigFlat struct {
	Streams []jsonStreamFlat `json:"streams"`
}

// Old nested format (for backward compatibility)
type jsonStream struct {
	ID             string   `json:"id"`
	AcestreamID    string   `json:"acestream_id"`
	Quality        string   `json:"quality"`
	Tags           []string `json:"tags"`
	NetworkCaching uint64   `json:"networkCaching"`
}

type jsonChannel struct {
	ID         string       `json:"id"`
	Title      string       `json:"title"`
	GuideID    string       `json:"guideId"`
	Logo       string       `json:"logo"`
	GroupTitle string       `json:"groupTitle"`
	Streams    []jsonStream `json:"streams"`
}

type jsonConfig struct {
	Channels []jsonChannel `json:"channels"`
}

// DEPRECATED: saveChannelsToFile saves channel views in the old nested format.
// This is kept for backward compatibility but should not be used for new code.
// Use saveStreamsToFile instead.
func saveChannelsToFile(filePath string, channels []*domain.Channel) error {
	config := jsonConfig{
		Channels: make([]jsonChannel, len(channels)),
	}

	for i, channel := range channels {
		jsonCh := jsonChannel{
			ID:         "", // Channel no longer has ID in new model
			Title:      channel.Title,
			GuideID:    channel.GuideID,
			Logo:       channel.Logo,
			GroupTitle: channel.GroupTitle,
			Streams:    make([]jsonStream, len(channel.Streams)),
		}

		for j, stream := range channel.Streams {
			jsonCh.Streams[j] = jsonStream{
				ID:             stream.ID.String(),
				AcestreamID:    stream.AcestreamID,
				Quality:        stream.Quality,
				Tags:           stream.Tags,
				NetworkCaching: stream.NetworkCaching,
			}
		}

		config.Channels[i] = jsonCh
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling channels: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing to file %s: %w", filePath, err)
	}

	return nil
}

func saveStreamsToFile(filePath string, streams []*domain.Stream) error {
	config := jsonConfigFlat{
		Streams: make([]jsonStreamFlat, len(streams)),
	}

	for i, stream := range streams {
		config.Streams[i] = jsonStreamFlat{
			ID:             stream.ID.String(),
			GuideID:        stream.GuideID,
			AcestreamID:    stream.AcestreamID,
			Quality:        stream.Quality,
			Tags:           stream.Tags,
			NetworkCaching: stream.NetworkCaching,
		}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling streams: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing to file %s: %w", filePath, err)
	}

	return nil
}
