package driven

import (
	"context"

	"github.com/alorle/iptv-manager/internal/channel"
)

// ChannelRepository defines the interface for channel persistence operations.
// This is a driven port that will be implemented by concrete adapters (e.g., BoltDB).
type ChannelRepository interface {
	// Save persists a channel. Returns channel.ErrChannelAlreadyExists if a channel
	// with the same name already exists.
	Save(ctx context.Context, ch channel.Channel) error

	// FindByName retrieves a channel by its name. Returns channel.ErrChannelNotFound
	// if the channel does not exist.
	FindByName(ctx context.Context, name string) (channel.Channel, error)

	// FindAll retrieves all channels.
	FindAll(ctx context.Context) ([]channel.Channel, error)

	// Delete removes a channel by its name. Returns channel.ErrChannelNotFound
	// if the channel does not exist.
	Delete(ctx context.Context, name string) error

	// Ping checks if the repository (database) is accessible and operational.
	// Returns nil if healthy, otherwise returns an error describing the issue.
	Ping(ctx context.Context) error
}
