package memory

import (
	"errors"
	"sync"

	domain "github.com/alorle/iptv-manager/internal"
	"github.com/google/uuid"
)

var (
	ErrChannelNotFound = errors.New("channel not found")
	ErrChannelExists   = errors.New("channel with this ID already exists")
)

// DEPRECATED: InMemoryChannelsRepository is deprecated.
// Channels are now derived views (grouping streams by GuideID).
// Use InMemoryStreamsRepository instead.
// This is kept for backward compatibility during transition.
type InMemoryChannelsRepository struct {
	channels []*domain.Channel
	mu       sync.RWMutex
	filePath string
}

func NewInMemoryChannelsRepository(channels []*domain.Channel) (*InMemoryChannelsRepository, error) {
	return &InMemoryChannelsRepository{
		channels: channels,
		filePath: "",
	}, nil
}

func (r *InMemoryChannelsRepository) SetFilePath(path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.filePath = path
}

func (r *InMemoryChannelsRepository) GetAll() ([]*domain.Channel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.channels, nil
}

// GetByID is deprecated - channels no longer have IDs in the new model
func (r *InMemoryChannelsRepository) GetByID(id uuid.UUID) (*domain.Channel, error) {
	return nil, errors.New("GetByID not supported: channels are now derived views without IDs")
}

// Create is deprecated - use StreamRepository.Create instead
func (r *InMemoryChannelsRepository) Create(channel *domain.Channel) error {
	return errors.New("Create not supported: use StreamRepository.Create to add streams")
}

// Update is deprecated - use StreamRepository.Update instead
func (r *InMemoryChannelsRepository) Update(id uuid.UUID, channel *domain.Channel) error {
	return errors.New("Update not supported: use StreamRepository.Update to modify streams")
}

// Delete is deprecated - use StreamRepository.Delete instead
func (r *InMemoryChannelsRepository) Delete(id uuid.UUID) error {
	return errors.New("Delete not supported: use StreamRepository.Delete to remove streams")
}

func (r *InMemoryChannelsRepository) saveToFile() error {
	if r.filePath == "" {
		return nil // No file path configured, skip persistence
	}
	return saveChannelsToFile(r.filePath, r.channels)
}
