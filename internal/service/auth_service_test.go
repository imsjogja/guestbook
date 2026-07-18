package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"guestflow/internal/domain"
)

type captureMailer struct {
	to      string
	subject string
	body    string
}

func (m *captureMailer) Send(_ context.Context, to, subject, body string) error {
	m.to, m.subject, m.body = to, subject, body
	return nil
}

func TestNewVerificationTokenStoresOnlyHash(t *testing.T) {
	raw, token, err := newVerificationToken(uuid.New())
	if err != nil {
		t.Fatalf("newVerificationToken returned error: %v", err)
	}
	if len(raw) != 64 {
		t.Fatalf("expected 64-character token, got %d", len(raw))
	}
	if token.TokenHash == raw || token.TokenHash != hashVerificationToken(raw) {
		t.Fatal("verification token must be stored as its hash")
	}
}

func TestSendVerificationEmailBuildsPublicVerificationURL(t *testing.T) {
	mailer := &captureMailer{}
	service := &AuthService{
		mailer:    mailer,
		publicURL: "https://guestflow.id/",
	}

	if err := service.sendVerificationEmail(context.Background(), "member@example.com", "Member", "raw-token"); err != nil {
		t.Fatalf("sendVerificationEmail returned error: %v", err)
	}
	if mailer.to != "member@example.com" || mailer.subject == "" {
		t.Fatalf("unexpected captured email metadata: %+v", mailer)
	}
	if !strings.Contains(mailer.body, "https://guestflow.id/verify-email?token=raw-token") {
		t.Fatalf("verification URL missing from email body: %s", mailer.body)
	}
	if !strings.Contains(mailer.body, "berlaku selama 24 jam") {
		t.Fatal("email expiry guidance missing")
	}
}

func TestNewAuthEmailTokenHasPurposeAndExpiry(t *testing.T) {
	before := time.Now().UTC()
	raw, token, err := newAuthEmailToken(uuid.New(), domain.AuthEmailTokenMagicLogin, 15*time.Minute)
	if err != nil {
		t.Fatalf("newAuthEmailToken returned error: %v", err)
	}
	if len(raw) != 64 || token.Purpose != domain.AuthEmailTokenMagicLogin {
		t.Fatalf("unexpected auth email token: %+v", token)
	}
	if !token.ExpiresAt.After(before.Add(14 * time.Minute)) {
		t.Fatal("auth email token expiry is too short")
	}
}
