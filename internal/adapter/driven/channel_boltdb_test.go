package driven

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/alorle/iptv-manager/internal/channel"
)

// setupTestDB creates a temporary BoltDB instance for testing.
func setupTestDB(t *testing.T) (*bbolt.DB, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestNewChannelBoltDBRepository(t *testing.T) {
	t.Run("creates repository and bucket successfully", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewChannelBoltDBRepository(db)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if repo == nil {
			t.Fatal("expected non-nil repository")
		}

		// Verify bucket was created
		err = db.View(func(tx *bbolt.Tx) error {
			bucket := tx.Bucket([]byte(channelsBucket))
			if bucket == nil {
				t.Error("expected channels bucket to exist")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed to verify bucket: %v", err)
		}
	})

	t.Run("returns error for nil database", func(t *testing.T) {
		repo, err := NewChannelBoltDBRepository(nil)
		if err == nil {
			t.Fatal("expected error for nil database")
		}
		if repo != nil {
			t.Error("expected nil repository")
		}
	})
}

func TestChannelBoltDBRepository_Save(t *testing.T) {
	t.Run("saves a new channel successfully", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewChannelBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ch, err := channel.NewChannel("HBO")
		if err != nil {
			t.Fatalf("failed to create channel: %v", err)
		}

		ctx := context.Background()
		err = repo.Save(ctx, ch)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify the channel was saved
		found, err := repo.FindByName(ctx, "HBO")
		if err != nil {
			t.Fatalf("failed to find saved channel: %v", err)
		}
		if found.Name() != "HBO" {
			t.Errorf("expected channel name 'HBO', got %q", found.Name())
		}
	})

	t.Run("returns ErrChannelAlreadyExists for duplicate channel", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewChannelBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ch, err := channel.NewChannel("ESPN")
		if err != nil {
			t.Fatalf("failed to create channel: %v", err)
		}

		ctx := context.Background()

		// Save the first time
		err = repo.Save(ctx, ch)
		if err != nil {
			t.Fatalf("expected no error on first save, got %v", err)
		}

		// Try to save again
		err = repo.Save(ctx, ch)
		if err != channel.ErrChannelAlreadyExists {
			t.Errorf("expected ErrChannelAlreadyExists, got %v", err)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewChannelBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ch, err := channel.NewChannel("CNN")
		if err != nil {
			t.Fatalf("failed to create channel: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err = repo.Save(ctx, ch)
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestChannelBoltDBRepository_FindByName(t *testing.T) {
	t.Run("finds existing channel", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewChannelBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ch, err := channel.NewChannel("Discovery")
		if err != nil {
			t.Fatalf("failed to create channel: %v", err)
		}

		ctx := context.Background()
		err = repo.Save(ctx, ch)
		if err != nil {
			t.Fatalf("failed to save channel: %v", err)
		}

		found, err := repo.FindByName(ctx, "Discovery")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if found.Name() != "Discovery" {
			t.Errorf("expected channel name 'Discovery', got %q", found.Name())
		}
	})

	t.Run("returns ErrChannelNotFound for non-existent channel", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewChannelBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()
		_, err = repo.FindByName(ctx, "NonExistent")
		if err != channel.ErrChannelNotFound {
			t.Errorf("expected ErrChannelNotFound, got %v", err)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewChannelBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err = repo.FindByName(ctx, "SomeChannel")
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestChannelBoltDBRepository_FindAll(t *testing.T) {
	t.Run("returns empty slice when no channels exist", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewChannelBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()
		channels, err := repo.FindAll(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if channels == nil {
			t.Error("expected non-nil slice")
		}
		if len(channels) != 0 {
			t.Errorf("expected empty slice, got %d channels", len(channels))
		}
	})

	t.Run("returns all saved channels", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewChannelBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()

		// Save multiple channels
		names := []string{"HBO", "ESPN", "CNN", "Discovery"}
		for _, name := range names {
			ch, err := channel.NewChannel(name)
			if err != nil {
				t.Fatalf("failed to create channel %q: %v", name, err)
			}
			err = repo.Save(ctx, ch)
			if err != nil {
				t.Fatalf("failed to save channel %q: %v", name, err)
			}
		}

		channels, err := repo.FindAll(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(channels) != len(names) {
			t.Errorf("expected %d channels, got %d", len(names), len(channels))
		}

		// Verify all channels are present
		foundNames := make(map[string]bool)
		for _, ch := range channels {
			foundNames[ch.Name()] = true
		}

		for _, name := range names {
			if !foundNames[name] {
				t.Errorf("expected to find channel %q", name)
			}
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewChannelBoltDBRepository(db)
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

func TestChannelBoltDBRepository_Delete(t *testing.T) {
	t.Run("deletes existing channel successfully", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewChannelBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ch, err := channel.NewChannel("NatGeo")
		if err != nil {
			t.Fatalf("failed to create channel: %v", err)
		}

		ctx := context.Background()
		err = repo.Save(ctx, ch)
		if err != nil {
			t.Fatalf("failed to save channel: %v", err)
		}

		err = repo.Delete(ctx, "NatGeo")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify the channel was deleted
		_, err = repo.FindByName(ctx, "NatGeo")
		if err != channel.ErrChannelNotFound {
			t.Errorf("expected ErrChannelNotFound after deletion, got %v", err)
		}
	})

	t.Run("returns ErrChannelNotFound for non-existent channel", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewChannelBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()
		err = repo.Delete(ctx, "NonExistent")
		if err != channel.ErrChannelNotFound {
			t.Errorf("expected ErrChannelNotFound, got %v", err)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewChannelBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err = repo.Delete(ctx, "SomeChannel")
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestChannelBoltDBRepository_Integration(t *testing.T) {
	t.Run("full lifecycle: save, find, update via delete+save, delete", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewChannelBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()

		// Create and save
		ch, err := channel.NewChannel("BBC One")
		if err != nil {
			t.Fatalf("failed to create channel: %v", err)
		}

		err = repo.Save(ctx, ch)
		if err != nil {
			t.Fatalf("failed to save channel: %v", err)
		}

		// Find
		found, err := repo.FindByName(ctx, "BBC One")
		if err != nil {
			t.Fatalf("failed to find channel: %v", err)
		}
		if found.Name() != "BBC One" {
			t.Errorf("expected 'BBC One', got %q", found.Name())
		}

		// Find all
		all, err := repo.FindAll(ctx)
		if err != nil {
			t.Fatalf("failed to find all channels: %v", err)
		}
		if len(all) != 1 {
			t.Errorf("expected 1 channel, got %d", len(all))
		}

		// Delete
		err = repo.Delete(ctx, "BBC One")
		if err != nil {
			t.Fatalf("failed to delete channel: %v", err)
		}

		// Verify deletion
		all, err = repo.FindAll(ctx)
		if err != nil {
			t.Fatalf("failed to find all channels after delete: %v", err)
		}
		if len(all) != 0 {
			t.Errorf("expected 0 channels after deletion, got %d", len(all))
		}
	})
}
