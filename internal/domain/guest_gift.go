package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	GuestGiftTypeCash     = "cash"
	GuestGiftTypeTransfer = "transfer"
	GuestGiftTypeGoods    = "goods"
	GuestGiftTypeOther    = "other"
)

// GuestGift stores the angpau/gift received from a guest for one event.
type GuestGift struct {
	Base
	TenantID     uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	EventID      uuid.UUID  `db:"event_id" json:"event_id"`
	GuestID      uuid.UUID  `db:"guest_id" json:"guest_id"`
	EventGuestID uuid.UUID  `db:"event_guest_id" json:"event_guest_id"`
	Amount       int64      `db:"amount" json:"amount"`
	GiftType     string     `db:"gift_type" json:"gift_type"`
	Notes        *string    `db:"notes" json:"notes,omitempty"`
	ReceivedAt   time.Time  `db:"received_at" json:"received_at"`
	RecordedBy   *uuid.UUID `db:"recorded_by" json:"recorded_by,omitempty"`
}

type GuestGiftUpsertRequest struct {
	Amount   int64  `json:"amount" validate:"required,min=1"`
	GiftType string `json:"gift_type,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

func IsValidGuestGiftType(value string) bool {
	switch value {
	case GuestGiftTypeCash, GuestGiftTypeTransfer, GuestGiftTypeGoods, GuestGiftTypeOther:
		return true
	default:
		return false
	}
}
