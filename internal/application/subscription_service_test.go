package application

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/alorle/iptv-manager/internal/epg"
	"github.com/alorle/iptv-manager/internal/subscription"
)

type countingEPGFetcher struct {
	channels  []epg.Channel
	callCount atomic.Int32
}

func (f *countingEPGFetcher) FetchEPG(ctx context.Context) ([]epg.Channel, error) {
	f.callCount.Add(1)
	return f.channels, nil
}

type stubSubscriptionRepo struct{}

func (r *stubSubscriptionRepo) Save(ctx context.Context, sub subscription.Subscription) error {
	return nil
}
func (r *stubSubscriptionRepo) FindAll(ctx context.Context) ([]subscription.Subscription, error) {
	return nil, nil
}
func (r *stubSubscriptionRepo) FindByEPGID(ctx context.Context, epgID string) (subscription.Subscription, error) {
	return subscription.Subscription{}, subscription.ErrSubscriptionNotFound
}
func (r *stubSubscriptionRepo) Delete(ctx context.Context, epgID string) error { return nil }

func TestSubscriptionService_ListAvailableEPGChannels_Caching(t *testing.T) {
	ch, _ := epg.NewChannel("ch1", "Channel 1", "", "General", "en", "ch1")
	fetcher := &countingEPGFetcher{channels: []epg.Channel{ch}}
	service := NewSubscriptionService(&stubSubscriptionRepo{}, fetcher)

	ctx := context.Background()

	// First call should fetch
	channels1, err := service.ListAvailableEPGChannels(ctx, ChannelFilter{})
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if len(channels1) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(channels1))
	}
	if fetcher.callCount.Load() != 1 {
		t.Fatalf("expected 1 fetch call, got %d", fetcher.callCount.Load())
	}

	// Second call should use cache
	channels2, err := service.ListAvailableEPGChannels(ctx, ChannelFilter{})
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if len(channels2) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(channels2))
	}
	if fetcher.callCount.Load() != 1 {
		t.Fatalf("expected still 1 fetch call after cache hit, got %d", fetcher.callCount.Load())
	}
}
