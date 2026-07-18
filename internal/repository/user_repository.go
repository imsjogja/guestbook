// Package repository provides data access layer implementations for GuestFlow.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"guestflow/internal/domain"
)

// UserRepository provides data access operations for the User domain model.
// It encapsulates all database interactions related to user management including
// creation, retrieval, updates, and soft deletion.
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user into the database.
// The user's ID, CreatedAt, and UpdatedAt fields must be pre-populated.
// Returns an error if the email already exists or database insertion fails.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, full_name, phone, avatar_url, 
		                   email_verified_at, mfa_enabled, status, created_at, updated_at, deleted_at)
		VALUES (:id, :email, :password_hash, :full_name, :phone, :avatar_url,
		        :email_verified_at, :mfa_enabled, :status, :created_at, :updated_at, :deleted_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// GetByID retrieves a user by their UUID. Returns domain-specific errors
// if the user is not found or has been soft-deleted.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	query := `
		SELECT id, email, password_hash, full_name, phone, avatar_url,
		       email_verified_at, mfa_enabled, status, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get user by id: %w", domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}

// GetByEmail retrieves a user by their email address.
// Returns an error if no user with the given email exists or the user is soft-deleted.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	query := `
		SELECT id, email, password_hash, full_name, phone, avatar_url,
		       email_verified_at, mfa_enabled, status, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get user by email: %w", domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &user, nil
}

// Update modifies an existing user's fields in the database.
// The UpdatedAt field should be refreshed before calling this method.
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET email = :email,
		    password_hash = :password_hash,
		    full_name = :full_name,
		    phone = :phone,
		    avatar_url = :avatar_url,
		    email_verified_at = :email_verified_at,
		    mfa_enabled = :mfa_enabled,
		    status = :status,
		    updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL
	`
	result, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check update result: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found or already deleted")
	}

	return nil
}

// SoftDelete marks a user as deleted by setting the deleted_at timestamp.
// The user record is retained in the database for audit and referential integrity.
func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()

	query := `
		UPDATE users
		SET deleted_at = $1, updated_at = $1, email = CONCAT(email, '.deleted.', id::text)
		WHERE id = $2 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("soft delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check delete result: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found or already deleted")
	}

	return nil
}

// EmailExists checks whether a user with the given email already exists
// in the database (excluding soft-deleted users).
func (r *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM users 
			WHERE email = $1 AND deleted_at IS NULL
			LIMIT 1
		)
	`
	err := r.db.GetContext(ctx, &exists, query, email)
	if err != nil {
		return false, fmt.Errorf("check email exists: %w", err)
	}
	return exists, nil
}
