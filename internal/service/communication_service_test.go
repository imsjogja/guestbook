package service

import (
	"strings"
	"testing"

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
