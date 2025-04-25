package handlers

import (
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

	for _, channel := range channels {
		encoder.AddChannel(&m3u.Channel{
			SeqId:    1,
			Title:    channel.FullTitle(),
			URI:      channel.GetStreamURL(h.acestreamURL),
			Duration: -1,
			TVGTags: &m3u.TVGTags{
				ID:         channel.GuideID,
				Name:       channel.FullTitle(),
				GroupTitle: channel.GroupTitle,
			},
		})
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/x-mpegURL")
	if err := encoder.Encode(w); err != nil {
		http.Error(w, "Error writing playlist", http.StatusInternalServerError)
		return
	}
}
