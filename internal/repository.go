package domain

import "github.com/google/uuid"

type ChannelRepository interface {
	GetAll() ([]*Channel, error)
	GetByID(id uuid.UUID) (*Channel, error)
	Create(channel *Channel) error
	Update(id uuid.UUID, channel *Channel) error
	Delete(id uuid.UUID) error
}
