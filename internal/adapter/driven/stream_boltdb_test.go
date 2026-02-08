package driven

import (
	"context"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/alorle/iptv-manager/internal/stream"
)

func TestNewStreamBoltDBRepository(t *testing.T) {
	t.Run("creates repository and bucket successfully", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if repo == nil {
			t.Fatal("expected non-nil repository")
		}

		// Verify bucket was created
		err = db.View(func(tx *bbolt.Tx) error {
			bucket := tx.Bucket([]byte(streamsBucket))
			if bucket == nil {
				t.Error("expected streams bucket to exist")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed to verify bucket: %v", err)
		}
	})

	t.Run("returns error for nil database", func(t *testing.T) {
		repo, err := NewStreamBoltDBRepository(nil)
		if err == nil {
			t.Fatal("expected error for nil database")
		}
		if repo != nil {
			t.Error("expected nil repository")
		}
	})
}

func TestStreamBoltDBRepository_Save(t *testing.T) {
	t.Run("saves a new stream successfully", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		s, err := stream.NewStream("abc123", "HBO")
		if err != nil {
			t.Fatalf("failed to create stream: %v", err)
		}

		ctx := context.Background()
		err = repo.Save(ctx, s)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify the stream was saved
		found, err := repo.FindByInfoHash(ctx, "abc123")
		if err != nil {
			t.Fatalf("failed to find saved stream: %v", err)
		}
		if found.InfoHash() != "abc123" {
			t.Errorf("expected infohash 'abc123', got %q", found.InfoHash())
		}
		if found.ChannelName() != "HBO" {
			t.Errorf("expected channel name 'HBO', got %q", found.ChannelName())
		}
	})

	t.Run("returns ErrStreamAlreadyExists for duplicate stream", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		s, err := stream.NewStream("def456", "ESPN")
		if err != nil {
			t.Fatalf("failed to create stream: %v", err)
		}

		ctx := context.Background()

		// Save the first time
		err = repo.Save(ctx, s)
		if err != nil {
			t.Fatalf("expected no error on first save, got %v", err)
		}

		// Try to save again
		err = repo.Save(ctx, s)
		if err != stream.ErrStreamAlreadyExists {
			t.Errorf("expected ErrStreamAlreadyExists, got %v", err)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		s, err := stream.NewStream("ghi789", "CNN")
		if err != nil {
			t.Fatalf("failed to create stream: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err = repo.Save(ctx, s)
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestStreamBoltDBRepository_FindByInfoHash(t *testing.T) {
	t.Run("finds existing stream", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		s, err := stream.NewStream("jkl012", "Discovery")
		if err != nil {
			t.Fatalf("failed to create stream: %v", err)
		}

		ctx := context.Background()
		err = repo.Save(ctx, s)
		if err != nil {
			t.Fatalf("failed to save stream: %v", err)
		}

		found, err := repo.FindByInfoHash(ctx, "jkl012")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if found.InfoHash() != "jkl012" {
			t.Errorf("expected infohash 'jkl012', got %q", found.InfoHash())
		}
		if found.ChannelName() != "Discovery" {
			t.Errorf("expected channel name 'Discovery', got %q", found.ChannelName())
		}
	})

	t.Run("returns ErrStreamNotFound for non-existent stream", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()
		_, err = repo.FindByInfoHash(ctx, "nonexistent")
		if err != stream.ErrStreamNotFound {
			t.Errorf("expected ErrStreamNotFound, got %v", err)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err = repo.FindByInfoHash(ctx, "somehash")
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestStreamBoltDBRepository_FindAll(t *testing.T) {
	t.Run("returns empty slice when no streams exist", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()
		streams, err := repo.FindAll(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if streams == nil {
			t.Error("expected non-nil slice")
		}
		if len(streams) != 0 {
			t.Errorf("expected empty slice, got %d streams", len(streams))
		}
	})

	t.Run("returns all saved streams", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()

		// Save multiple streams
		testStreams := []struct {
			infoHash    string
			channelName string
		}{
			{"hash1", "HBO"},
			{"hash2", "ESPN"},
			{"hash3", "CNN"},
			{"hash4", "Discovery"},
		}

		for _, ts := range testStreams {
			s, err := stream.NewStream(ts.infoHash, ts.channelName)
			if err != nil {
				t.Fatalf("failed to create stream %q: %v", ts.infoHash, err)
			}
			err = repo.Save(ctx, s)
			if err != nil {
				t.Fatalf("failed to save stream %q: %v", ts.infoHash, err)
			}
		}

		streams, err := repo.FindAll(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(streams) != len(testStreams) {
			t.Errorf("expected %d streams, got %d", len(testStreams), len(streams))
		}

		// Verify all streams are present
		foundHashes := make(map[string]bool)
		for _, s := range streams {
			foundHashes[s.InfoHash()] = true
		}

		for _, ts := range testStreams {
			if !foundHashes[ts.infoHash] {
				t.Errorf("expected to find stream %q", ts.infoHash)
			}
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err = repo.FindAll(ctx)
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestStreamBoltDBRepository_FindByChannelName(t *testing.T) {
	t.Run("returns empty slice when no streams exist for channel", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()
		streams, err := repo.FindByChannelName(ctx, "HBO")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if streams == nil {
			t.Error("expected non-nil slice")
		}
		if len(streams) != 0 {
			t.Errorf("expected empty slice, got %d streams", len(streams))
		}
	})

	t.Run("returns only streams for specified channel", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()

		// Save streams for multiple channels
		testStreams := []struct {
			infoHash    string
			channelName string
		}{
			{"hash1", "HBO"},
			{"hash2", "HBO"},
			{"hash3", "ESPN"},
			{"hash4", "HBO"},
			{"hash5", "CNN"},
		}

		for _, ts := range testStreams {
			s, err := stream.NewStream(ts.infoHash, ts.channelName)
			if err != nil {
				t.Fatalf("failed to create stream %q: %v", ts.infoHash, err)
			}
			err = repo.Save(ctx, s)
			if err != nil {
				t.Fatalf("failed to save stream %q: %v", ts.infoHash, err)
			}
		}

		// Find streams for HBO
		streams, err := repo.FindByChannelName(ctx, "HBO")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		expectedCount := 3 // hash1, hash2, hash4
		if len(streams) != expectedCount {
			t.Errorf("expected %d streams for HBO, got %d", expectedCount, len(streams))
		}

		// Verify all returned streams are for HBO
		for _, s := range streams {
			if s.ChannelName() != "HBO" {
				t.Errorf("expected channel 'HBO', got %q", s.ChannelName())
			}
		}

		// Verify correct infohashes
		expectedHashes := map[string]bool{"hash1": true, "hash2": true, "hash4": true}
		for _, s := range streams {
			if !expectedHashes[s.InfoHash()] {
				t.Errorf("unexpected stream infohash: %q", s.InfoHash())
			}
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err = repo.FindByChannelName(ctx, "HBO")
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestStreamBoltDBRepository_Delete(t *testing.T) {
	t.Run("deletes existing stream successfully", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		s, err := stream.NewStream("mno345", "NatGeo")
		if err != nil {
			t.Fatalf("failed to create stream: %v", err)
		}

		ctx := context.Background()
		err = repo.Save(ctx, s)
		if err != nil {
			t.Fatalf("failed to save stream: %v", err)
		}

		err = repo.Delete(ctx, "mno345")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify the stream was deleted
		_, err = repo.FindByInfoHash(ctx, "mno345")
		if err != stream.ErrStreamNotFound {
			t.Errorf("expected ErrStreamNotFound after deletion, got %v", err)
		}
	})

	t.Run("returns ErrStreamNotFound for non-existent stream", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()
		err = repo.Delete(ctx, "nonexistent")
		if err != stream.ErrStreamNotFound {
			t.Errorf("expected ErrStreamNotFound, got %v", err)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err = repo.Delete(ctx, "somehash")
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestStreamBoltDBRepository_DeleteByChannelName(t *testing.T) {
	t.Run("deletes all streams for a channel", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()

		// Save streams for multiple channels
		testStreams := []struct {
			infoHash    string
			channelName string
		}{
			{"hash1", "HBO"},
			{"hash2", "HBO"},
			{"hash3", "ESPN"},
			{"hash4", "HBO"},
			{"hash5", "CNN"},
		}

		for _, ts := range testStreams {
			s, err := stream.NewStream(ts.infoHash, ts.channelName)
			if err != nil {
				t.Fatalf("failed to create stream %q: %v", ts.infoHash, err)
			}
			err = repo.Save(ctx, s)
			if err != nil {
				t.Fatalf("failed to save stream %q: %v", ts.infoHash, err)
			}
		}

		// Delete all HBO streams
		err = repo.DeleteByChannelName(ctx, "HBO")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify HBO streams are gone
		hboStreams, err := repo.FindByChannelName(ctx, "HBO")
		if err != nil {
			t.Fatalf("failed to find HBO streams: %v", err)
		}
		if len(hboStreams) != 0 {
			t.Errorf("expected 0 HBO streams after deletion, got %d", len(hboStreams))
		}

		// Verify other streams still exist
		allStreams, err := repo.FindAll(ctx)
		if err != nil {
			t.Fatalf("failed to find all streams: %v", err)
		}
		expectedCount := 2 // ESPN and CNN
		if len(allStreams) != expectedCount {
			t.Errorf("expected %d streams after HBO deletion, got %d", expectedCount, len(allStreams))
		}

		// Verify remaining streams are correct
		for _, s := range allStreams {
			if s.ChannelName() == "HBO" {
				t.Errorf("found HBO stream after deletion: %q", s.InfoHash())
			}
		}
	})

	t.Run("succeeds even if no streams exist for channel", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()
		err = repo.DeleteByChannelName(ctx, "NonExistentChannel")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err = repo.DeleteByChannelName(ctx, "HBO")
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestStreamBoltDBRepository_Integration(t *testing.T) {
	t.Run("full lifecycle: save, find, delete", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()

		// Create and save
		s, err := stream.NewStream("integration123", "BBC One")
		if err != nil {
			t.Fatalf("failed to create stream: %v", err)
		}

		err = repo.Save(ctx, s)
		if err != nil {
			t.Fatalf("failed to save stream: %v", err)
		}

		// Find by infohash
		found, err := repo.FindByInfoHash(ctx, "integration123")
		if err != nil {
			t.Fatalf("failed to find stream: %v", err)
		}
		if found.InfoHash() != "integration123" {
			t.Errorf("expected 'integration123', got %q", found.InfoHash())
		}
		if found.ChannelName() != "BBC One" {
			t.Errorf("expected 'BBC One', got %q", found.ChannelName())
		}

		// Find by channel name
		channelStreams, err := repo.FindByChannelName(ctx, "BBC One")
		if err != nil {
			t.Fatalf("failed to find streams by channel: %v", err)
		}
		if len(channelStreams) != 1 {
			t.Errorf("expected 1 stream, got %d", len(channelStreams))
		}

		// Find all
		all, err := repo.FindAll(ctx)
		if err != nil {
			t.Fatalf("failed to find all streams: %v", err)
		}
		if len(all) != 1 {
			t.Errorf("expected 1 stream, got %d", len(all))
		}

		// Delete
		err = repo.Delete(ctx, "integration123")
		if err != nil {
			t.Fatalf("failed to delete stream: %v", err)
		}

		// Verify deletion
		all, err = repo.FindAll(ctx)
		if err != nil {
			t.Fatalf("failed to find all streams after delete: %v", err)
		}
		if len(all) != 0 {
			t.Errorf("expected 0 streams after deletion, got %d", len(all))
		}
	})

	t.Run("cascade delete: delete channel deletes all its streams", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()

		// Create multiple streams for the same channel
		channelName := "Sky Sports"
		for i := 1; i <= 3; i++ {
			s, err := stream.NewStream("skyhash"+string(rune('0'+i)), channelName)
			if err != nil {
				t.Fatalf("failed to create stream: %v", err)
			}
			err = repo.Save(ctx, s)
			if err != nil {
				t.Fatalf("failed to save stream: %v", err)
			}
		}

		// Create streams for another channel
		s, err := stream.NewStream("otherhash", "Other Channel")
		if err != nil {
			t.Fatalf("failed to create stream: %v", err)
		}
		err = repo.Save(ctx, s)
		if err != nil {
			t.Fatalf("failed to save stream: %v", err)
		}

		// Verify initial state
		allStreams, err := repo.FindAll(ctx)
		if err != nil {
			t.Fatalf("failed to find all streams: %v", err)
		}
		if len(allStreams) != 4 {
			t.Errorf("expected 4 streams initially, got %d", len(allStreams))
		}

		// Delete all streams for Sky Sports
		err = repo.DeleteByChannelName(ctx, channelName)
		if err != nil {
			t.Fatalf("failed to delete by channel name: %v", err)
		}

		// Verify Sky Sports streams are gone
		skyStreams, err := repo.FindByChannelName(ctx, channelName)
		if err != nil {
			t.Fatalf("failed to find Sky Sports streams: %v", err)
		}
		if len(skyStreams) != 0 {
			t.Errorf("expected 0 Sky Sports streams after cascade delete, got %d", len(skyStreams))
		}

		// Verify other channel still exists
		allStreams, err = repo.FindAll(ctx)
		if err != nil {
			t.Fatalf("failed to find all streams: %v", err)
		}
		if len(allStreams) != 1 {
			t.Errorf("expected 1 stream after cascade delete, got %d", len(allStreams))
		}
	})
}
