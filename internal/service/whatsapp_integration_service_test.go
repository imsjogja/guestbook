package service

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDefaultTenantDeviceIDIsDeterministicAndScoped(t *testing.T) {
	tenantID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	deviceID := defaultTenantDeviceID(tenantID)

	if deviceID != "guestflow-a0eebc999c0b4ef8bb6d6bb9bd380a11" {
		t.Fatalf("unexpected device ID: %s", deviceID)
	}
	if deviceID == defaultTenantDeviceID(uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12")) {
		t.Fatal("different tenants must not share a device ID")
	}
}

func TestValidateWhatsAppDeviceID(t *testing.T) {
	valid := []string{"guestflow-main", "tenant.device_01", "abc"}
	for _, value := range valid {
		if err := validateWhatsAppDeviceID(value); err != nil {
			t.Errorf("expected %q to be valid: %v", value, err)
		}
	}

	invalid := []string{"", "ab", strings.Repeat("a", 65), "tenant/device"}
	for _, value := range invalid {
		if err := validateWhatsAppDeviceID(value); err == nil {
			t.Errorf("expected %q to be invalid", value)
		}
	}
}

func TestPhoneNumberFromJID(t *testing.T) {
	tests := map[string]string{
		"6281234567890@s.whatsapp.net":    "6281234567890",
		"6281234567890:42@s.whatsapp.net": "6281234567890",
		"":                                "",
	}
	for jid, want := range tests {
		if got := phoneNumberFromJID(jid); got != want {
			t.Errorf("phoneNumberFromJID(%q) = %q, want %q", jid, got, want)
		}
	}
}
