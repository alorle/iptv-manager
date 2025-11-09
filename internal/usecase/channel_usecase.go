package usecase

import (
	domain "github.com/alorle/iptv-manager/internal"
	"github.com/google/uuid"
)

type GetChannelUseCase interface {
	GetChannels() ([]*domain.Channel, error)
	GetChannel(id uuid.UUID) (*domain.Channel, error)
	CreateChannel(channel *domain.Channel) error
	UpdateChannel(id uuid.UUID, channel *domain.Channel) error
	DeleteChannel(id uuid.UUID) error
}

type channelsUseCase struct {
	repo domain.ChannelRepository
}

func NewChannelsUseCase(repo domain.ChannelRepository) *channelsUseCase {
	return &channelsUseCase{repo: repo}
}

func (uc *channelsUseCase) GetChannels() ([]*domain.Channel, error) {
	return uc.repo.GetAll()
}

func (uc *channelsUseCase) GetChannel(id uuid.UUID) (*domain.Channel, error) {
	return uc.repo.GetByID(id)
}

func (uc *channelsUseCase) CreateChannel(channel *domain.Channel) error {
	return uc.repo.Create(channel)
}

func (uc *channelsUseCase) UpdateChannel(id uuid.UUID, channel *domain.Channel) error {
	return uc.repo.Update(id, channel)
}

func (uc *channelsUseCase) DeleteChannel(id uuid.UUID) error {
	return uc.repo.Delete(id)
}
