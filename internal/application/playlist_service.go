package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/alorle/iptv-manager/internal/port/driven"
)

// PlaylistService provides use cases for playlist generation.
// It depends only on port interfaces.
type PlaylistService struct {
	streamRepo driven.StreamRepository
}

// NewPlaylistService creates a new PlaylistService with the given stream repository.
func NewPlaylistService(streamRepo driven.StreamRepository) *PlaylistService {
	return &PlaylistService{
		streamRepo: streamRepo,
	}
}

// GenerateM3U generates an M3U playlist with all available streams.
// The host parameter is used to build the proxy URL for each stream.
// Returns a playlist with only the #EXTM3U header if no streams are found.
func (p *PlaylistService) GenerateM3U(ctx context.Context, host string) (string, error) {
	streams, err := p.streamRepo.FindAll(ctx)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString("#EXTM3U\n")

	for _, s := range streams {
		// Format: #EXTINF:-1 tvg-id="NombreCanal",NombreCanal - infohash
		fmt.Fprintf(&builder, "#EXTINF:-1 tvg-id=\"%s\",%s - %s\n",
			s.ChannelName(),
			s.ChannelName(),
			s.InfoHash())

		// Format: http://{host}/ace/getstream?id=infohash
		fmt.Fprintf(&builder, "http://%s/ace/getstream?id=%s\n",
			host,
			s.InfoHash())
	}

	return builder.String(), nil
}
