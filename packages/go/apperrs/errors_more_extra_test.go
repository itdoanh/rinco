// Extra tests for apperrs package.
package apperrs

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestCodeConstants(t *testing.T) {
	codes := map[string]string{
		CodeUnknown:          "unknown",
		CodeInvalidArgument:  "invalid_argument",
		CodeValidation:       "validation_failed",
		CodeUnauthenticated:  "unauthenticated",
		CodePermissionDenied: "permission_denied",
		CodeNotFound:         "not_found",
		CodeAlreadyExists:    "already_exists",
		CodeConflict:         "conflict",
		CodeRateLimited:      "rate_limited",
		CodeDeadlineExceeded: "deadline_exceeded",
		CodeUnavailable:      "unavailable",
		CodeInternal:         "internal",
	}
	for code, expected := range codes {
		if code != expected {
			t.Errorf("got %s, want %s", code, expected)
		}
	}
}

func TestAppError_Error_WithCause(t *testing.T) {
	cause := errors.New("root cause")
	e := Internal("operation failed").WithCause(cause)
	msg := e.Error()
	if msg == "" {
		t.Error("empty message")
	}
	// Should contain both code, message, and cause
	if !contains(msg, "internal") {
		t.Error("missing code")
	}
	if !contains(msg, "operation failed") {
		t.Error("missing message")
	}
	if !contains(msg, "root cause") {
		t.Error("missing cause")
	}
}

func TestAppError_Error_NoCause(t *testing.T) {
	e := NotFound("user not found")
	msg := e.Error()
	if msg == "" {
		t.Error("empty message")
	}
	if !contains(msg, "not_found") {
		t.Error("missing code")
	}
	if !contains(msg, "user not found") {
		t.Error("missing message")
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("root")
	e := Internal("op").WithCause(cause)
	if e.Unwrap() != cause {
		t.Error("Unwrap should return cause")
	}
}

func TestAppError_Is_MatchCode(t *testing.T) {
	e1 := NotFound("user 1")
	e2 := NotFound("user 2")
	if !errors.Is(e1, e2) {
		t.Error("same code should match")
	}
}

func TestAppError_Is_NoMatchDifferentCode(t *testing.T) {
	e1 := NotFound("x")
	e2 := Internal("y")
	if errors.Is(e1, e2) {
		t.Error("different codes should not match")
	}
}

func TestAppError_WithDetail(t *testing.T) {
	e := InvalidArgument("bad input").WithDetail("field", "name")
	if e.Details["field"] != "name" {
		t.Error("detail not set")
	}
}

func TestAppError_WithDetails(t *testing.T) {
	e := InvalidArgument("bad").WithDetails(map[string]any{"a": 1, "b": "two"})
	if e.Details["a"] != 1 {
		t.Error("detail a")
	}
	if e.Details["b"] != "two" {
		t.Error("detail b")
	}
}

func TestAppError_WithStack(t *testing.T) {
	e := Internal("error").WithStack("at func1\nat func2")
	if e.Stack == "" {
		t.Error("stack not set")
	}
}

func TestAppError_WithMessage(t *testing.T) {
	e := Internal("original").WithMessage("overridden")
	if e.Message != "overridden" {
		t.Error("message not overridden")
	}
}

func TestNewErr(t *testing.T) {
	e := newErr("test_code", "test_msg", 418, 13)
	if e.Code != "test_code" {
		t.Error("code")
	}
	if e.HTTPStatus != 418 {
		t.Errorf("http: %d", e.HTTPStatus)
	}
	if e.GRPCStatus != 13 {
		t.Errorf("grpc: %d", e.GRPCStatus)
	}
}

func TestInvalidArgument(t *testing.T) {
	e := InvalidArgument("bad")
	if e.HTTPStatus != http.StatusBadRequest {
		t.Errorf("http: %d", e.HTTPStatus)
	}
	if e.Code != CodeInvalidArgument {
		t.Error("code")
	}
}

func TestValidation(t *testing.T) {
	e := Validation("field required")
	if e.HTTPStatus != http.StatusBadRequest {
		t.Errorf("http: %d", e.HTTPStatus)
	}
}

func TestUnauthenticated(t *testing.T) {
	e := Unauthenticated("login required")
	if e.HTTPStatus != http.StatusUnauthorized {
		t.Errorf("http: %d", e.HTTPStatus)
	}
}

func TestPermissionDenied(t *testing.T) {
	e := PermissionDenied("forbidden")
	if e.HTTPStatus != http.StatusForbidden {
		t.Errorf("http: %d", e.HTTPStatus)
	}
}

func TestNotFound(t *testing.T) {
	e := NotFound("not here")
	if e.HTTPStatus != http.StatusNotFound {
		t.Errorf("http: %d", e.HTTPStatus)
	}
}

func TestAlreadyExists(t *testing.T) {
	e := AlreadyExists("dup")
	if e.HTTPStatus != http.StatusConflict {
		t.Errorf("http: %d", e.HTTPStatus)
	}
}

func TestConflict(t *testing.T) {
	e := Conflict("conflict")
	if e.HTTPStatus != http.StatusConflict {
		t.Errorf("http: %d", e.HTTPStatus)
	}
}

func TestRateLimited(t *testing.T) {
	e := RateLimited("slow down")
	if e.HTTPStatus != http.StatusTooManyRequests {
		t.Errorf("http: %d", e.HTTPStatus)
	}
}

func TestDeadlineExceeded(t *testing.T) {
	e := DeadlineExceeded("timeout")
	if e.HTTPStatus != http.StatusGatewayTimeout {
		t.Errorf("http: %d", e.HTTPStatus)
	}
}

func TestUnavailable(t *testing.T) {
	e := Unavailable("down")
	if e.HTTPStatus != http.StatusServiceUnavailable {
		t.Errorf("http: %d", e.HTTPStatus)
	}
}

func TestInternal(t *testing.T) {
	e := Internal("oops")
	if e.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("http: %d", e.HTTPStatus)
	}
}

