package driven

import (
	"context"
	"encoding/json"
	"errors"
	"time"

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
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	EPGMapping *epgMappingDTO `json:"epg_mapping,omitempty"`
}

// epgMappingDTO is used for JSON serialization of EPG mapping data.
type epgMappingDTO struct {
	EPGID      string `json:"epg_id"`
	Source     string `json:"source"`
	LastSynced string `json:"last_synced"`
}

func channelToDTO(ch channel.Channel) channelDTO {
	dto := channelDTO{
		Name:   ch.Name(),
		Status: string(ch.Status()),
	}
	if m := ch.EPGMapping(); m != nil {
		dto.EPGMapping = &epgMappingDTO{
			EPGID:      m.EPGID(),
			Source:     string(m.Source()),
			LastSynced: m.LastSynced().Format(time.RFC3339),
		}
	}
	return dto
}

func dtoToChannel(dto channelDTO) (channel.Channel, error) {
	status := channel.Status(dto.Status)
	if status == "" {
		status = channel.StatusActive
	}

	var mapping *channel.EPGMapping
	if dto.EPGMapping != nil {
		lastSynced, err := time.Parse(time.RFC3339, dto.EPGMapping.LastSynced)
		if err != nil {
			return channel.Channel{}, err
		}
		m, err := channel.NewEPGMapping(dto.EPGMapping.EPGID, channel.MappingSource(dto.EPGMapping.Source), lastSynced)
		if err != nil {
			return channel.Channel{}, err
		}
		mapping = &m
	}

	return channel.ReconstructChannel(dto.Name, status, mapping), nil
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

		data, err := json.Marshal(channelToDTO(ch))
		if err != nil {
			return err
		}

		return bucket.Put(key, data)
	})
}

// Update persists changes to an existing channel in BoltDB.
func (r *ChannelBoltDBRepository) Update(ctx context.Context, ch channel.Channel) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(channelsBucket))
		if bucket == nil {
			return errors.New("channels bucket not found")
		}

		key := []byte(ch.Name())

		if bucket.Get(key) == nil {
			return channel.ErrChannelNotFound
		}

		data, err := json.Marshal(channelToDTO(ch))
		if err != nil {
			return err
		}

		return bucket.Put(key, data)
	})
}

// FindByName retrieves a channel by its name from BoltDB.
func (r *ChannelBoltDBRepository) FindByName(ctx context.Context, name string) (channel.Channel, error) {
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

		reconstructed, err := dtoToChannel(dto)
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

			ch, err := dtoToChannel(dto)
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
