package security

import "testing"

func TestSecretRoundTrip(t *testing.T) {
	ciphertext, err := EncryptSecret("jwt-secret", "account-token")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	plaintext, err := DecryptSecret("jwt-secret", ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if plaintext != "account-token" {
		t.Fatalf("plaintext = %q, want account-token", plaintext)
	}
	if ciphertext == "account-token" {
		t.Fatal("ciphertext must not equal plaintext")
	}
}

func TestDecryptRejectsWrongKey(t *testing.T) {
	ciphertext, err := EncryptSecret("jwt-secret", "account-token")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if _, err := DecryptSecret("wrong-secret", ciphertext); err == nil {
		t.Fatal("expected wrong key to fail")
	}
}
