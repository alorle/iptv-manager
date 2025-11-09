package usecase

import (
	domain "github.com/alorle/iptv-manager/internal"
	"github.com/alorle/iptv-manager/internal/epg"
	"github.com/google/uuid"
)

// StreamUseCase handles stream operations and derives channel views
type StreamUseCase interface {
	// Stream operations
	GetStreams() ([]*domain.Stream, error)
	GetStream(id uuid.UUID) (*domain.Stream, error)
	CreateStream(stream *domain.Stream) error
	UpdateStream(id uuid.UUID, stream *domain.Stream) error
	DeleteStream(id uuid.UUID) error

	// Channel view operations (derived from streams + EPG)
	GetChannelViews() ([]*domain.Channel, error)
	GetChannelViewByGuideID(guideID string) (*domain.Channel, error)
}

type streamUseCase struct {
	streamRepo domain.StreamRepository
	epgRepo    epg.Repository
}

func NewStreamUseCase(streamRepo domain.StreamRepository, epgRepo epg.Repository) *streamUseCase {
	return &streamUseCase{
		streamRepo: streamRepo,
		epgRepo:    epgRepo,
	}
}

// Stream operations

func (uc *streamUseCase) GetStreams() ([]*domain.Stream, error) {
	return uc.streamRepo.GetAll()
}

func (uc *streamUseCase) GetStream(id uuid.UUID) (*domain.Stream, error) {
	return uc.streamRepo.GetByID(id)
}

func (uc *streamUseCase) CreateStream(stream *domain.Stream) error {
	return uc.streamRepo.Create(stream)
}

func (uc *streamUseCase) UpdateStream(id uuid.UUID, stream *domain.Stream) error {
	return uc.streamRepo.Update(id, stream)
}

func (uc *streamUseCase) DeleteStream(id uuid.UUID) error {
	return uc.streamRepo.Delete(id)
}

// Channel view operations (derived from streams grouped by GuideID)

func (uc *streamUseCase) GetChannelViews() ([]*domain.Channel, error) {
	streams, err := uc.streamRepo.GetAll()
	if err != nil {
		return nil, err
	}

	return uc.deriveChannelViews(streams), nil
}

func (uc *streamUseCase) GetChannelViewByGuideID(guideID string) (*domain.Channel, error) {
	streams, err := uc.streamRepo.GetByGuideID(guideID)
	if err != nil {
		return nil, err
	}

	if len(streams) == 0 {
		return nil, nil // Channel doesn't exist if no streams
	}

	// Get EPG metadata for this channel
	epgChannel := uc.epgRepo.FindByID(guideID)

	channel := &domain.Channel{
		GuideID: guideID,
		Streams: streams,
	}

	// Enrich with EPG data if available
	if epgChannel != nil {
		channel.Title = epgChannel.Name
		channel.Logo = epgChannel.Logo
		// GroupTitle could be derived from EPG or set manually
		// For now, we'll leave it empty unless EPG provides it
	} else {
		// Fallback: use GuideID as title if EPG not available
		channel.Title = guideID
	}

	return channel, nil
}

// deriveChannelViews groups streams by GuideID and enriches with EPG metadata
func (uc *streamUseCase) deriveChannelViews(streams []*domain.Stream) []*domain.Channel {
	// Group streams by GuideID
	channelMap := make(map[string][]*domain.Stream)
	for _, stream := range streams {
		channelMap[stream.GuideID] = append(channelMap[stream.GuideID], stream)
	}

	// Create channel views
	channels := make([]*domain.Channel, 0, len(channelMap))
	for guideID, streamGroup := range channelMap {
		// Get EPG metadata for this channel
		epgChannel := uc.epgRepo.FindByID(guideID)

		channel := &domain.Channel{
			GuideID: guideID,
			Streams: streamGroup,
		}

		// Enrich with EPG data if available
		if epgChannel != nil {
			channel.Title = epgChannel.Name
			channel.Logo = epgChannel.Logo
			// GroupTitle could be derived from EPG or set manually
		} else {
			// Fallback: use GuideID as title if EPG not available
			channel.Title = guideID
		}

		channels = append(channels, channel)
	}

	return channels
}
