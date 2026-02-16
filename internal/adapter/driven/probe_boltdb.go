package driven

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"time"

	"go.etcd.io/bbolt"

	"github.com/alorle/iptv-manager/internal/probe"
)

const probesBucket = "probes"

// ProbeBoltDBRepository implements the ProbeRepository port using BoltDB.
// It uses nested buckets: probes/<infoHash> with timestamp-keyed entries.
type ProbeBoltDBRepository struct {
	db *bbolt.DB
}

// NewProbeBoltDBRepository creates a new BoltDB-backed probe repository.
// It initializes the required top-level bucket if it doesn't exist.
func NewProbeBoltDBRepository(db *bbolt.DB) (*ProbeBoltDBRepository, error) {
	if db == nil {
		return nil, errors.New("db cannot be nil")
	}

	err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(probesBucket))
		return err
	})
	if err != nil {
		return nil, err
	}

	return &ProbeBoltDBRepository{db: db}, nil
}

// probeDTO is the JSON serialization format for a probe result.
type probeDTO struct {
	InfoHash       string `json:"infohash"`
	Timestamp      int64  `json:"timestamp"`
	Available      bool   `json:"available"`
	StartupLatency int64  `json:"startup_latency"`
	PeerCount      int    `json:"peer_count"`
	DownloadSpeed  int64  `json:"download_speed"`
	Status         string `json:"status"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

// Save persists a probe result to BoltDB.
func (r *ProbeBoltDBRepository) Save(ctx context.Context, result probe.Result) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		top := tx.Bucket([]byte(probesBucket))
		if top == nil {
			return errors.New("probes bucket not found")
		}

		// Create or get the sub-bucket for this infoHash
		sub, err := top.CreateBucketIfNotExists([]byte(result.InfoHash()))
		if err != nil {
			return err
		}

		dto := probeDTO{
			InfoHash:       result.InfoHash(),
			Timestamp:      result.Timestamp().UnixNano(),
			Available:      result.Available(),
			StartupLatency: result.StartupLatency().Nanoseconds(),
			PeerCount:      result.PeerCount(),
			DownloadSpeed:  result.DownloadSpeed(),
			Status:         result.Status(),
			ErrorMessage:   result.ErrorMessage(),
		}

		data, err := json.Marshal(dto)
		if err != nil {
			return err
		}

		key := timestampToKey(result.Timestamp())
		return sub.Put(key, data)
	})
}

// FindByInfoHash retrieves all probe results for a given stream,
// ordered by timestamp descending (most recent first).
func (r *ProbeBoltDBRepository) FindByInfoHash(ctx context.Context, infoHash string) ([]probe.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var results []probe.Result

	err := r.db.View(func(tx *bbolt.Tx) error {
		top := tx.Bucket([]byte(probesBucket))
		if top == nil {
			return errors.New("probes bucket not found")
		}

		sub := top.Bucket([]byte(infoHash))
		if sub == nil {
			return nil // No probes for this infoHash
		}

		// Iterate in reverse (most recent first)
		c := sub.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			result, err := dtoToResult(v)
			if err != nil {
				return err
			}
			results = append(results, result)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if results == nil {
		results = []probe.Result{}
	}

	return results, nil
}

// FindByInfoHashSince retrieves probe results for a stream since the
// given time, ordered by timestamp descending.
func (r *ProbeBoltDBRepository) FindByInfoHashSince(ctx context.Context, infoHash string, since time.Time) ([]probe.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var results []probe.Result

	err := r.db.View(func(tx *bbolt.Tx) error {
		top := tx.Bucket([]byte(probesBucket))
		if top == nil {
			return errors.New("probes bucket not found")
		}

		sub := top.Bucket([]byte(infoHash))
		if sub == nil {
			return nil
		}

		sinceKey := timestampToKey(since)

		// Iterate from the end backwards, collecting entries >= since
		c := sub.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			// Keys are big-endian timestamps; stop when we go before the cutoff
			if compareKeys(k, sinceKey) < 0 {
				break
			}

			result, err := dtoToResult(v)
			if err != nil {
				return err
			}
			results = append(results, result)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if results == nil {
		results = []probe.Result{}
	}

	return results, nil
}

// DeleteBefore removes all probe results older than the given time.
func (r *ProbeBoltDBRepository) DeleteBefore(ctx context.Context, before time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		top := tx.Bucket([]byte(probesBucket))
		if top == nil {
			return errors.New("probes bucket not found")
		}

		beforeKey := timestampToKey(before)

		// Iterate all sub-buckets (one per infoHash)
		return top.ForEach(func(k, v []byte) error {
			// v is nil for nested buckets
			if v != nil {
				return nil
			}

			sub := top.Bucket(k)
			if sub == nil {
				return nil
			}

			// Collect keys to delete (can't delete during iteration)
			var keysToDelete [][]byte
			c := sub.Cursor()
			for ck, _ := c.First(); ck != nil; ck, _ = c.Next() {
				if compareKeys(ck, beforeKey) < 0 {
					keyCopy := make([]byte, len(ck))
					copy(keyCopy, ck)
					keysToDelete = append(keysToDelete, keyCopy)
				} else {
					break // Keys are sorted, no need to continue
				}
			}

			for _, dk := range keysToDelete {
				if err := sub.Delete(dk); err != nil {
					return err
				}
			}

			return nil
		})
	})
}

// timestampToKey converts a time.Time to an 8-byte big-endian key.
// This ensures chronological ordering in BoltDB's byte-sorted keys.
func timestampToKey(t time.Time) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, uint64(t.UnixNano()))
	return key
}

// compareKeys compares two 8-byte big-endian keys.
// Returns -1, 0, or 1.
func compareKeys(a, b []byte) int {
	va := binary.BigEndian.Uint64(a)
	vb := binary.BigEndian.Uint64(b)
	if va < vb {
		return -1
	}
	if va > vb {
		return 1
	}
	return 0
}

// dtoToResult deserializes a JSON value into a probe.Result.
func dtoToResult(data []byte) (probe.Result, error) {
	var dto probeDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return probe.Result{}, err
	}

	return probe.ReconstructResult(
		dto.InfoHash,
		time.Unix(0, dto.Timestamp),
		dto.Available,
		time.Duration(dto.StartupLatency),
		dto.PeerCount,
		dto.DownloadSpeed,
		dto.Status,
		dto.ErrorMessage,
	), nil
}
