package application

import (
	"context"
	"errors"
	"testing"

	"github.com/alorle/iptv-manager/internal/channel"
	"github.com/alorle/iptv-manager/internal/stream"
)

// mockChannelRepository is a mock implementation of driven.ChannelRepository for testing.
type mockChannelRepository struct {
	saveFunc       func(ctx context.Context, ch channel.Channel) error
	updateFunc     func(ctx context.Context, ch channel.Channel) error
	findByNameFunc func(ctx context.Context, name string) (channel.Channel, error)
	findAllFunc    func(ctx context.Context) ([]channel.Channel, error)
	deleteFunc     func(ctx context.Context, name string) error
	pingFunc       func(ctx context.Context) error
}

func (m *mockChannelRepository) Save(ctx context.Context, ch channel.Channel) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, ch)
	}
	return nil
}

func (m *mockChannelRepository) Update(ctx context.Context, ch channel.Channel) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, ch)
	}
	return nil
}

func (m *mockChannelRepository) FindByName(ctx context.Context, name string) (channel.Channel, error) {
	if m.findByNameFunc != nil {
		return m.findByNameFunc(ctx, name)
	}
	return channel.Channel{}, channel.ErrChannelNotFound
}

func (m *mockChannelRepository) FindAll(ctx context.Context) ([]channel.Channel, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(ctx)
	}
	return []channel.Channel{}, nil
}

func (m *mockChannelRepository) Delete(ctx context.Context, name string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, name)
	}
	return nil
}

func (m *mockChannelRepository) Ping(ctx context.Context) error {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return nil
}

// mockStreamRepository is a mock implementation of driven.StreamRepository for testing.
type mockStreamRepository struct {
	saveFunc                func(ctx context.Context, s stream.Stream) error
	findByInfoHashFunc      func(ctx context.Context, infoHash string) (stream.Stream, error)
	findAllFunc             func(ctx context.Context) ([]stream.Stream, error)
	findByChannelNameFunc   func(ctx context.Context, channelName string) ([]stream.Stream, error)
	deleteFunc              func(ctx context.Context, infoHash string) error
	deleteByChannelNameFunc func(ctx context.Context, channelName string) error
}

func (m *mockStreamRepository) Save(ctx context.Context, s stream.Stream) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, s)
	}
	return nil
}

func (m *mockStreamRepository) FindByInfoHash(ctx context.Context, infoHash string) (stream.Stream, error) {
	if m.findByInfoHashFunc != nil {
		return m.findByInfoHashFunc(ctx, infoHash)
	}
	return stream.Stream{}, stream.ErrStreamNotFound
}

func (m *mockStreamRepository) FindAll(ctx context.Context) ([]stream.Stream, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(ctx)
	}
	return []stream.Stream{}, nil
}

func (m *mockStreamRepository) FindByChannelName(ctx context.Context, channelName string) ([]stream.Stream, error) {
	if m.findByChannelNameFunc != nil {
		return m.findByChannelNameFunc(ctx, channelName)
	}
	return []stream.Stream{}, nil
}

func (m *mockStreamRepository) Delete(ctx context.Context, infoHash string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, infoHash)
	}
	return nil
}

func (m *mockStreamRepository) DeleteByChannelName(ctx context.Context, channelName string) error {
	if m.deleteByChannelNameFunc != nil {
		return m.deleteByChannelNameFunc(ctx, channelName)
	}
	return nil
}

func TestChannelService_CreateChannel(t *testing.T) {
	t.Run("creates channel successfully", func(t *testing.T) {
		channelRepo := &mockChannelRepository{
			saveFunc: func(ctx context.Context, ch channel.Channel) error {
				if ch.Name() != "TestChannel" {
					t.Errorf("expected channel name 'TestChannel', got %q", ch.Name())
				}
				return nil
			},
		}
		streamRepo := &mockStreamRepository{}
		service := NewChannelService(channelRepo, streamRepo)

		ch, err := service.CreateChannel(context.Background(), "TestChannel")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if ch.Name() != "TestChannel" {
			t.Errorf("expected channel name 'TestChannel', got %q", ch.Name())
		}
	})

	t.Run("returns error for empty name", func(t *testing.T) {
		channelRepo := &mockChannelRepository{}
		streamRepo := &mockStreamRepository{}
		service := NewChannelService(channelRepo, streamRepo)

		_, err := service.CreateChannel(context.Background(), "")
		if !errors.Is(err, channel.ErrEmptyName) {
			t.Errorf("expected ErrEmptyName, got %v", err)
		}
	})

	t.Run("returns error if channel already exists", func(t *testing.T) {
		channelRepo := &mockChannelRepository{
			saveFunc: func(ctx context.Context, ch channel.Channel) error {
				return channel.ErrChannelAlreadyExists
			},
		}
		streamRepo := &mockStreamRepository{}
		service := NewChannelService(channelRepo, streamRepo)

		_, err := service.CreateChannel(context.Background(), "TestChannel")
		if !errors.Is(err, channel.ErrChannelAlreadyExists) {
			t.Errorf("expected ErrChannelAlreadyExists, got %v", err)
		}
	})
}

