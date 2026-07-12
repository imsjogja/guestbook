package rbac

import (
	"guestflow/internal/domain"
)

// GetPermissionsForRole returns the default permissions for a given role.
func GetPermissionsForRole(role string) []string {
	if perms, ok := domain.RolePermissions[role]; ok {
		return perms
	}
	return []string{}
}

// IsValidRole checks if the given role is a known system role.
func IsValidRole(role string) bool {
	_, ok := domain.RolePermissions[role]
	return ok
}

// AllRoles returns all valid role identifiers.
func AllRoles() []string {
	roles := make([]string, 0, len(domain.RolePermissions))
	for role := range domain.RolePermissions {
		roles = append(roles, role)
	}
	return roles
}

// GetDisplayName returns the human-readable display name for a role.
func GetDisplayName(role string) string {
	if name, ok := domain.RoleDisplayNames[role]; ok {
		return name
	}
	return role
}

// HasPermission checks if a permission is granted within a permission list.
func hasPermission(perms []string, permission string) bool {
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

// hasAnyPermission checks if any of the provided permissions are in the permission list.
func hasAnyPermission(perms []string, permissions ...string) bool {
	for _, perm := range permissions {
		if hasPermission(perms, perm) {
			return true
		}
	}
	return false
}

// hasAllPermissions checks if all provided permissions are in the permission list.
func hasAllPermissions(perms []string, permissions ...string) bool {
	for _, perm := range permissions {
		if !hasPermission(perms, perm) {
			return false
		}
	}
	return true
}
