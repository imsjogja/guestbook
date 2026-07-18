package domain

import "github.com/google/uuid"

// Predefined role constants.
const (
	RoleTenantOwner         = "tenant_owner"
	RoleEventManager        = "event_manager"
	RoleRSVPOfficer         = "rsvp_officer"
	RoleRegistrationOfficer = "registration_officer"
	RoleUsher               = "usher"
	RoleGiftOfficer         = "gift_officer"
	RoleViewer              = "viewer"
)

// Permission constants.
const (
	PermGuestRead          = "guest:read"
	PermGuestWrite         = "guest:write"
	PermGuestDelete        = "guest:delete"
	PermGuestImport        = "guest:import"
	PermGuestExport        = "guest:export"
	PermEventRead          = "event:read"
	PermEventWrite         = "event:write"
	PermEventDelete        = "event:delete"
	PermInvitationRead     = "invitation:read"
	PermInvitationWrite    = "invitation:write"
	PermInvitationSend     = "invitation:send"
	PermRSVPRead           = "rsvp:read"
	PermRSVPWrite          = "rsvp:write"
	PermCheckinRead        = "checkin:read"
	PermCheckinWrite       = "checkin:write"
	PermSeatingRead        = "seating:read"
	PermSeatingWrite       = "seating:write"
	PermReportRead         = "report:read"
	PermReportExport       = "report:export"
	PermCommunicationRead  = "communication:read"
	PermCommunicationWrite = "communication:write"
	PermCommunicationSend  = "communication:send"
	PermBillingRead        = "billing:read"
	PermBillingWrite       = "billing:write"
	PermTeamRead           = "team:read"
	PermTeamWrite          = "team:write"
	PermTeamInvite         = "team:invite"
	PermEventTeamRead      = "event_team:read"
	PermEventTeamWrite     = "event_team:write"
	PermSettingsRead       = "settings:read"
	PermSettingsWrite      = "settings:write"
	PermAuditRead          = "audit:read"
)

// Role defines a role within the system. Roles can be system-wide or tenant-specific.
type Role struct {
	Base
	Name        string     `db:"name" json:"name"`
	DisplayName string     `db:"display_name" json:"display_name"`
	Description string     `db:"description" json:"description"`
	Permissions []string   `db:"permissions" json:"permissions"`
	IsSystem    bool       `db:"is_system" json:"is_system"`
	TenantID    *uuid.UUID `db:"tenant_id" json:"tenant_id,omitempty"`
}

// AllPermissions returns the complete list of all defined permissions.
func AllPermissions() []string {
	return []string{
		PermGuestRead,
		PermGuestWrite,
		PermGuestDelete,
		PermGuestImport,
		PermGuestExport,
		PermEventRead,
		PermEventWrite,
		PermEventDelete,
		PermInvitationRead,
		PermInvitationWrite,
		PermInvitationSend,
		PermRSVPRead,
		PermRSVPWrite,
		PermCheckinRead,
		PermCheckinWrite,
		PermSeatingRead,
		PermSeatingWrite,
		PermReportRead,
		PermReportExport,
		PermCommunicationRead,
		PermCommunicationWrite,
		PermCommunicationSend,
		PermBillingRead,
		PermBillingWrite,
		PermTeamRead, PermTeamWrite,
		PermTeamInvite,
		PermEventTeamRead, PermEventTeamWrite,
		PermSettingsRead,
		PermSettingsWrite,
		PermAuditRead,
	}
}

// RolePermissions maps predefined roles to their granted permissions.
var RolePermissions = map[string][]string{
	RoleTenantOwner: {
		PermGuestRead, PermGuestWrite, PermGuestDelete, PermGuestImport, PermGuestExport,
		PermEventRead, PermEventWrite, PermEventDelete,
		PermInvitationRead, PermInvitationWrite, PermInvitationSend,
		PermRSVPRead, PermRSVPWrite,
		PermCheckinRead, PermCheckinWrite,
		PermSeatingRead, PermSeatingWrite,
		PermReportRead, PermReportExport,
		PermCommunicationRead, PermCommunicationWrite, PermCommunicationSend,
		PermBillingRead, PermBillingWrite,
		PermTeamRead, PermTeamWrite, PermTeamInvite,
		PermEventTeamRead, PermEventTeamWrite,
		PermSettingsRead, PermSettingsWrite,
		PermAuditRead,
	},
	RoleEventManager: {
		PermGuestRead, PermGuestWrite, PermGuestDelete, PermGuestImport, PermGuestExport,
		PermEventRead, PermEventWrite, PermEventDelete,
		PermInvitationRead, PermInvitationWrite, PermInvitationSend,
		PermRSVPRead, PermRSVPWrite,
		PermCheckinRead, PermCheckinWrite,
		PermSeatingRead, PermSeatingWrite,
		PermReportRead, PermReportExport,
		PermCommunicationRead, PermCommunicationWrite, PermCommunicationSend,
		PermTeamRead,
		PermEventTeamRead, PermEventTeamWrite,
		PermSettingsRead,
	},
	RoleRSVPOfficer: {
		PermGuestRead, PermGuestWrite,
		PermEventRead,
		PermInvitationRead, PermInvitationWrite, PermInvitationSend,
		PermRSVPRead, PermRSVPWrite,
		PermCommunicationRead, PermCommunicationWrite, PermCommunicationSend,
		PermReportRead,
		PermSettingsRead,
	},
	RoleRegistrationOfficer: {
		PermGuestRead, PermGuestWrite, PermGuestImport, PermGuestExport,
		PermEventRead,
		PermCheckinRead, PermCheckinWrite,
		PermReportRead, PermReportExport,
		PermSettingsRead,
	},
	RoleUsher: {
		PermGuestRead,
		PermEventRead,
		PermCheckinRead, PermCheckinWrite,
		PermSeatingRead,
	},
	RoleGiftOfficer: {
		PermGuestRead,
		PermEventRead,
		PermReportRead,
	},
	RoleViewer: {
		PermGuestRead,
		PermEventRead,
		PermInvitationRead,
		PermRSVPRead,
		PermCheckinRead,
		PermSeatingRead,
		PermReportRead,
		PermCommunicationRead,
		PermTeamRead,
		PermEventTeamRead,
		PermSettingsRead,
	},
}

// IsValidRole checks if the given role identifier is a known predefined role.
func IsValidRole(role string) bool {
	_, ok := RolePermissions[role]
	return ok
}

// RoleDisplayNames maps role identifiers to human-readable display names.
var RoleDisplayNames = map[string]string{
	RoleTenantOwner:         "Tenant Owner",
	RoleEventManager:        "Event Manager",
	RoleRSVPOfficer:         "RSVP Officer",
	RoleRegistrationOfficer: "Registration Officer",
	RoleUsher:               "Usher",
	RoleGiftOfficer:         "Gift Officer",
	RoleViewer:              "Viewer",
}
