package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alorle/iptv-manager/internal/channel"
	"github.com/alorle/iptv-manager/internal/epg"
	"github.com/alorle/iptv-manager/internal/port/driven"
	"github.com/alorle/iptv-manager/internal/stream"
)

const fuzzyMatchThreshold = 0.7

// EPGSyncService orchestrates the EPG sync workflow:
// fetch EPG data, match with Acestream sources, merge channels, and update streams.
type EPGSyncService struct {
	epgFetcher       driven.EPGFetcher
	acestreamSrc     driven.AcestreamSource
	channelRepo      driven.ChannelRepository
	streamRepo       driven.StreamRepository
	subscriptionRepo driven.SubscriptionRepository
	logger           *slog.Logger
}

// NewEPGSyncService creates a new EPG sync service with the required dependencies.
func NewEPGSyncService(
	epgFetcher driven.EPGFetcher,
	acestreamSrc driven.AcestreamSource,
	channelRepo driven.ChannelRepository,
	streamRepo driven.StreamRepository,
	subscriptionRepo driven.SubscriptionRepository,
	logger *slog.Logger,
) *EPGSyncService {
	return &EPGSyncService{
		epgFetcher:       epgFetcher,
		acestreamSrc:     acestreamSrc,
		channelRepo:      channelRepo,
		streamRepo:       streamRepo,
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
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

	newEraHashes, err := s.acestreamSrc.FetchHashes(ctx, stream.SourceNewEra)
	if err != nil {
		return fmt.Errorf("failed to fetch new-era hashes: %w", err)
	}

	elcanoHashes, err := s.acestreamSrc.FetchHashes(ctx, stream.SourceElcano)
	if err != nil {
		return fmt.Errorf("failed to fetch elcano hashes: %w", err)
	}

	allHashes := mergeTaggedHashMaps(
		tagHashMap(newEraHashes, stream.SourceNewEra),
		tagHashMap(elcanoHashes, stream.SourceElcano),
	)

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

		matchedHashes, matchScore := s.matchChannelWithHashes(epgChannel, allHashes)

		if matchScore < fuzzyMatchThreshold {
			s.logger.Debug("skipping epg channel, no automatic match", "channel", epgChannel.Name(), "epg_id", epgChannel.EPGID(), "score", matchScore)
			continue
		}

		if len(matchedHashes) == 0 {
			s.logger.Debug("skipping epg channel, matched but no hashes", "channel", epgChannel.Name(), "epg_id", epgChannel.EPGID())
			continue
		}

		if err := s.processChannel(ctx, epgChannel, matchedHashes, existingChannelMap); err != nil {
			// Log error but continue processing other channels
			s.logger.Error("failed to process channel", "channel", epgChannel.Name(), "error", err)
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
				s.logger.Error("failed to archive channel", "channel", existingChannel.Name(), "error", err)
			} else {
				s.logger.Info("archived channel, no longer in epg", "channel", existingChannel.Name())
			}
		}
	}

	return nil
}

func (s *EPGSyncService) matchChannelWithHashes(epgChannel epg.Channel, allHashes map[string][]taggedHash) ([]taggedHash, float64) {
	if hashes, ok := allHashes[epgChannel.EPGID()]; ok {
		return hashes, 1.0
	}

	var bestMatch string
	var bestScore float64

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

func (s *EPGSyncService) processChannel(
	ctx context.Context,
	epgChannel epg.Channel,
	hashes []taggedHash,
	existingChannels map[string]channel.Channel,
) error {
	channelName := epgChannel.Name()

	// Check if channel already exists
	existingChannel, exists := existingChannels[channelName]

	// Create EPG mapping (same for new and existing channels)
	mapping, err := channel.NewEPGMapping(epgChannel.EPGID(), channel.MappingAuto, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create EPG mapping: %w", err)
	}

	if !exists {
		ch, err := channel.NewChannel(channelName)
		if err != nil {
			return fmt.Errorf("failed to create channel: %w", err)
		}
		ch.SetEPGMapping(mapping)

		if err := s.channelRepo.Save(ctx, ch); err != nil {
			if errors.Is(err, channel.ErrChannelAlreadyExists) {
				s.logger.Warn("channel already exists, treating as update", "channel", channelName)
			} else {
				return fmt.Errorf("failed to save channel: %w", err)
			}
		}
	} else {
		existingChannel.SetEPGMapping(mapping)
		if err := s.channelRepo.Update(ctx, existingChannel); err != nil {
			return fmt.Errorf("failed to update channel: %w", err)
		}
	}

	// Update streams for this channel
	return s.updateChannelStreams(ctx, channelName, hashes)
}

func (s *EPGSyncService) updateChannelStreams(ctx context.Context, channelName string, hashes []taggedHash) error {
	existingStreams, err := s.streamRepo.FindByChannelName(ctx, channelName)
	if err != nil && !errors.Is(err, stream.ErrStreamNotFound) {
		return fmt.Errorf("failed to load existing streams: %w", err)
	}

	existingHashSet := make(map[string]bool)
	for _, s := range existingStreams {
		existingHashSet[s.InfoHash()] = true
	}

	for _, th := range hashes {
		if existingHashSet[th.hash] {
			continue
		}

		newStream, err := stream.NewStream(th.hash, channelName, th.source)
		if err != nil {
			s.logger.Error("failed to create stream", "channel", channelName, "hash", th.hash, "error", err)
			continue
		}

		if err := s.streamRepo.Save(ctx, newStream); err != nil {
			if !errors.Is(err, stream.ErrStreamAlreadyExists) {
				s.logger.Error("failed to save stream", "channel", channelName, "hash", th.hash, "error", err)
			}
			continue
		}
	}

	hashSet := make(map[string]bool)
	for _, th := range hashes {
		hashSet[th.hash] = true
	}

	for _, existingStream := range existingStreams {
		if !hashSet[existingStream.InfoHash()] {
			if err := s.streamRepo.Delete(ctx, existingStream.InfoHash()); err != nil {
				s.logger.Error("failed to delete obsolete stream", "hash", existingStream.InfoHash(), "channel", channelName, "error", err)
			}
		}
	}

	return nil
}

type taggedHash struct {
	hash   string
	source string
}

func tagHashMap(m map[string][]string, source string) map[string][]taggedHash {
	result := make(map[string][]taggedHash, len(m))
	for key, hashes := range m {
		tagged := make([]taggedHash, len(hashes))
		for i, h := range hashes {
			tagged[i] = taggedHash{hash: h, source: source}
		}
		result[key] = tagged
	}
	return result
}

func mergeTaggedHashMaps(maps ...map[string][]taggedHash) map[string][]taggedHash {
	result := make(map[string][]taggedHash)

	for _, m := range maps {
		for channelName, hashes := range m {
			existing := make(map[string]bool)
			for _, th := range result[channelName] {
				existing[th.hash] = true
			}
			for _, th := range hashes {
				if !existing[th.hash] {
					result[channelName] = append(result[channelName], th)
					existing[th.hash] = true
				}
			}
		}
	}

	return result
}
