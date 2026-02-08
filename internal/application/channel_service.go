package application

import (
	"context"

	"github.com/alorle/iptv-manager/internal/channel"
	"github.com/alorle/iptv-manager/internal/port/driven"
)

// ChannelService provides use cases for channel management.
// It depends only on domain packages and port interfaces.
type ChannelService struct {
	channelRepo driven.ChannelRepository
	streamRepo  driven.StreamRepository
}

// NewChannelService creates a new ChannelService with the given repositories.
func NewChannelService(channelRepo driven.ChannelRepository, streamRepo driven.StreamRepository) *ChannelService {
	return &ChannelService{
		channelRepo: channelRepo,
		streamRepo:  streamRepo,
	}
}

// CreateChannel creates a new channel with the given name.
// Returns channel.ErrEmptyName if the name is invalid.
// Returns channel.ErrChannelAlreadyExists if a channel with the same name already exists.
func (s *ChannelService) CreateChannel(ctx context.Context, name string) (channel.Channel, error) {
	ch, err := channel.NewChannel(name)
	if err != nil {
		return channel.Channel{}, err
	}

	if err := s.channelRepo.Save(ctx, ch); err != nil {
		return channel.Channel{}, err
	}

	return ch, nil
}

// GetChannel retrieves a channel by its name.
// Returns channel.ErrChannelNotFound if the channel does not exist.
func (s *ChannelService) GetChannel(ctx context.Context, name string) (channel.Channel, error) {
	return s.channelRepo.FindByName(ctx, name)
}

// ListChannels retrieves all channels.
func (s *ChannelService) ListChannels(ctx context.Context) ([]channel.Channel, error) {
	return s.channelRepo.FindAll(ctx)
}

// DeleteChannel removes a channel and all its associated streams (cascade delete).
// Returns channel.ErrChannelNotFound if the channel does not exist.
// If the channel exists but stream deletion fails, the error is returned and the channel is not deleted.
func (s *ChannelService) DeleteChannel(ctx context.Context, name string) error {
	// Verify the channel exists first
	_, err := s.channelRepo.FindByName(ctx, name)
	if err != nil {
		return err
	}

	// Delete all streams associated with this channel first (cascade delete)
	if err := s.streamRepo.DeleteByChannelName(ctx, name); err != nil {
		return err
	}

	// Delete the channel
	return s.channelRepo.Delete(ctx, name)
}
