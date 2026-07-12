package domain

import (
	"github.com/google/uuid"
)

// Household represents a family unit or group of guests.
type Household struct {
	TenantBase
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description,omitempty"`
	Address     *string   `db:"address" json:"address,omitempty"`
	City        *string   `db:"city" json:"city,omitempty"`
	MaxPax      *int      `db:"max_pax" json:"max_pax,omitempty"`
	Notes       *string   `db:"notes" json:"notes,omitempty"`
	CreatedBy   uuid.UUID `db:"created_by" json:"created_by"`
}

// HouseholdMember links guests to households.
type HouseholdMember struct {
	HouseholdID uuid.UUID `db:"household_id" json:"household_id"`
	GuestID     uuid.UUID `db:"guest_id" json:"guest_id"`
	IsPrimary   bool      `db:"is_primary" json:"is_primary"`
	Role        *string   `db:"role" json:"role,omitempty"` // e.g., 'head', 'spouse', 'child'
}

// HouseholdCreateRequest input for creating a household.
type HouseholdCreateRequest struct {
	Name        string      `json:"name" validate:"required,min=2,max=255"`
	Description string      `json:"description,omitempty"`
	Address     string      `json:"address,omitempty"`
	City        string      `json:"city,omitempty"`
	MaxPax      *int        `json:"max_pax,omitempty" validate:"omitempty,min=1"`
	Notes       string      `json:"notes,omitempty"`
	GuestIDs    []uuid.UUID `json:"guest_ids,omitempty"`
}

// HouseholdUpdateRequest input for updating a household.
type HouseholdUpdateRequest struct {
	Name        string `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Description string `json:"description,omitempty"`
	Address     string `json:"address,omitempty"`
	City        string `json:"city,omitempty"`
	MaxPax      *int   `json:"max_pax,omitempty" validate:"omitempty,min=1"`
	Notes       string `json:"notes,omitempty"`
}

// HouseholdListParams for filtering and paginating household lists.
type HouseholdListParams struct {
	TenantID uuid.UUID
	Search   string
	Page     int
	PerPage  int
}

// NewHousehold creates a new Household from a create request.
func NewHousehold(tenantID, createdBy uuid.UUID, req HouseholdCreateRequest) *Household {
	h := &Household{
		TenantBase: TenantBase{
			Base:     NewBase(),
			TenantID: tenantID,
		},
		Name:      req.Name,
		CreatedBy: createdBy,
	}

	if req.Description != "" {
		h.Description = &req.Description
	}
	if req.Address != "" {
		h.Address = &req.Address
	}
	if req.City != "" {
		h.City = &req.City
	}
	if req.MaxPax != nil {
		h.MaxPax = req.MaxPax
	}
	if req.Notes != "" {
		h.Notes = &req.Notes
	}

	return h
}

// ApplyUpdate applies non-zero fields from an update request to the household.
func (h *Household) ApplyUpdate(req HouseholdUpdateRequest) {
	h.Touch()
	if req.Name != "" {
		h.Name = req.Name
	}
	if req.Description != "" {
		h.Description = &req.Description
	}
	if req.Address != "" {
		h.Address = &req.Address
	}
	if req.City != "" {
		h.City = &req.City
	}
	if req.MaxPax != nil {
		h.MaxPax = req.MaxPax
	}
	if req.Notes != "" {
		h.Notes = &req.Notes
	}
}
