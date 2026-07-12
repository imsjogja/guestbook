package domain

import "github.com/google/uuid"

// Guest types
const (
	GuestTypeVIP         = "vip"
	GuestTypeVVIP        = "vvip"
	GuestTypeFamily      = "family"
	GuestTypeFriend      = "friend"
	GuestTypeColleague   = "colleague"
	GuestTypeGovernment  = "government"
	GuestTypeMedia       = "media"
	GuestTypeSponsor     = "sponsor"
	GuestTypeVendor      = "vendor"
	GuestTypeSpeaker     = "speaker"
	GuestTypeParticipant = "participant"
	GuestTypeInternal    = "internal"
	GuestTypeProtocol    = "protocol"
	GuestTypeSecurity    = "security"
	GuestTypeGeneral     = "general"
)

// TenantBase provides common fields for tenant-scoped domain models.
type TenantBase struct {
	Base
	TenantID uuid.UUID `db:"tenant_id" json:"tenant_id"`
}

// Guest represents a person in the guest database.
type Guest struct {
	TenantBase
	FullName             string     `db:"full_name" json:"full_name"`
	Nickname             *string    `db:"nickname" json:"nickname,omitempty"`
	Phone                *string    `db:"phone" json:"phone,omitempty"`
	Email                *string    `db:"email" json:"email,omitempty"`
	Address              *string    `db:"address" json:"address,omitempty"`
	City                 *string    `db:"city" json:"city,omitempty"`
	Country              *string    `db:"country" json:"country,omitempty"`
	Language             string     `db:"language" json:"language"` // 'id' or 'en'
	GuestType            string     `db:"guest_type" json:"guest_type"`
	Segment              *string    `db:"segment" json:"segment,omitempty"`
	Institution          *string    `db:"institution" json:"institution,omitempty"`
	Title                *string    `db:"title" json:"title,omitempty"`
	Relationship         *string    `db:"relationship" json:"relationship,omitempty"`
	PIC                  *string    `db:"pic" json:"pic,omitempty"` // Person In Charge
	AccessibilityNeeds   *string    `db:"accessibility_needs" json:"accessibility_needs,omitempty"`
	DietaryRestrictions  *string    `db:"dietary_restrictions" json:"dietary_restrictions,omitempty"`
	Allergies            *string    `db:"allergies" json:"allergies,omitempty"`
	Notes                *string    `db:"notes" json:"notes,omitempty"`
	ConsentCommunication bool       `db:"consent_communication" json:"consent_communication"`
	ConsentVersion       *string    `db:"consent_version" json:"consent_version,omitempty"`
	Source               *string    `db:"source" json:"source,omitempty"` // manual, import, referral
	IsActive             bool       `db:"is_active" json:"is_active"`
	CreatedBy            uuid.UUID  `db:"created_by" json:"created_by"`
	UpdatedBy            *uuid.UUID `db:"updated_by" json:"updated_by,omitempty"`
}

// GuestCreateRequest input for creating a guest.
type GuestCreateRequest struct {
	FullName             string `json:"full_name" validate:"required,min=2,max=255"`
	Nickname             string `json:"nickname,omitempty"`
	Phone                string `json:"phone,omitempty" validate:"omitempty,e164"`
	Email                string `json:"email,omitempty" validate:"omitempty,email"`
	Address              string `json:"address,omitempty"`
	City                 string `json:"city,omitempty"`
	Country              string `json:"country,omitempty"`
	Language             string `json:"language,omitempty" validate:"omitempty,oneof=id en"`
	GuestType            string `json:"guest_type" validate:"required,oneof=vip vvip family friend colleague government media sponsor vendor speaker participant internal protocol security general"`
	Segment              string `json:"segment,omitempty"`
	Institution          string `json:"institution,omitempty"`
	Title                string `json:"title,omitempty"`
	Relationship         string `json:"relationship,omitempty"`
	PIC                  string `json:"pic,omitempty"`
	AccessibilityNeeds   string `json:"accessibility_needs,omitempty"`
	DietaryRestrictions  string `json:"dietary_restrictions,omitempty"`
	Allergies            string `json:"allergies,omitempty"`
	Notes                string `json:"notes,omitempty"`
	ConsentCommunication bool   `json:"consent_communication"`
}

