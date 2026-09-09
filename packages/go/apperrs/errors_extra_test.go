// Additional tests for apperrs package.
package apperrs

import (
	"errors"
	"testing"
)

func TestExtra_Error_WithCause(t *testing.T) {
	cause := errors.New("underlying")
	e := Internal("server error").WithCause(cause)
	if e.Cause != cause {
		t.Error("cause not set")
	}
	if e.Error() == "" {
		t.Error("error message empty")
	}
}

func TestExtra_Error_WithDetails(t *testing.T) {
	e := Validation("invalid").WithDetail("field", "email").WithDetail("value", "bad")
	if len(e.Details) != 2 {
		t.Errorf("details: got %d", len(e.Details))
	}
}

func TestExtra_Error_WithDetailsMap(t *testing.T) {
	details := map[string]any{"a": 1, "b": "two"}
	e := InvalidArgument("bad").WithDetails(details)
	if e.Details["a"] != 1 {
		t.Error("detail a mismatch")
	}
}

func TestExtra_Error_WithStack(t *testing.T) {
	e := Internal("err").WithStack("main.go:42")
	if e.Stack != "main.go:42" {
		t.Errorf("stack: got %s", e.Stack)
	}
}

func TestExtra_Error_WithMessage(t *testing.T) {
	e := Internal("first").WithMessage("second")
	if e.Message != "second" {
		t.Errorf("message: got %s", e.Message)
	}
}

func TestExtra_Error_Unwrap(t *testing.T) {
	cause := errors.New("underlying")
	e := Internal("err").WithCause(cause)
	if e.Unwrap() != cause {
		t.Error("Unwrap should return cause")
	}
}

func TestExtra_Error_Is_SameCode(t *testing.T) {
	e1 := NotFound("a")
	e2 := NotFound("b")
	if !errors.Is(e1, e2) {
		t.Error("same code should match")
	}
}

func TestExtra_Error_Is_DiffCode(t *testing.T) {
	e1 := NotFound("a")
	e2 := Internal("b")
	if errors.Is(e1, e2) {
		t.Error("different codes should not match")
	}
}

func TestExtra_HTTPStatus_AppError(t *testing.T) {
	if got := HTTPStatus(NotFound("x")); got != 404 {
		t.Errorf("got %d", got)
	}
}

func TestExtra_HTTPStatus_PlainError(t *testing.T) {
	if got := HTTPStatus(errors.New("plain")); got != 500 {
		t.Errorf("got %d", got)
	}
}

func TestExtra_HTTPStatus_Nil(t *testing.T) {
	if got := HTTPStatus(nil); got != 500 {
		t.Errorf("nil should return 500: got %d", got)
	}
}

func TestExtra_GRPCStatus_AppError(t *testing.T) {
	if got := GRPCStatus(PermissionDenied("x")); got != 7 {
		t.Errorf("got %d", got)
	}
}

func TestExtra_GRPCStatus_PlainError(t *testing.T) {
	if got := GRPCStatus(errors.New("plain")); got != 2 {
		t.Errorf("got %d", got)
	}
}

func TestExtra_Code_AppError(t *testing.T) {
	if got := Code(AlreadyExists("x")); got != CodeAlreadyExists {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Code_PlainError(t *testing.T) {
	if got := Code(errors.New("plain")); got != CodeUnknown {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Code_Nil(t *testing.T) {
	if got := Code(nil); got != CodeUnknown {
		t.Errorf("nil should be unknown: got %s", got)
	}
}

func TestExtra_As_AppError(t *testing.T) {
	e := Internal("x")
	got := As(e)
	if got == nil {
		t.Fatal("As should return AppError")
	}
	if got.Code != CodeInternal {
		t.Errorf("code: got %s", got.Code)
	}
}

func TestExtra_As_PlainError(t *testing.T) {
	got := As(errors.New("plain"))
	if got != nil {
		t.Error("plain error should not be AppError")
	}
}

func TestExtra_As_Nil(t *testing.T) {
	if got := As(nil); got != nil {
		t.Error("nil should return nil")
	}
}

func TestExtra_Wrap_PlainError(t *testing.T) {
	cause := errors.New("underlying")
	e := Wrap(cause, CodeInternal, "wrapped")
	if e.Cause != cause {
		t.Error("cause not set")
	}
}

func TestExtra_Wrap_AppError(t *testing.T) {
	original := NotFound("original")
	wrapped := Wrap(original, CodeInternal, "wrapped")
	if wrapped != original {
		t.Error("AppError should be returned as-is")
	}
}

func TestExtra_Wrap_Nil(t *testing.T) {
	if got := Wrap(nil, CodeInternal, "msg"); got != nil {
		t.Error("nil cause should return nil")
	}
}

func TestExtra_AllConstructors_HaveCorrectStatus(t *testing.T) {
	tests := []struct {
		err       *AppError
		httpWant  int
		grpcWant  int
	}{
		{InvalidArgument("x"), 400, 3},
		{Unauthenticated("x"), 401, 16},
		{PermissionDenied("x"), 403, 7},
		{NotFound("x"), 404, 5},
		{AlreadyExists("x"), 409, 6},
		{Conflict("x"), 409, 10},
		{RateLimited("x"), 429, 8},
		{DeadlineExceeded("x"), 504, 4},
		{Unavailable("x"), 503, 14},
		{Internal("x"), 500, 13},
	}
	for _, tt := range tests {
		if tt.err.HTTPStatus != tt.httpWant {
			t.Errorf("%s: HTTP got %d, want %d", tt.err.Code, tt.err.HTTPStatus, tt.httpWant)
		}
		if tt.err.GRPCStatus != tt.grpcWant {
			t.Errorf("%s: gRPC got %d, want %d", tt.err.Code, tt.err.GRPCStatus, tt.grpcWant)
		}
	}
}

func TestExtra_Validation_Constructor(t *testing.T) {
	v := Validation("bad")
	if v.HTTPStatus != 400 {
		t.Errorf("HTTP: got %d", v.HTTPStatus)
	}
}

func TestExtra_Error_NilDetails(t *testing.T) {
	e := InvalidArgument("x")
	if e.Details != nil {
		t.Error("Details should be nil by default")
	}
	// Add detail
	e.WithDetail("k", "v")
	if e.Details == nil {
		t.Error("Details should be initialized after WithDetail")
	}
}
