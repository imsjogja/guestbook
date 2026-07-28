package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"guestflow/internal/service"

	"github.com/labstack/echo/v4"
)

func TestHandleAuthErrorVerificationFlows(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
		body string
	}{
		{name: "unverified email", err: service.ErrEmailNotVerified, code: 403, body: "EMAIL_NOT_VERIFIED"},
		{name: "email delivery failure", err: service.ErrEmailDelivery, code: 503, body: "EMAIL_DELIVERY_FAILED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
			ctx := e.NewContext(request, recorder)

			if err := (&AuthHandler{}).handleAuthError(ctx, tt.err); err != nil {
				t.Fatalf("handleAuthError returned error: %v", err)
			}
			if recorder.Code != tt.code {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.code)
			}

			var response map[string]string
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response["code"] != tt.body {
				t.Fatalf("code = %q, want %q", response["code"], tt.body)
			}
		})
	}
}
