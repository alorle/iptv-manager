package subscription_test

import (
	"errors"
	"testing"

	"github.com/alorle/iptv-manager/internal/subscription"
)

func TestNewSubscription(t *testing.T) {
	tests := []struct {
		name          string
		epgChannelID  string
		wantChannelID string
		wantEnabled   bool
		wantOverride  bool
		wantError     error
	}{
		{
			name:          "valid subscription",
			epgChannelID:  "ch-123",
			wantChannelID: "ch-123",
			wantEnabled:   true,
			wantOverride:  false,
			wantError:     nil,
		},
		{
			name:          "valid subscription with trimmed whitespace",
			epgChannelID:  "  ch-123  ",
			wantChannelID: "ch-123",
			wantEnabled:   true,
			wantOverride:  false,
			wantError:     nil,
		},
		{
			name:         "empty epg channel id",
			epgChannelID: "",
			wantError:    subscription.ErrEmptyEPGChannelID,
		},
		{
			name:         "whitespace only epg channel id",
			epgChannelID: "   ",
			wantError:    subscription.ErrEmptyEPGChannelID,
		},
		{
			name:         "tabs in epg channel id",
			epgChannelID: "\t\t",
			wantError:    subscription.ErrEmptyEPGChannelID,
		},
		{
			name:         "newlines in epg channel id",
			epgChannelID: "\n\n",
			wantError:    subscription.ErrEmptyEPGChannelID,
		},
		{
			name:         "mixed whitespace in epg channel id",
			epgChannelID: " \t\n ",
			wantError:    subscription.ErrEmptyEPGChannelID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, err := subscription.NewSubscription(tt.epgChannelID)

			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Errorf("NewSubscription() error = %v, wantError %v", err, tt.wantError)
				}
				return
			}

			if err != nil {
				t.Errorf("NewSubscription() unexpected error = %v", err)
				return
			}

			if got := sub.EPGChannelID(); got != tt.wantChannelID {
				t.Errorf("Subscription.EPGChannelID() = %q, want %q", got, tt.wantChannelID)
			}

			if got := sub.IsEnabled(); got != tt.wantEnabled {
				t.Errorf("Subscription.IsEnabled() = %v, want %v", got, tt.wantEnabled)
			}

			if got := sub.HasManualOverride(); got != tt.wantOverride {
				t.Errorf("Subscription.HasManualOverride() = %v, want %v", got, tt.wantOverride)
			}
		})
	}
}

func TestSubscriptionGetters(t *testing.T) {
	sub, err := subscription.NewSubscription("ch-123")
	if err != nil {
		t.Fatalf("NewSubscription() unexpected error = %v", err)
	}

	if got := sub.EPGChannelID(); got != "ch-123" {
		t.Errorf("EPGChannelID() = %q, want %q", got, "ch-123")
	}

	if got := sub.IsEnabled(); got != true {
		t.Errorf("IsEnabled() = %v, want %v", got, true)
	}

	if got := sub.HasManualOverride(); got != false {
		t.Errorf("HasManualOverride() = %v, want %v", got, false)
	}
}

func TestSubscriptionEnable(t *testing.T) {
	tests := []struct {
		name            string
		initialEnabled  bool
		initialOverride bool
		wantEnabled     bool
		wantOverride    bool
	}{
		{
			name:            "enable already enabled subscription",
			initialEnabled:  true,
			initialOverride: false,
			wantEnabled:     true,
			wantOverride:    false,
		},
		{
			name:            "enable disabled subscription (creates override)",
			initialEnabled:  false,
			initialOverride: false,
			wantEnabled:     true,
			wantOverride:    true,
		},
		{
			name:            "enable disabled subscription with existing override",
			initialEnabled:  false,
			initialOverride: true,
			wantEnabled:     true,
			wantOverride:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create initial subscription
			sub, err := subscription.NewSubscription("ch-123")
			if err != nil {
				t.Fatalf("NewSubscription() unexpected error = %v", err)
			}

			// Set up initial state
			if !tt.initialEnabled {
				sub = sub.Disable()
			}
			if tt.initialOverride && tt.initialEnabled {
				// Create override on enabled subscription
				sub = sub.Disable().Enable()
			}

			// Test Enable
			result := sub.Enable()

			if got := result.IsEnabled(); got != tt.wantEnabled {
				t.Errorf("after Enable() IsEnabled() = %v, want %v", got, tt.wantEnabled)
			}

			if got := result.HasManualOverride(); got != tt.wantOverride {
				t.Errorf("after Enable() HasManualOverride() = %v, want %v", got, tt.wantOverride)
			}

			// Verify immutability - original should be unchanged
			if tt.initialEnabled != sub.IsEnabled() {
				t.Errorf("original subscription was mutated: IsEnabled() = %v, want %v", sub.IsEnabled(), tt.initialEnabled)
			}
		})
	}
}

