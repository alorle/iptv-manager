package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/alorle/iptv-manager/internal/adapter/driven"
	"github.com/alorle/iptv-manager/internal/epg"
	"github.com/alorle/iptv-manager/internal/subscription"
)

// setupE2ETestDB creates a temporary BoltDB instance for E2E testing.
func setupE2ETestDB(t *testing.T) (*bbolt.DB, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "e2e-test.db")

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

// mockEPGFetcher provides controlled EPG data for E2E testing.
type mockEPGFetcher struct {
	channels []epg.Channel
	err      error
}

func (m *mockEPGFetcher) FetchEPG(ctx context.Context) ([]epg.Channel, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.channels, nil
}

// mockAcestreamSource provides controlled Acestream hash data for E2E testing.
type mockAcestreamSource struct {
	hashes map[string]map[string][]string
	err    error
}

func (m *mockAcestreamSource) FetchHashes(ctx context.Context, source string) (map[string][]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	if hashes, ok := m.hashes[source]; ok {
		return hashes, nil
	}
	return map[string][]string{}, nil
}

// TestEPGSyncService_SyncChannels_E2E performs a full end-to-end test of the sync workflow.
func TestEPGSyncService_SyncChannels_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	t.Run("full sync workflow creates channels and streams for subscriptions", func(t *testing.T) {
		// Setup test database
		db, cleanup := setupE2ETestDB(t)
		defer cleanup()

		// Create real repository adapters
		channelRepo, err := driven.NewChannelBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create channel repository: %v", err)
		}

		streamRepo, err := driven.NewStreamBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create stream repository: %v", err)
		}

		subscriptionRepo, err := driven.NewSubscriptionBoltDBRepository(db)
		if err != nil {
			t.Fatalf("failed to create subscription repository: %v", err)
		}

		// Setup mock EPG channels
		hboChannel, _ := epg.NewChannel("hbo.epg", "HBO", "https://example.com/hbo.png", "Movies", "en", "hbo.epg")
		espnChannel, _ := epg.NewChannel("espn.epg", "ESPN", "https://example.com/espn.png", "Sports", "en", "espn.epg")
		cnnChannel, _ := epg.NewChannel("cnn.epg", "CNN", "", "News", "en", "cnn.epg")
		discoveryChannel, _ := epg.NewChannel("discovery.epg", "Discovery", "https://example.com/discovery.png", "Documentary", "en", "discovery.epg")

		epgFetcher := &mockEPGFetcher{
			channels: []epg.Channel{hboChannel, espnChannel, cnnChannel, discoveryChannel},
		}

		// Setup mock Acestream sources with multiple streams per channel
		acestreamSource := &mockAcestreamSource{
			hashes: map[string]map[string][]string{
				"new-era": {
					"HBO": {"0123456789abcdef0123456789abcdef01234567"},
					"ESPN": {
						"fedcba9876543210fedcba9876543210fedcba98",
						"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					},
					"CNN": {"1111111111111111111111111111111111111111"},
				},
				"elcano": {
					"HBO": {
						"9999999999999999999999999999999999999999",
					},
					"Discovery": {"2222222222222222222222222222222222222222"},
				},
			},
		}

		// Create EPG sync service
		syncService := NewEPGSyncService(
			epgFetcher,
			acestreamSource,
			channelRepo,
			streamRepo,
			subscriptionRepo,
		)

		ctx := context.Background()

		// Step 1: Create subscriptions for HBO, ESPN, and CNN
		sub1, _ := subscription.NewSubscription("hbo.epg")
		sub2, _ := subscription.NewSubscription("espn.epg")
		sub3, _ := subscription.NewSubscription("cnn.epg")

		if err := subscriptionRepo.Save(ctx, sub1); err != nil {
			t.Fatalf("failed to save HBO subscription: %v", err)
		}
		if err := subscriptionRepo.Save(ctx, sub2); err != nil {
			t.Fatalf("failed to save ESPN subscription: %v", err)
		}
		if err := subscriptionRepo.Save(ctx, sub3); err != nil {
			t.Fatalf("failed to save CNN subscription: %v", err)
		}

		// Step 2: Run the sync
		if err := syncService.SyncChannels(ctx); err != nil {
			t.Fatalf("sync failed: %v", err)
		}

		// Step 3: Verify channels were created
		channels, err := channelRepo.FindAll(ctx)
		if err != nil {
			t.Fatalf("failed to list channels: %v", err)
		}

		if len(channels) != 3 {
			t.Fatalf("expected 3 channels, got %d", len(channels))
		}

		// Verify all expected channels exist
		channelNames := make(map[string]bool)
		for _, ch := range channels {
			channelNames[ch.Name()] = true
		}

		if !channelNames["HBO"] {
			t.Error("expected HBO channel to be created")
		}
		if !channelNames["ESPN"] {
			t.Error("expected ESPN channel to be created")
		}
		if !channelNames["CNN"] {
			t.Error("expected CNN channel to be created")
		}

		// Step 4: Verify streams were created (including multiple streams per channel)
		hboStreams, err := streamRepo.FindByChannelName(ctx, "HBO")
		if err != nil {
			t.Fatalf("failed to find HBO streams: %v", err)
		}
		// HBO should have 2 streams: one from new-era, one from elcano
		if len(hboStreams) != 2 {
			t.Errorf("expected 2 streams for HBO, got %d", len(hboStreams))
		}

		espnStreams, err := streamRepo.FindByChannelName(ctx, "ESPN")
		if err != nil {
			t.Fatalf("failed to find ESPN streams: %v", err)
		}
		// ESPN should have 2 streams: both from new-era
		if len(espnStreams) != 2 {
			t.Errorf("expected 2 streams for ESPN, got %d", len(espnStreams))
		}

		cnnStreams, err := streamRepo.FindByChannelName(ctx, "CNN")
		if err != nil {
			t.Fatalf("failed to find CNN streams: %v", err)
		}
		// CNN should have 1 stream: from new-era
		if len(cnnStreams) != 1 {
			t.Errorf("expected 1 stream for CNN, got %d", len(cnnStreams))
		}

		// Discovery should not have been created (not subscribed)
		_, err = channelRepo.FindByName(ctx, "Discovery")
		if err == nil {
			t.Error("expected Discovery not to exist (not subscribed)")
		}

		t.Logf("Successfully synced 3 subscribed channels with total streams: HBO=%d, ESPN=%d, CNN=%d",
			len(hboStreams), len(espnStreams), len(cnnStreams))
	})

	t.Run("merging behavior - existing channels are preserved during sync", func(t *testing.T) {
		// Setup test database
		db, cleanup := setupE2ETestDB(t)
		defer cleanup()

		channelRepo, _ := driven.NewChannelBoltDBRepository(db)
		streamRepo, _ := driven.NewStreamBoltDBRepository(db)
		subscriptionRepo, _ := driven.NewSubscriptionBoltDBRepository(db)

		ctx := context.Background()

		// Step 1: First sync - create HBO channel
		hboChannel, _ := epg.NewChannel("hbo.epg", "HBO", "https://example.com/hbo.png", "Movies", "en", "hbo.epg")
		epgFetcher := &mockEPGFetcher{
			channels: []epg.Channel{hboChannel},
		}

		acestreamSource := &mockAcestreamSource{
			hashes: map[string]map[string][]string{
				"new-era": {
					"HBO": {"0123456789abcdef0123456789abcdef01234567"},
				},
				"elcano": {},
			},
		}

		syncService := NewEPGSyncService(epgFetcher, acestreamSource, channelRepo, streamRepo, subscriptionRepo)

		sub, _ := subscription.NewSubscription("hbo.epg")
		if err := subscriptionRepo.Save(ctx, sub); err != nil {
			t.Fatalf("failed to save subscription: %v", err)
		}

		if err := syncService.SyncChannels(ctx); err != nil {
			t.Fatalf("first sync failed: %v", err)
		}

		// Verify channel and stream were created
		channels, err := channelRepo.FindAll(ctx)
		if err != nil {
			t.Fatalf("failed to list channels: %v", err)
		}
		if len(channels) != 1 {
			t.Fatalf("expected 1 channel after first sync, got %d", len(channels))
		}

		streams, err := streamRepo.FindByChannelName(ctx, "HBO")
		if err != nil {
			t.Fatalf("failed to find streams after first sync: %v", err)
		}
		if len(streams) != 1 {
			t.Fatalf("expected 1 stream after first sync, got %d", len(streams))
		}

		// Step 2: Second sync with same data - should not create duplicates
		if err := syncService.SyncChannels(ctx); err != nil {
			t.Fatalf("second sync failed: %v", err)
		}

		// Verify no duplicates were created
		channels, err = channelRepo.FindAll(ctx)
		if err != nil {
			t.Fatalf("failed to list channels after second sync: %v", err)
		}
		if len(channels) != 1 {
			t.Errorf("expected 1 channel after second sync (no duplicates), got %d", len(channels))
		}

		streams, err = streamRepo.FindByChannelName(ctx, "HBO")
		if err != nil {
			t.Fatalf("failed to find streams after second sync: %v", err)
		}
		if len(streams) != 1 {
			t.Errorf("expected 1 stream after second sync (no duplicates), got %d", len(streams))
		}

		t.Log("Successfully verified that existing channels are preserved during sync (no duplicates)")
	})

	t.Run("multiple streams per channel are created", func(t *testing.T) {
		// This test verifies that when a channel has multiple Acestream sources,
		// all streams are created correctly
		db, cleanup := setupE2ETestDB(t)
		defer cleanup()

		channelRepo, _ := driven.NewChannelBoltDBRepository(db)
		streamRepo, _ := driven.NewStreamBoltDBRepository(db)
		subscriptionRepo, _ := driven.NewSubscriptionBoltDBRepository(db)

		ctx := context.Background()

		// Setup HBO with multiple streams from different sources
		hboChannel, _ := epg.NewChannel("hbo.epg", "HBO", "https://example.com/hbo.png", "Movies", "en", "hbo.epg")
		epgFetcher := &mockEPGFetcher{
			channels: []epg.Channel{hboChannel},
		}

		acestreamSource := &mockAcestreamSource{
			hashes: map[string]map[string][]string{
				"new-era": {
					"HBO": {
						"hash1111111111111111111111111111111111111",
						"hash2222222222222222222222222222222222222",
					},
				},
				"elcano": {
					"HBO": {
						"hash3333333333333333333333333333333333333",
					},
				},
			},
		}

		syncService := NewEPGSyncService(epgFetcher, acestreamSource, channelRepo, streamRepo, subscriptionRepo)

		sub, _ := subscription.NewSubscription("hbo.epg")
		if err := subscriptionRepo.Save(ctx, sub); err != nil {
			t.Fatalf("failed to save subscription: %v", err)
		}

		if err := syncService.SyncChannels(ctx); err != nil {
			t.Fatalf("sync failed: %v", err)
		}

		// Verify all 3 streams were created (merged from both sources)
		streams, err := streamRepo.FindByChannelName(ctx, "HBO")
		if err != nil {
			t.Fatalf("failed to find streams: %v", err)
		}

		if len(streams) != 3 {
			t.Fatalf("expected 3 streams (merged from both sources), got %d", len(streams))
		}

		// Verify correct hashes
		hashSet := make(map[string]bool)
		for _, s := range streams {
			hashSet[s.InfoHash()] = true
		}

		if !hashSet["hash1111111111111111111111111111111111111"] {
			t.Error("expected hash1 from new-era source")
		}
		if !hashSet["hash2222222222222222222222222222222222222"] {
			t.Error("expected hash2 from new-era source")
		}
		if !hashSet["hash3333333333333333333333333333333333333"] {
			t.Error("expected hash3 from elcano source")
		}

		t.Log("Successfully created multiple streams per channel from different sources")
	})

	t.Run("disabled subscriptions are skipped during sync", func(t *testing.T) {
		// Setup test database
		db, cleanup := setupE2ETestDB(t)
		defer cleanup()

		channelRepo, _ := driven.NewChannelBoltDBRepository(db)
		streamRepo, _ := driven.NewStreamBoltDBRepository(db)
		subscriptionRepo, _ := driven.NewSubscriptionBoltDBRepository(db)

		ctx := context.Background()

		// Setup EPG and Acestream data
		hboChannel, _ := epg.NewChannel("hbo.epg", "HBO", "https://example.com/hbo.png", "Movies", "en", "hbo.epg")
		epgChannel, _ := epg.NewChannel("espn.epg", "ESPN", "https://example.com/espn.png", "Sports", "en", "espn.epg")

		epgFetcher := &mockEPGFetcher{
			channels: []epg.Channel{hboChannel, epgChannel},
		}

		acestreamSource := &mockAcestreamSource{
			hashes: map[string]map[string][]string{
				"new-era": {
					"HBO":  {"0123456789abcdef0123456789abcdef01234567"},
					"ESPN": {"fedcba9876543210fedcba9876543210fedcba98"},
				},
				"elcano": {},
			},
		}

		syncService := NewEPGSyncService(epgFetcher, acestreamSource, channelRepo, streamRepo, subscriptionRepo)

		// Create one enabled subscription and one disabled subscription
		enabledSub, _ := subscription.NewSubscription("hbo.epg")
		if err := subscriptionRepo.Save(ctx, enabledSub); err != nil {
			t.Fatalf("failed to save enabled subscription: %v", err)
		}

		disabledSub, _ := subscription.NewSubscription("espn.epg")
		disabledSub = disabledSub.Disable()
		if err := subscriptionRepo.Save(ctx, disabledSub); err != nil {
			t.Fatalf("failed to save disabled subscription: %v", err)
		}

		// Run sync
		if err := syncService.SyncChannels(ctx); err != nil {
			t.Fatalf("sync failed: %v", err)
		}

		// Verify only HBO was created (enabled subscription)
		channels, err := channelRepo.FindAll(ctx)
		if err != nil {
			t.Fatalf("failed to list channels: %v", err)
		}

		if len(channels) != 1 {
			t.Fatalf("expected 1 channel (only enabled), got %d", len(channels))
		}

		if channels[0].Name() != "HBO" {
			t.Errorf("expected HBO channel, got %q", channels[0].Name())
		}

		// Verify ESPN was not created
		_, err = channelRepo.FindByName(ctx, "ESPN")
		if err == nil {
			t.Error("expected ESPN channel not to exist (disabled subscription)")
		}

		t.Log("Successfully skipped disabled subscription during sync")
	})
}
