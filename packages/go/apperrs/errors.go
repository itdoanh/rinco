// Package apperrs provides typed application errors with HTTP and gRPC status
// mapping, structured metadata, and helpers for serialization.
//
// Use these errors at the boundary between business logic and transport (HTTP,
// gRPC, CLI, etc.) so that the appropriate status code can be derived
// without sprinkling transport-specific logic everywhere.
//
// Example:
//
//	if user == nil {
//	    return apperrs.NotFound("user does not exist").
//	        WithDetail("user_id", id)
//	}
//
// In an Echo handler:
//
//	if err != nil {
//	    return c.JSON(err.HTTPStatus(), err)
//	}
package apperrs

import (
	"errors"
	"fmt"
	"net/http"
)

// Standard error codes used across RINCO services.
const (
	CodeUnknown          = "unknown"
	CodeInvalidArgument  = "invalid_argument"
	CodeValidation       = "validation_failed"
	CodeUnauthenticated  = "unauthenticated"
	CodePermissionDenied = "permission_denied"
	CodeNotFound         = "not_found"
	CodeAlreadyExists    = "already_exists"
	CodeConflict         = "conflict"
	CodeRateLimited      = "rate_limited"
	CodeDeadlineExceeded = "deadline_exceeded"
	CodeUnavailable      = "unavailable"
	CodeInternal         = "internal"
)

// AppError is the canonical typed error type.
type AppError struct {
	Code       string            // machine-readable code
	Message    string            // human-readable message
	HTTPStatus int               // HTTP status code
	GRPCStatus int               // gRPC status code
	Cause      error             // optional wrapped cause
	Details    map[string]any    // structured context (key/value)
	Stack      string            // optional stack trace
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap exposes the wrapped cause for errors.Is / errors.As.
func (e *AppError) Unwrap() error { return e.Cause }

// Is matches AppError by Code so callers can compare types by code.
func (e *AppError) Is(target error) bool {
	var other *AppError
	if errors.As(target, &other) {
		return other.Code == e.Code
	}
	return false
}

// WithDetail adds a key/value detail.
func (e *AppError) WithDetail(key string, value any) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// WithDetails replaces the entire detail map.
func (e *AppError) WithDetails(details map[string]any) *AppError {
	e.Details = details
	return e
}

// WithCause wraps an underlying error.
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// WithStack sets a custom stack trace string.
func (e *AppError) WithStack(stack string) *AppError {
	e.Stack = stack
	return e
}

// WithMessage overrides the message.
func (e *AppError) WithMessage(msg string) *AppError {
	e.Message = msg
	return e
}

// ===== Constructors =====

func newErr(code, message string, httpStatus, grpcStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		GRPCStatus: grpcStatus,
	}
}

// InvalidArgument creates a 400 / InvalidArgument error.
func InvalidArgument(message string) *AppError {
	return newErr(CodeInvalidArgument, message, http.StatusBadRequest, 3)
}

// Validation creates a 400 / InvalidArgument error optimised for form
// validation with a structured details map.
func Validation(message string) *AppError {
	return InvalidArgument(message)
}

// Unauthenticated creates a 401 / Unauthenticated error.
func Unauthenticated(message string) *AppError {
	return newErr(CodeUnauthenticated, message, http.StatusUnauthorized, 16)
}

// PermissionDenied creates a 403 / PermissionDenied error.
func PermissionDenied(message string) *AppError {
	return newErr(CodePermissionDenied, message, http.StatusForbidden, 7)
}

// NotFound creates a 404 / NotFound error.
func NotFound(message string) *AppError {
	return newErr(CodeNotFound, message, http.StatusNotFound, 5)
}

// AlreadyExists creates a 409 / AlreadyExists error.
func AlreadyExists(message string) *AppError {
	return newErr(CodeAlreadyExists, message, http.StatusConflict, 6)
}

// Conflict creates a 409 / Aborted error.
func Conflict(message string) *AppError {
	return newErr(CodeConflict, message, http.StatusConflict, 10)
}

// RateLimited creates a 429 / ResourceExhausted error.
func RateLimited(message string) *AppError {
	return newErr(CodeRateLimited, message, http.StatusTooManyRequests, 8)
}

// DeadlineExceeded creates a 504 / DeadlineExceeded error.
func DeadlineExceeded(message string) *AppError {
	return newErr(CodeDeadlineExceeded, message, http.StatusGatewayTimeout, 4)
}

// Unavailable creates a 503 / Unavailable error.
func Unavailable(message string) *AppError {
	return newErr(CodeUnavailable, message, http.StatusServiceUnavailable, 14)
}

// Internal creates a 500 / Internal error.
func Internal(message string) *AppError {
	return newErr(CodeInternal, message, http.StatusInternalServerError, 13)
}

// Wrap wraps an arbitrary error with a fallback code; if cause is already an
// *AppError it is returned unchanged.
func Wrap(cause error, fallbackCode, fallbackMessage string) *AppError {
	if cause == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(cause, &appErr) {
		return appErr
	}
	e := newErr(fallbackCode, fallbackMessage, http.StatusInternalServerError, 13)
	e.Cause = cause
	return e
}

// As extracts an *AppError from the chain or returns nil.
func As(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// HTTPStatus returns the HTTP status code, falling back to 500.
func HTTPStatus(err error) int {
	if appErr := As(err); appErr != nil {
		return appErr.HTTPStatus
	}
	return http.StatusInternalServerError
}

// GRPCStatus returns the gRPC status code, falling back to Unknown (2).
func GRPCStatus(err error) int {
	if appErr := As(err); appErr != nil {
		return appErr.GRPCStatus
	}
	return 2
}

// Code returns the canonical error code, falling back to CodeUnknown.
func Code(err error) string {
	if appErr := As(err); appErr != nil {
		return appErr.Code
	}
	return CodeUnknown
}