package domain

import "errors"

// Domain errors used across the application.
var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists is returned when a resource already exists.
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrUnauthorized is returned when the caller lacks permission.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when the caller is not allowed to perform an action.
	ErrForbidden = errors.New("forbidden")

	// ErrInvalidInput is returned when request validation fails.
	ErrInvalidInput = errors.New("invalid input")

	// ErrTenantNotFound is returned when a tenant does not exist.
	ErrTenantNotFound = errors.New("tenant not found")

	// ErrUserNotFound is returned when a user does not exist.
	ErrUserNotFound = errors.New("user not found")

	// ErrDuplicateSlug is returned when a tenant slug already exists.
	ErrDuplicateSlug = errors.New("tenant slug already exists")

	// ErrInvalidRole is returned when an invalid role is specified.
	ErrInvalidRole = errors.New("invalid role")

	// ErrMembershipNotFound is returned when a tenant membership does not exist.
	ErrMembershipNotFound = errors.New("membership not found")

	// ErrCannotRemoveOwner is returned when attempting to remove the tenant owner.
	ErrCannotRemoveOwner = errors.New("cannot remove tenant owner")

	// ErrOwnerRoleImmutable is returned when attempting to change the owner's role.
	ErrOwnerRoleImmutable = errors.New("owner role cannot be changed")
)
