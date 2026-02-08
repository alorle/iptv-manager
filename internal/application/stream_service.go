package application

import (
	"context"

	"github.com/alorle/iptv-manager/internal/port/driven"
	"github.com/alorle/iptv-manager/internal/stream"
)

// StreamService provides use cases for stream management.
// It depends only on domain packages and port interfaces.
type StreamService struct {
	streamRepo  driven.StreamRepository
	channelRepo driven.ChannelRepository
}

// NewStreamService creates a new StreamService with the given repositories.
func NewStreamService(streamRepo driven.StreamRepository, channelRepo driven.ChannelRepository) *StreamService {
	return &StreamService{
		streamRepo:  streamRepo,
		channelRepo: channelRepo,
	}
}

// CreateStream creates a new stream with the given infohash and channel name.
// It validates that the channel exists before creating the stream.
// Returns stream.ErrEmptyInfoHash if the infohash is invalid.
// Returns stream.ErrEmptyChannelName if the channel name is invalid.
// Returns channel.ErrChannelNotFound if the referenced channel does not exist.
// Returns stream.ErrStreamAlreadyExists if a stream with the same infohash already exists.
func (s *StreamService) CreateStream(ctx context.Context, infoHash, channelName string) (stream.Stream, error) {
	// Validate the channel exists
	_, err := s.channelRepo.FindByName(ctx, channelName)
	if err != nil {
		return stream.Stream{}, err
	}

	// Create the stream entity
	st, err := stream.NewStream(infoHash, channelName)
	if err != nil {
		return stream.Stream{}, err
	}

	// Save the stream
	if err := s.streamRepo.Save(ctx, st); err != nil {
		return stream.Stream{}, err
	}

	return st, nil
}

// GetStream retrieves a stream by its infohash.
// Returns stream.ErrStreamNotFound if the stream does not exist.
func (s *StreamService) GetStream(ctx context.Context, infoHash string) (stream.Stream, error) {
	return s.streamRepo.FindByInfoHash(ctx, infoHash)
}

// ListStreams retrieves all streams.
func (s *StreamService) ListStreams(ctx context.Context) ([]stream.Stream, error) {
	return s.streamRepo.FindAll(ctx)
}

// DeleteStream removes a stream by its infohash.
// Returns stream.ErrStreamNotFound if the stream does not exist.
func (s *StreamService) DeleteStream(ctx context.Context, infoHash string) error {
	return s.streamRepo.Delete(ctx, infoHash)
}
