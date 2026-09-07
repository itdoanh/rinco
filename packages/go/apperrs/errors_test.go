package apperrs

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstructors(t *testing.T) {
	cases := []struct {
		name           string
		err            *AppError
		wantCode       string
		wantHTTPStatus int
		wantGRPCStatus int
	}{
		{"InvalidArgument", InvalidArgument("bad"), CodeInvalidArgument, 400, 3},
		{"Validation", Validation("nope"), CodeInvalidArgument, 400, 3},
		{"Unauthenticated", Unauthenticated("nope"), CodeUnauthenticated, 401, 16},
		{"PermissionDenied", PermissionDenied("nope"), CodePermissionDenied, 403, 7},
		{"NotFound", NotFound("nope"), CodeNotFound, 404, 5},
		{"AlreadyExists", AlreadyExists("nope"), CodeAlreadyExists, 409, 6},
		{"Conflict", Conflict("nope"), CodeConflict, 409, 10},
		{"RateLimited", RateLimited("nope"), CodeRateLimited, 429, 8},
		{"DeadlineExceeded", DeadlineExceeded("nope"), CodeDeadlineExceeded, 504, 4},
		{"Unavailable", Unavailable("nope"), CodeUnavailable, 503, 14},
		{"Internal", Internal("nope"), CodeInternal, 500, 13},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.wantCode, c.err.Code)
			assert.Equal(t, c.wantHTTPStatus, c.err.HTTPStatus)
			assert.Equal(t, c.wantGRPCStatus, c.err.GRPCStatus)
			assert.NotEmpty(t, c.err.Message)
		})
	}
}

func TestWithDetailAndDetails(t *testing.T) {
	e := NotFound("missing").WithDetail("user_id", "abc").WithDetail("tenant_id", "t1")
	require.Len(t, e.Details, 2)
	assert.Equal(t, "abc", e.Details["user_id"])

	e2 := NotFound("missing").WithDetails(map[string]any{"a": 1, "b": 2})
	require.Len(t, e2.Details, 2)
}

func TestWithCauseUnwrap(t *testing.T) {
	root := stderrors.New("boom")
	e := Internal("wrapped").WithCause(root)
	assert.True(t, stderrors.Is(e, root))

	var appE *AppError
	require.True(t, stderrors.As(e, &appE))
	assert.Equal(t, CodeInternal, appE.Code)
}

func TestIsMatchByCode(t *testing.T) {
	a := NotFound("a")
	b := NotFound("b")
	assert.True(t, stderrors.Is(a, b))
	assert.False(t, stderrors.Is(a, Unauthenticated("c")))
}

func TestWrap(t *testing.T) {
	// nil cause → nil error
	assert.Nil(t, Wrap(nil, CodeInternal, "x"))

	// already an AppError → returned as-is
	orig := NotFound("nope")
	wrapped := Wrap(orig, CodeInternal, "x")
	assert.Same(t, orig, wrapped)

	// generic error → wrapped with fallback
	plain := stderrors.New("plain")
	wrapped = Wrap(plain, CodeInternal, "internal")
	assert.Equal(t, CodeInternal, wrapped.Code)
	assert.True(t, stderrors.Is(wrapped, plain))
}

func TestHelpers(t *testing.T) {
	assert.Nil(t, As(nil))
	assert.Nil(t, As(stderrors.New("plain")))

	e := NotFound("nope")
	assert.Same(t, e, As(e))

	assert.Equal(t, 404, HTTPStatus(e))
	assert.Equal(t, 500, HTTPStatus(stderrors.New("plain")))
	assert.Equal(t, 5, GRPCStatus(e))
	assert.Equal(t, 2, GRPCStatus(stderrors.New("plain")))

	assert.Equal(t, CodeNotFound, Code(e))
	assert.Equal(t, CodeUnknown, Code(stderrors.New("plain")))
}

func TestWithMessageAndStack(t *testing.T) {
	e := NotFound("a").WithMessage("b").WithStack("stack-trace")
	assert.Equal(t, "b", e.Message)
	assert.Equal(t, "stack-trace", e.Stack)
}

func TestErrorString(t *testing.T) {
	plain := NotFound("missing")
	assert.Contains(t, plain.Error(), CodeNotFound)
	assert.Contains(t, plain.Error(), "missing")

	withCause := Internal("oops").WithCause(stderrors.New("boom"))
	s := withCause.Error()
	assert.Contains(t, s, "boom")
	assert.Contains(t, s, "oops")
}

func TestHTTPStatusForEdgeCases(t *testing.T) {
	// Wrap with HTTPStatus helper that respects chain - wrapped plain error uses fallback 500
	wrapped := Wrap(fmt.Errorf("network: %w", stderrors.New("refused")), CodeUnavailable, "downstream")
	assert.Equal(t, CodeUnavailable, wrapped.Code)
	assert.Equal(t, http.StatusInternalServerError, wrapped.HTTPStatus) // fallback for plain error
	assert.Equal(t, 13, wrapped.GRPCStatus)
}