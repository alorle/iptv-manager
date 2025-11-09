package domain

import "github.com/google/uuid"

// StreamRepository manages stream storage
type StreamRepository interface {
	GetAll() ([]*Stream, error)
	GetByID(id uuid.UUID) (*Stream, error)
	GetByGuideID(guideID string) ([]*Stream, error) // For grouping into channels
	Create(stream *Stream) error
	Update(id uuid.UUID, stream *Stream) error
	Delete(id uuid.UUID) error
}

// ChannelRepository kept for backward compatibility during transition
// DEPRECATED: Will be removed in v3.0
type ChannelRepository interface {
	GetAll() ([]*Channel, error)
	GetByID(id uuid.UUID) (*Channel, error)
	Create(channel *Channel) error
	Update(id uuid.UUID, channel *Channel) error
	Delete(id uuid.UUID) error
}
