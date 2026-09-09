// Tests for apperrs package (errors.go).
package apperrs

import (
	"errors"
	"net/http"
	"testing"
)

func TestExtra2_CodeConstants(t *testing.T) {
	if CodeUnknown == "" {
		t.Error("CodeUnknown should not be empty")
	}
	if CodeInvalidArgument == "" {
		t.Error("CodeInvalidArgument should not be empty")
	}
	if CodeValidation == "" {
		t.Error("CodeValidation should not be empty")
	}
	if CodeUnauthenticated == "" {
		t.Error("CodeUnauthenticated should not be empty")
	}
	if CodePermissionDenied == "" {
		t.Error("CodePermissionDenied should not be empty")
	}
	if CodeNotFound == "" {
		t.Error("CodeNotFound should not be empty")
	}
	if CodeAlreadyExists == "" {
		t.Error("CodeAlreadyExists should not be empty")
	}
	if CodeConflict == "" {
		t.Error("CodeConflict should not be empty")
	}
	if CodeRateLimited == "" {
		t.Error("CodeRateLimited should not be empty")
	}
	if CodeDeadlineExceeded == "" {
		t.Error("CodeDeadlineExceeded should not be empty")
	}
	if CodeUnavailable == "" {
		t.Error("CodeUnavailable should not be empty")
	}
	if CodeInternal == "" {
		t.Error("CodeInternal should not be empty")
	}
}

func TestExtra2_AppError_Fields(t *testing.T) {
	e := &AppError{
		Code:       CodeNotFound,
		Message:    "user not found",
		HTTPStatus: http.StatusNotFound,
		GRPCStatus: 5,
		Details:    map[string]any{"user_id": "123"},
		Stack:      "main.go:42",
	}
	if e.Code != CodeNotFound {
		t.Error("Code")
	}
	if e.Message != "user not found" {
		t.Error("Message")
	}
	if e.HTTPStatus != http.StatusNotFound {
		t.Error("HTTPStatus")
	}
	if e.GRPCStatus != 5 {
		t.Error("GRPCStatus")
	}
	if e.Details["user_id"] != "123" {
		t.Error("Details")
	}
	if e.Stack != "main.go:42" {
		t.Error("Stack")
	}
}

func TestExtra2_InvalidArgument(t *testing.T) {
	e := InvalidArgument("invalid input")
	if e.Code != CodeInvalidArgument {
		t.Errorf("Code: got %s", e.Code)
	}
	if e.HTTPStatus != http.StatusBadRequest {
		t.Errorf("HTTPStatus: got %d", e.HTTPStatus)
	}
	if e.GRPCStatus != 3 {
		t.Errorf("GRPCStatus: got %d", e.GRPCStatus)
	}
}

func TestExtra2_Validation(t *testing.T) {
	e := Validation("validation failed")
	// Validation is an alias for InvalidArgument
	if e.HTTPStatus != http.StatusBadRequest {
		t.Errorf("HTTPStatus: got %d", e.HTTPStatus)
	}
}

func TestExtra2_Unauthenticated(t *testing.T) {
	e := Unauthenticated("no token")
	if e.Code != CodeUnauthenticated {
		t.Errorf("Code: got %s", e.Code)
	}
	if e.HTTPStatus != http.StatusUnauthorized {
		t.Errorf("HTTPStatus: got %d", e.HTTPStatus)
	}
	if e.GRPCStatus != 16 {
		t.Errorf("GRPCStatus: got %d", e.GRPCStatus)
	}
}

func TestExtra2_PermissionDenied(t *testing.T) {
	e := PermissionDenied("access denied")
	if e.Code != CodePermissionDenied {
		t.Errorf("Code: got %s", e.Code)
	}
	if e.HTTPStatus != http.StatusForbidden {
		t.Errorf("HTTPStatus: got %d", e.HTTPStatus)
	}
	if e.GRPCStatus != 7 {
		t.Errorf("GRPCStatus: got %d", e.GRPCStatus)
	}
}

