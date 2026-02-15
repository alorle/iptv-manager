package application

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/alorle/iptv-manager/internal/channel"
	"github.com/alorle/iptv-manager/internal/epg"
	"github.com/alorle/iptv-manager/internal/port/driven"
	"github.com/alorle/iptv-manager/internal/stream"
)

const (
	// Minimum fuzzy match score to consider automatic matching successful
	fuzzyMatchThreshold = 0.7

	// Acestream sources to fetch from
	sourceNewEra = "new-era"
	sourceElcano = "elcano"
)

// EPGSyncService orchestrates the EPG sync workflow:
// fetch EPG data, match with Acestream sources, merge channels, and update streams.
type EPGSyncService struct {
	epgFetcher       driven.EPGFetcher
	acestreamSrc     driven.AcestreamSource
	channelRepo      driven.ChannelRepository
	streamRepo       driven.StreamRepository
	subscriptionRepo driven.SubscriptionRepository
}

// NewEPGSyncService creates a new EPG sync service with the required dependencies.
func NewEPGSyncService(
	epgFetcher driven.EPGFetcher,
	acestreamSrc driven.AcestreamSource,
	channelRepo driven.ChannelRepository,
	streamRepo driven.StreamRepository,
	subscriptionRepo driven.SubscriptionRepository,
) *EPGSyncService {
	return &EPGSyncService{
		epgFetcher:       epgFetcher,
		acestreamSrc:     acestreamSrc,
		channelRepo:      channelRepo,
		streamRepo:       streamRepo,
		subscriptionRepo: subscriptionRepo,
	}
}

// SyncChannels performs the full EPG synchronization workflow:
// 1. Fetch EPG channels from external source
// 2. Fetch Acestream hash lists from both sources (new-era, elcano)
// 3. Match EPG channels with Acestream hashes using fuzzy matching
// 4. Create/update channels and streams for subscribed EPG channels
// 5. Archive channels that disappeared from EPG
//
// Errors during individual channel processing are logged but do not stop the sync.
// Only critical errors (unable to fetch data, unable to load subscriptions) return an error.
func (s *EPGSyncService) SyncChannels(ctx context.Context) error {
	// Fetch EPG channels
	epgChannels, err := s.epgFetcher.FetchEPG(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch EPG data: %w", err)
	}

	// Fetch Acestream hashes from both sources
	newEraHashes, err := s.acestreamSrc.FetchHashes(ctx, sourceNewEra)
	if err != nil {
		return fmt.Errorf("failed to fetch new-era hashes: %w", err)
	}

	elcanoHashes, err := s.acestreamSrc.FetchHashes(ctx, sourceElcano)
	if err != nil {
		return fmt.Errorf("failed to fetch elcano hashes: %w", err)
	}

	// Merge hash maps from both sources
	allHashes := mergeHashMaps(newEraHashes, elcanoHashes)

	// Load all subscriptions
	subscriptions, err := s.subscriptionRepo.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to load subscriptions: %w", err)
	}

	// Build a set of subscribed EPG IDs for quick lookup
	subscribedEPGIDs := make(map[string]bool)
	for _, sub := range subscriptions {
		if sub.IsEnabled() {
			subscribedEPGIDs[sub.EPGChannelID()] = true
		}
	}

	// Load existing channels for comparison
	existingChannels, err := s.channelRepo.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to load existing channels: %w", err)
	}

	// Build a map of existing channels by name
	existingChannelMap := make(map[string]channel.Channel)
	for _, ch := range existingChannels {
		existingChannelMap[ch.Name()] = ch
	}

	// Track which channels were processed
	processedChannelNames := make(map[string]bool)

	// Process each EPG channel
	for _, epgChannel := range epgChannels {
		// Only process subscribed channels
		if !subscribedEPGIDs[epgChannel.EPGID()] {
			continue
		}

		// Match this EPG channel with Acestream hashes
		matchedHashes, matchScore := s.matchChannelWithHashes(epgChannel, allHashes)

		// Skip channels that fail automatic matching
		if matchScore < fuzzyMatchThreshold {
			log.Printf("Skipping EPG channel %s (EPGID: %s) - no automatic match found (score: %.2f)",
				epgChannel.Name(), epgChannel.EPGID(), matchScore)
			continue
		}

		// Skip channels with no matched hashes
		if len(matchedHashes) == 0 {
			log.Printf("Skipping EPG channel %s (EPGID: %s) - matched but no hashes found",
				epgChannel.Name(), epgChannel.EPGID())
			continue
		}

		// Process this channel (create or update)
		if err := s.processChannel(ctx, epgChannel, matchedHashes, existingChannelMap); err != nil {
			// Log error but continue processing other channels
			log.Printf("Error processing channel %s: %v", epgChannel.Name(), err)
			continue
		}

		// Mark this channel as processed
		processedChannelNames[epgChannel.Name()] = true
	}

	// Archive channels that disappeared from EPG (only if they were active)
	for _, existingChannel := range existingChannels {
		if existingChannel.Status() == channel.StatusActive && !processedChannelNames[existingChannel.Name()] {
			existingChannel.Archive()
			if err := s.channelRepo.Update(ctx, existingChannel); err != nil {
				log.Printf("Error archiving channel %s: %v", existingChannel.Name(), err)
			} else {
				log.Printf("Archived channel %s (no longer in EPG)", existingChannel.Name())
			}
		}
	}

	return nil
}

