// Unit tests for auth-service. These tests do NOT require a database
// or Valkey — they cover preflight validation, password hashing, and
// PASETO round-trip.
package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func newTestAuthServer() *Server {
	return NewServer(nil, nil)
}

// ============================================================================
// Email validation
// ============================================================================

func TestValidEmail_Accepts(t *testing.T) {
	good := []string{
		"a@b.co",
		"john.doe@apex-fintech.vn",
		"user+tag@example.com",
		"x_y@sub.domain.io",
	}
	for _, e := range good {
		t.Run(e, func(t *testing.T) {
			assert.NoError(t, validEmail(e))
		})
	}
}

func TestValidEmail_Rejects(t *testing.T) {
	bad := []string{
		"",
		"not-an-email",
		"@nodomain.com",
		"missing-at.com",
		"spaces in@email.com",
		"a@b",
		"a@b.c",       // TLD too short
		strings.Repeat("a", 250) + "@x.com", // too long
	}
	for _, e := range bad {
		t.Run(e, func(t *testing.T) {
			assert.Error(t, validEmail(e))
		})
	}
}

// ============================================================================
// Password validation
// ============================================================================

func TestValidPassword_Accepts(t *testing.T) {
	good := []string{
		"Password123",
		"abc12345",
		"Th!s1sStrong",
	}
	for _, p := range good {
		t.Run(p, func(t *testing.T) {
			assert.NoError(t, validPassword(p))
		})
	}
}

func TestValidPassword_Rejects(t *testing.T) {
	bad := []string{
		"",
		"short1",      // too short
		"allletters",  // no digit
		"12345678",    // no letter
		strings.Repeat("a", 200), // too long
	}
	for _, p := range bad {
		t.Run(p, func(t *testing.T) {
			assert.Error(t, validPassword(p))
		})
	}
}

// ============================================================================
// Argon2id hash + verify round trip
// ============================================================================

func TestPasswordHash_RoundTrip(t *testing.T) {
	passwords := []string{
		"CorrectHorseBatteryStaple1",
		"aA1!aA1!",
		strings.Repeat("P", 100) + "1",
	}
	for _, p := range passwords {
		t.Run(p[:8], func(t *testing.T) {
			h, err := hashPassword(p)
			assert.NoError(t, err)
			assert.NotEmpty(t, h)
			assert.True(t, strings.HasPrefix(h, "$argon2id$"),
				"hash must use argon2id variant")

			ok, err := verifyPassword(h, p)
			assert.NoError(t, err)
			assert.True(t, ok, "correct password must verify")
		})
	}
}

func TestPasswordHash_WrongPasswordRejected(t *testing.T) {
	h, err := hashPassword("CorrectPassword1")
	assert.NoError(t, err)
	ok, err := verifyPassword(h, "WrongPassword1")
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestPasswordHash_SaltsAreUnique(t *testing.T) {
	h1, _ := hashPassword("samePassword1")
	h2, _ := hashPassword("samePassword1")
	assert.NotEqual(t, h1, h2,
		"two hashes of the same password must differ (random salt)")
}

func TestPasswordHash_ConstantTimeVerify(t *testing.T) {
	// Constant-time comparison is critical. We can't directly test timing,
	// but we can at least assert that verifyPassword never returns true for
	// tampered encoded strings.
	h, _ := hashPassword("CorrectPassword1")
	ok, err := verifyPassword(h[:len(h)-2]+"AA", "CorrectPassword1")
	assert.NoError(t, err)
	assert.False(t, ok)
}

// ============================================================================
// PASETO token round-trip
// ============================================================================

func TestIssueAndVerifyAccessToken_Success(t *testing.T) {
	s := newTestAuthServer()
	tok, exp, err := s.issueAccessToken("u-1", "t-1", "agent")
	assert.NoError(t, err)
	assert.NotEmpty(t, tok)
	assert.Greater(t, exp, 0)

	claims, err := s.verifyAccessToken(tok)
	assert.NoError(t, err)
	assert.Equal(t, "u-1", claims.UserID)
	assert.Equal(t, "t-1", claims.TenantID)
	assert.Equal(t, "agent", claims.Role)
}

func TestIssueAndVerifyAccessToken_TamperedFails(t *testing.T) {
	s := newTestAuthServer()
	tok, _, err := s.issueAccessToken("u-1", "t-1", "agent")
	assert.NoError(t, err)

	// Flip one byte of the token.
	tampered := tok[:10] + "X" + tok[11:]
	_, err = s.verifyAccessToken(tampered)
	assert.Error(t, err, "tampered tokens must be rejected")
}

func TestIssueAndVerifyAccessToken_DifferentKeyFails(t *testing.T) {
	s1 := newTestAuthServer()
	s2 := newTestAuthServer() // random key

	tok, _, err := s1.issueAccessToken("u-1", "t-1", "agent")
	assert.NoError(t, err)

	_, err = s2.verifyAccessToken(tok)
	assert.Error(t, err, "tokens from one key must not verify with another")
}

func TestGenerateRefreshToken_Unique(t *testing.T) {
	t1, err := generateRefreshToken()
	assert.NoError(t, err)
	t2, err := generateRefreshToken()
	assert.NoError(t, err)
	assert.NotEqual(t, t1, t2)
	assert.Len(t, t1, 64, "refresh tokens must be 256-bit hex (64 chars)")
}

func TestGenerateRefreshToken_HasHighEntropy(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		tok, err := generateRefreshToken()
		assert.NoError(t, err)
		assert.False(t, seen[tok], "duplicate refresh token after %d iterations", i)
		seen[tok] = true
	}
}

// ============================================================================
// HTTP handler shape tests
// ============================================================================

func newCtx(method, path, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestLogin_BadJSON_Returns400(t *testing.T) {
	s := newTestAuthServer()
	c, rec := newCtx(http.MethodPost, "/auth/login", "not json")
	err := s.Login(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_MissingEmail_Returns400(t *testing.T) {
	s := newTestAuthServer()
	c, rec := newCtx(http.MethodPost, "/auth/login", `{"password":"x"}`)
	err := s.Login(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_MissingPassword_Returns400(t *testing.T) {
	s := newTestAuthServer()
	c, rec := newCtx(http.MethodPost, "/auth/login", `{"email":"a@b.co"}`)
	err := s.Login(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_ShortPassword_Returns400(t *testing.T) {
	s := newTestAuthServer()
	c, rec := newCtx(http.MethodPost, "/auth/register",
		`{"email":"a@b.co","password":"short","full_name":"John"}`)
	err := s.Register(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_NoDigitPassword_Returns400(t *testing.T) {
	s := newTestAuthServer()
	c, rec := newCtx(http.MethodPost, "/auth/register",
		`{"email":"a@b.co","password":"onlyletters","full_name":"John"}`)
	err := s.Register(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_MissingFullName_Returns400(t *testing.T) {
	s := newTestAuthServer()
	c, rec := newCtx(http.MethodPost, "/auth/register",
		`{"email":"a@b.co","password":"Good1Password"}`)
	err := s.Register(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRefresh_EmptyToken_Returns400(t *testing.T) {
	s := newTestAuthServer()
	c, rec := newCtx(http.MethodPost, "/auth/refresh", `{}`)
	err := s.Refresh(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogout_Returns204(t *testing.T) {
	s := newTestAuthServer()
	c, rec := newCtx(http.MethodPost, "/auth/logout", "")
	err := s.Logout(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}
