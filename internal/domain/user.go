package domain

import "time"

// User status values.
const (
	UserStatusActive   = "active"
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
	AvatarURL       *string    `db:"avatar_url" json:"avatar_url,omitempty"`
	EmailVerifiedAt *time.Time `db:"email_verified_at" json:"email_verified_at,omitempty"`
	MFAEnabled      bool       `db:"mfa_enabled" json:"mfa_enabled"`
	MFASecret       *string    `db:"mfa_secret" json:"-"`
	Status          string     `db:"status" json:"status"`
}

// RegisterRequest contains the payload required to create a new user.
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"required,min=2,max=255"`
	Phone    string `json:"phone,omitempty" validate:"omitempty,e164"`
}

// LoginRequest contains the payload required for authentication.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// IsActive reports whether the account is active.
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// Sanitize removes sensitive fields before returning a user outside the service layer.
func (u *User) Sanitize() {
	u.PasswordHash = ""
	u.MFASecret = nil
}

// UserInfo contains minimal user information for API responses.
type UserInfo struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Phone     string `json:"phone,omitempty"`
	Role      string `json:"role,omitempty"`
}
