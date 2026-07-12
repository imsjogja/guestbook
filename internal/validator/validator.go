// Package validator provides Echo-compatible request validation using
// go-playground/validator. It integrates with Echo's Validator interface
// and translates validation errors into our standard error format.
package validator

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	apperrors "guestflow/pkg/errors"

	playground "github.com/go-playground/validator/v10"
)

// Validator implements Echo's Validator interface using go-playground/validator.
type Validator struct {
	validate *playground.Validate
}

// pool reduces GC pressure by reusing ValidationError slices.
var validationErrorPool = sync.Pool{
	New: func() interface{} {
		return make([]apperrors.ErrorDetail, 0, 8)
	},
}

// New creates a new Validator with custom configuration.
// It registers custom validation tags and sets up tag name lookup
// to use "json" tags for field names in error messages.
func New() *Validator {
	v := playground.New()

	// Use json tag names in validation error messages
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return fld.Name
		}
		return name
	})

	return &Validator{validate: v}
}

// Validate performs struct validation. It implements Echo's Validator interface.
// Returns an *apperrors.AppError with code VALIDATION_ERROR if validation fails.
func (v *Validator) Validate(i interface{}) error {
	if err := v.validate.Struct(i); err != nil {
		validationErrs, ok := err.(playground.ValidationErrors)
		if !ok {
			return apperrors.ValidationError("invalid validation error")
		}

		details := formatValidationErrors(validationErrs)
		return apperrors.ValidationError("Request validation failed", details...)
	}
	return nil
}

// ValidateStruct validates a struct and returns detailed error information.
// This is a convenience method for direct validation calls outside of Echo.
func (v *Validator) ValidateStruct(i interface{}) *apperrors.AppError {
	if err := v.Validate(i); err != nil {
		if appErr, ok := apperrors.IsAppError(err); ok {
			return appErr
		}
		return apperrors.ValidationError("validation failed")
	}
	return nil
}

// formatValidationErrors converts playground validation errors into our
// standard ErrorDetail format for consistent API responses.
func formatValidationErrors(errs playground.ValidationErrors) []apperrors.ErrorDetail {
	details := validationErrorPool.Get().([]apperrors.ErrorDetail)
	defer validationErrorPool.Put(details[:0])

	result := make([]apperrors.ErrorDetail, 0, len(errs))
	for _, err := range errs {
		result = append(result, apperrors.ErrorDetail{
			Field:   err.Field(),
			Message: formatErrorMessage(err),
		})
	}
	return result
}

// formatErrorMessage creates a human-readable message for a single validation error.
func formatErrorMessage(err playground.FieldError) string {
	tag := err.Tag()
	param := err.Param()
	field := err.Field()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, param)
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, param)
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, param)
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, param)
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, param)
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, param)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, strings.ReplaceAll(param, " ", ", "))
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "datetime":
		return fmt.Sprintf("%s must be a valid datetime with format %s", field, param)
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters", field, param)
	default:
		if param != "" {
			return fmt.Sprintf("%s failed validation on '%s' (param: %s)", field, tag, param)
		}
		return fmt.Sprintf("%s failed validation on '%s'", field, tag)
	}
}
