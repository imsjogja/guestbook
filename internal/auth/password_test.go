package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	t.Run("successful hashing", func(t *testing.T) {
		hash, err := HashPassword("secure-password-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hash == "" {
			t.Error("expected non-empty hash")
		}
		// bcrypt hashes start with $2a$, $2b$, or $2y$ and have cost factor
		if !strings.HasPrefix(hash, "$2") {
			t.Errorf("expected bcrypt hash prefix, got: %s", hash[:10])
		}
		// Verify cost is 12
		cost, err := bcrypt.Cost([]byte(hash))
		if err != nil {
			t.Fatalf("failed to get bcrypt cost: %v", err)
		}
		if cost != bcryptCost {
			t.Errorf("expected cost %d, got %d", bcryptCost, cost)
		}
	})

	t.Run("different hashes for same password", func(t *testing.T) {
		password := "same-password"
		hash1, _ := HashPassword(password)
		hash2, _ := HashPassword(password)
		if hash1 == hash2 {
			t.Error("expected different hashes due to random salt")
		}
	})

	t.Run("long password", func(t *testing.T) {
		longPassword := strings.Repeat("a", 100)
		_, err := HashPassword(longPassword)
		if err != nil {
			t.Fatalf("unexpected error for long password: %v", err)
		}
	})
}

func TestCheckPassword(t *testing.T) {
	t.Run("correct password", func(t *testing.T) {
		password := "my-secret-password"
		hash, _ := HashPassword(password)
		if !CheckPassword(password, hash) {
			t.Error("expected password to match")
		}
	})

	t.Run("incorrect password", func(t *testing.T) {
		hash, _ := HashPassword("correct-password")
		if CheckPassword("wrong-password", hash) {
			t.Error("expected password to not match")
		}
	})

	t.Run("empty password", func(t *testing.T) {
		hash, _ := HashPassword("")
		if !CheckPassword("", hash) {
			t.Error("expected empty password to match")
		}
		if CheckPassword("not-empty", hash) {
			t.Error("expected non-empty password to not match empty hash")
		}
	})

	t.Run("invalid hash", func(t *testing.T) {
		if CheckPassword("password", "not-a-valid-hash") {
			t.Error("expected invalid hash to return false")
		}
	})
}
