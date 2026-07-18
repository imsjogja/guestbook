package whatsapp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"guestflow/internal/config"
)

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "indonesian local format", input: "0812-3456-7890", want: "6281234567890"},
		{name: "international format", input: "+62 812 3456 7890", want: "6281234567890"},
		{name: "empty", input: "  ", wantErr: ErrPhoneMissing},
		{name: "unsupported country", input: "081234", wantErr: ErrInvalidPhone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizePhone(tt.input)
			if got != tt.want {
				t.Fatalf("NormalizePhone() = %q, want %q", got, tt.want)
			}
			if err != tt.wantErr {
				t.Fatalf("NormalizePhone() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientSendUsesBlastrHeadersAndPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer account-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("X-Sender-Token"); got != "sender-token" {
			t.Errorf("X-Sender-Token = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message_id":"blastr-123"}`))
	}))
	defer server.Close()

	client := NewClient(config.WhatsAppConfig{
		Enabled:      true,
		APIURL:       server.URL,
		AccountToken: "account-token",
		SenderToken:  "sender-token",
	})

	receipt, err := client.Send(context.Background(), "081234567890", "Halo")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if receipt.ExternalID != "blastr-123" {
		t.Fatalf("Send() external id = %q, want %q", receipt.ExternalID, "blastr-123")
	}
	if receipt.HTTPStatus != http.StatusOK {
		t.Fatalf("Send() HTTP status = %d, want %d", receipt.HTTPStatus, http.StatusOK)
	}
}

func TestClientSendPreservesProviderHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid recipient"}`))
	}))
	defer server.Close()

	client := NewClient(config.WhatsAppConfig{
		Enabled:      true,
		APIURL:       server.URL,
		AccountToken: "account-token",
		SenderToken:  "sender-token",
	})

	receipt, err := client.Send(context.Background(), "081234567890", "Halo")
	if err == nil {
		t.Fatal("Send() error = nil, want provider error")
	}
	providerErr, ok := err.(*ProviderError)
	if !ok {
		t.Fatalf("Send() error type = %T, want *ProviderError", err)
	}
	if providerErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("provider status = %d, want %d", providerErr.StatusCode, http.StatusBadRequest)
	}
	if receipt.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("receipt HTTP status = %d, want %d", receipt.HTTPStatus, http.StatusBadRequest)
	}
}
