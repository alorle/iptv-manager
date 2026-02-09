package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/alorle/iptv-manager/internal/epg"
	"github.com/alorle/iptv-manager/internal/port/driven"
	"github.com/alorle/iptv-manager/internal/subscription"
)

// SubscriptionService manages user channel subscriptions.
// It orchestrates operations between the subscription repository and EPG fetcher
// to handle subscription lifecycle and available channel listing.
type SubscriptionService struct {
	subscriptionRepo driven.SubscriptionRepository
	epgFetcher       driven.EPGFetcher
}

// NewSubscriptionService creates a new subscription service with the required dependencies.
func NewSubscriptionService(
	subscriptionRepo driven.SubscriptionRepository,
	epgFetcher driven.EPGFetcher,
) *SubscriptionService {
	return &SubscriptionService{
		subscriptionRepo: subscriptionRepo,
		epgFetcher:       epgFetcher,
	}
}

// Subscribe creates a new subscription for the given EPG channel ID.
// Returns subscription.ErrSubscriptionAlreadyExists if already subscribed.
// Returns subscription.ErrEmptyEPGChannelID if epgChannelID is empty.
func (s *SubscriptionService) Subscribe(ctx context.Context, epgChannelID string) error {
	// Create new subscription (domain validates the ID)
	sub, err := subscription.NewSubscription(epgChannelID)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	// Persist subscription
	if err := s.subscriptionRepo.Save(ctx, sub); err != nil {
		return fmt.Errorf("failed to save subscription: %w", err)
	}

	return nil
}

// Unsubscribe removes the subscription for the given EPG channel ID.
// Returns subscription.ErrSubscriptionNotFound if not subscribed.
func (s *SubscriptionService) Unsubscribe(ctx context.Context, epgChannelID string) error {
	// Delete subscription
	if err := s.subscriptionRepo.Delete(ctx, epgChannelID); err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	return nil
}

// ListSubscriptions retrieves all current subscriptions.
func (s *SubscriptionService) ListSubscriptions(ctx context.Context) ([]subscription.Subscription, error) {
	subscriptions, err := s.subscriptionRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	return subscriptions, nil
}

// ChannelFilter represents filtering criteria for available EPG channels.
type ChannelFilter struct {
	// Category filters channels by category (case-insensitive, partial match)
	Category string
	// SearchTerm filters channels by name (case-insensitive, partial match)
	SearchTerm string
}

// ListAvailableEPGChannels retrieves available EPG channels from the external source,
// filtered by the provided criteria. Returns all channels if filter is empty.
func (s *SubscriptionService) ListAvailableEPGChannels(ctx context.Context, filter ChannelFilter) ([]epg.Channel, error) {
	// Fetch all EPG channels from external source
	channels, err := s.epgFetcher.FetchEPG(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch EPG channels: %w", err)
	}

	// Return all channels if no filter specified
	if filter.Category == "" && filter.SearchTerm == "" {
		return channels, nil
	}

	// Apply filters
	filtered := make([]epg.Channel, 0, len(channels))
	for _, ch := range channels {
		if s.matchesFilter(ch, filter) {
			filtered = append(filtered, ch)
		}
	}

	return filtered, nil
}

// matchesFilter checks if a channel matches the provided filter criteria.
// All non-empty filter fields must match (AND logic).
func (s *SubscriptionService) matchesFilter(ch epg.Channel, filter ChannelFilter) bool {
	// Category filter (case-insensitive, partial match)
	if filter.Category != "" {
		categoryLower := strings.ToLower(ch.Category())
		filterCategoryLower := strings.ToLower(filter.Category)
		if !strings.Contains(categoryLower, filterCategoryLower) {
			return false
		}
	}

	// SearchTerm filter on name (case-insensitive, partial match)
	if filter.SearchTerm != "" {
		nameLower := strings.ToLower(ch.Name())
		searchLower := strings.ToLower(filter.SearchTerm)
		if !strings.Contains(nameLower, searchLower) {
			return false
		}
	}

	return true
}

// IsSubscribed checks if a given EPG channel ID is currently subscribed.
// Returns false if subscription doesn't exist or any error occurs.
func (s *SubscriptionService) IsSubscribed(ctx context.Context, epgChannelID string) bool {
	_, err := s.subscriptionRepo.FindByEPGID(ctx, epgChannelID)
	// If subscription exists without error, return true
	// If not found or any error, return false
	return err == nil
}

// GetSubscription retrieves a specific subscription by EPG channel ID.
// Returns subscription.ErrSubscriptionNotFound if not subscribed.
func (s *SubscriptionService) GetSubscription(ctx context.Context, epgChannelID string) (subscription.Subscription, error) {
	sub, err := s.subscriptionRepo.FindByEPGID(ctx, epgChannelID)
	if err != nil {
		if errors.Is(err, subscription.ErrSubscriptionNotFound) {
			return subscription.Subscription{}, subscription.ErrSubscriptionNotFound
		}
		return subscription.Subscription{}, fmt.Errorf("failed to get subscription: %w", err)
	}

	return sub, nil
}
