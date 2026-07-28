package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"guestflow/internal/whatsapp"

	"github.com/labstack/echo/v4"
)

type githubNotificationSenderMock struct {
	recipients []string
	messages   []string
	err        error
}

func (m *githubNotificationSenderMock) SendTo(_ context.Context, recipient, message string) (whatsapp.SendReceipt, error) {
	m.recipients = append(m.recipients, recipient)
	m.messages = append(m.messages, message)
	if m.err != nil {
		return whatsapp.SendReceipt{}, m.err
	}
	return whatsapp.SendReceipt{}, nil
}

func TestGitHubIssueNotificationHandlerSendsToConfiguredRecipients(t *testing.T) {
	sender := &githubNotificationSenderMock{}
	handler := NewGitHubIssueNotificationHandler(sender, "webhook-secret", "0812-1; 0812-2", "imsjogja/guestbook-ui")
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github/issues", strings.NewReader(`{
		"action":"closed",
		"repository":"imsjogja/guestbook-ui",
		"issue_number":15,
		"title":"Riwayat pesan",
		"issue_url":"https://github.com/imsjogja/guestbook-ui/issues/15",
		"delivery_id":"run-15"
	}`))
	req.Header.Set("Authorization", "Bearer webhook-secret")
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	ctx := e.NewContext(req, res)

	if err := handler.Handle(ctx); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if len(sender.recipients) != 2 {
		t.Fatalf("send count = %d, want 2", len(sender.recipients))
	}
	if !strings.Contains(sender.messages[0], "Issue #15") || !strings.Contains(sender.messages[0], "closed") {
		t.Fatalf("unexpected message: %q", sender.messages[0])
	}
}

func TestGitHubIssueNotificationHandlerRejectsInvalidToken(t *testing.T) {
	sender := &githubNotificationSenderMock{}
	handler := NewGitHubIssueNotificationHandler(sender, "webhook-secret", "0812-1", "imsjogja/guestbook-ui")
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github/issues", strings.NewReader(`{"action":"opened","issue_number":1,"title":"Test"}`))
	req.Header.Set("Authorization", "Bearer wrong-secret")
	res := httptest.NewRecorder()
	ctx := e.NewContext(req, res)

	if err := handler.Handle(ctx); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
	if len(sender.recipients) != 0 {
		t.Fatalf("sender called %d times, want 0", len(sender.recipients))
	}
}

func TestGitHubIssueNotificationHandlerReturnsUnavailableWhenAllSendsFail(t *testing.T) {
	sender := &githubNotificationSenderMock{err: context.Canceled}
	handler := NewGitHubIssueNotificationHandler(sender, "webhook-secret", "0812-1", "imsjogja/guestbook-ui")
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github/issues", strings.NewReader(`{"action":"closed","repository":"imsjogja/guestbook-ui","issue_number":1,"title":"Test","delivery_id":"run-1"}`))
	req.Header.Set("Authorization", "Bearer webhook-secret")
	res := httptest.NewRecorder()
	ctx := e.NewContext(req, res)

	if err := handler.Handle(ctx); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
}

func TestGitHubIssueNotificationHandlerRejectsRepositoryOutsideAllowlist(t *testing.T) {
	sender := &githubNotificationSenderMock{}
	handler := NewGitHubIssueNotificationHandler(sender, "webhook-secret", "0812-1", "imsjogja/guestbook-ui")
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github/issues", strings.NewReader(`{"action":"opened","repository":"other/repository","issue_number":1,"title":"Test","delivery_id":"run-other"}`))
	req.Header.Set("Authorization", "Bearer webhook-secret")
	res := httptest.NewRecorder()
	ctx := e.NewContext(req, res)

	if err := handler.Handle(ctx); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusForbidden)
	}
	if len(sender.recipients) != 0 {
		t.Fatalf("sender called %d times, want 0", len(sender.recipients))
	}
}

func TestGitHubIssueNotificationHandlerDeduplicatesDelivery(t *testing.T) {
	sender := &githubNotificationSenderMock{}
	handler := NewGitHubIssueNotificationHandler(sender, "webhook-secret", "0812-1", "imsjogja/guestbook-ui")
	e := echo.New()
	body := `{"action":"closed","repository":"imsjogja/guestbook-ui","issue_number":1,"title":"Test","delivery_id":"run-duplicate"}`

	for attempt := 0; attempt < 2; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github/issues", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer webhook-secret")
		res := httptest.NewRecorder()
		ctx := e.NewContext(req, res)
		if err := handler.Handle(ctx); err != nil {
			t.Fatalf("Handle() attempt %d error = %v", attempt+1, err)
		}
		if res.Code != http.StatusOK {
			t.Fatalf("attempt %d status = %d, want %d", attempt+1, res.Code, http.StatusOK)
		}
	}
	if len(sender.recipients) != 1 {
		t.Fatalf("send count = %d, want 1", len(sender.recipients))
	}
}
