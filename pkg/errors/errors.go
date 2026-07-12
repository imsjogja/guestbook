// Package errors provides custom error types for GuestFlow.
// All application errors are wrapped in AppError with consistent codes and HTTP status mappings.
package errors

import (
	"fmt"
	"net/http"
)

// ErrorDetail represents a single validation error detail with field and message.
type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// AppError is the standard application error type used throughout GuestFlow.
// It provides structured error information including a machine-readable code,
// human-readable message, HTTP status, and optional validation details.
type AppError struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Status  int           `json:"status"`
	Details []ErrorDetail `json:"details,omitempty"`
	// internal stores the original error for logging purposes
	internal error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.internal != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.internal)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the internal error for errors.Is and errors.As compatibility.
func (e *AppError) Unwrap() error {
	return e.internal
}

// WithInternal attaches an internal error for logging while keeping the
// user-facing message unchanged.
func (e *AppError) WithInternal(err error) *AppError {
	e.internal = err
	return e
}

// WithDetails adds validation error details to the error.
func (e *AppError) WithDetails(details ...ErrorDetail) *AppError {
	e.Details = append(e.Details, details...)
	return e
}

// IsAppError checks if an error is an *AppError.
func IsAppError(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	if ae, ok := err.(*AppError); ok {
		return ae, true
	}
	return nil, false
}

// ------------------------------------------------------------------------------
// Error constructors - create standard application errors
// ------------------------------------------------------------------------------

// BadRequest creates a 400 Bad Request error.
func BadRequest(message string) *AppError {
	if message == "" {
		message = "Invalid request"
	}
	return &AppError{
		Code:    "BAD_REQUEST",
		Message: message,
		Status:  http.StatusBadRequest,
	}
}

// Unauthorized creates a 401 Unauthorized error.
func Unauthorized(message string) *AppError {
	if message == "" {
		message = "Authentication required"
	}
	return &AppError{
		Code:    "UNAUTHORIZED",
		Message: message,
		Status:  http.StatusUnauthorized,
	}
}

// Forbidden creates a 403 Forbidden error.
func Forbidden(message string) *AppError {
	if message == "" {
		message = "Insufficient permissions"
	}
	return &AppError{
		Code:    "FORBIDDEN",
		Message: message,
		Status:  http.StatusForbidden,
	}
}

// NotFound creates a 404 Not Found error.
func NotFound(resource string) *AppError {
	message := "Resource not found"
	if resource != "" {
		message = fmt.Sprintf("%s not found", resource)
	}
	return &AppError{
		Code:    "NOT_FOUND",
		Message: message,
		Status:  http.StatusNotFound,
	}
}

// Conflict creates a 409 Conflict error.
func Conflict(message string) *AppError {
	if message == "" {
		message = "Resource conflict"
	}
	return &AppError{
		Code:    "CONFLICT",
		Message: message,
		Status:  http.StatusConflict,
	}
}

// ValidationError creates a 422 Unprocessable Entity error with field details.
func ValidationError(message string, details ...ErrorDetail) *AppError {
	if message == "" {
		message = "Validation failed"
	}
	return &AppError{
		Code:    "VALIDATION_ERROR",
		Message: message,
		Status:  http.StatusUnprocessableEntity,
		Details: details,
	}
}

// Internal creates a 500 Internal Server Error.
func Internal(message string) *AppError {
	if message == "" {
		message = "An unexpected error occurred"
	}
	return &AppError{
		Code:    "INTERNAL_ERROR",
		Message: message,
		Status:  http.StatusInternalServerError,
	}
}

// RateLimited creates a 429 Too Many Requests error.
func RateLimited(message string) *AppError {
	if message == "" {
		message = "Too many requests"
	}
	return &AppError{
		Code:    "RATE_LIMITED",
		Message: message,
		Status:  http.StatusTooManyRequests,
	}
}

// TenantRequired creates a 400 error for missing tenant header.
func TenantRequired() *AppError {
	return &AppError{
		Code:    "TENANT_REQUIRED",
		Message: "X-Tenant-ID header is required",
		Status:  http.StatusBadRequest,
	}
}

// InvalidTenant creates a 403 error for invalid tenant access.
func InvalidTenant() *AppError {
	return &AppError{
		Code:    "INVALID_TENANT",
		Message: "Tenant access denied or tenant does not exist",
		Status:  http.StatusForbidden,
	}
}

// ------------------------------------------------------------------------------
// Wrap constructors - wrap an existing error with context
// ------------------------------------------------------------------------------

// WrapInternal wraps an internal error with a user-friendly message.
func WrapInternal(err error, message string) *AppError {
	return Internal(message).WithInternal(err)
}

// WrapBadRequest wraps an error as a bad request.
func WrapBadRequest(err error, message string) *AppError {
	return BadRequest(message).WithInternal(err)
}

// WrapNotFound wraps an error as not found.
func WrapNotFound(err error, resource string) *AppError {
	return NotFound(resource).WithInternal(err)
}
