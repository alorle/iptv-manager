package domain

import "testing"

func TestIsValidContentID(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{
			name:  "valid lowercase hex ID",
			id:    "1234567890abcdef1234567890abcdef12345678",
			valid: true,
		},
		{
			name:  "valid uppercase hex ID",
			id:    "1234567890ABCDEF1234567890ABCDEF12345678",
			valid: true,
		},
		{
			name:  "valid mixed case hex ID",
			id:    "1234567890AbCdEf1234567890aBcDeF12345678",
			valid: true,
		},
		{
			name:  "all zeros",
			id:    "0000000000000000000000000000000000000000",
			valid: true,
		},
		{
			name:  "all fs",
			id:    "ffffffffffffffffffffffffffffffffffffffff",
			valid: true,
		},
		{
			name:  "too short (39 chars)",
			id:    "1234567890abcdef1234567890abcdef1234567",
			valid: false,
		},
		{
			name:  "too long (41 chars)",
			id:    "1234567890abcdef1234567890abcdef123456789",
			valid: false,
		},
		{
			name:  "empty string",
			id:    "",
			valid: false,
		},
		{
			name:  "contains non-hex character (g)",
			id:    "1234567890abcdefg234567890abcdef12345678",
			valid: false,
		},
		{
			name:  "contains non-hex character (space)",
			id:    "1234567890abcdef 234567890abcdef12345678",
			valid: false,
		},
		{
			name:  "contains non-hex character (dash)",
			id:    "1234567890abcdef-234567890abcdef12345678",
			valid: false,
		},
		{
			name:  "contains special character",
			id:    "1234567890abcdef@234567890abcdef12345678",
			valid: false,
		},
		{
			name:  "unicode characters",
			id:    "1234567890abcdef1234567890abcdef1234567ü",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidContentID(tt.id)
			if result != tt.valid {
				t.Errorf("IsValidContentID(%q) = %v, want %v", tt.id, result, tt.valid)
			}
		})
	}
}
