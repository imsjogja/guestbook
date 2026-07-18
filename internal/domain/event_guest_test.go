package domain

import "testing"

func TestIsValidEventGuestSource(t *testing.T) {
	tests := []struct {
		source string
		valid  bool
	}{
		{EventGuestSourceManual, true},
		{EventGuestSourceInvitation, true},
		{EventGuestSourceWalkIn, true},
		{"tenant_master", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := IsValidEventGuestSource(tt.source); got != tt.valid {
				t.Fatalf("IsValidEventGuestSource(%q) = %v, want %v", tt.source, got, tt.valid)
			}
		})
	}
}
