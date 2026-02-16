package channel_test

import (
	"errors"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/internal/channel"
)

func TestNewChannel(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantError error
	}{
		{
			name:      "valid channel name",
			input:     "HBO",
			wantName:  "HBO",
			wantError: nil,
		},
		{
			name:      "valid channel name with spaces trimmed",
			input:     "  HBO  ",
			wantName:  "HBO",
			wantError: nil,
		},
		{
			name:      "valid channel name with multiple words",
			input:     "HBO Max",
			wantName:  "HBO Max",
			wantError: nil,
		},
		{
			name:      "empty string",
			input:     "",
			wantName:  "",
			wantError: channel.ErrEmptyName,
		},
		{
			name:      "only whitespace",
			input:     "   ",
			wantName:  "",
			wantError: channel.ErrEmptyName,
		},
		{
			name:      "only tabs",
			input:     "\t\t",
			wantName:  "",
			wantError: channel.ErrEmptyName,
		},
		{
			name:      "only newlines",
			input:     "\n\n",
			wantName:  "",
			wantError: channel.ErrEmptyName,
		},
		{
			name:      "mixed whitespace",
			input:     " \t\n ",
			wantName:  "",
			wantError: channel.ErrEmptyName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch, err := channel.NewChannel(tt.input)

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

			if got := ch.Name(); got != tt.wantName {
				t.Errorf("Channel.Name() = %q, want %q", got, tt.wantName)
			}

			if got := ch.Status(); got != channel.StatusActive {
				t.Errorf("Channel.Status() = %v, want %v", got, channel.StatusActive)
			}

			if got := ch.EPGMapping(); got != nil {
				t.Errorf("Channel.EPGMapping() = %v, want nil", got)
			}
		})
	}
}

func TestChannelStatus(t *testing.T) {
	ch, err := channel.NewChannel("HBO")
	if err != nil {
		t.Fatalf("NewChannel() unexpected error = %v", err)
	}

	if got := ch.Status(); got != channel.StatusActive {
		t.Errorf("initial Status() = %v, want %v", got, channel.StatusActive)
	}

	ch.Archive()

	if got := ch.Status(); got != channel.StatusArchived {
		t.Errorf("Status() after Archive() = %v, want %v", got, channel.StatusArchived)
	}
}

func TestNewEPGMapping(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		epgID      string
		source     channel.MappingSource
		lastSynced time.Time
		wantEPGID  string
		wantSource channel.MappingSource
		wantSynced time.Time
		wantError  error
	}{
		{
			name:       "valid auto mapping",
			epgID:      "hbo-hd",
			source:     channel.MappingAuto,
			lastSynced: now,
			wantEPGID:  "hbo-hd",
			wantSource: channel.MappingAuto,
			wantSynced: now,
			wantError:  nil,
		},
		{
			name:       "valid manual mapping",
			epgID:      "hbo-max",
			source:     channel.MappingManual,
			lastSynced: now,
			wantEPGID:  "hbo-max",
			wantSource: channel.MappingManual,
			wantSynced: now,
			wantError:  nil,
		},
		{
			name:       "trims epg id whitespace",
			epgID:      "  hbo-hd  ",
			source:     channel.MappingAuto,
			lastSynced: now,
			wantEPGID:  "hbo-hd",
			wantSource: channel.MappingAuto,
			wantSynced: now,
			wantError:  nil,
		},
		{
			name:       "invalid mapping source",
			epgID:      "hbo-hd",
			source:     channel.MappingSource("invalid"),
			lastSynced: now,
			wantError:  channel.ErrInvalidMappingSource,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping, err := channel.NewEPGMapping(tt.epgID, tt.source, tt.lastSynced)

			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Errorf("NewEPGMapping() error = %v, wantError %v", err, tt.wantError)
				}
				return
			}

			if err != nil {
				t.Errorf("NewEPGMapping() unexpected error = %v", err)
				return
			}

			if got := mapping.EPGID(); got != tt.wantEPGID {
				t.Errorf("EPGMapping.EPGID() = %q, want %q", got, tt.wantEPGID)
			}

			if got := mapping.Source(); got != tt.wantSource {
				t.Errorf("EPGMapping.Source() = %v, want %v", got, tt.wantSource)
			}

			if got := mapping.LastSynced(); !got.Equal(tt.wantSynced) {
				t.Errorf("EPGMapping.LastSynced() = %v, want %v", got, tt.wantSynced)
			}
		})
	}
}

