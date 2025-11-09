package memory

import (
	"errors"
	"sync"

	domain "github.com/alorle/iptv-manager/internal"
	"github.com/google/uuid"
)

var (
	ErrStreamNotFound = errors.New("stream not found")
	ErrStreamExists   = errors.New("stream with this ID already exists")
)

type InMemoryStreamsRepository struct {
	streams  []*domain.Stream
	mu       sync.RWMutex
	filePath string
}

func NewInMemoryStreamsRepository(streams []*domain.Stream) (*InMemoryStreamsRepository, error) {
	return &InMemoryStreamsRepository{
		streams:  streams,
		filePath: "",
	}, nil
}

func (r *InMemoryStreamsRepository) SetFilePath(path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.filePath = path
}

func (r *InMemoryStreamsRepository) GetAll() ([]*domain.Stream, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.streams, nil
}

func (r *InMemoryStreamsRepository) GetByID(id uuid.UUID) (*domain.Stream, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, stream := range r.streams {
		if stream.ID == id {
			return stream, nil
		}
	}
	return nil, ErrStreamNotFound
}

func (r *InMemoryStreamsRepository) GetByGuideID(guideID string) ([]*domain.Stream, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Stream
	for _, stream := range r.streams {
		if stream.GuideID == guideID {
			result = append(result, stream)
		}
	}
	return result, nil
}

func (r *InMemoryStreamsRepository) Create(stream *domain.Stream) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if stream with this ID already exists
	for _, s := range r.streams {
		if s.ID == stream.ID {
			return ErrStreamExists
		}
	}

	// Generate UUID if not set
	if stream.ID == uuid.Nil {
		stream.ID = uuid.New()
	}

	r.streams = append(r.streams, stream)
	return r.saveToFile()
}

func (r *InMemoryStreamsRepository) Update(id uuid.UUID, stream *domain.Stream) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, s := range r.streams {
		if s.ID == id {
			// Preserve the original ID
			stream.ID = id
			r.streams[i] = stream
			return r.saveToFile()
		}
	}
	return ErrStreamNotFound
}

func (r *InMemoryStreamsRepository) Delete(id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, stream := range r.streams {
		if stream.ID == id {
			r.streams = append(r.streams[:i], r.streams[i+1:]...)
			return r.saveToFile()
		}
	}
	return ErrStreamNotFound
}

func (r *InMemoryStreamsRepository) saveToFile() error {
	if r.filePath == "" {
		return nil // No file path configured, skip persistence
	}
	return saveStreamsToFile(r.filePath, r.streams)
}
