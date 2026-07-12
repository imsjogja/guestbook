package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewJWTService(t *testing.T) {
	svc := NewJWTService("access-secret", "refresh-secret", 15*time.Minute, 7*24*time.Hour)
	if svc == nil {
		t.Fatal("expected non-nil JWTService")
	}
	if svc.issuer != DefaultIssuer {
		t.Errorf("expected issuer %q, got %q", DefaultIssuer, svc.issuer)
	}
}

func TestJWTService_GenerateTokenPair(t *testing.T) {
	svc := NewJWTService("test-access-secret", "test-refresh-secret", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	t.Run("successful generation", func(t *testing.T) {
		pair, err := svc.GenerateTokenPair(userID, "test@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pair.AccessToken == "" {
			t.Error("expected non-empty access token")
		}
		if pair.RefreshToken == "" {
			t.Error("expected non-empty refresh token")
		}
		if pair.ExpiresIn != 900 {
			t.Errorf("expected expires_in 900, got %d", pair.ExpiresIn)
		}
	})

	t.Run("different users get different tokens", func(t *testing.T) {
		userID2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		pair1, _ := svc.GenerateTokenPair(userID, "user1@example.com")
		pair2, _ := svc.GenerateTokenPair(userID2, "user2@example.com")

		if pair1.AccessToken == pair2.AccessToken {
			t.Error("expected different access tokens for different users")
		}
		if pair1.RefreshToken == pair2.RefreshToken {
			t.Error("expected different refresh tokens for different users")
		}
	})
}

func TestJWTService_GenerateAccessToken(t *testing.T) {
	svc := NewJWTService("test-access-secret", "test-refresh-secret", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	tenantID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	t.Run("with tenant and role", func(t *testing.T) {
		token, err := svc.GenerateAccessToken(userID, "test@example.com", tenantID, "admin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token == "" {
			t.Error("expected non-empty token")
		}
	})

	t.Run("without tenant and role", func(t *testing.T) {
		token, err := svc.GenerateAccessToken(userID, "test@example.com", uuid.Nil, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token == "" {
			t.Error("expected non-empty token")
		}
	})
}

func TestJWTService_ValidateAccessToken(t *testing.T) {
	svc := NewJWTService("test-access-secret", "test-refresh-secret", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	t.Run("valid token", func(t *testing.T) {
		token, err := svc.GenerateAccessToken(userID, "test@example.com", uuid.Nil, "user")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		claims, err := svc.ValidateAccessToken(token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims.UserID != userID {
			t.Errorf("expected userID %v, got %v", userID, claims.UserID)
		}
		if claims.Email != "test@example.com" {
			t.Errorf("expected email test@example.com, got %s", claims.Email)
		}
		if claims.Role != "user" {
			t.Errorf("expected role user, got %s", claims.Role)
		}
	})

	t.Run("invalid token format", func(t *testing.T) {
		_, err := svc.ValidateAccessToken("not-a-valid-token")
		if err == nil {
			t.Error("expected error for invalid token")
		}
	})

	t.Run("token signed with wrong secret", func(t *testing.T) {
		otherSvc := NewJWTService("different-secret", "different-refresh", 15*time.Minute, 7*24*time.Hour)
		token, _ := otherSvc.GenerateAccessToken(userID, "test@example.com", uuid.Nil, "")

		_, err := svc.ValidateAccessToken(token)
		if err == nil {
			t.Error("expected error for token signed with different secret")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		shortSvc := NewJWTService("test-access-secret", "test-refresh-secret", -1*time.Second, 7*24*time.Hour)
		token, _ := shortSvc.GenerateAccessToken(userID, "test@example.com", uuid.Nil, "")

		time.Sleep(10 * time.Millisecond)
		_, err := svc.ValidateAccessToken(token)
		if err == nil {
			t.Error("expected error for expired token")
		}
	})
}

func TestJWTService_ValidateRefreshToken(t *testing.T) {
	svc := NewJWTService("test-access-secret", "test-refresh-secret", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	t.Run("valid refresh token", func(t *testing.T) {
		pair, err := svc.GenerateTokenPair(userID, "test@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		claims, err := svc.ValidateRefreshToken(pair.RefreshToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims.UserID != userID {
			t.Errorf("expected userID %v, got %v", userID, claims.UserID)
		}
	})

	t.Run("access token fails refresh validation", func(t *testing.T) {
		pair, _ := svc.GenerateTokenPair(userID, "test@example.com")

		_, err := svc.ValidateRefreshToken(pair.AccessToken)
		if err == nil {
			t.Error("expected error when validating access token as refresh token")
		}
	})
}

func TestTokenPair(t *testing.T) {
	pair := &TokenPair{
		AccessToken:  "access123",
		RefreshToken: "refresh456",
		ExpiresIn:    900,
	}
	if pair.AccessToken != "access123" {
		t.Error("access token mismatch")
	}
	if pair.RefreshToken != "refresh456" {
		t.Error("refresh token mismatch")
	}
	if pair.ExpiresIn != 900 {
		t.Errorf("expires_in mismatch: got %d, want 900", pair.ExpiresIn)
	}
}
