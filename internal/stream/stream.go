package stream

import (
	"errors"
	"strings"
)

// Domain errors
var (
	ErrEmptyInfoHash       = errors.New("infohash cannot be empty")
	ErrEmptyChannelName    = errors.New("channel name cannot be empty")
	ErrStreamNotFound      = errors.New("stream not found")
	ErrStreamAlreadyExists = errors.New("stream already exists")
)

// Stream represents an AceStream associated with a channel in the domain.
// It is the core entity for managing IPTV streams.
type Stream struct {
	infoHash    string
	channelName string
}

// NewStream creates a new Stream with the given infohash and channel name.
// It validates that both infohash and channelName are not empty and trims whitespace.
// Returns ErrEmptyInfoHash if the infohash is empty or contains only whitespace.
// Returns ErrEmptyChannelName if the channelName is empty or contains only whitespace.
func NewStream(infoHash, channelName string) (Stream, error) {
	trimmedHash := strings.TrimSpace(infoHash)
	if trimmedHash == "" {
		return Stream{}, ErrEmptyInfoHash
	}

	trimmedName := strings.TrimSpace(channelName)
	if trimmedName == "" {
		return Stream{}, ErrEmptyChannelName
	}

	return Stream{
		infoHash:    trimmedHash,
		channelName: trimmedName,
	}, nil
}

// InfoHash returns the stream's infohash identifier.
func (s Stream) InfoHash() string {
	return s.infoHash
}

// ChannelName returns the name of the channel this stream is associated with.
func (s Stream) ChannelName() string {
	return s.channelName
}
