package usecase

import (
	domain "github.com/alorle/iptv-manager/internal"
)

type GetChannelUseCase interface {
	GetChannels() ([]*domain.Channel, error)
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
