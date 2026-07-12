// Package service provides business logic implementations for GuestFlow.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"guestflow/internal/auth"
	"guestflow/internal/domain"
	"guestflow/internal/repository"
)

// Common authentication errors.
var (
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrTokenInvalid       = errors.New("invalid or expired token")
)

// AuthService handles all authentication-related business logic including
// user registration, login, logout, token refresh, and session management.
type AuthService struct {
	db         *sqlx.DB
	jwtService *auth.JWTService
	refreshSvc *auth.RefreshTokenService
	userRepo   *repository.UserRepository
	tokenRepo  *repository.RefreshTokenRepository
}

// NewAuthService creates a new AuthService with all required dependencies.
func NewAuthService(
	db *sqlx.DB,
	jwtService *auth.JWTService,
	refreshSvc *auth.RefreshTokenService,
) *AuthService {
	return &AuthService{
		db:         db,
		jwtService: jwtService,
		refreshSvc: refreshSvc,
		userRepo:   repository.NewUserRepository(db),
		tokenRepo:  repository.NewRefreshTokenRepository(db),
	}
}

// Register creates a new user account with the provided registration data.
// It validates email uniqueness, hashes the password, and returns the created
// user along with a fresh token pair.
//
// Returns ErrEmailExists if the email is already registered.
func (s *AuthService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.User, *auth.TokenPair, error) {
	// Check email uniqueness
	exists, err := s.userRepo.EmailExists(ctx, req.Email)
	if err != nil {
		return nil, nil, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return nil, nil, ErrEmailExists
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	// Create user
	user := &domain.User{
		Base:         domain.NewBase(),
		Email:        req.Email,
		PasswordHash: passwordHash,
		FullName:     req.FullName,
		Status:       string(domain.UserStatusActive),
	}
	if req.Phone != "" {
		user.Phone = &req.Phone
	}

	// Insert user into database within a transaction
	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO users (id, email, password_hash, full_name, phone, avatar_url,
		                   email_verified_at, mfa_enabled, status, created_at, updated_at, deleted_at)
		VALUES (:id, :email, :password_hash, :full_name, :phone, :avatar_url,
		        :email_verified_at, :mfa_enabled, :status, :created_at, :updated_at, :deleted_at)
	`
	_, err = tx.NamedExecContext(ctx, query, user)
	if err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	// Generate tokens
	tokenPair, err := s.jwtService.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	// Store refresh token
	_, _, err = s.refreshSvc.Create(ctx, user.ID, "default")
	if err != nil {
		return nil, nil, fmt.Errorf("create refresh token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit transaction: %w", err)
	}

	user.Sanitize()
	return user, tokenPair, nil
}

// Login authenticates a user with their email and password.
// Returns the user and a fresh token pair upon successful authentication.
//
// Returns ErrInvalidCredentials for invalid email/password combinations
// to prevent user enumeration attacks.
func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.User, *auth.TokenPair, error) {
	// Retrieve user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		// Use generic error to prevent email enumeration
		return nil, nil, ErrInvalidCredentials
	}

	// Check account status
	if !user.IsActive() {
		return nil, nil, ErrUserInactive
	}

	// Verify password
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		return nil, nil, ErrInvalidCredentials
	}

	// Generate token pair
	tokenPair, err := s.jwtService.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	// Create and store refresh token
	_, rawRefreshToken, err := s.refreshSvc.Create(ctx, user.ID, "default")
	if err != nil {
		return nil, nil, fmt.Errorf("create refresh token: %w", err)
	}

	tokenPair.RefreshToken = rawRefreshToken

	user.Sanitize()
	return user, tokenPair, nil
}

// Logout invalidates a user's refresh token and ends their session.
// The access token will expire naturally based on its short TTL.
func (s *AuthService) Logout(ctx context.Context, userID, refreshTokenID uuid.UUID) error {
	if err := s.refreshSvc.Revoke(ctx, refreshTokenID, userID); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}

// Refresh creates a new access token from a valid refresh token.
// This implements token rotation where the old refresh token is invalidated
// and a new one is issued along with the new access token.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	// Validate the refresh token in the database
	storedToken, err := s.refreshSvc.Validate(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	// Retrieve the user to include in new tokens
	user, err := s.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("refresh: user not found: %w", err)
	}

	// Generate new access token
	accessToken, err := s.jwtService.GenerateAccessToken(user.ID, user.Email, uuid.Nil, "")
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// Revoke old refresh token and create new one
	if err := s.refreshSvc.Revoke(ctx, storedToken.ID, user.ID); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}

	_, newRawToken, err := s.refreshSvc.Create(ctx, user.ID, "default")
	if err != nil {
		return nil, fmt.Errorf("create new refresh token: %w", err)
	}

	return &auth.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRawToken,
		ExpiresIn:    900, // 15 minutes in seconds
	}, nil
}

// Me returns the current user's profile information.
func (s *AuthService) Me(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserNotFound, err)
	}
	user.Sanitize()
	return user, nil
}
