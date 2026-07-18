package service

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"guestflow/internal/domain"
)

func TestDefaultInvitationTemplates(t *testing.T) {
	templates := DefaultInvitationTemplates()
	if len(templates) != 2 {
		t.Fatalf("default template count = %d, want 2", len(templates))
	}

	seenChannels := make(map[string]bool)
	for _, template := range templates {
		seenChannels[template.Channel] = true
		if template.Type != domain.MsgTypeInvitation {
			t.Errorf("template %q type = %q, want invitation", template.Name, template.Type)
		}
		if template.Body == "" || len(template.Variables) == 0 {
			t.Errorf("template %q must have body and variables", template.Name)
		}
		for _, variable := range template.Variables {
			if !strings.Contains(template.Body+template.Subject, "{{"+variable+"}}") {
				t.Errorf("template %q does not use variable %q", template.Name, variable)
			}
		}
	}
	if !seenChannels[domain.ChannelWhatsApp] || !seenChannels[domain.ChannelEmail] {
		t.Fatalf("default templates must include WhatsApp and email: %#v", seenChannels)
	}
}

func TestBuildRenderVariablesUsesCanonicalInvitationURL(t *testing.T) {
	guest := &domain.Guest{FullName: "Bambang Kusniawan", GuestType: "friend"}
	event := &domain.Event{Name: "Acara Demo", StartDate: time.Date(2026, 7, 18, 19, 0, 0, 0, time.UTC)}
	invitation := &domain.Invitation{
		Base:  domain.Base{ID: uuid.New()},
		Token: "opaque-token",
		URL:   "https://guestflow.id/i/opaque-token",
	}

	vars := BuildRenderVariables(guest, event, invitation, "https://guestflow.id")
	if got, want := vars["rsvp_link"], invitation.URL; got != want {
		t.Fatalf("rsvp_link = %q, want %q", got, want)
	}
	if strings.Contains(vars["rsvp_link"], "/rsvp/") {
		t.Fatalf("rsvp_link contains legacy path: %q", vars["rsvp_link"])
	}
}

func TestBuildRenderVariablesNormalizesLegacyInvitationURL(t *testing.T) {
	guest := &domain.Guest{FullName: "Bambang Kusniawan", GuestType: "friend"}
	event := &domain.Event{Name: "Acara Demo", StartDate: time.Now()}
	invitation := &domain.Invitation{Token: "opaque-token", URL: "https://guestflow.id/rsvp/opaque-token"}

	vars := BuildRenderVariables(guest, event, invitation, "https://guestflow.id")
	if got, want := vars["rsvp_link"], "https://guestflow.id/i/opaque-token"; got != want {
		t.Fatalf("rsvp_link = %q, want %q", got, want)
	}
}
