// Package service provides business logic layer implementations for GuestFlow.
package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"guestflow/internal/audit"
	"guestflow/internal/domain"
	"guestflow/internal/repository"

	"github.com/google/uuid"
)

// SeatingService encapsulates business logic for seating management.
type SeatingService struct {
	seatingRepo *repository.SeatingRepository
	guestRepo   *repository.GuestRepository
	checkinRepo *repository.CheckinRepository
	auditSvc    *audit.Service
}

// NewSeatingService creates a new SeatingService.
func NewSeatingService(
	seatingRepo *repository.SeatingRepository,
	guestRepo *repository.GuestRepository,
	checkinRepo *repository.CheckinRepository,
	auditSvc *audit.Service,
) *SeatingService {
	return &SeatingService{
		seatingRepo: seatingRepo,
		guestRepo:   guestRepo,
		checkinRepo: checkinRepo,
		auditSvc:    auditSvc,
	}
}

// ─── Venue Zone Operations ────────────────────────────────────────────────────

// CreateZone creates a new venue zone for an event.
func (s *SeatingService) CreateZone(ctx context.Context, tenantID uuid.UUID, req struct {
	EventID     uuid.UUID `json:"event_id" validate:"required"`
	Name        string    `json:"name" validate:"required,min=1,max=100"`
	Description string    `json:"description,omitempty"`
	SortOrder   int       `json:"sort_order,omitempty"`
}) (*domain.VenueZone, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("zone name is required: %w", domain.ErrInvalidInput)
	}

	zone := &domain.VenueZone{
		TenantBase: domain.TenantBase{
			Base:     domain.NewBase(),
			TenantID: tenantID,
		},
		EventID:   req.EventID,
		Name:      name,
		SortOrder: req.SortOrder,
	}

	if req.Description != "" {
		zone.Description = &req.Description
	}

	if err := s.seatingRepo.CreateZone(ctx, zone); err != nil {
		return nil, fmt.Errorf("create zone: %w", err)
	}

	return zone, nil
}

// GetZone retrieves a venue zone by ID.
func (s *SeatingService) GetZone(ctx context.Context, tenantID, zoneID uuid.UUID) (*domain.VenueZone, error) {
	zone, err := s.seatingRepo.GetZone(ctx, tenantID, zoneID)
	if err != nil {
		return nil, fmt.Errorf("get zone: %w", err)
	}
	return zone, nil
}

// ListZones lists all venue zones for an event.
func (s *SeatingService) ListZones(ctx context.Context, tenantID, eventID uuid.UUID) ([]domain.VenueZone, error) {
	zones, err := s.seatingRepo.ListZonesByEvent(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("list zones: %w", err)
	}
	return zones, nil
}

// DeleteZone removes a venue zone.
func (s *SeatingService) DeleteZone(ctx context.Context, tenantID, zoneID uuid.UUID) error {
	if err := s.seatingRepo.SoftDeleteZone(ctx, tenantID, zoneID); err != nil {
		return fmt.Errorf("delete zone: %w", err)
	}
	return nil
}

// ─── Table Operations ─────────────────────────────────────────────────────────

// CreateTable creates a new table for an event.
func (s *SeatingService) CreateTable(ctx context.Context, tenantID, eventID uuid.UUID, req domain.TableCreateRequest) (*domain.Table, error) {
	// Validate table name
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("table name is required: %w", domain.ErrInvalidInput)
	}

	// Validate capacity
	if req.Capacity < 1 {
		return nil, fmt.Errorf("table capacity must be at least 1: %w", domain.ErrInvalidInput)
	}
	if req.Capacity > 999 {
		return nil, fmt.Errorf("table capacity cannot exceed 999: %w", domain.ErrInvalidInput)
	}

	// Validate shape
	shape := req.Shape
	if shape == "" {
		shape = domain.TableShapeRound
	}
	if !isValidTableShape(shape) {
		return nil, fmt.Errorf("invalid table shape: %w", domain.ErrInvalidInput)
	}

	// Validate position if provided
	if req.PositionX != nil && req.PositionY != nil {
		if *req.PositionX < 0 || *req.PositionY < 0 {
			return nil, fmt.Errorf("table position coordinates must be non-negative: %w", domain.ErrInvalidInput)
		}
	}

	table := &domain.Table{
		Base: domain.NewBase(),
		TenantID:      tenantID,
		EventID:       eventID,
		ZoneID:        req.ZoneID,
		Name:          name,
		Capacity:      req.Capacity,
		Shape:         shape,
		PositionX:     req.PositionX,
		PositionY:     req.PositionY,
		IsLocked:      req.IsLocked,
		Accessibility: req.Accessibility,
		VIPOnly:       req.VIPOnly,
	}

	if req.Notes != "" {
		table.Notes = &req.Notes
	}

	if err := s.seatingRepo.CreateTable(ctx, table); err != nil {
		return nil, fmt.Errorf("create table: %w", err)
	}

	return table, nil
}