func TestExtra2_NotFound(t *testing.T) {
	e := NotFound("user not found")
	if e.Code != CodeNotFound {
		t.Errorf("Code: got %s", e.Code)
	}
	if e.HTTPStatus != http.StatusNotFound {
		t.Errorf("HTTPStatus: got %d", e.HTTPStatus)
	}
	if e.GRPCStatus != 5 {
		t.Errorf("GRPCStatus: got %d", e.GRPCStatus)
	}
}

func TestExtra2_AlreadyExists(t *testing.T) {
	e := AlreadyExists("user exists")
	if e.Code != CodeAlreadyExists {
		t.Errorf("Code: got %s", e.Code)
	}
	if e.HTTPStatus != http.StatusConflict {
		t.Errorf("HTTPStatus: got %d", e.HTTPStatus)
	}
	if e.GRPCStatus != 6 {
		t.Errorf("GRPCStatus: got %d", e.GRPCStatus)
	}
}

func TestExtra2_RateLimited(t *testing.T) {
	e := RateLimited("too many requests")
	if e.Code != CodeRateLimited {
		t.Errorf("Code: got %s", e.Code)
	}
	if e.HTTPStatus != http.StatusTooManyRequests {
		t.Errorf("HTTPStatus: got %d", e.HTTPStatus)
	}
	if e.GRPCStatus != 8 {
		t.Errorf("GRPCStatus: got %d", e.GRPCStatus)
	}
}

func TestExtra2_DeadlineExceeded(t *testing.T) {
	e := DeadlineExceeded("timeout")
	if e.Code != CodeDeadlineExceeded {
		t.Errorf("Code: got %s", e.Code)
	}
	if e.HTTPStatus != http.StatusGatewayTimeout {
		t.Errorf("HTTPStatus: got %d", e.HTTPStatus)
	}
	if e.GRPCStatus != 4 {
		t.Errorf("GRPCStatus: got %d", e.GRPCStatus)
	}
}

func TestExtra2_Unavailable(t *testing.T) {
	e := Unavailable("service down")
	if e.Code != CodeUnavailable {
		t.Errorf("Code: got %s", e.Code)
	}
	if e.HTTPStatus != http.StatusServiceUnavailable {
		t.Errorf("HTTPStatus: got %d", e.HTTPStatus)
	}
	if e.GRPCStatus != 14 {
		t.Errorf("GRPCStatus: got %d", e.GRPCStatus)
	}
}

func TestExtra2_Internal(t *testing.T) {
	e := Internal("internal error")
	if e.Code != CodeInternal {
		t.Errorf("Code: got %s", e.Code)
	}
	if e.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("HTTPStatus: got %d", e.HTTPStatus)
	}
	if e.GRPCStatus != 13 {
		t.Errorf("GRPCStatus: got %d", e.GRPCStatus)
	}
}

func TestExtra2_Wrap_Nil(t *testing.T) {
	e := Wrap(nil, "test", "test")
	if e != nil {
		t.Error("Wrap(nil) should return nil")
	}
}

func TestExtra2_Wrap_AlreadyAppError(t *testing.T) {
	original := NotFound("original")
	wrapped := Wrap(original, "internal", "should not change")
	if wrapped.Code != CodeNotFound {
		t.Error("Wrap should preserve existing AppError code")
	}
}

func TestExtra2_Wrap_RegularError(t *testing.T) {
	cause := errors.New("underlying")
	wrapped := Wrap(cause, CodeInternal, "internal error")
	if wrapped.Cause != cause {
		t.Error("wrapped cause should be preserved")
	}
	if wrapped.Code != CodeInternal {
		t.Error("wrapped code should be fallback")
	}
}

func TestExtra2_As_Nil(t *testing.T) {
	if As(nil) != nil {
		t.Error("As(nil) should return nil")
	}
}

