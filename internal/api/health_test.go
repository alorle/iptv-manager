package api

import (
	"context"
	"testing"
	"time"
)

func TestGetHealth(t *testing.T) {
	server := NewServer()

	t.Run("returns healthy status", func(t *testing.T) {
		resp, err := server.GetHealth(context.Background(), GetHealthRequestObject{})

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		healthResp, ok := resp.(GetHealth200JSONResponse)
		if !ok {
			t.Fatalf("expected GetHealth200JSONResponse, got %T", resp)
		}

		if healthResp.Status != "healthy" {
			t.Errorf("expected status 'healthy', got '%s'", healthResp.Status)
		}

		if healthResp.Version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got '%s'", healthResp.Version)
		}

		if healthResp.Timestamp.IsZero() {
			t.Error("expected non-zero timestamp")
		}

		// Verify timestamp is recent (within 1 second)
		now := time.Now().UTC()
		diff := now.Sub(healthResp.Timestamp)
		if diff < 0 {
			diff = -diff
		}
		if diff > time.Second {
			t.Errorf("timestamp not recent: %v (diff: %v)", healthResp.Timestamp, diff)
		}
	})

	t.Run("timestamp is in UTC", func(t *testing.T) {
		resp, err := server.GetHealth(context.Background(), GetHealthRequestObject{})

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		healthResp, ok := resp.(GetHealth200JSONResponse)
		if !ok {
			t.Fatalf("expected GetHealth200JSONResponse, got %T", resp)
		}

		if healthResp.Timestamp.Location() != time.UTC {
			t.Errorf("expected UTC timezone, got %v", healthResp.Timestamp.Location())
		}
	})
}