// GuestUpdateRequest input for updating a guest.
type GuestUpdateRequest struct {
	FullName             string `json:"full_name,omitempty" validate:"omitempty,min=2,max=255"`
	Nickname             string `json:"nickname,omitempty"`
	Phone                string `json:"phone,omitempty" validate:"omitempty,e164"`
	Email                string `json:"email,omitempty" validate:"omitempty,email"`
	Address              string `json:"address,omitempty"`
	City                 string `json:"city,omitempty"`
	Country              string `json:"country,omitempty"`
	Language             string `json:"language,omitempty" validate:"omitempty,oneof=id en"`
	GuestType            string `json:"guest_type,omitempty" validate:"omitempty,oneof=vip vvip family friend colleague government media sponsor vendor speaker participant internal protocol security general"`
	Segment              string `json:"segment,omitempty"`
	Institution          string `json:"institution,omitempty"`
	Title                string `json:"title,omitempty"`
	Relationship         string `json:"relationship,omitempty"`
	PIC                  string `json:"pic,omitempty"`
	AccessibilityNeeds   string `json:"accessibility_needs,omitempty"`
	DietaryRestrictions  string `json:"dietary_restrictions,omitempty"`
	Allergies            string `json:"allergies,omitempty"`
	Notes                string `json:"notes,omitempty"`
	ConsentCommunication *bool  `json:"consent_communication,omitempty"`
	IsActive             *bool  `json:"is_active,omitempty"`
}

// GuestListParams for filtering and paginating guest lists.
type GuestListParams struct {
	TenantID  uuid.UUID
	Search    string
	GuestType string
	Segment   string
	Status    *bool
	Page      int
	PerPage   int
}

// GuestImportRow represents a row from CSV import with validation errors.
type GuestImportRow struct {
	RowNum              int      `json:"row_num"`
	FullName            string   `json:"full_name"`
	Nickname            string   `json:"nickname"`
	Phone               string   `json:"phone"`
	Email               string   `json:"email"`
	Address             string   `json:"address"`
	City                string   `json:"city"`
	Country             string   `json:"country"`
	GuestType           string   `json:"guest_type"`
	Segment             string   `json:"segment"`
	Institution         string   `json:"institution"`
	Title               string   `json:"title"`
	Relationship        string   `json:"relationship"`
	PIC                 string   `json:"pic"`
	AccessibilityNeeds  string   `json:"accessibility_needs"`
	DietaryRestrictions string   `json:"dietary_restrictions"`
	Allergies           string   `json:"allergies"`
	Notes               string   `json:"notes"`
	Errors              []string `json:"errors,omitempty"`
}

// GuestImportResult holds the outcome of a bulk import operation.
type GuestImportResult struct {
	TotalRows    int              `json:"total_rows"`
	SuccessCount int              `json:"success_count"`
	ErrorCount   int              `json:"error_count"`
	Errors       []GuestImportRow `json:"errors,omitempty"`
}

// GuestTag represents a tag/label for guests.
type GuestTag struct {
	Base
	TenantID    uuid.UUID `db:"tenant_id" json:"tenant_id"`
	Name        string    `db:"name" json:"name"`
	Color       *string   `db:"color" json:"color,omitempty"`
	Description *string   `db:"description" json:"description,omitempty"`
}

// GuestTagAssignment links guests to tags.
type GuestTagAssignment struct {
	GuestID uuid.UUID `db:"guest_id" json:"guest_id"`
	TagID   uuid.UUID `db:"tag_id" json:"tag_id"`
}

// GuestNote represents an internal note about a guest.
type GuestNote struct {
	Base
	GuestID  uuid.UUID `db:"guest_id" json:"guest_id"`
	UserID   uuid.UUID `db:"user_id" json:"user_id"`
	Content  string    `db:"content" json:"content"`
	IsPinned bool      `db:"is_pinned" json:"is_pinned"`
}

