package subscription

import (
	"strings"
)

// Subscription represents a user's subscription to an EPG channel.
// It tracks which EPG channels are enabled and whether the user has manually overridden the default state.
type Subscription struct {
	epgChannelID   string
	enabled        bool
	manualOverride bool
}

// NewSubscription creates a new Subscription for the given EPG channel.
// The subscription is enabled by default with no manual override.
// Returns ErrEmptyEPGChannelID if the epgChannelID is empty or contains only whitespace.
func NewSubscription(epgChannelID string) (Subscription, error) {
	trimmedID := strings.TrimSpace(epgChannelID)
	if trimmedID == "" {
		return Subscription{}, ErrEmptyEPGChannelID
	}

	return Subscription{
		epgChannelID:   trimmedID,
		enabled:        true,
		manualOverride: false,
	}, nil
}

// EPGChannelID returns the EPG channel identifier this subscription refers to.
func (s Subscription) EPGChannelID() string {
	return s.epgChannelID
}

// IsEnabled returns whether this subscription is currently enabled.
func (s Subscription) IsEnabled() bool {
	return s.enabled
}

// HasManualOverride returns whether the user has manually changed the enabled state.
func (s Subscription) HasManualOverride() bool {
	return s.manualOverride
}

// Enable enables the subscription.
// If the subscription was previously disabled, this is considered a manual override.
func (s Subscription) Enable() Subscription {
	return Subscription{
		epgChannelID:   s.epgChannelID,
		enabled:        true,
		manualOverride: !s.enabled || s.manualOverride,
	}
}

// Disable disables the subscription.
// If the subscription was previously enabled, this is considered a manual override.
func (s Subscription) Disable() Subscription {
	return Subscription{
		epgChannelID:   s.epgChannelID,
		enabled:        false,
		manualOverride: s.enabled || s.manualOverride,
	}
}

// ClearOverride resets the manual override flag and enables the subscription.
// This returns the subscription to its default enabled state.
func (s Subscription) ClearOverride() Subscription {
	return Subscription{
		epgChannelID:   s.epgChannelID,
		enabled:        true,
		manualOverride: false,
	}
}