func TestWrap_Nil(t *testing.T) {
	if Wrap(nil, "x", "y") != nil {
		t.Error("wrap nil should return nil")
	}
}

func TestWrap_ExistingAppError(t *testing.T) {
	original := NotFound("user")
	wrapped := Wrap(original, "other_code", "other_msg")
	if wrapped != original {
		t.Error("existing AppError should pass through")
	}
}

func TestWrap_PlainError(t *testing.T) {
	err := errors.New("plain")
	wrapped := Wrap(err, "internal", "oops")
	if wrapped == nil {
		t.Fatal("nil wrap")
	}
	if wrapped.Cause != err {
		t.Error("cause not set")
	}
	if wrapped.Code != "internal" {
		t.Error("code")
	}
}

func TestAs_Nil(t *testing.T) {
	if As(nil) != nil {
		t.Error("As(nil) should return nil")
	}
}

func TestAs_AppError(t *testing.T) {
	e := NotFound("x")
	got := As(e)
	if got != e {
		t.Error("As should return original")
	}
}

func TestAs_WrappedAppError(t *testing.T) {
	e := NotFound("x")
	wrapped := fmt.Errorf("wrapped: %w", e)
	got := As(wrapped)
	if got != e {
		t.Error("As should unwrap")
	}
}

func TestAs_PlainError(t *testing.T) {
	err := errors.New("plain")
	if As(err) != nil {
		t.Error("As of plain error should return nil")
	}
}

func TestHTTPStatus_AppError(t *testing.T) {
	if HTTPStatus(NotFound("x")) != http.StatusNotFound {
		t.Error("status mismatch")
	}
}

func TestHTTPStatus_PlainError(t *testing.T) {
	if HTTPStatus(errors.New("plain")) != http.StatusInternalServerError {
		t.Error("plain error should return 500")
	}
}

func TestHTTPStatus_Nil(t *testing.T) {
	if HTTPStatus(nil) != http.StatusInternalServerError {
		t.Error("nil should return 500")
	}
}

func TestGRPCStatus_AppError(t *testing.T) {
	if GRPCStatus(NotFound("x")) != 5 {
		t.Errorf("got %d", GRPCStatus(NotFound("x")))
	}
}

func TestGRPCStatus_PlainError(t *testing.T) {
	if GRPCStatus(errors.New("plain")) != 2 {
		t.Error("plain error should return Unknown (2)")
	}
}

func TestCode_AppError(t *testing.T) {
	if Code(NotFound("x")) != CodeNotFound {
		t.Error("code mismatch")
	}
}

func TestCode_PlainError(t *testing.T) {
	if Code(errors.New("plain")) != CodeUnknown {
		t.Error("plain error should return CodeUnknown")
	}
}

func TestCode_Nil(t *testing.T) {
	if Code(nil) != CodeUnknown {
		t.Error("nil should return CodeUnknown")
	}
}

func containsMy(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
