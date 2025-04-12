package usecase

import (
	domain "github.com/alorle/iptv-manager/internal"
)

type ChannelUseCase struct {
	channelRepo domain.ChannelRepository
}

func NewChannelUseCase(repo domain.ChannelRepository) *ChannelUseCase {
	return &ChannelUseCase{
		channelRepo: repo,
	}
}

func (uc *ChannelUseCase) GetAllChannels() ([]*domain.Channel, error) {
	return uc.channelRepo.GetAll()
}
