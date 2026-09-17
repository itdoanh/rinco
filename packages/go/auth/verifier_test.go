// Tests for the shared PASETO verifier (verifier.go).
//
// We deliberately stay in the same package so we can mint test tokens with
// the existing Paseto / KeyRing helpers without exporting anything new.
package auth

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	// Two distinct 32-byte hex keys used across the suite.  Generated
	// once and reused so the tests are deterministic.
	testCurrentHex  = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	testPreviousHex = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	testKeyringKid1 = "current"
	testKeyringKid2 = "previous"
)

// mintAccessToken produces a signed access token with the given claims.
// We keep this in the test file rather than paseto_extra_test.go so that
// anyone reading verifier.go can see exactly what shape the verifier
// expects.
func mintAccessToken(t *testing.T, keyHex, kid string, claims Claims) string {
	t.Helper()
	p, err := NewPaseto(keyHex, kid)
	if err != nil {
		t.Fatalf("mint: new paseto: %v", err)
	}
	tok, err := p.Encrypt(claims)
	if err != nil {
		t.Fatalf("mint: encrypt: %v", err)
	}
	return tok
}

func TestVerifier_HappyPath(t *testing.T) {
	v, err := NewVerifier(testCurrentHex, "")
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	tok := mintAccessToken(t, testCurrentHex, testKeyringKid1, Claims{
		Subject:   "user-1",
		UserID:    "user-1",
		TenantID:  "tenant-42",
		Roles:     []string{"tenant_admin"},
		Issuer:    "rinco",
		Audience:  "rinco-app",
		IssuedAt:  time.Now().Add(-time.Minute),
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})

	got, err := v.VerifyAccessToken(tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.UserID != "user-1" {
		t.Errorf("UserID: want user-1, got %q", got.UserID)
	}
	if got.TenantID != "tenant-42" {
		t.Errorf("TenantID: want tenant-42, got %q", got.TenantID)
	}
	if got.Role != "tenant_admin" {
		t.Errorf("Role: want tenant_admin, got %q", got.Role)
	}
	if got.Issuer != "rinco" {
		t.Errorf("Issuer: want rinco, got %q", got.Issuer)
	}
	if got.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be populated")
	}
}

func TestVerifier_StripsBearerPrefix(t *testing.T) {
	v, err := NewVerifier(testCurrentHex, "")
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}
	tok := mintAccessToken(t, testCurrentHex, "k", Claims{
		Subject:   "u1",
		TenantID:  "t1",
		ExpiresAt: time.Now().Add(time.Minute),
	})

	got, err := v.VerifyAccessToken("Bearer " + tok)
	if err != nil {
		t.Fatalf("verify with bearer prefix: %v", err)
	}
	if got.UserID != "u1" {
		t.Errorf("UserID: %q", got.UserID)
	}
	// Case-insensitive prefix should also work.
	if _, err := v.VerifyAccessToken("bearer " + tok); err != nil {
		t.Errorf("lowercase bearer should be accepted: %v", err)
	}
}

func TestVerifier_RotatedKey(t *testing.T) {
	v, err := NewVerifier(testCurrentHex, testPreviousHex)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	// Token signed with the *previous* key (still inside the rotation
	// window) must verify.
	tok := mintAccessToken(t, testPreviousHex, testKeyringKid2, Claims{
		Subject:   "u1-old",
		TenantID:  "t1",
		ExpiresAt: time.Now().Add(time.Minute),
	})
	got, err := v.VerifyAccessToken(tok)
	if err != nil {
		t.Fatalf("verify rotated key: %v", err)
	}
	if got.UserID != "u1-old" {
		t.Errorf("expected u1-old, got %q", got.UserID)
	}
}

