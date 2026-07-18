package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

var ErrInvalidSecret = errors.New("invalid encrypted secret")

// EncryptSecret encrypts a provider credential for storage in tenant settings.
// The application JWT secret is used only as the key derivation input; the
// credential itself is never returned through an API response.
func EncryptSecret(key, plaintext string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", errors.New("encryption key is empty")
	}
	block, err := aes.NewCipher(deriveKey(key))
	if err != nil {
		return "", fmt.Errorf("create secret cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create secret gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate secret nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	encoded := base64.RawStdEncoding.EncodeToString(append(nonce, ciphertext...))
	return "v1:" + encoded, nil
}

// DecryptSecret decrypts a credential previously produced by EncryptSecret.
func DecryptSecret(key, encrypted string) (string, error) {
	if strings.TrimSpace(key) == "" || !strings.HasPrefix(encrypted, "v1:") {
		return "", ErrInvalidSecret
	}
	data, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(encrypted, "v1:"))
	if err != nil {
		return "", ErrInvalidSecret
	}
	block, err := aes.NewCipher(deriveKey(key))
	if err != nil {
		return "", ErrInvalidSecret
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(data) < gcm.NonceSize() {
		return "", ErrInvalidSecret
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrInvalidSecret
	}
	return string(plaintext), nil
}

func deriveKey(key string) []byte {
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}

// MaskSecret returns a non-sensitive representation suitable for UI status.
func MaskSecret(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "..." + value[len(value)-4:]
}
