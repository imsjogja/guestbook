// Package response provides standardized API response helpers for GuestFlow.
// All HTTP handlers should use these functions to ensure consistent response formats.
package response

import (
	"net/http"

	apperrors "guestflow/pkg/errors"

	"github.com/labstack/echo/v4"
)

// Response is the standard success response envelope.
type Response struct {
	Data interface{} `json:"data"`
	Meta *Meta       `json:"meta,omitempty"`
}

// PaginatedResponse is the response envelope for paginated list endpoints.
type PaginatedResponse struct {
	Data interface{} `json:"data"`
	Meta Meta        `json:"meta"`
}

// ErrorResponse is the standard error response envelope.
type ErrorResponse struct {
	Error   string                  `json:"error"`
	Code    string                  `json:"code"`
	Details []apperrors.ErrorDetail `json:"details,omitempty"`
}

// Meta contains pagination metadata for list responses.
type Meta struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	Total       int `json:"total"`
	TotalPages  int `json:"total_pages"`
}

// ------------------------------------------------------------------------------
// Success responses
// ------------------------------------------------------------------------------

// Success sends a 200 OK response with data.
func Success(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, Response{
		Data: data,
	})
}

// Created sends a 201 Created response with data.
func Created(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusCreated, Response{
		Data: data,
	})
}

// NoContent sends a 204 No Content response.
func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// ------------------------------------------------------------------------------
// Paginated response
// ------------------------------------------------------------------------------

// Paginated sends a 200 OK response with paginated data and metadata.
func Paginated(c echo.Context, data interface{}, meta Meta) error {
	return c.JSON(http.StatusOK, PaginatedResponse{
		Data: data,
		Meta: meta,
	})
}

// ------------------------------------------------------------------------------
// Error responses
// ------------------------------------------------------------------------------

// Error sends an error response with the appropriate HTTP status code.
func Error(c echo.Context, err *apperrors.AppError) error {
	return c.JSON(err.Status, ErrorResponse{
		Error:   err.Message,
		Code:    err.Code,
		Details: err.Details,
	})
}

// BadRequest sends a 400 Bad Request response.
func BadRequest(c echo.Context, message string) error {
	return Error(c, apperrors.BadRequest(message))
}

// Unauthorized sends a 401 Unauthorized response.
func Unauthorized(c echo.Context, message string) error {
	return Error(c, apperrors.Unauthorized(message))
}

// Forbidden sends a 403 Forbidden response.
func Forbidden(c echo.Context, message string) error {
	return Error(c, apperrors.Forbidden(message))
}

// NotFound sends a 404 Not Found response.
func NotFound(c echo.Context, resource string) error {
	return Error(c, apperrors.NotFound(resource))
}

// Conflict sends a 409 Conflict response.
func Conflict(c echo.Context, message string) error {
	return Error(c, apperrors.Conflict(message))
}

// ValidationError sends a 422 Unprocessable Entity response.
func ValidationError(c echo.Context, message string, details ...apperrors.ErrorDetail) error {
	return Error(c, apperrors.ValidationError(message, details...))
}

// InternalError sends a 500 Internal Server Error response.
func InternalError(c echo.Context, message string) error {
	return Error(c, apperrors.Internal(message))
}

// ServiceUnavailable sends a 503 response for temporarily unavailable integrations.
func ServiceUnavailable(c echo.Context, message string) error {
	return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
		Error: message,
		Code:  "SERVICE_UNAVAILABLE",
	})
}