// NewGuest creates a new Guest from a create request.
func NewGuest(tenantID, createdBy uuid.UUID, req GuestCreateRequest) *Guest {
	lang := req.Language
	if lang == "" {
		lang = "id"
	}
	country := req.Country
	if country == "" {
		country = "Indonesia"
	}
	source := "manual"

	g := &Guest{
		TenantBase: TenantBase{
			Base:     NewBase(),
			TenantID: tenantID,
		},
		FullName:             req.FullName,
		Language:             lang,
		GuestType:            req.GuestType,
		ConsentCommunication: req.ConsentCommunication,
		Country:              &country,
		IsActive:             true,
		CreatedBy:            createdBy,
		Source:               &source,
	}

	if req.Nickname != "" {
		g.Nickname = &req.Nickname
	}
	if req.Phone != "" {
		g.Phone = &req.Phone
	}
	if req.Email != "" {
		g.Email = &req.Email
	}
	if req.Address != "" {
		g.Address = &req.Address
	}
	if req.City != "" {
		g.City = &req.City
	}
	if req.Segment != "" {
		g.Segment = &req.Segment
	}
	if req.Institution != "" {
		g.Institution = &req.Institution
	}
	if req.Title != "" {
		g.Title = &req.Title
	}
	if req.Relationship != "" {
		g.Relationship = &req.Relationship
	}
	if req.PIC != "" {
		g.PIC = &req.PIC
	}
	if req.AccessibilityNeeds != "" {
		g.AccessibilityNeeds = &req.AccessibilityNeeds
	}
	if req.DietaryRestrictions != "" {
		g.DietaryRestrictions = &req.DietaryRestrictions
	}
	if req.Allergies != "" {
		g.Allergies = &req.Allergies
	}
	if req.Notes != "" {
		g.Notes = &req.Notes
	}

	return g
}

// ApplyUpdate applies non-zero fields from an update request to the guest.
func (g *Guest) ApplyUpdate(req GuestUpdateRequest, updatedBy uuid.UUID) {
	g.Touch()
	g.UpdatedBy = &updatedBy

	if req.FullName != "" {
		g.FullName = req.FullName
	}
	if req.Nickname != "" {
		g.Nickname = &req.Nickname
	}
	if req.Phone != "" {
		g.Phone = &req.Phone
	} else if req.Phone == "" && g.Phone != nil {
		// Allow clearing by sending empty string
		g.Phone = nil
	}
	if req.Email != "" {
		g.Email = &req.Email
	} else if req.Email == "" && g.Email != nil {
		g.Email = nil
	}
	if req.Address != "" {
		g.Address = &req.Address
	}
	if req.City != "" {
		g.City = &req.City
	}
	if req.Country != "" {
		g.Country = &req.Country
	}
	if req.Language != "" {
		g.Language = req.Language
	}
	if req.GuestType != "" {
		g.GuestType = req.GuestType
	}
	if req.Segment != "" {
		g.Segment = &req.Segment
	}
	if req.Institution != "" {
		g.Institution = &req.Institution
	}
	if req.Title != "" {
		g.Title = &req.Title
	}
	if req.Relationship != "" {
		g.Relationship = &req.Relationship
	}
	if req.PIC != "" {
		g.PIC = &req.PIC
	}
	if req.AccessibilityNeeds != "" {
		g.AccessibilityNeeds = &req.AccessibilityNeeds
	}
	if req.DietaryRestrictions != "" {
		g.DietaryRestrictions = &req.DietaryRestrictions
	}
	if req.Allergies != "" {
		g.Allergies = &req.Allergies
	}
	if req.Notes != "" {
		g.Notes = &req.Notes
	}
	if req.ConsentCommunication != nil {
		g.ConsentCommunication = *req.ConsentCommunication
	}
	if req.IsActive != nil {
		g.IsActive = *req.IsActive
	}
}

// IsValidGuestType checks if the given guest type is valid.
func IsValidGuestType(gt string) bool {
	switch gt {
	case GuestTypeVIP, GuestTypeVVIP, GuestTypeFamily, GuestTypeFriend,
		GuestTypeColleague, GuestTypeGovernment, GuestTypeMedia, GuestTypeSponsor,
		GuestTypeVendor, GuestTypeSpeaker, GuestTypeParticipant, GuestTypeInternal,
		GuestTypeProtocol, GuestTypeSecurity, GuestTypeGeneral:
		return true
	}
	return false
}
