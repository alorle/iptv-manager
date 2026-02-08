package channel

import (
	"errors"
	"strings"
)

// Domain errors
var (
	ErrEmptyName            = errors.New("channel name cannot be empty")
	ErrChannelNotFound      = errors.New("channel not found")
	ErrChannelAlreadyExists = errors.New("channel already exists")
)

// Channel represents a TV channel in the domain.
// It is the core entity for managing IPTV channels.
type Channel struct {
	name string
}

// NewChannel creates a new Channel with the given name.
// It validates that the name is not empty and trims whitespace.
// Returns ErrEmptyName if the name is empty or contains only whitespace.
func NewChannel(name string) (Channel, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return Channel{}, ErrEmptyName
	}
	return Channel{name: trimmed}, nil
}

// Name returns the channel's name.
func (c Channel) Name() string {
	return c.name
}
