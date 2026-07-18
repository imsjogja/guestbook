package domain

import (
	"time"

	"github.com/google/uuid"
)

// Communication channels
const (
	ChannelWhatsApp = "whatsapp"
	ChannelEmail    = "email"
	ChannelSMS      = "sms"
)

// Message types
const (
	MsgTypeSaveTheDate  = "save_the_date"
	MsgTypeInvitation   = "invitation"
	MsgTypeRSVPRequest  = "rsvp_request"
	MsgTypeRSVPFollowUp = "rsvp_followup"
	MsgTypeReminder     = "reminder"
	MsgTypeReminderH7   = "reminder_h7"
	MsgTypeReminderH1   = "reminder_h1"
	MsgTypeReminderDay  = "reminder_day"
	MsgTypeConfirmation = "confirmation"
	MsgTypeQRCard       = "qr_card"
	MsgTypeChangeNotice = "change_notice"
	MsgTypeEmergency    = "emergency"
	MsgTypeThankYou     = "thank_you"
	MsgTypeSurvey       = "survey"
	MsgTypeGallery      = "gallery"
)

// Message statuses
const (
	MessageStatusDraft     = "draft"
	MessageStatusQueued    = "queued"
	MessageStatusSent      = "sent"
	MessageStatusDelivered = "delivered"
	MessageStatusRead      = "read"
	MessageStatusFailed    = "failed"
	MessageStatusCancelled = "cancelled"
)

// Campaign statuses
const (
	CampaignStatusDraft     = "draft"
	CampaignStatusScheduled = "scheduled"
	CampaignStatusSending   = "sending"
	CampaignStatusCompleted = "completed"
	CampaignStatusCancelled = "cancelled"
)

// CommunicationTemplate represents a message template
type CommunicationTemplate struct {
	TenantBase
	Name        string          `db:"name" json:"name"`
	Channel     string          `db:"channel" json:"channel"`                   // whatsapp, email, sms
	Type        string          `db:"type" json:"type"`                         // message type
	Subject     *string         `db:"subject" json:"subject,omitempty"`         // For email
	Body        string          `db:"body" json:"body"`                         // Template body with {{variables}}
	Variables   JSONStringSlice `db:"variables" json:"variables"`               // Available variables
	IsActive    bool            `db:"is_active" json:"is_active"`               // Whether template is active
	IsSystem    bool            `db:"is_system" json:"is_system"`               // System-provided template
	Description *string         `db:"description" json:"description,omitempty"` // Human-readable description
	Language    string          `db:"language" json:"language"`                 // e.g., 'id', 'en'
}

// CommunicationCampaign represents a broadcast campaign
type CommunicationCampaign struct {
	TenantBase
	EventID         uuid.UUID  `db:"event_id" json:"event_id"`
	Name            string     `db:"name" json:"name"`
	TemplateID      uuid.UUID  `db:"template_id" json:"template_id"`
	Channel         string     `db:"channel" json:"channel"`
	Type            string     `db:"type" json:"type"`
	Status          string     `db:"status" json:"status"` // draft, scheduled, sending, completed, cancelled
	RecipientFilter JSONMap    `db:"recipient_filter" json:"recipient_filter"`
	ScheduledAt     *time.Time `db:"scheduled_at" json:"scheduled_at,omitempty"`
	SentAt          *time.Time `db:"sent_at" json:"sent_at,omitempty"`
	CompletedAt     *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	TotalRecipients int        `db:"total_recipients" json:"total_recipients"`
	SentCount       int        `db:"sent_count" json:"sent_count"`
	FailedCount     int        `db:"failed_count" json:"failed_count"`
	CreatedBy       uuid.UUID  `db:"created_by" json:"created_by"`
}

