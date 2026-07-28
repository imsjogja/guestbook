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
		publicURL: "https://app.guestflow.id/",
	}

	if err := service.sendVerificationEmail(context.Background(), "member@example.com", "Member", "raw-token"); err != nil {
		t.Fatalf("sendVerificationEmail returned error: %v", err)
	}
	if mailer.to != "member@example.com" || mailer.subject == "" {
		t.Fatalf("unexpected captured email metadata: %+v", mailer)
	}
	if !strings.Contains(mailer.body, "https://app.guestflow.id/verify-email?token=raw-token") {
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

func TestNewRegistrationTenantCreatesOwnerWorkspace(t *testing.T) {
	userID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a21")
	tenant := newRegistrationTenant(domain.RegisterRequest{
		FullName:   "Bambang Kusniawan",
		TenantName: "GuestFlow Partner",
	}, userID)

	if tenant.Name != "GuestFlow Partner" {
		t.Fatalf("unexpected tenant name: %q", tenant.Name)
	}
	if tenant.Slug != "guestflow-partner-a0eebc99" {
		t.Fatalf("unexpected tenant slug: %q", tenant.Slug)
	}
	if tenant.Status != domain.TenantStatusTrial || tenant.TrialEndsAt == nil {
		t.Fatalf("new registration tenant should start a trial: %+v", tenant)
	}
}

func TestNewRegistrationTenantFallsBackToUserWorkspaceName(t *testing.T) {
	userID := uuid.MustParse("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22")
	tenant := newRegistrationTenant(domain.RegisterRequest{FullName: "New Member"}, userID)

	if tenant.Name != "New Member Workspace" {
		t.Fatalf("unexpected fallback tenant name: %q", tenant.Name)
	}
	if tenant.Slug != "new-member-workspace-b0eebc99" {
		t.Fatalf("unexpected fallback tenant slug: %q", tenant.Slug)
	}
}