// matchChannelWithHashes finds the best matching Acestream channel for an EPG channel
// using fuzzy name matching. Returns the matched hashes and the match score.
func (s *EPGSyncService) matchChannelWithHashes(epgChannel epg.Channel, allHashes map[string][]string) ([]string, float64) {
	var bestMatch string
	var bestScore float64

	// Find the best matching channel name in the hash map
	for acestreamName := range allHashes {
		score := channel.FuzzyMatch(epgChannel.Name(), acestreamName)
		if score > bestScore {
			bestScore = score
			bestMatch = acestreamName
		}
	}

	if bestScore < fuzzyMatchThreshold {
		return nil, bestScore
	}

	return allHashes[bestMatch], bestScore
}

// processChannel creates or updates a channel and its streams.
// It merges new channels (doesn't overwrite existing) and updates streams if hashes changed.
func (s *EPGSyncService) processChannel(
	ctx context.Context,
	epgChannel epg.Channel,
	hashes []string,
	existingChannels map[string]channel.Channel,
) error {
	channelName := epgChannel.Name()

	// Check if channel already exists
	existingChannel, exists := existingChannels[channelName]

	var ch channel.Channel
	var isNew bool

	if !exists {
		// Create new channel
		newChannel, err := channel.NewChannel(channelName)
		if err != nil {
			return fmt.Errorf("failed to create channel: %w", err)
		}

		// Set EPG mapping
		mapping, err := channel.NewEPGMapping(epgChannel.EPGID(), channel.MappingAuto, time.Now())
		if err != nil {
			return fmt.Errorf("failed to create EPG mapping: %w", err)
		}
		newChannel.SetEPGMapping(mapping)

		ch = newChannel
		isNew = true
	} else {
		// Use existing channel, update EPG mapping
		ch = existingChannel

		// Update EPG mapping with current timestamp
		mapping, err := channel.NewEPGMapping(epgChannel.EPGID(), channel.MappingAuto, time.Now())
		if err != nil {
			return fmt.Errorf("failed to create EPG mapping: %w", err)
		}
		ch.SetEPGMapping(mapping)
	}

	// Save or update channel
	if isNew {
		if err := s.channelRepo.Save(ctx, ch); err != nil {
			if errors.Is(err, channel.ErrChannelAlreadyExists) {
				log.Printf("Channel %s already exists, treating as update", channelName)
			} else {
				return fmt.Errorf("failed to save channel: %w", err)
			}
		}
	} else {
		if err := s.channelRepo.Update(ctx, ch); err != nil {
			return fmt.Errorf("failed to update channel: %w", err)
		}
	}

	// Update streams for this channel
	return s.updateChannelStreams(ctx, channelName, hashes)
}

// updateChannelStreams creates or updates streams for a channel.
// Creates multiple stream entries when multiple hashes exist for a channel.
// Updates existing streams if hashes changed.
func (s *EPGSyncService) updateChannelStreams(ctx context.Context, channelName string, hashes []string) error {
	// Get existing streams for this channel
	existingStreams, err := s.streamRepo.FindByChannelName(ctx, channelName)
	if err != nil && !errors.Is(err, stream.ErrStreamNotFound) {
		return fmt.Errorf("failed to load existing streams: %w", err)
	}

	// Build a set of existing stream hashes
	existingHashSet := make(map[string]bool)
	for _, s := range existingStreams {
		existingHashSet[s.InfoHash()] = true
	}

	// Create new streams for hashes that don't exist
	for _, hash := range hashes {
		if existingHashSet[hash] {
			// Stream already exists, no update needed
			continue
		}

		// Create new stream
		newStream, err := stream.NewStream(hash, channelName)
		if err != nil {
			log.Printf("Error creating stream for channel %s, hash %s: %v", channelName, hash, err)
			continue
		}

		if err := s.streamRepo.Save(ctx, newStream); err != nil {
			// If stream already exists, it's fine (may have been created by another process)
			if !errors.Is(err, stream.ErrStreamAlreadyExists) {
				log.Printf("Error saving stream for channel %s, hash %s: %v", channelName, hash, err)
			}
			continue
		}
	}

	// Remove streams that are no longer in the hash list
	hashSet := make(map[string]bool)
	for _, hash := range hashes {
		hashSet[hash] = true
	}

	for _, existingStream := range existingStreams {
		if !hashSet[existingStream.InfoHash()] {
			if err := s.streamRepo.Delete(ctx, existingStream.InfoHash()); err != nil {
				log.Printf("Error deleting obsolete stream %s for channel %s: %v",
					existingStream.InfoHash(), channelName, err)
			}
		}
	}

	return nil
}

// mergeHashMaps merges multiple hash maps into a single map.
// If a channel appears in multiple sources, all hashes are combined.
func mergeHashMaps(maps ...map[string][]string) map[string][]string {
	result := make(map[string][]string)

	for _, m := range maps {
		for channelName, hashes := range m {
			// Append hashes, avoiding duplicates
			existingHashes := result[channelName]
			hashSet := make(map[string]bool)
			for _, h := range existingHashes {
				hashSet[h] = true
			}

			for _, h := range hashes {
				if !hashSet[h] {
					result[channelName] = append(result[channelName], h)
					hashSet[h] = true
				}
			}
		}
	}

	return result
}