// CommunicationMessage represents an individual message
type CommunicationMessage struct {
	Base
	TenantID           uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	CampaignID         *uuid.UUID `db:"campaign_id" json:"campaign_id,omitempty"`
	EventID            uuid.UUID  `db:"event_id" json:"event_id"`
	GuestID            uuid.UUID  `db:"guest_id" json:"guest_id"`
	EventGuestID       *uuid.UUID `db:"event_guest_id" json:"event_guest_id,omitempty"`
	InvitationID       *uuid.UUID `db:"invitation_id" json:"invitation_id,omitempty"`
	Channel            string     `db:"channel" json:"channel"`
	Type               string     `db:"type" json:"type"`
	Subject            *string    `db:"subject" json:"subject,omitempty"`
	Body               string     `db:"body" json:"body"`
	Status             string     `db:"status" json:"status"`
	SentAt             *time.Time `db:"sent_at" json:"sent_at,omitempty"`
	DeliveredAt        *time.Time `db:"delivered_at" json:"delivered_at,omitempty"`
	ReadAt             *time.Time `db:"read_at" json:"read_at,omitempty"`
	FailedAt           *time.Time `db:"failed_at" json:"failed_at,omitempty"`
	ErrorMessage       *string    `db:"error_message" json:"error_message,omitempty"`
	ExternalID         *string    `db:"external_id" json:"external_id,omitempty"` // Provider message ID
	ProviderHTTPStatus *int       `db:"provider_http_status" json:"provider_http_status,omitempty"`
	Cost               *float64   `db:"cost" json:"cost,omitempty"`
}

// CommunicationTemplateCreateRequest input for creating a template
type CommunicationTemplateCreateRequest struct {
	Name        string   `json:"name" validate:"required,min=2,max=255"`
	Channel     string   `json:"channel" validate:"required,oneof=whatsapp email sms"`
	Type        string   `json:"type" validate:"required"`
	Subject     string   `json:"subject,omitempty"`
	Body        string   `json:"body" validate:"required"`
	Variables   []string `json:"variables,omitempty"`
	Description string   `json:"description,omitempty"`
	Language    string   `json:"language,omitempty"`
}

// CommunicationTemplateUpdateRequest input for updating a template
type CommunicationTemplateUpdateRequest struct {
	Name        string   `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Channel     string   `json:"channel,omitempty" validate:"omitempty,oneof=whatsapp email sms"`
	Type        string   `json:"type,omitempty"`
	Subject     string   `json:"subject,omitempty"`
	Body        string   `json:"body,omitempty"`
	Variables   []string `json:"variables,omitempty"`
	Description string   `json:"description,omitempty"`
	Language    string   `json:"language,omitempty"`
	IsActive    *bool    `json:"is_active,omitempty"`
}

// SendMessageRequest for manual message sending
type SendMessageRequest struct {
	GuestIDs   []uuid.UUID `json:"guest_ids" validate:"required,min=1"`
	TemplateID uuid.UUID   `json:"template_id" validate:"required"`
	Variables  JSONMap     `json:"variables,omitempty"`
}

// CommunicationCampaignCreateRequest input for creating a campaign
type CommunicationCampaignCreateRequest struct {
	Name            string     `json:"name" validate:"required,min=2,max=255"`
	TemplateID      uuid.UUID  `json:"template_id" validate:"required"`
	Channel         string     `json:"channel" validate:"required,oneof=whatsapp email sms"`
	Type            string     `json:"type" validate:"required"`
	RecipientFilter JSONMap    `json:"recipient_filter,omitempty"`
	ScheduledAt     *time.Time `json:"scheduled_at,omitempty"`
}

// MessageStatusUpdate for updating message status from provider webhooks
type MessageStatusUpdate struct {
	Status       string     `json:"status" validate:"required"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
	ReadAt       *time.Time `json:"read_at,omitempty"`
	FailedAt     *time.Time `json:"failed_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	ExternalID   string     `json:"external_id,omitempty"`
}

// TemplateListParams for filtering templates
type TemplateListParams struct {
	TenantID uuid.UUID
	Channel  string
	Type     string
	IsActive *bool
	Page     int
	PerPage  int
}

// CampaignListParams for filtering campaigns
type CampaignListParams struct {
	TenantID uuid.UUID
	EventID  uuid.UUID
	Status   string
	Page     int
	PerPage  int
}

// MessageListParams for filtering messages
type MessageListParams struct {
	TenantID   uuid.UUID
	EventID    uuid.UUID
	CampaignID *uuid.UUID
	GuestID    *uuid.UUID
	Status     string
	Page       int
	PerPage    int
}

// Communication errors
var (
	ErrTemplateNotFound   = NewDomainError("template not found")
	ErrCampaignNotFound   = NewDomainError("campaign not found")
	ErrMessageNotFound    = NewDomainError("message not found")
	ErrInvalidChannel     = NewDomainError("invalid communication channel")
	ErrInvalidMessageType = NewDomainError("invalid message type")
	ErrTemplateInactive   = NewDomainError("template is inactive")
	ErrEmptyRecipientList = NewDomainError("empty recipient list")
	ErrCampaignStarted    = NewDomainError("campaign has already started")
)