func TestSubscriptionDisable(t *testing.T) {
	tests := []struct {
		name            string
		initialEnabled  bool
		initialOverride bool
		wantEnabled     bool
		wantOverride    bool
	}{
		{
			name:            "disable enabled subscription (creates override)",
			initialEnabled:  true,
			initialOverride: false,
			wantEnabled:     false,
			wantOverride:    true,
		},
		{
			name:            "disable already disabled subscription with override",
			initialEnabled:  false,
			initialOverride: true,
			wantEnabled:     false,
			wantOverride:    true,
		},
		{
			name:            "disable enabled subscription with existing override",
			initialEnabled:  true,
			initialOverride: true,
			wantEnabled:     false,
			wantOverride:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create initial subscription
			sub, err := subscription.NewSubscription("ch-123")
			if err != nil {
				t.Fatalf("NewSubscription() unexpected error = %v", err)
			}

			// Set up initial state
			if !tt.initialEnabled {
				sub = sub.Disable()
			}
			if tt.initialOverride && tt.initialEnabled {
				// Create override on enabled subscription
				sub = sub.Disable().Enable()
			}

			// Test Disable
			result := sub.Disable()

			if got := result.IsEnabled(); got != tt.wantEnabled {
				t.Errorf("after Disable() IsEnabled() = %v, want %v", got, tt.wantEnabled)
			}

			if got := result.HasManualOverride(); got != tt.wantOverride {
				t.Errorf("after Disable() HasManualOverride() = %v, want %v", got, tt.wantOverride)
			}

			// Verify immutability - original should be unchanged
			if tt.initialEnabled != sub.IsEnabled() {
				t.Errorf("original subscription was mutated: IsEnabled() = %v, want %v", sub.IsEnabled(), tt.initialEnabled)
			}
		})
	}
}

func TestSubscriptionClearOverride(t *testing.T) {
	tests := []struct {
		name            string
		initialEnabled  bool
		initialOverride bool
		wantEnabled     bool
		wantOverride    bool
	}{
		{
			name:            "clear override on enabled subscription",
			initialEnabled:  true,
			initialOverride: false,
			wantEnabled:     true,
			wantOverride:    false,
		},
		{
			name:            "clear override on disabled subscription (re-enables)",
			initialEnabled:  false,
			initialOverride: true,
			wantEnabled:     true,
			wantOverride:    false,
		},
		{
			name:            "clear override on enabled subscription with override",
			initialEnabled:  true,
			initialOverride: true,
			wantEnabled:     true,
			wantOverride:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create initial subscription
			sub, err := subscription.NewSubscription("ch-123")
			if err != nil {
				t.Fatalf("NewSubscription() unexpected error = %v", err)
			}

			// Set up initial state
			if !tt.initialEnabled {
				sub = sub.Disable()
			}
			if tt.initialOverride && tt.initialEnabled {
				// Create override on enabled subscription
				sub = sub.Disable().Enable()
			}

			// Test ClearOverride
			result := sub.ClearOverride()

			if got := result.IsEnabled(); got != tt.wantEnabled {
				t.Errorf("after ClearOverride() IsEnabled() = %v, want %v", got, tt.wantEnabled)
			}

			if got := result.HasManualOverride(); got != tt.wantOverride {
				t.Errorf("after ClearOverride() HasManualOverride() = %v, want %v", got, tt.wantOverride)
			}

			// EPG channel ID should remain the same
			if got := result.EPGChannelID(); got != "ch-123" {
				t.Errorf("after ClearOverride() EPGChannelID() = %q, want %q", got, "ch-123")
			}

			// Verify immutability - original should be unchanged
			if tt.initialEnabled != sub.IsEnabled() {
				t.Errorf("original subscription was mutated: IsEnabled() = %v, want %v", sub.IsEnabled(), tt.initialEnabled)
			}
		})
	}
}

