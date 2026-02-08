package stream_test

import (
	"errors"
	"testing"

	"github.com/alorle/iptv-manager/internal/stream"
)

func TestNewStream(t *testing.T) {
	tests := []struct {
		name            string
		infoHash        string
		channelName     string
		wantInfoHash    string
		wantChannelName string
		wantError       error
	}{
		{
			name:            "valid stream",
			infoHash:        "94c2fd8fa9b16211252c5e9f0b836d94155b505a",
			channelName:     "HBO",
			wantInfoHash:    "94c2fd8fa9b16211252c5e9f0b836d94155b505a",
			wantChannelName: "HBO",
			wantError:       nil,
		},
		{
			name:            "valid stream with trimmed spaces",
			infoHash:        "  94c2fd8fa9b16211252c5e9f0b836d94155b505a  ",
			channelName:     "  HBO  ",
			wantInfoHash:    "94c2fd8fa9b16211252c5e9f0b836d94155b505a",
			wantChannelName: "HBO",
			wantError:       nil,
		},
		{
			name:            "valid stream with channel name containing spaces",
			infoHash:        "94c2fd8fa9b16211252c5e9f0b836d94155b505a",
			channelName:     "HBO Max",
			wantInfoHash:    "94c2fd8fa9b16211252c5e9f0b836d94155b505a",
			wantChannelName: "HBO Max",
			wantError:       nil,
		},
		{
			name:            "empty infohash",
			infoHash:        "",
			channelName:     "HBO",
			wantInfoHash:    "",
			wantChannelName: "",
			wantError:       stream.ErrEmptyInfoHash,
		},
		{
			name:            "infohash with only whitespace",
			infoHash:        "   ",
			channelName:     "HBO",
			wantInfoHash:    "",
			wantChannelName: "",
			wantError:       stream.ErrEmptyInfoHash,
		},
		{
			name:            "infohash with only tabs",
			infoHash:        "\t\t",
			channelName:     "HBO",
			wantInfoHash:    "",
			wantChannelName: "",
			wantError:       stream.ErrEmptyInfoHash,
		},
		{
			name:            "infohash with only newlines",
			infoHash:        "\n\n",
			channelName:     "HBO",
			wantInfoHash:    "",
			wantChannelName: "",
			wantError:       stream.ErrEmptyInfoHash,
		},
		{
			name:            "infohash with mixed whitespace",
			infoHash:        " \t\n ",
			channelName:     "HBO",
			wantInfoHash:    "",
			wantChannelName: "",
			wantError:       stream.ErrEmptyInfoHash,
		},
		{
			name:            "empty channel name",
			infoHash:        "94c2fd8fa9b16211252c5e9f0b836d94155b505a",
			channelName:     "",
			wantInfoHash:    "",
			wantChannelName: "",
			wantError:       stream.ErrEmptyChannelName,
		},
		{
			name:            "channel name with only whitespace",
			infoHash:        "94c2fd8fa9b16211252c5e9f0b836d94155b505a",
			channelName:     "   ",
			wantInfoHash:    "",
			wantChannelName: "",
			wantError:       stream.ErrEmptyChannelName,
		},
		{
			name:            "channel name with only tabs",
			infoHash:        "94c2fd8fa9b16211252c5e9f0b836d94155b505a",
			channelName:     "\t\t",
			wantInfoHash:    "",
			wantChannelName: "",
			wantError:       stream.ErrEmptyChannelName,
		},
		{
			name:            "channel name with only newlines",
			infoHash:        "94c2fd8fa9b16211252c5e9f0b836d94155b505a",
			channelName:     "\n\n",
			wantInfoHash:    "",
			wantChannelName: "",
			wantError:       stream.ErrEmptyChannelName,
		},
		{
			name:            "channel name with mixed whitespace",
			infoHash:        "94c2fd8fa9b16211252c5e9f0b836d94155b505a",
			channelName:     " \t\n ",
			wantInfoHash:    "",
			wantChannelName: "",
			wantError:       stream.ErrEmptyChannelName,
		},
		{
			name:            "both empty",
			infoHash:        "",
			channelName:     "",
			wantInfoHash:    "",
			wantChannelName: "",
			wantError:       stream.ErrEmptyInfoHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := stream.NewStream(tt.infoHash, tt.channelName)

			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Errorf("NewStream() error = %v, wantError %v", err, tt.wantError)
				}
				return
			}

			if err != nil {
				t.Errorf("NewStream() unexpected error = %v", err)
				return
			}

			if got := s.InfoHash(); got != tt.wantInfoHash {
				t.Errorf("Stream.InfoHash() = %q, want %q", got, tt.wantInfoHash)
			}

			if got := s.ChannelName(); got != tt.wantChannelName {
				t.Errorf("Stream.ChannelName() = %q, want %q", got, tt.wantChannelName)
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
			name: "ErrEmptyInfoHash",
			err:  stream.ErrEmptyInfoHash,
			msg:  "infohash cannot be empty",
		},
		{
			name: "ErrEmptyChannelName",
			err:  stream.ErrEmptyChannelName,
			msg:  "channel name cannot be empty",
		},
		{
			name: "ErrStreamNotFound",
			err:  stream.ErrStreamNotFound,
			msg:  "stream not found",
		},
		{
			name: "ErrStreamAlreadyExists",
			err:  stream.ErrStreamAlreadyExists,
			msg:  "stream already exists",
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
