// Package service provides business logic implementations for GuestFlow.
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"guestflow/internal/auth"
	"guestflow/internal/domain"
	"guestflow/internal/email"
	"guestflow/internal/repository"
)

// Common authentication errors.
var (
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrTokenInvalid       = errors.New("invalid or expired token")
	ErrEmailNotVerified   = errors.New("email address is not verified")
	ErrEmailDelivery      = errors.New("verification email could not be sent")
)

// AuthService handles all authentication-related business logic including
// user registration, login, logout, token refresh, and session management.
type AuthService struct {
	db                 *sqlx.DB
	jwtService         *auth.JWTService
	refreshSvc         *auth.RefreshTokenService
	userRepo           *repository.UserRepository
	tokenRepo          *repository.RefreshTokenRepository
	verificationRepo   *repository.EmailVerificationRepository
	mailer             email.Mailer
	verifyEmailEnabled bool
	publicURL          string
}

// NewAuthService creates a new AuthService with all required dependencies.
func NewAuthService(
	db *sqlx.DB,
	jwtService *auth.JWTService,
	refreshSvc *auth.RefreshTokenService,
	mailer email.Mailer,
	verifyEmailEnabled bool,
	publicURL string,
) *AuthService {
	return &AuthService{
		db:                 db,
		jwtService:         jwtService,
		refreshSvc:         refreshSvc,
		userRepo:           repository.NewUserRepository(db),
		tokenRepo:          repository.NewRefreshTokenRepository(db),
		verificationRepo:   repository.NewEmailVerificationRepository(db),
		mailer:             mailer,
		verifyEmailEnabled: verifyEmailEnabled,
		publicURL:          strings.TrimRight(publicURL, "/"),
	}
}

// Register creates a new user account with the provided registration data.
// It validates email uniqueness and hashes the password. When email delivery is
// enabled, the account is created without a session until the address is
// verified; otherwise it keeps the development login behavior.
//
// Returns ErrEmailExists if the email is already registered.
func (s *AuthService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.User, *auth.TokenPair, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
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

	var rawVerificationToken string
	var verificationToken *domain.EmailVerificationToken
	if s.verifyEmailEnabled {
		var err error
		rawVerificationToken, verificationToken, err = newVerificationToken(user.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("create email verification token: %w", err)
		}
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

	var tokenPair *auth.TokenPair
	if s.verifyEmailEnabled {
		if err := s.verificationRepo.Create(ctx, tx, verificationToken); err != nil {
			return nil, nil, err
		}
	} else {
		// Store the refresh token in the same transaction as the user. The user
		// foreign key is not visible to another connection until this transaction
		// commits, so using the shared transaction avoids a registration-time FK
		// violation.
		tokenPair, err = s.jwtService.GenerateTokenPair(user.ID, user.Email)
		if err != nil {
			return nil, nil, fmt.Errorf("generate tokens: %w", err)
		}
		_, rawRefreshToken, err := s.refreshSvc.CreateWithExecutor(ctx, tx, user.ID, "default")
		if err != nil {
			return nil, nil, fmt.Errorf("create refresh token: %w", err)
		}
		tokenPair.RefreshToken = rawRefreshToken
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit transaction: %w", err)
	}

	user.Sanitize()
	if s.verifyEmailEnabled {
		if err := s.sendVerificationEmail(ctx, user.Email, user.FullName, rawVerificationToken); err != nil {
			return user, nil, fmt.Errorf("%w: %v", ErrEmailDelivery, err)
		}
	}
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
	if s.verifyEmailEnabled && user.EmailVerifiedAt == nil {
		return nil, nil, ErrEmailNotVerified
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

// VerifyEmail consumes a one-time verification token.
func (s *AuthService) VerifyEmail(ctx context.Context, rawToken string) error {
	if !s.verifyEmailEnabled {
		return nil
	}
	if strings.TrimSpace(rawToken) == "" {
		return ErrTokenInvalid
	}
	return s.verificationRepo.Consume(ctx, hashVerificationToken(rawToken), time.Now().UTC())
}

// ResendVerification issues a new token. It intentionally returns success for
// unknown or already verified addresses to avoid account enumeration.
func (s *AuthService) ResendVerification(ctx context.Context, emailAddress string) error {
	if !s.verifyEmailEnabled {
		return nil
	}
	user, err := s.userRepo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(emailAddress)))
	if err != nil || user.EmailVerifiedAt != nil {
		return nil
	}

	rawToken, token, err := newVerificationToken(user.ID)
	if err != nil {
		return fmt.Errorf("create email verification token: %w", err)
	}
	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin resend verification transaction: %w", err)
	}
	defer tx.Rollback()
	if err := s.verificationRepo.InvalidateActive(ctx, tx, user.ID); err != nil {
		return err
	}
	if err := s.verificationRepo.Create(ctx, tx, token); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit resend verification transaction: %w", err)
	}
	if err := s.sendVerificationEmail(ctx, user.Email, user.FullName, rawToken); err != nil {
		return fmt.Errorf("%w: %v", ErrEmailDelivery, err)
	}
	return nil
}

func newVerificationToken(userID uuid.UUID) (string, *domain.EmailVerificationToken, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	rawToken := hex.EncodeToString(raw)
	return rawToken, &domain.EmailVerificationToken{
		ID: uuid.New(), UserID: userID, TokenHash: hashVerificationToken(rawToken),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour), CreatedAt: time.Now().UTC(),
	}, nil
}

func hashVerificationToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}

func (s *AuthService) sendVerificationEmail(ctx context.Context, recipient, fullName, rawToken string) error {
	if s.mailer == nil {
		return fmt.Errorf("email mailer is not configured")
	}
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", strings.TrimRight(s.publicURL, "/"), url.QueryEscape(rawToken))
	body := fmt.Sprintf("Halo %s,\n\nKlik tautan berikut untuk memverifikasi email akun GuestFlow Anda:\n%s\n\nTautan ini berlaku selama 24 jam dan hanya dapat digunakan sekali.\n\nJika Anda tidak membuat akun GuestFlow, abaikan email ini.\n", fullName, verifyURL)
	return s.mailer.Send(ctx, recipient, "Verifikasi email GuestFlow", body)
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
