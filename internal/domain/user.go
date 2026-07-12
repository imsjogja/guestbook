package domain

// User represents a user in the system (minimal definition for compilation).
type User struct {
	Base
	Email     string `db:"email" json:"email"`
	FirstName string `db:"first_name" json:"first_name"`
	LastName  string `db:"last_name" json:"last_name"`
	Status    string `db:"status" json:"status"` // active, inactive, pending
}

// UserInfo contains minimal user information for API responses.
type UserInfo struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role,omitempty"`
}
