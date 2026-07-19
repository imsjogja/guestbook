package service

import (
	"testing"

	"guestflow/internal/domain"
)

func TestNormalizeImportPhone(t *testing.T) {
	tests := map[string]string{
		"620000000001":      "+620000000001",
		"0813-2993-6537":    "+6281329936537",
		"+62 813 2993 6537": "+6281329936537",
		"006281329936537":   "+6281329936537",
	}

	for input, want := range tests {
		if got := normalizeImportPhone(input); got != want {
			t.Errorf("normalizeImportPhone(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeImportGuestType(t *testing.T) {
	tests := map[string]string{
		"VIP":      domain.GuestTypeVIP,
		"Regular":  domain.GuestTypeGeneral,
		"Vendor":   domain.GuestTypeVendor,
		"Keluarga": domain.GuestTypeFamily,
	}

	for input, want := range tests {
		if got := normalizeImportGuestType(input); got != want {
			t.Errorf("normalizeImportGuestType(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseRowNormalizesDummyImportValues(t *testing.T) {
	row := parseRow(
		[]string{"Andi Pratama", "620000000001", "VIP"},
		map[string]int{"full_name": 0, "phone": 1, "guest_type": 2},
		2,
	)
	validateRow(&row)

	if row.Phone != "+620000000001" {
		t.Fatalf("phone = %q, want normalized E.164 value", row.Phone)
	}
	if row.GuestType != domain.GuestTypeVIP {
		t.Fatalf("guest type = %q, want %q", row.GuestType, domain.GuestTypeVIP)
	}
	if len(row.Errors) != 0 {
		t.Fatalf("normalized row has validation errors: %v", row.Errors)
	}
}
