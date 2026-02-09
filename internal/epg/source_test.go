package epg_test

import (
	"errors"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/internal/epg"
)

func TestNewSource(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name            string
		url             string
		lastFetchedTime time.Time
		wantURL         string
		wantLastFetched time.Time
		wantError       error
	}{
		{
			name:            "valid source",
			url:             "https://example.com/epg.xml",
			lastFetchedTime: now,
			wantURL:         "https://example.com/epg.xml",
			wantLastFetched: now,
			wantError:       nil,
		},
		{
			name:            "valid source with trimmed whitespace",
			url:             "  https://example.com/epg.xml  ",
			lastFetchedTime: now,
			wantURL:         "https://example.com/epg.xml",
			wantLastFetched: now,
			wantError:       nil,
		},
		{
			name:            "valid source with zero time",
			url:             "https://example.com/epg.xml",
			lastFetchedTime: time.Time{},
			wantURL:         "https://example.com/epg.xml",
			wantLastFetched: time.Time{},
			wantError:       nil,
		},
		{
			name:            "empty url",
			url:             "",
			lastFetchedTime: now,
			wantError:       epg.ErrEmptyURL,
		},
		{
			name:            "whitespace only url",
			url:             "   ",
			lastFetchedTime: now,
			wantError:       epg.ErrEmptyURL,
		},
		{
			name:            "tabs in url",
			url:             "\t\t",
			lastFetchedTime: now,
			wantError:       epg.ErrEmptyURL,
		},
		{
			name:            "newlines in url",
			url:             "\n\n",
			lastFetchedTime: now,
			wantError:       epg.ErrEmptyURL,
		},
		{
			name:            "mixed whitespace in url",
			url:             " \t\n ",
			lastFetchedTime: now,
			wantError:       epg.ErrEmptyURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := epg.NewSource(tt.url, tt.lastFetchedTime)

			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Errorf("NewSource() error = %v, wantError %v", err, tt.wantError)
				}
				return
			}

			if err != nil {
				t.Errorf("NewSource() unexpected error = %v", err)
				return
			}

			if got := source.URL(); got != tt.wantURL {
				t.Errorf("Source.URL() = %q, want %q", got, tt.wantURL)
			}

			if got := source.LastFetchedTime(); !got.Equal(tt.wantLastFetched) {
				t.Errorf("Source.LastFetchedTime() = %v, want %v", got, tt.wantLastFetched)
			}
		})
	}
}

func TestSourceGetters(t *testing.T) {
	now := time.Now()
	source, err := epg.NewSource("https://example.com/epg.xml", now)
	if err != nil {
		t.Fatalf("NewSource() unexpected error = %v", err)
	}

	if got := source.URL(); got != "https://example.com/epg.xml" {
		t.Errorf("URL() = %q, want %q", got, "https://example.com/epg.xml")
	}

	if got := source.LastFetchedTime(); !got.Equal(now) {
		t.Errorf("LastFetchedTime() = %v, want %v", got, now)
	}
}

func TestSourceWithLastFetchedTime(t *testing.T) {
	initialTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedTime := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	source, err := epg.NewSource("https://example.com/epg.xml", initialTime)
	if err != nil {
		t.Fatalf("NewSource() unexpected error = %v", err)
	}

	if got := source.LastFetchedTime(); !got.Equal(initialTime) {
		t.Errorf("initial LastFetchedTime() = %v, want %v", got, initialTime)
	}

	// Create new source with updated time (immutable pattern)
	updatedSource := source.WithLastFetchedTime(updatedTime)

	// Original should be unchanged
	if got := source.LastFetchedTime(); !got.Equal(initialTime) {
		t.Errorf("original LastFetchedTime() after update = %v, want %v", got, initialTime)
	}

	// Updated source should have new time
	if got := updatedSource.LastFetchedTime(); !got.Equal(updatedTime) {
		t.Errorf("updated LastFetchedTime() = %v, want %v", got, updatedTime)
	}

	// URL should remain the same
	if got := updatedSource.URL(); got != "https://example.com/epg.xml" {
		t.Errorf("updated URL() = %q, want %q", got, "https://example.com/epg.xml")
	}
}

func TestSourceImmutability(t *testing.T) {
	time1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	time3 := time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC)

	source1, err := epg.NewSource("https://example.com/epg.xml", time1)
	if err != nil {
		t.Fatalf("NewSource() unexpected error = %v", err)
	}

	source2 := source1.WithLastFetchedTime(time2)
	source3 := source1.WithLastFetchedTime(time3)

	// All sources should maintain their own state
	if got := source1.LastFetchedTime(); !got.Equal(time1) {
		t.Errorf("source1.LastFetchedTime() = %v, want %v", got, time1)
	}

	if got := source2.LastFetchedTime(); !got.Equal(time2) {
		t.Errorf("source2.LastFetchedTime() = %v, want %v", got, time2)
	}

	if got := source3.LastFetchedTime(); !got.Equal(time3) {
		t.Errorf("source3.LastFetchedTime() = %v, want %v", got, time3)
	}
}
