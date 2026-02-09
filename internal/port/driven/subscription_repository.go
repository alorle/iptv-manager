package driven

import (
	"context"

	"github.com/alorle/iptv-manager/internal/subscription"
)

// SubscriptionRepository defines the interface for subscription persistence operations.
// This is a driven port that will be implemented by concrete adapters (e.g., BoltDB).
type SubscriptionRepository interface {
	// Save persists a subscription. Returns subscription.ErrSubscriptionAlreadyExists
	// if a subscription with the same EPG channel ID already exists.
	Save(ctx context.Context, sub subscription.Subscription) error

	// FindAll retrieves all subscriptions.
	FindAll(ctx context.Context) ([]subscription.Subscription, error)

	// FindByEPGID retrieves a subscription by its EPG channel ID.
	// Returns subscription.ErrSubscriptionNotFound if the subscription does not exist.
	FindByEPGID(ctx context.Context, epgChannelID string) (subscription.Subscription, error)

	// Delete removes a subscription by its EPG channel ID.
	// Returns subscription.ErrSubscriptionNotFound if the subscription does not exist.
	Delete(ctx context.Context, epgChannelID string) error
}
