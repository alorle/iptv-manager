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

func (r *InMemoryChannelsRepository) GetByID(id uuid.UUID) (*domain.Channel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, channel := range r.channels {
		if channel.ID == id {
			return channel, nil
		}
	}
	return nil, ErrChannelNotFound
}

func (r *InMemoryChannelsRepository) Create(channel *domain.Channel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if channel with this ID already exists
	for _, c := range r.channels {
		if c.ID == channel.ID {
			return ErrChannelExists
		}
	}

	// Generate UUIDs for channel and its streams if not set
	if channel.ID == uuid.Nil {
		channel.ID = uuid.New()
	}
	for _, stream := range channel.Streams {
		if stream.ID == uuid.Nil {
			stream.ID = uuid.New()
		}
		stream.ChannelID = channel.ID
	}

	r.channels = append(r.channels, channel)
	return r.saveToFile()
}

func (r *InMemoryChannelsRepository) Update(id uuid.UUID, channel *domain.Channel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, c := range r.channels {
		if c.ID == id {
			// Preserve the original ID
			channel.ID = id
			// Ensure stream IDs are set and channel references are correct
			for _, stream := range channel.Streams {
				if stream.ID == uuid.Nil {
					stream.ID = uuid.New()
				}
				stream.ChannelID = id
			}
			r.channels[i] = channel
			return r.saveToFile()
		}
	}
	return ErrChannelNotFound
}

func (r *InMemoryChannelsRepository) Delete(id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, channel := range r.channels {
		if channel.ID == id {
			r.channels = append(r.channels[:i], r.channels[i+1:]...)
			return r.saveToFile()
		}
	}
	return ErrChannelNotFound
}

func (r *InMemoryChannelsRepository) saveToFile() error {
	if r.filePath == "" {
		return nil // No file path configured, skip persistence
	}
	return saveChannelsToFile(r.filePath, r.channels)
}
