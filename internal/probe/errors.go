package probe

import "errors"

var (
	ErrEmptyInfoHash    = errors.New("probe infohash cannot be empty")
	ErrInvalidTimestamp = errors.New("probe timestamp must not be zero")
	ErrNoProbeData      = errors.New("no probe data available")
)
