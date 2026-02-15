package channel

import (
	"errors"
	"strings"
	"time"
	"unicode"
)

// Domain errors
var (
	ErrEmptyName            = errors.New("channel name cannot be empty")
	ErrChannelNotFound      = errors.New("channel not found")
	ErrChannelAlreadyExists = errors.New("channel already exists")
	ErrInvalidMappingSource = errors.New("invalid mapping source")
)

// Status represents the operational status of a channel.
type Status string

const (
	StatusActive   Status = "active"   // Channel is currently available
	StatusArchived Status = "archived" // Channel disappeared from source
)

// MappingSource indicates how the EPG mapping was established.
type MappingSource string

const (
	MappingAuto   MappingSource = "auto"   // Automatically matched via correlation
	MappingManual MappingSource = "manual" // Manually specified by user
)

// EPGMapping holds the EPG correlation data for a channel.
type EPGMapping struct {
	epgID      string
	source     MappingSource
	lastSynced time.Time
}

// NewEPGMapping creates a new EPGMapping with the given attributes.
// Returns ErrInvalidMappingSource if the source is not valid.
func NewEPGMapping(epgID string, source MappingSource, lastSynced time.Time) (EPGMapping, error) {
	if source != MappingAuto && source != MappingManual {
		return EPGMapping{}, ErrInvalidMappingSource
	}
	return EPGMapping{
		epgID:      strings.TrimSpace(epgID),
		source:     source,
		lastSynced: lastSynced,
	}, nil
}

// EPGID returns the EPG identifier this channel is mapped to.
func (m EPGMapping) EPGID() string {
	return m.epgID
}

// Source returns how this mapping was established.
func (m EPGMapping) Source() MappingSource {
	return m.source
}

// LastSynced returns when this mapping was last synchronized.
func (m EPGMapping) LastSynced() time.Time {
	return m.lastSynced
}

// Channel represents a TV channel in the domain.
// It is the core entity for managing IPTV channels.
type Channel struct {
	name       string
	status     Status
	epgMapping *EPGMapping
}

// NewChannel creates a new Channel with the given name.
// It validates that the name is not empty and trims whitespace.
// Returns ErrEmptyName if the name is empty or contains only whitespace.
// The channel is created with StatusActive and no EPG mapping.
func NewChannel(name string) (Channel, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return Channel{}, ErrEmptyName
	}
	return Channel{
		name:   trimmed,
		status: StatusActive,
	}, nil
}

// ReconstructChannel rebuilds a Channel from persisted state.
// This is intended for repository adapters only — it bypasses the validation
// and defaults applied by NewChannel.
func ReconstructChannel(name string, status Status, epgMapping *EPGMapping) Channel {
	return Channel{
		name:       name,
		status:     status,
		epgMapping: epgMapping,
	}
}

// Name returns the channel's name.
func (c Channel) Name() string {
	return c.name
}

// Status returns the channel's operational status.
func (c Channel) Status() Status {
	return c.status
}

// EPGMapping returns the channel's EPG mapping if present.
// Returns nil if no mapping has been established.
func (c Channel) EPGMapping() *EPGMapping {
	return c.epgMapping
}

// Archive marks the channel as archived (disappeared from source).
func (c *Channel) Archive() {
	c.status = StatusArchived
}

// SetEPGMapping assigns or updates the EPG mapping for this channel.
func (c *Channel) SetEPGMapping(mapping EPGMapping) {
	c.epgMapping = &mapping
}

// ClearEPGMapping removes the EPG mapping from this channel.
func (c *Channel) ClearEPGMapping() {
	c.epgMapping = nil
}

// NormalizeName returns a normalized version of the channel name for comparison.
// It converts to lowercase, removes extra whitespace, and strips common punctuation.
func NormalizeName(name string) string {
	// Convert to lowercase
	normalized := strings.ToLower(name)

	// Replace punctuation and special characters with spaces
	normalized = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}
		return ' '
	}, normalized)

	// Collapse multiple spaces to single space
	fields := strings.Fields(normalized)
	return strings.Join(fields, " ")
}

// FuzzyMatch calculates a simple similarity score between two channel names.
// Returns a value between 0.0 (no match) and 1.0 (exact match).
// Uses normalized name comparison and common substring matching.
func FuzzyMatch(name1, name2 string) float64 {
	n1 := NormalizeName(name1)
	n2 := NormalizeName(name2)

	// Exact match after normalization
	if n1 == n2 {
		return 1.0
	}

	// If either is empty, no match
	if n1 == "" || n2 == "" {
		return 0.0
	}

	// Check if one contains the other
	if strings.Contains(n1, n2) || strings.Contains(n2, n1) {
		return 0.8
	}

	// Calculate token overlap
	tokens1 := strings.Fields(n1)
	tokens2 := strings.Fields(n2)

	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	// Count matching tokens
	matchCount := 0
	for _, t1 := range tokens1 {
		for _, t2 := range tokens2 {
			if t1 == t2 {
				matchCount++
				break
			}
		}
	}

	// Return ratio of matches to average token count
	avgTokens := float64(len(tokens1)+len(tokens2)) / 2.0
	return float64(matchCount) / avgTokens
}