func TestExtra2_As_RegularError(t *testing.T) {
	err := errors.New("regular error")
	if As(err) != nil {
		t.Error("As(regular error) should return nil")
	}
}

func TestExtra2_As_AppError(t *testing.T) {
	appErr := NotFound("user")
	err := As(appErr)
	if err == nil {
		t.Fatal("As(appErr) should not return nil")
	}
	if err.Code != CodeNotFound {
		t.Error("Code should be preserved")
	}
}

func TestExtra2_HTTPStatus_AppError(t *testing.T) {
	e := NotFound("test")
	if HTTPStatus(e) != http.StatusNotFound {
		t.Errorf("got %d", HTTPStatus(e))
	}
}

func TestExtra2_HTTPStatus_RegularError(t *testing.T) {
	if HTTPStatus(errors.New("test")) != http.StatusInternalServerError {
		t.Error("regular error should return 500")
	}
}

func TestExtra2_HTTPStatus_Nil(t *testing.T) {
	if HTTPStatus(nil) != http.StatusInternalServerError {
		t.Error("nil should return 500")
	}
}

func TestExtra2_GRPCStatus_AppError(t *testing.T) {
	e := NotFound("test")
	if GRPCStatus(e) != 5 {
		t.Errorf("got %d", GRPCStatus(e))
	}
}

func TestExtra2_GRPCStatus_RegularError(t *testing.T) {
	if GRPCStatus(errors.New("test")) != 2 { // Unknown
		t.Error("regular error should return 2 (Unknown)")
	}
}

func TestExtra2_Code_AppError(t *testing.T) {
	e := NotFound("test")
	if Code(e) != CodeNotFound {
		t.Errorf("got %s", Code(e))
	}
}

func TestExtra2_Code_RegularError(t *testing.T) {
	if Code(errors.New("test")) != CodeUnknown {
		t.Error("regular error should return CodeUnknown")
	}
}

func TestExtra2_AppError_Error_WithCause(t *testing.T) {
	cause := errors.New("underlying cause")
	e := &AppError{Code: "test", Message: "test message", Cause: cause}
	msg := e.Error()
	if msg == "" {
		t.Error("Error() should not return empty")
	}
}

func TestExtra2_AppError_Error_WithoutCause(t *testing.T) {
	e := &AppError{Code: "test", Message: "test message"}
	msg := e.Error()
	if msg == "" {
		t.Error("Error() should not return empty")
	}
}

func TestExtra2_WithDetail_Chaining(t *testing.T) {
	e := NotFound("user").
		WithDetail("user_id", "123").
		WithDetail("tenant_id", "t1")
	
	if len(e.Details) != 2 {
		t.Errorf("got %d details", len(e.Details))
	}
	if e.Details["user_id"] != "123" {
		t.Error("user_id mismatch")
	}
}

func TestExtra2_WithDetails_Replaces(t *testing.T) {
	e := NotFound("user").
		WithDetail("a", "1").
		WithDetails(map[string]any{"b": "2"})
	
	if len(e.Details) != 1 {
		t.Errorf("got %d details", len(e.Details))
	}
	if e.Details["a"] != nil {
		t.Error("old detail should be replaced")
	}
	if e.Details["b"] != "2" {
		t.Error("new detail not set")
	}
}

func TestExtra2_Conflict(t *testing.T) {
	// conflict is unexported, but we can test via the CodeConflict constant
	if CodeConflict == "" {
		t.Error("CodeConflict should not be empty")
	}
}

func TestExtra2_AppError_Is_DifferentCodes(t *testing.T) {
	e1 := NotFound("a")
	e2 := AlreadyExists("b")
	if errors.Is(e1, e2) {
		t.Error("different codes should not match")
	}
}

func TestExtra2_AppError_Is_OtherNil(t *testing.T) {
	e := NotFound("a")
	if errors.Is(e, nil) {
		t.Error("should not match nil")
	}
}
