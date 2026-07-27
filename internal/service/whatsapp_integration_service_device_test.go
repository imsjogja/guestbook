package service

import "testing"

func TestIsGOWADeviceNotFoundAcceptsV8ErrorResponse(t *testing.T) {
	body := []byte(`{"code":"INTERNAL_SERVER_ERROR","message":"device guestflow-example not found"}`)
	if !isGOWADeviceNotFound(body, 500) {
		t.Fatal("expected GOWA v8 device-not-found response to be recognized")
	}
}

func TestIsGOWADeviceNotFoundRejectsOtherErrors(t *testing.T) {
	body := []byte(`{"code":"INTERNAL_SERVER_ERROR","message":"database unavailable"}`)
	if isGOWADeviceNotFound(body, 500) {
		t.Fatal("unexpectedly classified unrelated GOWA error as device-not-found")
	}
}
