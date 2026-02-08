package driven

import (
	"context"

	"github.com/alorle/iptv-manager/internal/stream"
)

// StreamRepository defines the interface for stream persistence operations.
// This is a driven port that will be implemented by concrete adapters (e.g., BoltDB).
type StreamRepository interface {
	// Save persists a stream. Returns stream.ErrStreamAlreadyExists if a stream
	// with the same infohash already exists.
	Save(ctx context.Context, s stream.Stream) error

	// FindByInfoHash retrieves a stream by its infohash. Returns stream.ErrStreamNotFound
	// if the stream does not exist.
	FindByInfoHash(ctx context.Context, infoHash string) (stream.Stream, error)

	// FindAll retrieves all streams.
	FindAll(ctx context.Context) ([]stream.Stream, error)

	// FindByChannelName retrieves all streams associated with a specific channel.
	FindByChannelName(ctx context.Context, channelName string) ([]stream.Stream, error)

	// Delete removes a stream by its infohash. Returns stream.ErrStreamNotFound
	// if the stream does not exist.
	Delete(ctx context.Context, infoHash string) error

	// DeleteByChannelName removes all streams associated with a specific channel.
	// This supports cascade delete when a channel is removed.
	DeleteByChannelName(ctx context.Context, channelName string) error
}
