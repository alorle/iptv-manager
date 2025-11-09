package usecase

import "github.com/alorle/iptv-manager/internal/epg"

// EPGUseCase handles EPG-related business logic
type EPGUseCase struct {
	repo epg.Repository
}

// NewEPGUseCase creates a new EPG use case
func NewEPGUseCase(repo epg.Repository) *EPGUseCase {
	return &EPGUseCase{
		repo: repo,
	}
}

// ListChannels returns all EPG channels
func (uc *EPGUseCase) ListChannels() []epg.EPGChannel {
	return uc.repo.GetAll()
}

// SearchChannels returns EPG channels matching the query
func (uc *EPGUseCase) SearchChannels(query string) []epg.EPGChannel {
	return uc.repo.Search(query)
}

// ValidateGuideID checks if a guide ID exists in the EPG
func (uc *EPGUseCase) ValidateGuideID(guideID string) bool {
	return uc.repo.FindByID(guideID) != nil
}

// IsEPGAvailable returns true if EPG data is available
func (uc *EPGUseCase) IsEPGAvailable() bool {
	return uc.repo.IsAvailable()
}

// GetChannelByID returns an EPG channel by ID, or nil if not found
func (uc *EPGUseCase) GetChannelByID(id string) *epg.EPGChannel {
	return uc.repo.FindByID(id)
}
