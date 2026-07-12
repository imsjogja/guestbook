// Package crypto provides cryptographic utilities for GuestFlow.
// Includes password hashing (bcrypt), random token generation, and SHA-256 hashing.
package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// ------------------------------------------------------------------------------
// Password hashing (bcrypt)
// ------------------------------------------------------------------------------

// Default bcrypt cost factor. Increase for production environments.
// Cost 12 provides a good balance between security and performance.
const bcryptCost = 12

// HashPassword hashes a plaintext password using bcrypt and returns the encoded hash.
// Returns an error if the password cannot be hashed.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword compares a plaintext password against a bcrypt hash.
// Returns true if the password matches the hash, false otherwise.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ------------------------------------------------------------------------------
// Random token generation
// ------------------------------------------------------------------------------

// GenerateRandomToken generates a cryptographically secure random token
// of the specified byte length, encoded as a URL-safe base64 string.
// The resulting string length will be longer than the byte length due to encoding.
//
// Example: GenerateRandomToken(32) generates a 256-bit token.
func GenerateRandomToken(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("token length must be positive, got %d", length)
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateRandomHex generates a cryptographically secure random hex string
// of the specified byte length. The resulting string is 2*length characters long.
//
// Example: GenerateRandomHex(16) generates a 32-character hex string.
func GenerateRandomHex(byteLength int) (string, error) {
	if byteLength <= 0 {
		return "", fmt.Errorf("byte length must be positive, got %d", byteLength)
	}

	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// ------------------------------------------------------------------------------
// SHA-256 hashing
// ------------------------------------------------------------------------------

// SHA256Hash computes the SHA-256 hash of the input string and returns
// it as a lowercase hex string. This is used for deterministic hashing
// such as refresh token storage (store hash, not raw token).
func SHA256Hash(input string) string {
	h := sha256.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

// ------------------------------------------------------------------------------
// Secure comparison
// ------------------------------------------------------------------------------

// SecureCompare performs a constant-time comparison of two strings
// to prevent timing attacks. Returns true if the strings are equal.
func SecureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