func TestVerifier_RejectsUnknownKey(t *testing.T) {
	v, err := NewVerifier(testCurrentHex, testPreviousHex)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	rogueHex := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	tok := mintAccessToken(t, rogueHex, "rogue", Claims{
		Subject:   "evil",
		TenantID:  "t1",
		ExpiresAt: time.Now().Add(time.Minute),
	})

	if _, err := v.VerifyAccessToken(tok); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestVerifier_RejectsExpired(t *testing.T) {
	v, err := NewVerifier(testCurrentHex, "")
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	tok := mintAccessToken(t, testCurrentHex, "k", Claims{
		Subject:   "u1",
		ExpiresAt: time.Now().Add(-2 * time.Hour),
	})
	if _, err := v.VerifyAccessToken(tok); !errors.Is(err, ErrExpired) {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestVerifier_EmptyToken(t *testing.T) {
	v, err := NewVerifier(testCurrentHex, "")
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}
	if _, err := v.VerifyAccessToken(""); !errors.Is(err, ErrEmptyToken) {
		t.Errorf("expected ErrEmptyToken for empty string, got %v", err)
	}
	if _, err := v.VerifyAccessToken("   "); !errors.Is(err, ErrEmptyToken) {
		t.Errorf("expected ErrEmptyToken for whitespace, got %v", err)
	}
	if _, err := v.VerifyAccessToken("Bearer "); !errors.Is(err, ErrEmptyToken) {
		t.Errorf("expected ErrEmptyToken for Bearer + nothing, got %v", err)
	}
}

func TestVerifier_IssuerPinning(t *testing.T) {
	v, err := NewVerifier(testCurrentHex, "", WithIssuer("rinco"))
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	tokOK := mintAccessToken(t, testCurrentHex, "k", Claims{
		Issuer:    "rinco",
		ExpiresAt: time.Now().Add(time.Minute),
	})
	if _, err := v.VerifyAccessToken(tokOK); err != nil {
		t.Errorf("matching issuer should verify: %v", err)
	}

	tokBad := mintAccessToken(t, testCurrentHex, "k", Claims{
		Issuer:    "evil-iss",
		ExpiresAt: time.Now().Add(time.Minute),
	})
	if _, err := v.VerifyAccessToken(tokBad); !errors.Is(err, ErrInvalidIssuer) {
		t.Errorf("expected ErrInvalidIssuer, got %v", err)
	}

	// Empty issuer in token is treated as "no claim" and passes the pin
	// silently (some auth-service versions skip `iss` for compat).
	tokEmpty := mintAccessToken(t, testCurrentHex, "k", Claims{
		ExpiresAt: time.Now().Add(time.Minute),
	})
	if _, err := v.VerifyAccessToken(tokEmpty); err != nil {
		t.Errorf("empty-issuer token should be accepted, got %v", err)
	}
}

func TestVerifier_RoleFromScope(t *testing.T) {
	v, err := NewVerifier(testCurrentHex, "")
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	tok := mintAccessToken(t, testCurrentHex, "k", Claims{
		Subject:   "u1",
		TenantID:  "t1",
		Scope:     "role:agent read:leads write:contacts",
		ExpiresAt: time.Now().Add(time.Minute),
	})
	got, err := v.VerifyAccessToken(tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.Role != "agent" {
		t.Errorf("Role from scope: want agent, got %q", got.Role)
	}
}

func TestVerifier_SuperAdminHelpers(t *testing.T) {
	cases := []struct {
		role     string
		expected bool
	}{
		{"super_admin", true},
		{"platform_admin", true},
		{"tenant_admin", false},
		{"agent", false},
		{"", false},
	}
	for _, tc := range cases {
		c := &AccessClaims{Role: tc.role}
		if c.IsSuperAdmin() != tc.expected {
			t.Errorf("role=%q IsSuperAdmin=%v, want %v", tc.role, c.IsSuperAdmin(), tc.expected)
		}
	}
}

func TestVerifier_IsExpired(t *testing.T) {
	now := time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC)
	c := &AccessClaims{ExpiresAt: now.Add(-30 * time.Second)}

	// 1m skew: token expired 30s ago is within tolerance → NOT expired.
	if c.IsExpired(now, time.Minute) {
		t.Error("token expired 30s ago with 1m skew should NOT be expired (within tolerance)")
	}
	// 10s skew: token expired 30s ago is past tolerance → expired.
	if !c.IsExpired(now, 10*time.Second) {
		t.Error("token expired 30s ago with 10s skew should be expired")
	}

	// Zero expiry => never expired.
	if (&AccessClaims{}).IsExpired(now, time.Hour) {
		t.Error("zero ExpiresAt should not be expired")
	}

	// Future expiry => never expired.
	if (&AccessClaims{ExpiresAt: now.Add(time.Hour)}).IsExpired(now, time.Hour) {
		t.Error("future expiry should not be expired")
	}
}

func TestVerifier_NewVerifierFromEnv_MissingKey(t *testing.T) {
	t.Setenv("PASETO_KEY_CURRENT", "")
	if _, err := NewVerifierFromEnv(); err == nil {
		t.Error("expected error when PASETO_KEY_CURRENT is empty")
	}
}

func TestVerifier_NewVerifierFromEnv_OK(t *testing.T) {
	t.Setenv("PASETO_KEY_CURRENT", testCurrentHex)
	t.Setenv("PASETO_KEY_PREVIOUS", testPreviousHex)
	t.Setenv("AUTH_CLOCK_SKEW_SECONDS", "30")

	v, err := NewVerifierFromEnv()
	if err != nil {
		t.Fatalf("NewVerifierFromEnv: %v", err)
	}
	if v.clockSkew != 30*time.Second {
		t.Errorf("clockSkew: want 30s, got %v", v.clockSkew)
	}
	if v.CurrentKid() != testKeyringKid1 {
		t.Errorf("CurrentKid: want %q, got %q", testKeyringKid1, v.CurrentKid())
	}
}

func TestVerifier_NewVerifier_BadCurrentKey(t *testing.T) {
	if _, err := NewVerifier("", ""); err == nil {
		t.Error("expected error for empty current key")
	}
	if _, err := NewVerifier("not-hex", ""); err == nil {
		t.Error("expected error for malformed current key")
	}
	if _, err := NewVerifier("deadbeef", ""); err == nil {
		t.Error("expected error for short current key")
	}
}

func TestVerifier_MalformedToken(t *testing.T) {
	v, err := NewVerifier(testCurrentHex, "")
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	for _, bad := range []string{
		"not-a-paseto",
		"v2.local.invalid",
		strings.Repeat("x", 100),
	} {
		if _, err := v.VerifyAccessToken(bad); err == nil {
			t.Errorf("expected error for malformed token %q", bad)
		}
	}
}

func TestVerifier_NilReceiver(t *testing.T) {
	var v *Verifier
	if _, err := v.VerifyAccessToken("anything"); err == nil {
		t.Error("nil verifier should error")
	}
}
