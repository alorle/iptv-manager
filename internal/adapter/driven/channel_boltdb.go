package driven

import (
	"context"
	"encoding/json"
	"errors"

	"go.etcd.io/bbolt"

	"github.com/alorle/iptv-manager/internal/channel"
)

const (
	channelsBucket = "channels"
)

// ChannelBoltDBRepository implements the ChannelRepository port using BoltDB.
type ChannelBoltDBRepository struct {
	db *bbolt.DB
}

// NewChannelBoltDBRepository creates a new BoltDB-backed channel repository.
// It initializes the required bucket if it doesn't exist.
func NewChannelBoltDBRepository(db *bbolt.DB) (*ChannelBoltDBRepository, error) {
	if db == nil {
		return nil, errors.New("db cannot be nil")
	}

	// Create the channels bucket if it doesn't exist
	err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(channelsBucket))
		return err
	})
	if err != nil {
		return nil, err
	}

	return &ChannelBoltDBRepository{db: db}, nil
}

// channelDTO is used for JSON serialization.
type channelDTO struct {
	Name string `json:"name"`
}

// Save persists a channel to BoltDB.
func (r *ChannelBoltDBRepository) Save(ctx context.Context, ch channel.Channel) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(channelsBucket))
		if bucket == nil {
			return errors.New("channels bucket not found")
		}

		key := []byte(ch.Name())

		// Check if channel already exists
		if bucket.Get(key) != nil {
			return channel.ErrChannelAlreadyExists
		}

		// Serialize channel
		dto := channelDTO{Name: ch.Name()}
		data, err := json.Marshal(dto)
		if err != nil {
			return err
		}

		return bucket.Put(key, data)
	})
}

// FindByName retrieves a channel by its name from BoltDB.
func (r *ChannelBoltDBRepository) FindByName(ctx context.Context, name string) (channel.Channel, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return channel.Channel{}, err
	}

	var ch channel.Channel

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(channelsBucket))
		if bucket == nil {
			return errors.New("channels bucket not found")
		}

		data := bucket.Get([]byte(name))
		if data == nil {
			return channel.ErrChannelNotFound
		}

		var dto channelDTO
		if err := json.Unmarshal(data, &dto); err != nil {
			return err
		}

		// Reconstruct domain entity
		reconstructed, err := channel.NewChannel(dto.Name)
		if err != nil {
			return err
		}

		ch = reconstructed
		return nil
	})

	return ch, err
}

// FindAll retrieves all channels from BoltDB.
func (r *ChannelBoltDBRepository) FindAll(ctx context.Context) ([]channel.Channel, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var channels []channel.Channel

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(channelsBucket))
		if bucket == nil {
			return errors.New("channels bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var dto channelDTO
			if err := json.Unmarshal(v, &dto); err != nil {
				return err
			}

			ch, err := channel.NewChannel(dto.Name)
			if err != nil {
				return err
			}

			channels = append(channels, ch)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no channels found
	if channels == nil {
		channels = []channel.Channel{}
	}

	return channels, nil
}

// Delete removes a channel by its name from BoltDB.
func (r *ChannelBoltDBRepository) Delete(ctx context.Context, name string) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(channelsBucket))
		if bucket == nil {
			return errors.New("channels bucket not found")
		}

		key := []byte(name)

		// Check if channel exists before deleting
		if bucket.Get(key) == nil {
			return channel.ErrChannelNotFound
		}

		return bucket.Delete(key)
	})
}

// Ping checks if the BoltDB database is accessible and operational.
func (r *ChannelBoltDBRepository) Ping(ctx context.Context) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Perform a simple read transaction to verify DB is accessible
	return r.db.View(func(tx *bbolt.Tx) error {
		// Simply verify we can start a transaction and access a bucket
		bucket := tx.Bucket([]byte(channelsBucket))
		if bucket == nil {
			return errors.New("channels bucket not found")
		}
		return nil
	})
}
