// Package auth provides authentication utilities for GuestFlow including
// JWT token management, password hashing, and refresh token handling.
package auth

import (
	"guestflow/pkg/crypto"
)

// HashPassword hashes a plaintext password using bcrypt with cost 12.
// Returns the bcrypt hash string which includes the salt and cost factor.
// Delegates to the crypto package for the actual implementation.
//
// Example:
//
//		hash, err := auth.HashPassword("my-secret-password")
//		if err != nil {
//		    // handle error
//	}
func HashPassword(password string) (string, error) {
	return crypto.HashPassword(password)
}

// CheckPassword verifies a plaintext password against a bcrypt hash.
// Returns true if the password matches the hash, false otherwise.
// Uses constant-time comparison to prevent timing attacks.
// Delegates to the crypto package for the actual implementation.
//
// Example:
//
//		if !auth.CheckPassword(password, storedHash) {
//		    // authentication failed
//	}
func CheckPassword(password, hash string) bool {
	return crypto.CheckPassword(password, hash)
}
