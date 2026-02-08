package channel_test

import (
	"errors"
	"testing"

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.msg)
			}
		})
	}
}
