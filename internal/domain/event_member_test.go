package domain

import "testing"

func TestEventMemberRoles(t *testing.T) {
	validRoles := []string{
		RoleRSVPOfficer,
		RoleRegistrationOfficer,
		RoleUsher,
		RoleGiftOfficer,
		RoleViewer,
	}

	for _, role := range validRoles {
		if !IsValidEventMemberRole(role) {
			t.Errorf("expected %q to be a valid event member role", role)
		}
	}

	for _, role := range []string{RoleTenantOwner, RoleEventManager, "unknown"} {
		if IsValidEventMemberRole(role) {
			t.Errorf("expected %q to be rejected as an event member role", role)
		}
	}
}

func TestEventManagerUsesEventTeamPermission(t *testing.T) {
	managerPermissions := RolePermissions[RoleEventManager]
	if !hasPermission(managerPermissions, PermEventTeamRead) || !hasPermission(managerPermissions, PermEventTeamWrite) {
		t.Fatal("event manager must be able to manage event staff")
	}
	if hasPermission(managerPermissions, PermTeamWrite) {
		t.Fatal("event manager must not be able to change tenant team membership roles")
	}
}

func TestOnlyTenantManagementRolesCanWriteEvents(t *testing.T) {
	for _, role := range []string{RoleTenantOwner, RoleEventManager} {
		if !hasPermission(RolePermissions[role], PermEventWrite) {
			t.Errorf("role %q must be able to create and update events", role)
		}
	}

	for _, role := range []string{RoleRSVPOfficer, RoleRegistrationOfficer, RoleUsher, RoleGiftOfficer, RoleViewer} {
		if hasPermission(RolePermissions[role], PermEventWrite) {
			t.Errorf("operational role %q must not be able to create or update events", role)
		}
	}
}

func TestAllPermissionsAreUnique(t *testing.T) {
	seen := make(map[string]struct{})
	for _, permission := range AllPermissions() {
		if _, exists := seen[permission]; exists {
			t.Fatalf("permission %q is listed more than once", permission)
		}
		seen[permission] = struct{}{}
	}
}

func hasPermission(permissions []string, wanted string) bool {
	for _, permission := range permissions {
		if permission == wanted {
			return true
		}
	}
	return false
}