func TestSubscriptionImmutability(t *testing.T) {
	original, err := subscription.NewSubscription("ch-123")
	if err != nil {
		t.Fatalf("NewSubscription() unexpected error = %v", err)
	}

	// Perform various operations
	disabled := original.Disable()
	enabled := original.Enable()
	cleared := original.ClearOverride()

	// Original should remain unchanged
	if !original.IsEnabled() {
		t.Errorf("original.IsEnabled() = false, want true")
	}
	if original.HasManualOverride() {
		t.Errorf("original.HasManualOverride() = true, want false")
	}

	// Each operation should produce the expected state
	if disabled.IsEnabled() {
		t.Errorf("disabled.IsEnabled() = true, want false")
	}
	if !disabled.HasManualOverride() {
		t.Errorf("disabled.HasManualOverride() = false, want true")
	}

	if !enabled.IsEnabled() {
		t.Errorf("enabled.IsEnabled() = false, want true")
	}

	if !cleared.IsEnabled() {
		t.Errorf("cleared.IsEnabled() = false, want true")
	}
	if cleared.HasManualOverride() {
		t.Errorf("cleared.HasManualOverride() = true, want false")
	}

	// All should maintain the same EPG channel ID
	if original.EPGChannelID() != "ch-123" {
		t.Errorf("original.EPGChannelID() = %q, want %q", original.EPGChannelID(), "ch-123")
	}
	if disabled.EPGChannelID() != "ch-123" {
		t.Errorf("disabled.EPGChannelID() = %q, want %q", disabled.EPGChannelID(), "ch-123")
	}
	if enabled.EPGChannelID() != "ch-123" {
		t.Errorf("enabled.EPGChannelID() = %q, want %q", enabled.EPGChannelID(), "ch-123")
	}
	if cleared.EPGChannelID() != "ch-123" {
		t.Errorf("cleared.EPGChannelID() = %q, want %q", cleared.EPGChannelID(), "ch-123")
	}
}

func TestSubscriptionWorkflow(t *testing.T) {
	// Test a realistic workflow of subscription state changes

	// Create a new subscription (enabled by default, no override)
	sub, err := subscription.NewSubscription("ch-123")
	if err != nil {
		t.Fatalf("NewSubscription() unexpected error = %v", err)
	}

	if !sub.IsEnabled() || sub.HasManualOverride() {
		t.Fatalf("initial state: enabled=%v override=%v, want enabled=true override=false",
			sub.IsEnabled(), sub.HasManualOverride())
	}

	// User disables the subscription (creates manual override)
	sub = sub.Disable()
	if sub.IsEnabled() || !sub.HasManualOverride() {
		t.Fatalf("after Disable: enabled=%v override=%v, want enabled=false override=true",
			sub.IsEnabled(), sub.HasManualOverride())
	}

	// User re-enables the subscription (maintains manual override)
	sub = sub.Enable()
	if !sub.IsEnabled() || !sub.HasManualOverride() {
		t.Fatalf("after Enable: enabled=%v override=%v, want enabled=true override=true",
			sub.IsEnabled(), sub.HasManualOverride())
	}

	// User clears the override (returns to default enabled state)
	sub = sub.ClearOverride()
	if !sub.IsEnabled() || sub.HasManualOverride() {
		t.Fatalf("after ClearOverride: enabled=%v override=%v, want enabled=true override=false",
			sub.IsEnabled(), sub.HasManualOverride())
	}
}

func TestSubscriptionDomainErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "ErrEmptyEPGChannelID",
			err:  subscription.ErrEmptyEPGChannelID,
			msg:  "subscription epg channel id cannot be empty",
		},
		{
			name: "ErrSubscriptionNotFound",
			err:  subscription.ErrSubscriptionNotFound,
			msg:  "subscription not found",
		},
		{
			name: "ErrSubscriptionAlreadyExists",
			err:  subscription.ErrSubscriptionAlreadyExists,
			msg:  "subscription already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.msg)
			}
		})
	}
}
