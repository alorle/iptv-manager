package driven

import (
	"context"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/alorle/iptv-manager/internal/subscription"
)

func TestNewSubscriptionBoltDBRepository(t *testing.T) {
	t.Run("creates repository and bucket successfully", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewSubscriptionBoltDBRepository(db)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if repo == nil {
			t.Fatal("expected non-nil repository")
		}

		// Verify bucket was created
		err = db.View(func(tx *bbolt.Tx) error {
			bucket := tx.Bucket([]byte(subscriptionsBucket))
			if bucket == nil {
				t.Error("expected subscriptions bucket to exist")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed to verify bucket: %v", err)
		}
	})

	t.Run("returns error for nil database", func(t *testing.T) {
		repo, err := NewSubscriptionBoltDBRepository(nil)
		if err == nil {
			t.Fatal("expected error for nil database")
		}
		if repo != nil {
			t.Error("expected nil repository")
		}
	})
}

func TestSubscriptionBoltDBRepository_Save(t *testing.T) {
	t.Run("saves a new subscription successfully", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewSubscriptionBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		sub, err := subscription.NewSubscription("hbo-channel-1")
		if err != nil {
			t.Fatalf("failed to create subscription: %v", err)
		}

		ctx := context.Background()
		err = repo.Save(ctx, sub)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify the subscription was saved
		found, err := repo.FindByEPGID(ctx, "hbo-channel-1")
		if err != nil {
			t.Fatalf("failed to find saved subscription: %v", err)
		}
		if found.EPGChannelID() != "hbo-channel-1" {
			t.Errorf("expected EPG channel ID 'hbo-channel-1', got %q", found.EPGChannelID())
		}
		if !found.IsEnabled() {
			t.Error("expected subscription to be enabled by default")
		}
		if found.HasManualOverride() {
			t.Error("expected no manual override by default")
		}
	})

	t.Run("returns error when subscription already exists", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewSubscriptionBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		sub, err := subscription.NewSubscription("hbo-channel-1")
		if err != nil {
			t.Fatalf("failed to create subscription: %v", err)
		}

		ctx := context.Background()
		err = repo.Save(ctx, sub)
		if err != nil {
			t.Fatalf("failed to save first subscription: %v", err)
		}

		// Try to save again
		err = repo.Save(ctx, sub)
		if err != subscription.ErrSubscriptionAlreadyExists {
			t.Fatalf("expected ErrSubscriptionAlreadyExists, got %v", err)
		}
	})

	t.Run("saves subscription with disabled state", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewSubscriptionBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		sub, err := subscription.NewSubscription("hbo-channel-1")
		if err != nil {
			t.Fatalf("failed to create subscription: %v", err)
		}

		// Disable the subscription
		sub = sub.Disable()

		ctx := context.Background()
		err = repo.Save(ctx, sub)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify the subscription was saved with disabled state
		found, err := repo.FindByEPGID(ctx, "hbo-channel-1")
		if err != nil {
			t.Fatalf("failed to find saved subscription: %v", err)
		}
		if found.IsEnabled() {
			t.Error("expected subscription to be disabled")
		}
		if !found.HasManualOverride() {
			t.Error("expected manual override flag to be set")
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewSubscriptionBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		sub, err := subscription.NewSubscription("hbo-channel-1")
		if err != nil {
			t.Fatalf("failed to create subscription: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err = repo.Save(ctx, sub)
		if err == nil {
			t.Fatal("expected error due to cancelled context")
		}
	})
}

func TestSubscriptionBoltDBRepository_FindAll(t *testing.T) {
	t.Run("returns empty slice when no subscriptions exist", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewSubscriptionBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()
		subscriptions, err := repo.FindAll(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if subscriptions == nil {
			t.Fatal("expected non-nil slice")
		}
		if len(subscriptions) != 0 {
			t.Errorf("expected empty slice, got %d subscriptions", len(subscriptions))
		}
	})

	t.Run("returns all saved subscriptions", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewSubscriptionBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()

		// Save multiple subscriptions
		sub1, _ := subscription.NewSubscription("hbo-channel-1")
		sub2, _ := subscription.NewSubscription("espn-channel-2")
		sub3, _ := subscription.NewSubscription("cnn-channel-3")
		sub3 = sub3.Disable()

		err = repo.Save(ctx, sub1)
		if err != nil {
			t.Fatalf("failed to save subscription 1: %v", err)
		}
		err = repo.Save(ctx, sub2)
		if err != nil {
			t.Fatalf("failed to save subscription 2: %v", err)
		}
		err = repo.Save(ctx, sub3)
		if err != nil {
			t.Fatalf("failed to save subscription 3: %v", err)
		}

		// Retrieve all subscriptions
		subscriptions, err := repo.FindAll(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(subscriptions) != 3 {
			t.Fatalf("expected 3 subscriptions, got %d", len(subscriptions))
		}

		// Verify subscriptions have correct state
		foundDisabled := false
		for _, sub := range subscriptions {
			if sub.EPGChannelID() == "cnn-channel-3" {
				if sub.IsEnabled() {
					t.Error("expected cnn-channel-3 to be disabled")
				}
				foundDisabled = true
			}
		}
		if !foundDisabled {
			t.Error("did not find the disabled subscription")
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewSubscriptionBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err = repo.FindAll(ctx)
		if err == nil {
			t.Fatal("expected error due to cancelled context")
		}
	})
}

func TestSubscriptionBoltDBRepository_FindByEPGID(t *testing.T) {
	t.Run("finds existing subscription", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewSubscriptionBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		sub, err := subscription.NewSubscription("hbo-channel-1")
		if err != nil {
			t.Fatalf("failed to create subscription: %v", err)
		}

		ctx := context.Background()
		err = repo.Save(ctx, sub)
		if err != nil {
			t.Fatalf("failed to save subscription: %v", err)
		}

		found, err := repo.FindByEPGID(ctx, "hbo-channel-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if found.EPGChannelID() != "hbo-channel-1" {
			t.Errorf("expected EPG channel ID 'hbo-channel-1', got %q", found.EPGChannelID())
		}
	})

	t.Run("returns error when subscription not found", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewSubscriptionBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()
		_, err = repo.FindByEPGID(ctx, "nonexistent-channel")
		if err != subscription.ErrSubscriptionNotFound {
			t.Fatalf("expected ErrSubscriptionNotFound, got %v", err)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewSubscriptionBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err = repo.FindByEPGID(ctx, "hbo-channel-1")
		if err == nil {
			t.Fatal("expected error due to cancelled context")
		}
	})
}

func TestSubscriptionBoltDBRepository_Delete(t *testing.T) {
	t.Run("deletes existing subscription", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewSubscriptionBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		sub, err := subscription.NewSubscription("hbo-channel-1")
		if err != nil {
			t.Fatalf("failed to create subscription: %v", err)
		}

		ctx := context.Background()
		err = repo.Save(ctx, sub)
		if err != nil {
			t.Fatalf("failed to save subscription: %v", err)
		}

		err = repo.Delete(ctx, "hbo-channel-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify subscription was deleted
		_, err = repo.FindByEPGID(ctx, "hbo-channel-1")
		if err != subscription.ErrSubscriptionNotFound {
			t.Fatalf("expected ErrSubscriptionNotFound, got %v", err)
		}
	})

	t.Run("returns error when subscription not found", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewSubscriptionBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx := context.Background()
		err = repo.Delete(ctx, "nonexistent-channel")
		if err != subscription.ErrSubscriptionNotFound {
			t.Fatalf("expected ErrSubscriptionNotFound, got %v", err)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewSubscriptionBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err = repo.Delete(ctx, "hbo-channel-1")
		if err == nil {
			t.Fatal("expected error due to cancelled context")
		}
	})
}
