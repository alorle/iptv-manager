package driven

import (
	"context"
	"testing"
	"time"
)

func TestEPGXMLFetcher_FetchEPG_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("fetch from actual EPG source", func(t *testing.T) {
		fetcher := NewEPGXMLFetcher("", nil) // Use default URL

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		channels, err := fetcher.FetchEPG(ctx)
		if err != nil {
			t.Fatalf("failed to fetch EPG: %v", err)
		}

		if len(channels) == 0 {
			t.Error("expected at least one channel, got none")
		}

		// Verify first channel has required fields
		if len(channels) > 0 {
			ch := channels[0]
			if ch.ID() == "" {
				t.Error("expected channel to have ID")
			}
			if ch.Name() == "" {
				t.Error("expected channel to have name")
			}
			if ch.EPGID() == "" {
				t.Error("expected channel to have EPG ID")
			}

			t.Logf("Successfully fetched %d channels", len(channels))
			t.Logf("First channel: ID=%q, Name=%q, Logo=%q",
				ch.ID(), ch.Name(), ch.Logo())
		}
	})
}
