package domain

import (
	"fmt"
	"net/url"
	"strings"
)

type Channel struct {
	ID             string
	Title          string
	GuideID        string
	GroupTitle     string
	Quality        string
	Tags           []string
	StreamID       string
	NetworkCaching uint64
}

func (c *Channel) FullTitle() string {
	w := strings.Builder{}
	w.WriteString(c.Title)
	if c.Quality != "" {
		w.WriteString(fmt.Sprintf(" [%s]", c.Quality))
	}
	if len(c.Tags) > 0 {
		w.WriteString(fmt.Sprintf(" [%s]", strings.Join(c.Tags, " ")))
	}
	return w.String()
}

func (c *Channel) GetStreamURL(baseURL *url.URL) string {
	q := baseURL.Query()
	q.Set("id", c.StreamID)
	q.Set("network-caching", fmt.Sprintf("%d", c.NetworkCaching))
	baseURL.RawQuery = q.Encode()
	return baseURL.String()
}
