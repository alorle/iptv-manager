package application

import (
	"context"
	"errors"
	"testing"

	"github.com/alorle/iptv-manager/internal/channel"
	"github.com/alorle/iptv-manager/internal/stream"
)

func TestStreamService_CreateStream(t *testing.T) {
	t.Run("creates stream successfully", func(t *testing.T) {
		ch, _ := channel.NewChannel("TestChannel")
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				if name != "TestChannel" {
					t.Errorf("expected channel name 'TestChannel', got %q", name)
				}
				return ch, nil
			},
		}
		streamRepo := &mockStreamRepository{
			saveFunc: func(ctx context.Context, s stream.Stream) error {
				if s.InfoHash() != "abc123" {
					t.Errorf("expected infohash 'abc123', got %q", s.InfoHash())
				}
				if s.ChannelName() != "TestChannel" {
					t.Errorf("expected channel name 'TestChannel', got %q", s.ChannelName())
				}
				return nil
			},
		}
		service := NewStreamService(streamRepo, channelRepo)

		st, err := service.CreateStream(context.Background(), "abc123", "TestChannel")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if st.InfoHash() != "abc123" {
			t.Errorf("expected infohash 'abc123', got %q", st.InfoHash())
		}
		if st.ChannelName() != "TestChannel" {
			t.Errorf("expected channel name 'TestChannel', got %q", st.ChannelName())
		}
	})

	t.Run("returns error if channel not found", func(t *testing.T) {
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				return channel.Channel{}, channel.ErrChannelNotFound
			},
		}
		streamRepo := &mockStreamRepository{}
		service := NewStreamService(streamRepo, channelRepo)

		_, err := service.CreateStream(context.Background(), "abc123", "NonExistent")
		if !errors.Is(err, channel.ErrChannelNotFound) {
			t.Errorf("expected ErrChannelNotFound, got %v", err)
		}
	})

	t.Run("returns error for empty infohash", func(t *testing.T) {
		ch, _ := channel.NewChannel("TestChannel")
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				return ch, nil
			},
		}
		streamRepo := &mockStreamRepository{}
		service := NewStreamService(streamRepo, channelRepo)

		_, err := service.CreateStream(context.Background(), "", "TestChannel")
		if !errors.Is(err, stream.ErrEmptyInfoHash) {
			t.Errorf("expected ErrEmptyInfoHash, got %v", err)
		}
	})

	t.Run("returns error for empty channel name", func(t *testing.T) {
		ch, _ := channel.NewChannel("TestChannel")
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				return ch, nil
			},
		}
		streamRepo := &mockStreamRepository{}
		service := NewStreamService(streamRepo, channelRepo)

		_, err := service.CreateStream(context.Background(), "abc123", "")
		if !errors.Is(err, stream.ErrEmptyChannelName) {
			t.Errorf("expected ErrEmptyChannelName, got %v", err)
		}
	})

	t.Run("returns error if stream already exists", func(t *testing.T) {
		ch, _ := channel.NewChannel("TestChannel")
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				return ch, nil
			},
		}
		streamRepo := &mockStreamRepository{
			saveFunc: func(ctx context.Context, s stream.Stream) error {
				return stream.ErrStreamAlreadyExists
			},
		}
		service := NewStreamService(streamRepo, channelRepo)

		_, err := service.CreateStream(context.Background(), "abc123", "TestChannel")
		if !errors.Is(err, stream.ErrStreamAlreadyExists) {
			t.Errorf("expected ErrStreamAlreadyExists, got %v", err)
		}
	})
}

func TestStreamService_GetStream(t *testing.T) {
	t.Run("gets stream successfully", func(t *testing.T) {
		expectedStream, _ := stream.NewStream("abc123", "TestChannel")
		streamRepo := &mockStreamRepository{
			findByInfoHashFunc: func(ctx context.Context, infoHash string) (stream.Stream, error) {
				if infoHash != "abc123" {
					t.Errorf("expected infohash 'abc123', got %q", infoHash)
				}
				return expectedStream, nil
			},
		}
		channelRepo := &mockChannelRepository{}
		service := NewStreamService(streamRepo, channelRepo)

		st, err := service.GetStream(context.Background(), "abc123")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if st.InfoHash() != "abc123" {
			t.Errorf("expected infohash 'abc123', got %q", st.InfoHash())
		}
		if st.ChannelName() != "TestChannel" {
			t.Errorf("expected channel name 'TestChannel', got %q", st.ChannelName())
		}
	})

	t.Run("returns error if stream not found", func(t *testing.T) {
		streamRepo := &mockStreamRepository{
			findByInfoHashFunc: func(ctx context.Context, infoHash string) (stream.Stream, error) {
				return stream.Stream{}, stream.ErrStreamNotFound
			},
		}
		channelRepo := &mockChannelRepository{}
		service := NewStreamService(streamRepo, channelRepo)

		_, err := service.GetStream(context.Background(), "nonexistent")
		if !errors.Is(err, stream.ErrStreamNotFound) {
			t.Errorf("expected ErrStreamNotFound, got %v", err)
		}
	})
}

func TestStreamService_ListStreams(t *testing.T) {
	t.Run("lists all streams successfully", func(t *testing.T) {
		st1, _ := stream.NewStream("abc123", "Channel1")
		st2, _ := stream.NewStream("def456", "Channel2")
		expectedStreams := []stream.Stream{st1, st2}

		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return expectedStreams, nil
			},
		}
		channelRepo := &mockChannelRepository{}
		service := NewStreamService(streamRepo, channelRepo)

		streams, err := service.ListStreams(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(streams) != 2 {
			t.Fatalf("expected 2 streams, got %d", len(streams))
		}
		if streams[0].InfoHash() != "abc123" || streams[1].InfoHash() != "def456" {
			t.Errorf("unexpected stream infohashes: %q, %q", streams[0].InfoHash(), streams[1].InfoHash())
		}
	})

	t.Run("returns empty list when no streams exist", func(t *testing.T) {
		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return []stream.Stream{}, nil
			},
		}
		channelRepo := &mockChannelRepository{}
		service := NewStreamService(streamRepo, channelRepo)

		streams, err := service.ListStreams(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(streams) != 0 {
			t.Errorf("expected empty list, got %d streams", len(streams))
		}
	})
}

func TestStreamService_DeleteStream(t *testing.T) {
	t.Run("deletes stream successfully", func(t *testing.T) {
		deleteCalled := false
		streamRepo := &mockStreamRepository{
			deleteFunc: func(ctx context.Context, infoHash string) error {
				deleteCalled = true
				if infoHash != "abc123" {
					t.Errorf("expected infohash 'abc123', got %q", infoHash)
				}
				return nil
			},
		}
		channelRepo := &mockChannelRepository{}
		service := NewStreamService(streamRepo, channelRepo)

		err := service.DeleteStream(context.Background(), "abc123")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !deleteCalled {
			t.Error("delete was not called")
		}
	})

	t.Run("returns error if stream not found", func(t *testing.T) {
		streamRepo := &mockStreamRepository{
			deleteFunc: func(ctx context.Context, infoHash string) error {
				return stream.ErrStreamNotFound
			},
		}
		channelRepo := &mockChannelRepository{}
		service := NewStreamService(streamRepo, channelRepo)

		err := service.DeleteStream(context.Background(), "nonexistent")
		if !errors.Is(err, stream.ErrStreamNotFound) {
			t.Errorf("expected ErrStreamNotFound, got %v", err)
		}
	})
}