func TestChannelService_GetChannel(t *testing.T) {
	t.Run("gets channel successfully", func(t *testing.T) {
		expectedChannel, _ := channel.NewChannel("TestChannel")
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				if name != "TestChannel" {
					t.Errorf("expected name 'TestChannel', got %q", name)
				}
				return expectedChannel, nil
			},
		}
		streamRepo := &mockStreamRepository{}
		service := NewChannelService(channelRepo, streamRepo)

		ch, err := service.GetChannel(context.Background(), "TestChannel")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if ch.Name() != "TestChannel" {
			t.Errorf("expected channel name 'TestChannel', got %q", ch.Name())
		}
	})

	t.Run("returns error if channel not found", func(t *testing.T) {
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				return channel.Channel{}, channel.ErrChannelNotFound
			},
		}
		streamRepo := &mockStreamRepository{}
		service := NewChannelService(channelRepo, streamRepo)

		_, err := service.GetChannel(context.Background(), "NonExistent")
		if !errors.Is(err, channel.ErrChannelNotFound) {
			t.Errorf("expected ErrChannelNotFound, got %v", err)
		}
	})
}

func TestChannelService_ListChannels(t *testing.T) {
	t.Run("lists all channels successfully", func(t *testing.T) {
		ch1, _ := channel.NewChannel("Channel1")
		ch2, _ := channel.NewChannel("Channel2")
		expectedChannels := []channel.Channel{ch1, ch2}

		channelRepo := &mockChannelRepository{
			findAllFunc: func(ctx context.Context) ([]channel.Channel, error) {
				return expectedChannels, nil
			},
		}
		streamRepo := &mockStreamRepository{}
		service := NewChannelService(channelRepo, streamRepo)

		channels, err := service.ListChannels(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(channels) != 2 {
			t.Fatalf("expected 2 channels, got %d", len(channels))
		}
		if channels[0].Name() != "Channel1" || channels[1].Name() != "Channel2" {
			t.Errorf("unexpected channel names: %q, %q", channels[0].Name(), channels[1].Name())
		}
	})

	t.Run("returns empty list when no channels exist", func(t *testing.T) {
		channelRepo := &mockChannelRepository{
			findAllFunc: func(ctx context.Context) ([]channel.Channel, error) {
				return []channel.Channel{}, nil
			},
		}
		streamRepo := &mockStreamRepository{}
		service := NewChannelService(channelRepo, streamRepo)

		channels, err := service.ListChannels(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(channels) != 0 {
			t.Errorf("expected empty list, got %d channels", len(channels))
		}
	})
}

func TestChannelService_DeleteChannel(t *testing.T) {
	t.Run("deletes channel and streams successfully", func(t *testing.T) {
		streamDeleteCalled := false
		channelDeleteCalled := false

		ch, _ := channel.NewChannel("TestChannel")
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				if name != "TestChannel" {
					t.Errorf("expected name 'TestChannel', got %q", name)
				}
				return ch, nil
			},
			deleteFunc: func(ctx context.Context, name string) error {
				channelDeleteCalled = true
				if !streamDeleteCalled {
					t.Error("channel delete called before stream delete")
				}
				return nil
			},
		}
		streamRepo := &mockStreamRepository{
			deleteByChannelNameFunc: func(ctx context.Context, channelName string) error {
				streamDeleteCalled = true
				if channelName != "TestChannel" {
					t.Errorf("expected channel name 'TestChannel', got %q", channelName)
				}
				return nil
			},
		}
		service := NewChannelService(channelRepo, streamRepo)

		err := service.DeleteChannel(context.Background(), "TestChannel")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !streamDeleteCalled {
			t.Error("stream delete was not called")
		}
		if !channelDeleteCalled {
			t.Error("channel delete was not called")
		}
	})

	t.Run("returns error if channel not found", func(t *testing.T) {
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				return channel.Channel{}, channel.ErrChannelNotFound
			},
		}
		streamRepo := &mockStreamRepository{}
		service := NewChannelService(channelRepo, streamRepo)

		err := service.DeleteChannel(context.Background(), "NonExistent")
		if !errors.Is(err, channel.ErrChannelNotFound) {
			t.Errorf("expected ErrChannelNotFound, got %v", err)
		}
	})

	t.Run("does not delete channel if stream deletion fails", func(t *testing.T) {
		channelDeleteCalled := false
		streamDeleteError := errors.New("stream delete failed")

		ch, _ := channel.NewChannel("TestChannel")
		channelRepo := &mockChannelRepository{
			findByNameFunc: func(ctx context.Context, name string) (channel.Channel, error) {
				return ch, nil
			},
			deleteFunc: func(ctx context.Context, name string) error {
				channelDeleteCalled = true
				return nil
			},
		}
		streamRepo := &mockStreamRepository{
			deleteByChannelNameFunc: func(ctx context.Context, channelName string) error {
				return streamDeleteError
			},
		}
		service := NewChannelService(channelRepo, streamRepo)

		err := service.DeleteChannel(context.Background(), "TestChannel")
		if !errors.Is(err, streamDeleteError) {
			t.Errorf("expected stream delete error, got %v", err)
		}
		if channelDeleteCalled {
			t.Error("channel delete should not be called if stream delete fails")
		}
	})
}
