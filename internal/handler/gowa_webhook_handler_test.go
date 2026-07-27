package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestVerifyGOWASignature(t *testing.T) {
	secret := "webhook-secret"
	body := []byte(`{"event":"message.ack"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !verifyGOWASignature(secret, body, signature) {
		t.Fatal("verifyGOWASignature() rejected a valid signature")
	}
	if verifyGOWASignature(secret, body, "sha256="+strings.Repeat("0", 64)) {
		t.Fatal("verifyGOWASignature() accepted an invalid signature")
	}
}
