package domain

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

type Channel struct {
	ID         uuid.UUID
	Title      string
	GuideID    string
	Logo       string
	GroupTitle string
	Streams    []*Stream
}

type Stream struct {
	ID             uuid.UUID
	ChannelID      uuid.UUID
	AcestreamID    string
	Quality        string
	Tags           []string
	NetworkCaching uint64
}

func (s *Stream) FullTitle(channelTitle string) string {
	w := strings.Builder{}
	w.WriteString(channelTitle)
	if s.Quality != "" {
		w.WriteString(fmt.Sprintf(" [%s]", s.Quality))
	}
	if len(s.Tags) > 0 {
		w.WriteString(fmt.Sprintf(" [%s]", strings.Join(s.Tags, " ")))
	}
	return w.String()
}

func (s *Stream) GetStreamURL(baseURL *url.URL) string {
	q := baseURL.Query()
	q.Set("id", s.AcestreamID)
	q.Set("network-caching", fmt.Sprintf("%d", s.NetworkCaching))
	baseURL.RawQuery = q.Encode()
	return baseURL.String()
}
