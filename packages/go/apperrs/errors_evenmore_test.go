package apperrs

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestCodesDefined(t *testing.T) {
	codes := []string{
		CodeUnknown, CodeInvalidArgument, CodeValidation,
		CodeUnauthenticated, CodePermissionDenied, CodeNotFound,
		CodeAlreadyExists, CodeConflict, CodeRateLimited,
		CodeDeadlineExceeded, CodeUnavailable, CodeInternal,
	}
	if len(codes) != 12 {
		t.Errorf("expected 12 codes, got %d", len(codes))
	}
}

func TestErrorWithoutCause(t *testing.T) {
	e := InvalidArgument("test msg")
	if !errors.Is(e, e) {
		t.Error("Is should match itself")
	}
	if e.Error() == "" {
		t.Error("non-empty error message")
	}
}

func TestErrorWithCauseUnwrap(t *testing.T) {
	base := errors.New("base")
	e := Internal("oops").WithCause(base)
	if e.Cause != base {
		t.Error("cause not stored")
	}
	if errors.Unwrap(e) != base {
		t.Error("Unwrap should return base")
	}
}

func TestErrorMessageFormat(t *testing.T) {
	base := errors.New("base")
	e := Internal("msg").WithCause(base)
	got := e.Error()
	if got == "" {
		t.Fatal("empty")
	}
	if !contains(got, "internal: msg") {
		t.Errorf("missing code:msg in %q", got)
	}
	if !contains(got, "base") {
		t.Errorf("missing cause in %q", got)
	}
}

func TestIsByCode(t *testing.T) {
	e1 := NotFound("a")
	e2 := NotFound("b")
	e3 := Internal("c")
	if !errors.Is(e1, e2) {
		t.Error("same code should match")
	}
	if errors.Is(e1, e3) {
		t.Error("different code should not match")
	}
}

func TestWithDetail(t *testing.T) {
	e := NotFound("nf").WithDetail("id", "abc")
	if e.Details["id"] != "abc" {
		t.Error("detail not added")
	}
}

func TestWithDetailsBulk(t *testing.T) {
	details := map[string]any{
		"a": 1,
		"b": "two",
	}
	e := NotFound("nf").WithDetails(details)
	if e.Details["a"] != 1 || e.Details["b"] != "two" {
		t.Error("details not replaced")
	}
}

func TestWithStack(t *testing.T) {
	e := Internal("x").WithStack("at line 1")
	if e.Stack == "" {
		t.Error("stack not set")
	}
}

func TestWithMessage(t *testing.T) {
	e := Internal("original").WithMessage("updated")
	if e.Message != "updated" {
		t.Errorf("message = %q", e.Message)
	}
}

func TestConstructorStatusCodes(t *testing.T) {
	tests := []struct {
		ctor  func(string) *AppError
		http  int
		grpc  int
	}{
		{InvalidArgument, 400, 3},
		{Unauthenticated, 401, 16},
		{PermissionDenied, 403, 7},
		{NotFound, 404, 5},
		{AlreadyExists, 409, 6},
		{Conflict, 409, 10},
		{RateLimited, 429, 8},
		{DeadlineExceeded, 504, 4},
		{Unavailable, 503, 14},
		{Internal, 500, 13},
	}
	for _, tc := range tests {
		e := tc.ctor("msg")
		if e.HTTPStatus != tc.http {
			t.Errorf("%T http = %d, want %d", tc.ctor, e.HTTPStatus, tc.http)
		}
		if e.GRPCStatus != tc.grpc {
			t.Errorf("%T grpc = %d, want %d", tc.ctor, e.GRPCStatus, tc.grpc)
		}
	}
}

func TestValidationAlias(t *testing.T) {
	v := Validation("form error")
	if v.HTTPStatus != http.StatusBadRequest {
		t.Errorf("http = %d", v.HTTPStatus)
	}
}

func TestWrapNil(t *testing.T) {
	if Wrap(nil, "x", "y") != nil {
		t.Error("Wrap(nil) should be nil")
	}
}

func TestWrapAppError(t *testing.T) {
	original := NotFound("orig")
	wrapped := Wrap(original, "different", "msg")
	if wrapped != original {
		t.Error("should return same AppError")
	}
}

func TestWrapGenericError(t *testing.T) {
	original := errors.New("base")
	wrapped := Wrap(original, "code", "msg")
	if wrapped.Cause != original {
		t.Error("cause should be set")
	}
	if wrapped.Code != "code" {
		t.Errorf("code = %q", wrapped.Code)
	}
}

func TestAsNil(t *testing.T) {
	if As(nil) != nil {
		t.Error("As(nil) should be nil")
	}
}

func TestAsExtracts(t *testing.T) {
	e := NotFound("x")
	if As(e) != e {
		t.Error("As should extract same ptr")
	}
}

func TestAsGenericError(t *testing.T) {
	e := errors.New("plain")
	if As(e) != nil {
		t.Error("As should not extract plain error")
	}
}

func TestHTTPStatusFunction(t *testing.T) {
	if HTTPStatus(NotFound("x")) != 404 {
		t.Error("HTTPStatus(NotFound) should be 404")
	}
	if HTTPStatus(errors.New("plain")) != 500 {
		t.Error("plain should be 500")
	}
	if HTTPStatus(nil) != 500 {
		t.Error("nil should be 500")
	}
}

func TestGRPCStatusFunction(t *testing.T) {
	if GRPCStatus(NotFound("x")) != 5 {
		t.Error("GRPCStatus(NotFound) should be 5")
	}
	if GRPCStatus(errors.New("plain")) != 2 {
		t.Error("plain should be Unknown 2")
	}
	if GRPCStatus(nil) != 2 {
		t.Error("nil should be 2")
	}
}

func TestCodeFunction(t *testing.T) {
	if Code(NotFound("x")) != CodeNotFound {
		t.Errorf("code = %q", Code(NotFound("x")))
	}
	if Code(errors.New("plain")) != CodeUnknown {
		t.Error("plain should be unknown")
	}
	if Code(nil) != CodeUnknown {
		t.Error("nil should be unknown")
	}
}

func TestIsMatchingNonAppError(t *testing.T) {
	e := NotFound("x")
	plain := errors.New("plain")
	if e.Is(plain) {
		t.Error("should not match non-AppError")
	}
}

func TestChainWrapping(t *testing.T) {
	base := errors.New("db")
	cause := fmt.Errorf("query: %w", base)
	appErr := Internal("db failure").WithCause(cause)
	if !errors.Is(appErr, base) {
		t.Error("errors.Is should find base in chain")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
