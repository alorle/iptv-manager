package handlers

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/alorle/iptv-manager/internal/m3u"
	"github.com/alorle/iptv-manager/internal/usecase"
)

type playlistHandler struct {
	channelUseCase usecase.GetChannelUseCase
	acestreamURL   *url.URL
	epgURL         string
}

var _ http.Handler = (*playlistHandler)(nil)

func NewPlaylistHandler(channelUseCase usecase.GetChannelUseCase, acestreamURL *url.URL, epgURL string) *playlistHandler {
	return &playlistHandler{
		channelUseCase: channelUseCase,
		acestreamURL:   acestreamURL,
		epgURL:         epgURL,
	}
}

func (h *playlistHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	guideUrls := []string{}
	if h.epgURL != "" {
		guideUrls = append(guideUrls, h.epgURL)
	}

	encoder := m3u.NewEncoder(guideUrls)

	channels, err := h.channelUseCase.GetChannels()
	if err != nil {
		http.Error(w, "Error retrieving channels", http.StatusInternalServerError)
		return
	}

	// Track title occurrences for incremental numbering
	titleCounts := make(map[string]int)

	// Flatten channels → streams for M3U generation
	for _, channel := range channels {
		for _, stream := range channel.Streams {
			// Increment counter for this channel title
			titleCounts[channel.Title]++
			count := titleCounts[channel.Title]

			// Generate display title with incremental numbering if needed
			displayTitle := channel.Title
			if count > 1 {
				displayTitle = fmt.Sprintf("%s (#%d)", channel.Title, count)
			}

			// Add quality and tags to the title
			fullTitle := stream.FullTitle(displayTitle)

			encoder.AddChannel(&m3u.Channel{
				SeqId:    1,
				Title:    fullTitle,
				URI:      stream.GetStreamURL(h.acestreamURL),
				Duration: -1,
				TVGTags: &m3u.TVGTags{
					ID:         channel.GuideID,
					Name:       fullTitle,
					GroupTitle: channel.GroupTitle,
				},
			})
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/x-mpegURL")
	if err := encoder.Encode(w); err != nil {
		http.Error(w, "Error writing playlist", http.StatusInternalServerError)
		return
	}
}