// GetTable retrieves a table by ID with occupancy information.
func (s *SeatingService) GetTable(ctx context.Context, tenantID, eventID, tableID uuid.UUID) (*domain.TableWithOccupancy, error) {
	table, err := s.seatingRepo.GetTableWithOccupancy(ctx, tenantID, eventID, tableID)
	if err != nil {
		return nil, fmt.Errorf("get table: %w", err)
	}

	// Load assigned guests
	assignedGuests, err := s.seatingRepo.ListAssignmentsByTable(ctx, tableID)
	if err != nil {
		return nil, fmt.Errorf("get table: %w", err)
	}
	table.AssignedGuests = assignedGuests

	return table, nil
}

// ListTables lists all tables for an event with occupancy information.
func (s *SeatingService) ListTables(ctx context.Context, tenantID, eventID uuid.UUID) ([]domain.TableWithOccupancy, error) {
	tables, err := s.seatingRepo.ListTablesByEvent(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	return tables, nil
}

// UpdateTable updates an existing table.
func (s *SeatingService) UpdateTable(ctx context.Context, tenantID, eventID, tableID uuid.UUID, req domain.TableUpdateRequest) (*domain.Table, error) {
	// Get existing table
	existing, err := s.seatingRepo.GetTable(ctx, tenantID, eventID, tableID)
	if err != nil {
		return nil, fmt.Errorf("update table: %w", err)
	}

	// Validate capacity if provided
	if req.Capacity != nil {
		if *req.Capacity < 1 {
			return nil, fmt.Errorf("table capacity must be at least 1: %w", domain.ErrInvalidInput)
		}
		if *req.Capacity > 999 {
			return nil, fmt.Errorf("table capacity cannot exceed 999: %w", domain.ErrInvalidInput)
		}
		// Check that new capacity is not less than current occupancy
		occupancy, err := s.seatingRepo.CountAssignmentsByTable(ctx, tableID)
		if err != nil {
			return nil, fmt.Errorf("update table: %w", err)
		}
		if *req.Capacity < occupancy {
			return nil, fmt.Errorf("new capacity cannot be less than current occupancy (%d): %w", occupancy, domain.ErrInvalidInput)
		}
	}

	// Validate shape if provided
	if req.Shape != "" && !isValidTableShape(req.Shape) {
		return nil, fmt.Errorf("invalid table shape: %w", domain.ErrInvalidInput)
	}

	// Validate position if provided
	if req.PositionX != nil && req.PositionY != nil {
		if *req.PositionX < 0 || *req.PositionY < 0 {
			return nil, fmt.Errorf("table position coordinates must be non-negative: %w", domain.ErrInvalidInput)
		}
	}

	// Update fields
	if req.Name != "" {
		existing.Name = strings.TrimSpace(req.Name)
	}
	if req.Capacity != nil {
		existing.Capacity = *req.Capacity
	}
	if req.Shape != "" {
		existing.Shape = req.Shape
	}
	if req.PositionX != nil {
		existing.PositionX = req.PositionX
	}
	if req.PositionY != nil {
		existing.PositionY = req.PositionY
	}
	if req.IsLocked != nil {
		existing.IsLocked = *req.IsLocked
	}
	if req.Accessibility != nil {
		existing.Accessibility = *req.Accessibility
	}
	if req.VIPOnly != nil {
		existing.VIPOnly = *req.VIPOnly
	}
	if req.Notes != "" {
		existing.Notes = &req.Notes
	}
	if req.ZoneID != nil {
		existing.ZoneID = req.ZoneID
	}

	existing.Touch()

	if err := s.seatingRepo.UpdateTable(ctx, existing); err != nil {
		return nil, fmt.Errorf("update table: %w", err)
	}

	return existing, nil
}

// DeleteTable soft-deletes a table and removes all seat assignments.
func (s *SeatingService) DeleteTable(ctx context.Context, tenantID, eventID, tableID uuid.UUID, deletedBy uuid.UUID) error {
	if err := s.seatingRepo.SoftDeleteTable(ctx, tenantID, eventID, tableID); err != nil {
		return fmt.Errorf("delete table: %w", err)
	}

	// Audit log
	_ = s.auditSvc.LogWithUser(ctx, deletedBy, tenantID, domain.AuditActionDelete, "table", tableID, map[string]interface{}{
		"event_id": eventID.String(),
		"deleted":  true,
	}, nil)

	return nil
}

// ─── Seat Assignment Operations ───────────────────────────────────────────────

// AssignGuest assigns a guest to a table, checking capacity constraints.
func (s *SeatingService) AssignGuest(ctx context.Context, tenantID, eventID, tableID uuid.UUID, req domain.AssignGuestRequest, assignedBy uuid.UUID) error {
	// Get table info
	table, err := s.seatingRepo.GetTable(ctx, tenantID, eventID, tableID)
	if err != nil {
		return fmt.Errorf("assign guest: %w", err)
	}

	// Check if table is locked
	if table.IsLocked {
		return fmt.Errorf("table is locked and cannot be modified: %w", domain.ErrForbidden)
	}

	// Get guest info
	guest, err := s.guestRepo.GetByIDForTenant(ctx, tenantID, req.GuestID)
	if err != nil {
		return fmt.Errorf("assign guest: %w", err)
	}

	// Check VIP constraint
	if table.VIPOnly {
		if guest.GuestType != domain.GuestTypeVIP && guest.GuestType != domain.GuestTypeVVIP {
			return fmt.Errorf("only VIP/VVIP guests can be assigned to this table: %w", domain.ErrForbidden)
		}
	}

	// Check accessibility constraint
	if table.Accessibility {
		if guest.AccessibilityNeeds == nil || *guest.AccessibilityNeeds == "" {
			// Allow assignment but log warning - accessibility tables are prioritized
			// for guests with accessibility needs but not strictly reserved
		}
	}

	// Check table capacity
	occupancy, err := s.seatingRepo.CountAssignmentsByTable(ctx, tableID)
	if err != nil {
		return fmt.Errorf("assign guest: %w", err)
	}

	// Check if guest is already assigned to this table
	existingAssignment, err := s.seatingRepo.GetGuestAssignment(ctx, req.GuestID)
	if err != nil {
		return fmt.Errorf("assign guest: %w", err)
	}

	// If guest is already at this table, this is an update (seat number change)
	isUpdate := existingAssignment != nil && existingAssignment.TableID == tableID

	if !isUpdate && occupancy >= table.Capacity {
		return fmt.Errorf("table is at full capacity (%d/%d): %w", occupancy, table.Capacity, domain.ErrInvalidInput)
	}

	assignment := &domain.SeatAssignment{
		TableID:    tableID,
		GuestID:    req.GuestID,
		SeatNumber: req.SeatNumber,
		AssignedBy: assignedBy,
		AssignedAt: time.Now().UTC(),
	}

	if err := s.seatingRepo.AssignGuest(ctx, assignment); err != nil {
		return fmt.Errorf("assign guest: %w", err)
	}

	return nil
}

// UnassignGuest removes a guest's seat assignment from a table.
func (s *SeatingService) UnassignGuest(ctx context.Context, tenantID, eventID, tableID, guestID uuid.UUID) error {
	// Verify table exists
	_, err := s.seatingRepo.GetTable(ctx, tenantID, eventID, tableID)
	if err != nil {
		return fmt.Errorf("unassign guest: %w", err)
	}

	if err := s.seatingRepo.UnassignGuest(ctx, tableID, guestID); err != nil {
		return fmt.Errorf("unassign guest: %w", err)
	}

	return nil
}

// AutoAssign automatically assigns guests to tables based on availability and constraints.
func (s *SeatingService) AutoAssign(ctx context.Context, tenantID, eventID uuid.UUID, assignedBy uuid.UUID) (map[string]interface{}, error) {
	// Get all tables for the event
	tables, err := s.seatingRepo.ListTablesByEvent(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("auto assign: %w", err)
	}

	// Get all checked-in guests for the event
	params := domain.CheckinListParams{
		TenantID: tenantID,
		EventID:  eventID,
		Page:     1,
		PerPage:  1000,
	}
	checkins, err := s.checkinRepo.ListByEvent(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("auto assign: %w", err)
	}

	// Get current assignments to avoid re-assigning
	assignments, err := s.seatingRepo.ListAssignmentsByEvent(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("auto assign: %w", err)
	}

	assignedGuestIDs := make(map[uuid.UUID]bool)
	for _, a := range assignments {
		assignedGuestIDs[a.GuestID] = true
	}

	// Track table capacities
	tableCapacities := make(map[uuid.UUID]int)
	tableOccupancies := make(map[uuid.UUID]int)
	for _, t := range tables {
		if !t.IsLocked {
			tableCapacities[t.ID] = t.Capacity
			tableOccupancies[t.ID] = t.Occupancy
		}
	}

	assigned := 0
	skipped := 0

	for _, checkin := range checkins {
		if assignedGuestIDs[checkin.GuestID] {
			skipped++
			continue
		}

		// Find first available table
		assignedTable := uuid.Nil
		for _, t := range tables {
			if t.IsLocked {
				continue
			}
			currentOccupancy := tableOccupancies[t.ID]
			if currentOccupancy < tableCapacities[t.ID] {
				// Check VIP constraint
				if t.VIPOnly {
					guest, err := s.guestRepo.GetByIDForTenant(ctx, tenantID, checkin.GuestID)
					if err != nil {
						continue
					}
					if guest.GuestType != domain.GuestTypeVIP && guest.GuestType != domain.GuestTypeVVIP {
						continue
					}
				}
				assignedTable = t.ID
				break
			}
		}

		if assignedTable == uuid.Nil {
			skipped++
			continue
		}

		seatAssignment := &domain.SeatAssignment{
			TableID:    assignedTable,
			GuestID:    checkin.GuestID,
			AssignedBy: assignedBy,
			AssignedAt: time.Now().UTC(),
		}

		if err := s.seatingRepo.AssignGuest(ctx, seatAssignment); err != nil {
			skipped++
			continue
		}

		tableOccupancies[assignedTable]++
		assigned++
	}

	return map[string]interface{}{
		"assigned": assigned,
		"skipped":  skipped,
	}, nil
}

// GetLayout returns the full seating layout for an event.
func (s *SeatingService) GetLayout(ctx context.Context, tenantID, eventID uuid.UUID) (*domain.SeatingLayout, error) {
	// Get zones
	zones, err := s.seatingRepo.ListZonesByEvent(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get seating layout: %w", err)
	}

	// Get tables with occupancy
	tables, err := s.seatingRepo.ListTablesByEvent(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get seating layout: %w", err)
	}

	// For each table, load assigned guests
	for i := range tables {
		assignedGuests, err := s.seatingRepo.ListAssignmentsByTable(ctx, tables[i].ID)
		if err != nil {
			return nil, fmt.Errorf("get seating layout: %w", err)
		}
		tables[i].AssignedGuests = assignedGuests
	}

	// Count unassigned guests
	unassigned, err := s.seatingRepo.CountUnassignedGuests(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get seating layout: %w", err)
	}

	// Count total checked-in guests
	totalGuests, err := s.checkinRepo.CountByEvent(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get seating layout: %w", err)
	}

	return &domain.SeatingLayout{
		EventID:     eventID,
		Zones:       zones,
		Tables:      tables,
		Unassigned:  unassigned,
		TotalGuests: totalGuests,
	}, nil
}

// ─── Internal Helpers ─────────────────────────────────────────────────────────

// isValidTableShape checks if the given shape is a valid table shape.
func isValidTableShape(shape string) bool {
	switch shape {
	case domain.TableShapeRound, domain.TableShapeRectangular, domain.TableShapeSquare,
		domain.TableShapeOval, domain.TableShapeUShape:
		return true
	}
	return false
}
