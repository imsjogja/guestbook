package payment

import (
	"crypto/sha512"
	"fmt"
	"testing"
)

func TestVerifySignature(t *testing.T) {
	client := NewClient("server-key", "client-key", false)
	raw := "GF-order2001000server-key"
	hash := sha512.Sum512([]byte(raw))

	if !client.VerifySignature("GF-order", "200", "1000", fmt.Sprintf("%x", hash)) {
		t.Fatal("expected valid Midtrans signature")
	}
	if client.VerifySignature("GF-order", "200", "1000", "invalid") {
		t.Fatal("expected invalid Midtrans signature")
	}
}

func TestTransactionStatusHelpers(t *testing.T) {
	if !IsSuccessStatus("settlement", "") {
		t.Fatal("settlement should be successful")
	}
	if !IsSuccessStatus("capture", "accept") {
		t.Fatal("accepted capture should be successful")
	}
	if IsSuccessStatus("capture", "deny") {
		t.Fatal("denied capture should not be successful")
	}
	if !IsFailedStatus("cancel") || !IsExpiredStatus("expire") {
		t.Fatal("expected failed and expired status helpers to match")
	}
}
