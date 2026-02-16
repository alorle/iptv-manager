package driven

import (
	"context"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/internal/probe"
)

func TestNewProbeBoltDBRepository(t *testing.T) {
	t.Run("nil db returns error", func(t *testing.T) {
		_, err := NewProbeBoltDBRepository(nil)
		if err == nil {
			t.Fatal("expected error for nil db, got nil")
		}
	})

	t.Run("valid db succeeds", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		repo, err := NewProbeBoltDBRepository(db)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo == nil {
			t.Fatal("expected non-nil repository")
		}
	})
}

func TestProbeBoltDBRepository_SaveAndFindByInfoHash(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo, err := NewProbeBoltDBRepository(db)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	r1, _ := probe.NewResult("abc123", now, true, 2*time.Second, 10, 100000, "dl", "")
	r2, _ := probe.NewResult("abc123", now.Add(-30*time.Minute), false, 0, 0, 0, "", "timeout")
	r3, _ := probe.NewResult("def456", now, true, time.Second, 5, 50000, "dl", "")

	// Save all results
	for _, r := range []probe.Result{r1, r2, r3} {
		if err := repo.Save(ctx, r); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// Find by infoHash "abc123" — should return 2 results, most recent first
	results, err := repo.FindByInfoHash(ctx, "abc123")
	if err != nil {
		t.Fatalf("FindByInfoHash failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// Most recent first
	if !results[0].Timestamp().After(results[1].Timestamp()) {
		t.Error("results should be ordered most recent first")
	}
	if results[0].Available() != true {
		t.Error("first result should be available")
	}
	if results[1].Available() != false {
		t.Error("second result should not be available")
	}

	// Find by infoHash "def456" — should return 1 result
	results, err = repo.FindByInfoHash(ctx, "def456")
	if err != nil {
		t.Fatalf("FindByInfoHash failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Find by non-existent infoHash — should return empty slice
	results, err = repo.FindByInfoHash(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("FindByInfoHash failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestProbeBoltDBRepository_FindByInfoHashSince(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo, err := NewProbeBoltDBRepository(db)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Create probes at different times
	r1, _ := probe.NewResult("abc123", now, true, time.Second, 10, 100000, "dl", "")
	r2, _ := probe.NewResult("abc123", now.Add(-1*time.Hour), true, 2*time.Second, 8, 80000, "dl", "")
	r3, _ := probe.NewResult("abc123", now.Add(-3*time.Hour), true, 3*time.Second, 5, 50000, "dl", "")
	r4, _ := probe.NewResult("abc123", now.Add(-25*time.Hour), true, 4*time.Second, 3, 30000, "dl", "")

	for _, r := range []probe.Result{r1, r2, r3, r4} {
		if err := repo.Save(ctx, r); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// Find since 2 hours ago — should return r1 and r2
	since := now.Add(-2 * time.Hour)
	results, err := repo.FindByInfoHashSince(ctx, "abc123", since)
	if err != nil {
		t.Fatalf("FindByInfoHashSince failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results since 2h ago, got %d", len(results))
	}

	// Find since 24 hours ago — should return r1, r2, r3
	since = now.Add(-24 * time.Hour)
	results, err = repo.FindByInfoHashSince(ctx, "abc123", since)
	if err != nil {
		t.Fatalf("FindByInfoHashSince failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results since 24h ago, got %d", len(results))
	}

	// Find since future time — should return 0
	since = now.Add(time.Hour)
	results, err = repo.FindByInfoHashSince(ctx, "abc123", since)
	if err != nil {
		t.Fatalf("FindByInfoHashSince failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for future since, got %d", len(results))
	}

	// Non-existent infoHash — should return empty slice
	results, err = repo.FindByInfoHashSince(ctx, "nonexistent", now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("FindByInfoHashSince failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestProbeBoltDBRepository_DeleteBefore(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo, err := NewProbeBoltDBRepository(db)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Create probes at different times across two infoHashes
	r1, _ := probe.NewResult("abc123", now, true, time.Second, 10, 100000, "dl", "")
	r2, _ := probe.NewResult("abc123", now.Add(-25*time.Hour), true, 2*time.Second, 5, 50000, "dl", "")
	r3, _ := probe.NewResult("def456", now, true, time.Second, 8, 80000, "dl", "")
	r4, _ := probe.NewResult("def456", now.Add(-25*time.Hour), false, 0, 0, 0, "", "timeout")

	for _, r := range []probe.Result{r1, r2, r3, r4} {
		if err := repo.Save(ctx, r); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// Delete everything older than 24 hours
	cutoff := now.Add(-24 * time.Hour)
	if err := repo.DeleteBefore(ctx, cutoff); err != nil {
		t.Fatalf("DeleteBefore failed: %v", err)
	}

	// abc123 should have 1 result left (the recent one)
	results, err := repo.FindByInfoHash(ctx, "abc123")
	if err != nil {
		t.Fatalf("FindByInfoHash failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for abc123 after cleanup, got %d", len(results))
	}

	// def456 should have 1 result left (the recent one)
	results, err = repo.FindByInfoHash(ctx, "def456")
	if err != nil {
		t.Fatalf("FindByInfoHash failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for def456 after cleanup, got %d", len(results))
	}
}

func TestProbeBoltDBRepository_ContextCancellation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo, err := NewProbeBoltDBRepository(db)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	r, _ := probe.NewResult("abc123", time.Now(), true, time.Second, 10, 100000, "dl", "")

	if err := repo.Save(ctx, r); err == nil {
		t.Error("expected error for cancelled context on Save")
	}

	if _, err := repo.FindByInfoHash(ctx, "abc123"); err == nil {
		t.Error("expected error for cancelled context on FindByInfoHash")
	}

	if _, err := repo.FindByInfoHashSince(ctx, "abc123", time.Now()); err == nil {
		t.Error("expected error for cancelled context on FindByInfoHashSince")
	}

	if err := repo.DeleteBefore(ctx, time.Now()); err == nil {
		t.Error("expected error for cancelled context on DeleteBefore")
	}
}

func TestProbeBoltDBRepository_ResultFieldPreservation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo, err := NewProbeBoltDBRepository(db)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	ctx := context.Background()
	now := time.Now().Truncate(time.Nanosecond) // Truncate for comparison

	original, _ := probe.NewResult("abc123", now, true, 2500*time.Millisecond, 42, 987654, "dl", "")

	if err := repo.Save(ctx, original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	results, err := repo.FindByInfoHash(ctx, "abc123")
	if err != nil {
		t.Fatalf("FindByInfoHash failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	got := results[0]

	if got.InfoHash() != original.InfoHash() {
		t.Errorf("InfoHash() = %q, want %q", got.InfoHash(), original.InfoHash())
	}
	if got.Available() != original.Available() {
		t.Errorf("Available() = %v, want %v", got.Available(), original.Available())
	}
	if got.StartupLatency() != original.StartupLatency() {
		t.Errorf("StartupLatency() = %v, want %v", got.StartupLatency(), original.StartupLatency())
	}
	if got.PeerCount() != original.PeerCount() {
		t.Errorf("PeerCount() = %d, want %d", got.PeerCount(), original.PeerCount())
	}
	if got.DownloadSpeed() != original.DownloadSpeed() {
		t.Errorf("DownloadSpeed() = %d, want %d", got.DownloadSpeed(), original.DownloadSpeed())
	}
	if got.Status() != original.Status() {
		t.Errorf("Status() = %q, want %q", got.Status(), original.Status())
	}
}
