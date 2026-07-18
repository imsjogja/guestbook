package domain

import "time"

// User status values.
const (
	UserStatusActive    = "active"
	UserStatusInactive  = "inactive"
	UserStatusPending   = "pending"
	UserStatusSuspended = "suspended"
	UserStatusDeleted   = "deleted"
)

// User represents a platform user account.
type User struct {
	Base
	Email           string     `db:"email" json:"email"`
	PasswordHash    string     `db:"password_hash" json:"-"`
	FullName        string     `db:"full_name" json:"full_name"`
	Phone           *string    `db:"phone" json:"phone,omitempty"`
	Position        *string    `db:"position" json:"position,omitempty"`
	Bio             *string    `db:"bio" json:"bio,omitempty"`
	AvatarURL       *string    `db:"avatar_url" json:"avatar_url,omitempty"`
	EmailVerifiedAt *time.Time `db:"email_verified_at" json:"email_verified_at,omitempty"`
	MFAEnabled      bool       `db:"mfa_enabled" json:"mfa_enabled"`
	MFASecret       *string    `db:"mfa_secret" json:"-"`
	Status          string     `db:"status" json:"status"`
}

// RegisterRequest captures the data needed to create a new account.
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"required,min=2,max=255"`
	Phone    string `json:"phone,omitempty" validate:"omitempty,e164"`
}

// LoginRequest captures the credentials required to authenticate.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// UserProfileUpdateRequest captures editable account profile fields.
type UserProfileUpdateRequest struct {
	Email    string `json:"email" validate:"required,email"`
	FullName string `json:"full_name" validate:"required,min=2,max=255"`
	Phone    string `json:"phone,omitempty" validate:"omitempty,max=32"`
	Position string `json:"position,omitempty" validate:"omitempty,max=255"`
	Bio      string `json:"bio,omitempty" validate:"omitempty,max=2000"`
}

// ChangePasswordRequest captures the current and replacement passwords.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

// UserInfo contains minimal user information for API responses.
type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role,omitempty"`
}

// IsActive returns true when the user can authenticate.
func (u *User) IsActive() bool {
	if u == nil {
		return false
	}
	return u.Status == UserStatusActive && u.DeletedAt == nil
}

// Sanitize clears sensitive fields before returning a user to clients.
func (u *User) Sanitize() {
	if u == nil {
		return
	}
	u.PasswordHash = ""
	u.MFASecret = nil
}
