package epg

import (
	"strings"
	"time"
)

// Source represents an EPG data source in the domain.
// It is a value object containing metadata about where EPG data is fetched from.
type Source struct {
	url             string
	lastFetchedTime time.Time
}

// NewSource creates a new EPG Source with the given URL and last fetched time.
// It validates that the URL is not empty and trims whitespace.
// Returns ErrEmptyURL if the URL is empty or contains only whitespace.
func NewSource(url string, lastFetchedTime time.Time) (Source, error) {
	trimmedURL := strings.TrimSpace(url)
	if trimmedURL == "" {
		return Source{}, ErrEmptyURL
	}

	return Source{
		url:             trimmedURL,
		lastFetchedTime: lastFetchedTime,
	}, nil
}

// URL returns the EPG source URL.
func (s Source) URL() string {
	return s.url
}

// LastFetchedTime returns the timestamp of when the EPG data was last fetched.
func (s Source) LastFetchedTime() time.Time {
	return s.lastFetchedTime
}

// WithLastFetchedTime returns a new Source with an updated last fetched time.
// This maintains immutability of the value object.
func (s Source) WithLastFetchedTime(t time.Time) Source {
	return Source{
		url:             s.url,
		lastFetchedTime: t,
	}
}
