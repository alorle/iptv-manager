package epg

import "errors"

// Domain errors for EPG operations.
var (
	// Channel validation errors
	ErrEmptyID    = errors.New("epg channel id cannot be empty")
	ErrEmptyName  = errors.New("epg channel name cannot be empty")
	ErrEmptyEPGID = errors.New("epg channel epg id cannot be empty")
	ErrEmptyURL   = errors.New("epg source url cannot be empty")

	// EPG operation errors
	ErrInvalidEPGFormat = errors.New("invalid epg format")
	ErrChannelNotFound  = errors.New("epg channel not found")
	ErrSourceNotFound   = errors.New("epg source not found")
)
