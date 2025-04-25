package memory

import (
	domain "github.com/alorle/iptv-manager/internal"
)

type InMemoryChannelsRepository struct {
	channels []*domain.Channel
}

func NewInMemoryChannelsRepository(channels []*domain.Channel) (*InMemoryChannelsRepository, error) {
	return &InMemoryChannelsRepository{channels: channels}, nil
}

func (r *InMemoryChannelsRepository) GetAll() ([]*domain.Channel, error) {
	return r.channels, nil
}
