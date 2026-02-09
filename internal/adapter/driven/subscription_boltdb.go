package driven

import (
	"context"
	"encoding/json"
	"errors"

	"go.etcd.io/bbolt"

	"github.com/alorle/iptv-manager/internal/subscription"
)

const (
	subscriptionsBucket = "subscriptions"
)

// SubscriptionBoltDBRepository implements the SubscriptionRepository port using BoltDB.
type SubscriptionBoltDBRepository struct {
	db *bbolt.DB
}

// NewSubscriptionBoltDBRepository creates a new BoltDB-backed subscription repository.
// It initializes the required bucket if it doesn't exist.
func NewSubscriptionBoltDBRepository(db *bbolt.DB) (*SubscriptionBoltDBRepository, error) {
	if db == nil {
		return nil, errors.New("db cannot be nil")
	}

	// Create the subscriptions bucket if it doesn't exist
	err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(subscriptionsBucket))
		return err
	})
	if err != nil {
		return nil, err
	}

	return &SubscriptionBoltDBRepository{db: db}, nil
}

// subscriptionDTO is used for JSON serialization.
type subscriptionDTO struct {
	EPGChannelID   string `json:"epg_channel_id"`
	Enabled        bool   `json:"enabled"`
	ManualOverride bool   `json:"manual_override"`
}

// Save persists a subscription to BoltDB.
func (r *SubscriptionBoltDBRepository) Save(ctx context.Context, sub subscription.Subscription) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(subscriptionsBucket))
		if bucket == nil {
			return errors.New("subscriptions bucket not found")
		}

		key := []byte(sub.EPGChannelID())

		// Check if subscription already exists
		if bucket.Get(key) != nil {
			return subscription.ErrSubscriptionAlreadyExists
		}

		// Serialize subscription
		dto := subscriptionDTO{
			EPGChannelID:   sub.EPGChannelID(),
			Enabled:        sub.IsEnabled(),
			ManualOverride: sub.HasManualOverride(),
		}
		data, err := json.Marshal(dto)
		if err != nil {
			return err
		}

		return bucket.Put(key, data)
	})
}

// FindAll retrieves all subscriptions from BoltDB.
func (r *SubscriptionBoltDBRepository) FindAll(ctx context.Context) ([]subscription.Subscription, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var subscriptions []subscription.Subscription

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(subscriptionsBucket))
		if bucket == nil {
			return errors.New("subscriptions bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var dto subscriptionDTO
			if err := json.Unmarshal(v, &dto); err != nil {
				return err
			}

			// Reconstruct domain entity
			sub, err := subscription.NewSubscription(dto.EPGChannelID)
			if err != nil {
				return err
			}

			// Apply state based on persisted data
			if !dto.Enabled {
				sub = sub.Disable()
			}
			if dto.ManualOverride && dto.Enabled {
				sub = sub.Enable()
			}

			subscriptions = append(subscriptions, sub)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no subscriptions found
	if subscriptions == nil {
		subscriptions = []subscription.Subscription{}
	}

	return subscriptions, nil
}

// FindByEPGID retrieves a subscription by its EPG channel ID from BoltDB.
func (r *SubscriptionBoltDBRepository) FindByEPGID(ctx context.Context, epgChannelID string) (subscription.Subscription, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return subscription.Subscription{}, err
	}

	var sub subscription.Subscription

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(subscriptionsBucket))
		if bucket == nil {
			return errors.New("subscriptions bucket not found")
		}

		data := bucket.Get([]byte(epgChannelID))
		if data == nil {
			return subscription.ErrSubscriptionNotFound
		}

		var dto subscriptionDTO
		if err := json.Unmarshal(data, &dto); err != nil {
			return err
		}

		// Reconstruct domain entity
		reconstructed, err := subscription.NewSubscription(dto.EPGChannelID)
		if err != nil {
			return err
		}

		// Apply state based on persisted data
		if !dto.Enabled {
			reconstructed = reconstructed.Disable()
		}
		if dto.ManualOverride && dto.Enabled {
			reconstructed = reconstructed.Enable()
		}

		sub = reconstructed
		return nil
	})

	return sub, err
}

// Delete removes a subscription by its EPG channel ID from BoltDB.
func (r *SubscriptionBoltDBRepository) Delete(ctx context.Context, epgChannelID string) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(subscriptionsBucket))
		if bucket == nil {
			return errors.New("subscriptions bucket not found")
		}

		key := []byte(epgChannelID)

		// Check if subscription exists before deleting
		if bucket.Get(key) == nil {
			return subscription.ErrSubscriptionNotFound
		}

		return bucket.Delete(key)
	})
}
