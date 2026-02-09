package epg_test

import (
	"errors"
	"testing"

	"github.com/alorle/iptv-manager/internal/epg"
)

func TestNewChannel(t *testing.T) {
	tests := []struct {
		name         string
		id           string
		channelName  string
		logo         string
		category     string
		language     string
		epgID        string
		wantID       string
		wantName     string
		wantLogo     string
		wantCategory string
		wantLanguage string
		wantEPGID    string
		wantError    error
	}{
		{
			name:         "valid channel with all fields",
			id:           "ch-123",
			channelName:  "HBO HD",
			logo:         "https://example.com/hbo.png",
			category:     "Movies",
			language:     "en",
			epgID:        "hbo-hd",
			wantID:       "ch-123",
			wantName:     "HBO HD",
			wantLogo:     "https://example.com/hbo.png",
			wantCategory: "Movies",
			wantLanguage: "en",
			wantEPGID:    "hbo-hd",
			wantError:    nil,
		},
		{
			name:         "valid channel with trimmed whitespace",
			id:           "  ch-123  ",
			channelName:  "  HBO HD  ",
			logo:         "  https://example.com/hbo.png  ",
			category:     "  Movies  ",
			language:     "  en  ",
			epgID:        "  hbo-hd  ",
			wantID:       "ch-123",
			wantName:     "HBO HD",
			wantLogo:     "https://example.com/hbo.png",
			wantCategory: "Movies",
			wantLanguage: "en",
			wantEPGID:    "hbo-hd",
			wantError:    nil,
		},
		{
			name:         "valid channel with empty optional fields",
			id:           "ch-123",
			channelName:  "HBO HD",
			logo:         "",
			category:     "",
			language:     "",
			epgID:        "hbo-hd",
			wantID:       "ch-123",
			wantName:     "HBO HD",
			wantLogo:     "",
			wantCategory: "",
			wantLanguage: "",
			wantEPGID:    "hbo-hd",
			wantError:    nil,
		},
		{
			name:        "empty id",
			id:          "",
			channelName: "HBO HD",
			logo:        "",
			category:    "",
			language:    "",
			epgID:       "hbo-hd",
			wantError:   epg.ErrEmptyID,
		},
		{
			name:        "whitespace only id",
			id:          "   ",
			channelName: "HBO HD",
			logo:        "",
			category:    "",
			language:    "",
			epgID:       "hbo-hd",
			wantError:   epg.ErrEmptyID,
		},
		{
			name:        "empty name",
			id:          "ch-123",
			channelName: "",
			logo:        "",
			category:    "",
			language:    "",
			epgID:       "hbo-hd",
			wantError:   epg.ErrEmptyName,
		},
		{
			name:        "whitespace only name",
			id:          "ch-123",
			channelName: "   ",
			logo:        "",
			category:    "",
			language:    "",
			epgID:       "hbo-hd",
			wantError:   epg.ErrEmptyName,
		},
		{
			name:        "empty epg id",
			id:          "ch-123",
			channelName: "HBO HD",
			logo:        "",
			category:    "",
			language:    "",
			epgID:       "",
			wantError:   epg.ErrEmptyEPGID,
		},
		{
			name:        "whitespace only epg id",
			id:          "ch-123",
			channelName: "HBO HD",
			logo:        "",
			category:    "",
			language:    "",
			epgID:       "   ",
			wantError:   epg.ErrEmptyEPGID,
		},
		{
			name:        "tabs in id",
			id:          "\t\t",
			channelName: "HBO HD",
			logo:        "",
			category:    "",
			language:    "",
			epgID:       "hbo-hd",
			wantError:   epg.ErrEmptyID,
		},
		{
			name:        "newlines in name",
			id:          "ch-123",
			channelName: "\n\n",
			logo:        "",
			category:    "",
			language:    "",
			epgID:       "hbo-hd",
			wantError:   epg.ErrEmptyName,
		},
		{
			name:        "mixed whitespace in epg id",
			id:          "ch-123",
			channelName: "HBO HD",
			logo:        "",
			category:    "",
			language:    "",
			epgID:       " \t\n ",
			wantError:   epg.ErrEmptyEPGID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch, err := epg.NewChannel(tt.id, tt.channelName, tt.logo, tt.category, tt.language, tt.epgID)

			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Errorf("NewChannel() error = %v, wantError %v", err, tt.wantError)
				}
				return
			}

			if err != nil {
				t.Errorf("NewChannel() unexpected error = %v", err)
				return
			}

			if got := ch.ID(); got != tt.wantID {
				t.Errorf("Channel.ID() = %q, want %q", got, tt.wantID)
			}

			if got := ch.Name(); got != tt.wantName {
				t.Errorf("Channel.Name() = %q, want %q", got, tt.wantName)
			}

			if got := ch.Logo(); got != tt.wantLogo {
				t.Errorf("Channel.Logo() = %q, want %q", got, tt.wantLogo)
			}

			if got := ch.Category(); got != tt.wantCategory {
				t.Errorf("Channel.Category() = %q, want %q", got, tt.wantCategory)
			}

			if got := ch.Language(); got != tt.wantLanguage {
				t.Errorf("Channel.Language() = %q, want %q", got, tt.wantLanguage)
			}

			if got := ch.EPGID(); got != tt.wantEPGID {
				t.Errorf("Channel.EPGID() = %q, want %q", got, tt.wantEPGID)
			}
		})
	}
}

func TestChannelGetters(t *testing.T) {
	ch, err := epg.NewChannel("ch-123", "HBO HD", "https://example.com/hbo.png", "Movies", "en", "hbo-hd")
	if err != nil {
		t.Fatalf("NewChannel() unexpected error = %v", err)
	}

	if got := ch.ID(); got != "ch-123" {
		t.Errorf("ID() = %q, want %q", got, "ch-123")
	}

	if got := ch.Name(); got != "HBO HD" {
		t.Errorf("Name() = %q, want %q", got, "HBO HD")
	}

	if got := ch.Logo(); got != "https://example.com/hbo.png" {
		t.Errorf("Logo() = %q, want %q", got, "https://example.com/hbo.png")
	}

	if got := ch.Category(); got != "Movies" {
		t.Errorf("Category() = %q, want %q", got, "Movies")
	}

	if got := ch.Language(); got != "en" {
		t.Errorf("Language() = %q, want %q", got, "en")
	}

	if got := ch.EPGID(); got != "hbo-hd" {
		t.Errorf("EPGID() = %q, want %q", got, "hbo-hd")
	}
}

func TestEPGDomainErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "ErrEmptyID",
			err:  epg.ErrEmptyID,
			msg:  "epg channel id cannot be empty",
		},
		{
			name: "ErrEmptyName",
			err:  epg.ErrEmptyName,
			msg:  "epg channel name cannot be empty",
		},
		{
			name: "ErrEmptyEPGID",
			err:  epg.ErrEmptyEPGID,
			msg:  "epg channel epg id cannot be empty",
		},
		{
			name: "ErrEmptyURL",
			err:  epg.ErrEmptyURL,
			msg:  "epg source url cannot be empty",
		},
		{
			name: "ErrInvalidEPGFormat",
			err:  epg.ErrInvalidEPGFormat,
			msg:  "invalid epg format",
		},
		{
			name: "ErrChannelNotFound",
			err:  epg.ErrChannelNotFound,
			msg:  "epg channel not found",
		},
		{
			name: "ErrSourceNotFound",
			err:  epg.ErrSourceNotFound,
			msg:  "epg source not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.msg)
			}
		})
	}
}
