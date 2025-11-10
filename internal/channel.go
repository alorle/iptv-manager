package domain

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

// Channel is now a derived view (not stored directly)
// It represents streams grouped by GuideID with metadata from EPG
type Channel struct {
	Title      string    // From EPG (via GuideID)
	GuideID    string    // Group key
	Logo       string    // From EPG
	GroupTitle string    // From EPG
	Streams    []*Stream // Streams with matching GuideID
}

// Stream is the primary entity (stored directly)
type Stream struct {
	GuideID        string
	AcestreamID    string
	Quality        string
	Tags           []string
	NetworkCaching uint64
	ID             uuid.UUID
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
