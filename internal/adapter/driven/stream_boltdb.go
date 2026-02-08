package driven

import (
	"context"
	"encoding/json"
	"errors"

	"go.etcd.io/bbolt"

	"github.com/alorle/iptv-manager/internal/stream"
)

const (
	streamsBucket = "streams"
)

// StreamBoltDBRepository implements the StreamRepository port using BoltDB.
type StreamBoltDBRepository struct {
	db *bbolt.DB
}

// NewStreamBoltDBRepository creates a new BoltDB-backed stream repository.
// It initializes the required bucket if it doesn't exist.
func NewStreamBoltDBRepository(db *bbolt.DB) (*StreamBoltDBRepository, error) {
	if db == nil {
		return nil, errors.New("db cannot be nil")
	}

	// Create the streams bucket if it doesn't exist
	err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(streamsBucket))
		return err
	})
	if err != nil {
		return nil, err
	}

	return &StreamBoltDBRepository{db: db}, nil
}

// streamDTO is used for JSON serialization.
type streamDTO struct {
	InfoHash    string `json:"infohash"`
	ChannelName string `json:"channel_name"`
}

// Save persists a stream to BoltDB.
func (r *StreamBoltDBRepository) Save(ctx context.Context, s stream.Stream) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(streamsBucket))
		if bucket == nil {
			return errors.New("streams bucket not found")
		}

		key := []byte(s.InfoHash())

		// Check if stream already exists
		if bucket.Get(key) != nil {
			return stream.ErrStreamAlreadyExists
		}

		// Serialize stream
		dto := streamDTO{
			InfoHash:    s.InfoHash(),
			ChannelName: s.ChannelName(),
		}
		data, err := json.Marshal(dto)
		if err != nil {
			return err
		}

		return bucket.Put(key, data)
	})
}

// FindByInfoHash retrieves a stream by its infohash from BoltDB.
func (r *StreamBoltDBRepository) FindByInfoHash(ctx context.Context, infoHash string) (stream.Stream, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return stream.Stream{}, err
	}

	var s stream.Stream

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(streamsBucket))
		if bucket == nil {
			return errors.New("streams bucket not found")
		}

		data := bucket.Get([]byte(infoHash))
		if data == nil {
			return stream.ErrStreamNotFound
		}

		var dto streamDTO
		if err := json.Unmarshal(data, &dto); err != nil {
			return err
		}

		// Reconstruct domain entity
		reconstructed, err := stream.NewStream(dto.InfoHash, dto.ChannelName)
		if err != nil {
			return err
		}

		s = reconstructed
		return nil
	})

	return s, err
}

// FindAll retrieves all streams from BoltDB.
func (r *StreamBoltDBRepository) FindAll(ctx context.Context) ([]stream.Stream, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var streams []stream.Stream

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(streamsBucket))
		if bucket == nil {
			return errors.New("streams bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var dto streamDTO
			if err := json.Unmarshal(v, &dto); err != nil {
				return err
			}

			s, err := stream.NewStream(dto.InfoHash, dto.ChannelName)
			if err != nil {
				return err
			}

			streams = append(streams, s)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no streams found
	if streams == nil {
		streams = []stream.Stream{}
	}

	return streams, nil
}

// FindByChannelName retrieves all streams associated with a specific channel from BoltDB.
func (r *StreamBoltDBRepository) FindByChannelName(ctx context.Context, channelName string) ([]stream.Stream, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var streams []stream.Stream

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(streamsBucket))
		if bucket == nil {
			return errors.New("streams bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var dto streamDTO
			if err := json.Unmarshal(v, &dto); err != nil {
				return err
			}

			// Filter by channel name
			if dto.ChannelName != channelName {
				return nil
			}

			s, err := stream.NewStream(dto.InfoHash, dto.ChannelName)
			if err != nil {
				return err
			}

			streams = append(streams, s)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no streams found
	if streams == nil {
		streams = []stream.Stream{}
	}

	return streams, nil
}

// Delete removes a stream by its infohash from BoltDB.
func (r *StreamBoltDBRepository) Delete(ctx context.Context, infoHash string) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(streamsBucket))
		if bucket == nil {
			return errors.New("streams bucket not found")
		}

		key := []byte(infoHash)

		// Check if stream exists before deleting
		if bucket.Get(key) == nil {
			return stream.ErrStreamNotFound
		}

		return bucket.Delete(key)
	})
}

// DeleteByChannelName removes all streams associated with a specific channel from BoltDB.
// This supports cascade delete when a channel is removed.
func (r *StreamBoltDBRepository) DeleteByChannelName(ctx context.Context, channelName string) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(streamsBucket))
		if bucket == nil {
			return errors.New("streams bucket not found")
		}

		// Collect keys to delete (we can't delete while iterating)
		var keysToDelete [][]byte

		err := bucket.ForEach(func(k, v []byte) error {
			var dto streamDTO
			if err := json.Unmarshal(v, &dto); err != nil {
				return err
			}

			if dto.ChannelName == channelName {
				// Make a copy of the key
				keyCopy := make([]byte, len(k))
				copy(keyCopy, k)
				keysToDelete = append(keysToDelete, keyCopy)
			}

			return nil
		})

		if err != nil {
			return err
		}

		// Delete all matching streams
		for _, key := range keysToDelete {
			if err := bucket.Delete(key); err != nil {
				return err
			}
		}

		return nil
	})
}