func TestChannelEPGMapping(t *testing.T) {
	ch, err := channel.NewChannel("HBO")
	if err != nil {
		t.Fatalf("NewChannel() unexpected error = %v", err)
	}

	if got := ch.EPGMapping(); got != nil {
		t.Fatalf("initial EPGMapping() = %v, want nil", got)
	}

	now := time.Now()
	mapping, err := channel.NewEPGMapping("hbo-hd", channel.MappingAuto, now)
	if err != nil {
		t.Fatalf("NewEPGMapping() unexpected error = %v", err)
	}

	ch.SetEPGMapping(mapping)

	if got := ch.EPGMapping(); got == nil {
		t.Fatal("EPGMapping() after SetEPGMapping() = nil, want non-nil")
	}

	if got := ch.EPGMapping().EPGID(); got != "hbo-hd" {
		t.Errorf("EPGMapping().EPGID() = %q, want %q", got, "hbo-hd")
	}

	if got := ch.EPGMapping().Source(); got != channel.MappingAuto {
		t.Errorf("EPGMapping().Source() = %v, want %v", got, channel.MappingAuto)
	}

	ch.ClearEPGMapping()

	if got := ch.EPGMapping(); got != nil {
		t.Errorf("EPGMapping() after ClearEPGMapping() = %v, want nil", got)
	}
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase conversion",
			input: "HBO",
			want:  "hbo",
		},
		{
			name:  "multiple spaces collapsed",
			input: "HBO  Max",
			want:  "hbo max",
		},
		{
			name:  "punctuation removed",
			input: "HBO-Max!",
			want:  "hbo max",
		},
		{
			name:  "mixed case and punctuation",
			input: "HBO-Max: Premium!",
			want:  "hbo max premium",
		},
		{
			name:  "leading and trailing spaces",
			input: "  HBO Max  ",
			want:  "hbo max",
		},
		{
			name:  "numbers preserved and quality suffix stripped",
			input: "Channel 5 HD",
			want:  "channel 5",
		},
		{
			name:  "fhd suffix stripped",
			input: "DAZN 1 FHD",
			want:  "dazn 1",
		},
		{
			name:  "hd suffix stripped",
			input: "DAZN LaLiga HD",
			want:  "dazn laliga",
		},
		{
			name:  "complex punctuation",
			input: "HBO+Max (Premium)",
			want:  "hbo max premium",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channel.NormalizeName(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name      string
		name1     string
		name2     string
		wantScore float64
		minScore  float64
	}{
		{
			name:      "exact match",
			name1:     "HBO",
			name2:     "HBO",
			wantScore: 1.0,
			minScore:  1.0,
		},
		{
			name:      "exact match with different case",
			name1:     "HBO",
			name2:     "hbo",
			wantScore: 1.0,
			minScore:  1.0,
		},
		{
			name:      "exact match with punctuation",
			name1:     "HBO-Max",
			name2:     "HBO Max",
			wantScore: 1.0,
			minScore:  1.0,
		},
		{
			name:      "substring match",
			name1:     "HBO",
			name2:     "HBO Max",
			wantScore: 0.8,
			minScore:  0.8,
		},
		{
			name:      "substring match reversed",
			name1:     "HBO Max",
			name2:     "HBO",
			wantScore: 0.8,
			minScore:  0.8,
		},
		{
			name:     "partial token match",
			name1:    "HBO Max Premium",
			name2:    "HBO Sports",
			minScore: 0.25,
		},
		{
			name:      "no match",
			name1:     "HBO",
			name2:     "Netflix",
			wantScore: 0.0,
			minScore:  0.0,
		},
		{
			name:      "empty first name",
			name1:     "",
			name2:     "HBO",
			wantScore: 0.0,
			minScore:  0.0,
		},
		{
			name:      "empty second name",
			name1:     "HBO",
			name2:     "",
			wantScore: 0.0,
			minScore:  0.0,
		},
		{
			name:      "both empty",
			name1:     "",
			name2:     "",
			wantScore: 0.0,
			minScore:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channel.FuzzyMatch(tt.name1, tt.name2)

			if tt.wantScore > 0 {
				if got != tt.wantScore {
					t.Errorf("FuzzyMatch(%q, %q) = %v, want %v", tt.name1, tt.name2, got, tt.wantScore)
				}
			} else if got < tt.minScore {
				t.Errorf("FuzzyMatch(%q, %q) = %v, want >= %v", tt.name1, tt.name2, got, tt.minScore)
			}

			if got < 0.0 || got > 1.0 {
				t.Errorf("FuzzyMatch(%q, %q) = %v, want score in range [0.0, 1.0]", tt.name1, tt.name2, got)
			}
		})
	}
}

func TestDomainErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "ErrEmptyName",
			err:  channel.ErrEmptyName,
			msg:  "channel name cannot be empty",
		},
		{
			name: "ErrChannelNotFound",
			err:  channel.ErrChannelNotFound,
			msg:  "channel not found",
		},
		{
			name: "ErrChannelAlreadyExists",
			err:  channel.ErrChannelAlreadyExists,
			msg:  "channel already exists",
		},
		{
			name: "ErrInvalidMappingSource",
			err:  channel.ErrInvalidMappingSource,
			msg:  "invalid mapping source",
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
