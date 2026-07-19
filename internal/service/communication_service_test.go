package service

import (
	"errors"
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

func TestDefaultRSVPReminderTemplates(t *testing.T) {
	templates := DefaultRSVPReminderTemplates()
	if len(templates) != 2 {
		t.Fatalf("default reminder template count = %d, want 2", len(templates))
	}

	seenChannels := make(map[string]bool)
	for _, template := range templates {
		seenChannels[template.Channel] = true
		if template.Type != domain.MsgTypeRSVPFollowUp {
			t.Errorf("template %q type = %q, want rsvp_followup", template.Name, template.Type)
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
		t.Fatalf("default reminder templates must include WhatsApp and email: %#v", seenChannels)
	}
}

func TestValidateRSVPReminderTemplate(t *testing.T) {
	tests := []struct {
		name         string
		channel      string
		templateType string
		wantErr      error
	}{
		{name: "whatsapp follow up", channel: domain.ChannelWhatsApp, templateType: domain.MsgTypeRSVPFollowUp},
		{name: "email follow up", channel: domain.ChannelEmail, templateType: domain.MsgTypeRSVPFollowUp},
		{name: "sms is unsupported", channel: domain.ChannelSMS, templateType: domain.MsgTypeRSVPFollowUp, wantErr: domain.ErrInvalidChannel},
		{name: "invitation type is unsupported", channel: domain.ChannelWhatsApp, templateType: domain.MsgTypeInvitation, wantErr: domain.ErrInvalidMessageType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRSVPReminderTemplate(&domain.CommunicationTemplate{Channel: tt.channel, Type: tt.templateType})
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestFilterRSVPReminderCandidatesThrottleAndForce(t *testing.T) {
	now := time.Now().UTC()
	recent := now.Add(-2 * time.Hour)
	old := now.Add(-48 * time.Hour)
	phone := "081234567890"

	candidates := []*domain.RSVPReminderCandidate{
		{GuestID: uuid.New(), FullName: "Baru Diingatkan", Phone: &phone, LastReminderAt: &recent},
		{GuestID: uuid.New(), FullName: "Lama Diingatkan", Phone: &phone, LastReminderAt: &old},
		{GuestID: uuid.New(), FullName: "Belum Pernah", Phone: &phone},
	}

	eligible, skipped := filterRSVPReminderCandidates(candidates, nil, false, domain.ChannelWhatsApp, now)
	if len(eligible) != 2 {
		t.Fatalf("eligible = %d, want 2 (throttled guest excluded)", len(eligible))
	}
	if len(skipped) != 1 || skipped[0].FullName != "Baru Diingatkan" {
		t.Fatalf("skipped = %#v, want the recently reminded guest", skipped)
	}

	eligible, skipped = filterRSVPReminderCandidates(candidates, nil, true, domain.ChannelWhatsApp, now)
	if len(eligible) != 3 || len(skipped) != 0 {
		t.Fatalf("force should include all candidates, got eligible=%d skipped=%d", len(eligible), len(skipped))
	}
}

func TestFilterRSVPReminderCandidatesChannelRequirements(t *testing.T) {
	now := time.Now().UTC()
	phone := "081234567890"
	badPhone := "12345"
	email := "tamu@example.com"
	empty := "   "

	candidates := []*domain.RSVPReminderCandidate{
		{GuestID: uuid.New(), FullName: "Punya Nomor", Phone: &phone, Email: &email},
		{GuestID: uuid.New(), FullName: "Nomor Salah", Phone: &badPhone, Email: &email},
		{GuestID: uuid.New(), FullName: "Tanpa Kontak", Email: &empty},
	}

	eligible, skipped := filterRSVPReminderCandidates(candidates, nil, false, domain.ChannelWhatsApp, now)
	if len(eligible) != 1 || eligible[0].FullName != "Punya Nomor" {
		t.Fatalf("whatsapp eligible = %#v, want only guest with valid phone", eligible)
	}
	if len(skipped) != 2 {
		t.Fatalf("whatsapp skipped = %d, want 2", len(skipped))
	}

	eligible, skipped = filterRSVPReminderCandidates(candidates, nil, false, domain.ChannelEmail, now)
	if len(eligible) != 2 {
		t.Fatalf("email eligible = %d, want 2", len(eligible))
	}
	if len(skipped) != 1 || skipped[0].FullName != "Tanpa Kontak" {
		t.Fatalf("email skipped = %#v, want guest without email", skipped)
	}
}

func TestFilterRSVPReminderCandidatesSubsetSelection(t *testing.T) {
	now := time.Now().UTC()
	phone := "081234567890"
	inRoster := uuid.New()
	other := uuid.New()
	notCandidate := uuid.New()

	candidates := []*domain.RSVPReminderCandidate{
		{GuestID: inRoster, FullName: "Dipilih", Phone: &phone},
		{GuestID: other, FullName: "Tidak Dipilih", Phone: &phone},
	}

	eligible, skipped := filterRSVPReminderCandidates(candidates, []uuid.UUID{inRoster, notCandidate}, false, domain.ChannelWhatsApp, now)
	if len(eligible) != 1 || eligible[0].GuestID != inRoster {
		t.Fatalf("eligible = %#v, want only the requested candidate", eligible)
	}
	if len(skipped) != 1 || skipped[0].GuestID != notCandidate {
		t.Fatalf("skipped = %#v, want the requested non-candidate", skipped)
	}
}
