package epg

import (
	"strings"
)

// Channel represents an EPG (Electronic Program Guide) channel in the domain.
// It contains metadata about a channel from an external EPG source.
type Channel struct {
	id       string
	name     string
	logo     string
	category string
	language string
	epgID    string
}

// NewChannel creates a new EPG Channel with the given attributes.
// It validates that id, name, and epgID are not empty and trims whitespace.
// Returns ErrEmptyID if the id is empty or contains only whitespace.
// Returns ErrEmptyName if the name is empty or contains only whitespace.
// Returns ErrEmptyEPGID if the epgID is empty or contains only whitespace.
func NewChannel(id, name, logo, category, language, epgID string) (Channel, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return Channel{}, ErrEmptyID
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Channel{}, ErrEmptyName
	}

	trimmedEPGID := strings.TrimSpace(epgID)
	if trimmedEPGID == "" {
		return Channel{}, ErrEmptyEPGID
	}

	return Channel{
		id:       trimmedID,
		name:     trimmedName,
		logo:     strings.TrimSpace(logo),
		category: strings.TrimSpace(category),
		language: strings.TrimSpace(language),
		epgID:    trimmedEPGID,
	}, nil
}

// ID returns the channel's unique identifier.
func (c Channel) ID() string {
	return c.id
}

// Name returns the channel's name.
func (c Channel) Name() string {
	return c.name
}

// Logo returns the channel's logo URL.
func (c Channel) Logo() string {
	return c.logo
}

// Category returns the channel's category (e.g., Sports, News, Entertainment).
func (c Channel) Category() string {
	return c.category
}

// Language returns the channel's language.
func (c Channel) Language() string {
	return c.language
}

// EPGID returns the channel's EPG identifier used to match with program data.
func (c Channel) EPGID() string {
	return c.epgID
}
