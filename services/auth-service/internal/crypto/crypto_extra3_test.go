// Extra tests for auth-service crypto package.
package crypto

import (
	"strings"
	"testing"
	"time"
)

func TestClaims_JSONRoundTrip(t *testing.T) {
	c := Claims{
		Issuer:    "rinco",
		Audience:  "rinco-app",
		Subject:   "user-123",
		UserID:    "user-123",
		TenantID:  "tenant-1",
		Scope:     "read write",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	tok, err := newKeyRing().Encrypt(c)
	if err != nil {
		t.Fatal(err)
	}
	got, err := newKeyRing().Decrypt(tok)
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != c.UserID {
		t.Errorf("UserID: %s vs %s", got.UserID, c.UserID)
	}
	if got.TenantID != c.TenantID {
		t.Errorf("TenantID: %s vs %s", got.TenantID, c.TenantID)
	}
}

func TestClaims_RolesRoundTrip(t *testing.T) {
	c := Claims{
		UserID: "user-1",
		Roles:  []string{"admin", "user"},
	}
	tok, _ := newKeyRing().Encrypt(c)
	got, _ := newKeyRing().Decrypt(tok)
	if len(got.Roles) != 2 {
		t.Errorf("roles: %d", len(got.Roles))
	}
}

func TestToken_ThreeParts(t *testing.T) {
	c := Claims{UserID: "u1"}
	tok, _ := newKeyRing().Encrypt(c)
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Errorf("expected 3 parts: %s", tok)
	}
}

func TestDecrypt_InvalidFormat(t *testing.T) {
	_, err := newKeyRing().Decrypt("not-a-token")
	if err == nil {
		t.Error("expected error")
	}
}

func TestDecrypt_TwoParts(t *testing.T) {
	_, err := newKeyRing().Decrypt("only.two")
	if err == nil {
		t.Error("expected error")
	}
}

func TestDecrypt_BadBase64(t *testing.T) {
	_, err := newKeyRing().Decrypt("!!!.???.@@@")
	if err == nil {
		t.Error("expected error")
	}
}

func TestDecrypt_Tampered(t *testing.T) {
	c := Claims{UserID: "u1"}
	tok, _ := newKeyRing().Encrypt(c)
	parts := strings.Split(tok, ".")
	// Tamper with signature
	parts[2] = parts[2] + "X"
	tampered := strings.Join(parts, ".")
	_, err := newKeyRing().Decrypt(tampered)
	if err == nil {
		t.Error("expected error for tampered token")
	}
}

func TestDecrypt_ExpiredToken(t *testing.T) {
	c := Claims{
		UserID:    "u1",
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	tok, _ := newKeyRing().Encrypt(c)
	_, err := newKeyRing().Decrypt(tok)
	if err != ErrExpiredToken {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestGenerateRandomToken_Length(t *testing.T) {
	tok, err := GenerateRandomToken(16)
	if err != nil {
		t.Fatal(err)
	}
	// 16 bytes -> 32 hex chars
	if len(tok) != 32 {
		t.Errorf("got %d", len(tok))
	}
}

func TestGenerateRandomToken_Unique(t *testing.T) {
	a, _ := GenerateRandomToken(16)
	b, _ := GenerateRandomToken(16)
	if a == b {
		t.Error("tokens should be unique")
	}
}

func TestGenerateKey_Length(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	// 32 bytes -> 64 hex chars
	if len(key) != 64 {
		t.Errorf("got %d", len(key))
	}
}

func TestSHA256Hex_Length(t *testing.T) {
	h := SHA256Hex([]byte("hello"))
	if len(h) != 64 {
		t.Errorf("got %d", len(h))
	}
}

func TestTokenLifetimeBytes_Length(t *testing.T) {
	b := TokenLifetimeBytes(time.Second)
	if len(b) != 8 {
		t.Errorf("got %d", len(b))
	}
}

func TestNewKeyRing_TooShort(t *testing.T) {
	_, err := NewKeyRing([]byte("short"), nil, "v1", "")
	if err == nil {
		t.Error("expected error for short key")
	}
}

func TestNewKeyRingFromHex_BadHex(t *testing.T) {
	_, err := NewKeyRingFromHex("not-hex", "", "v1", "")
	if err == nil {
		t.Error("expected error")
	}
}

// helper
func newKeyRing() *KeyRing {
	r, _ := NewKeyRingFromHex(strings.Repeat("a", 64), strings.Repeat("0", 64), "v1", "v0")
	return r
}

func TestHeader_Fields(t *testing.T) {
	h := Header{Alg: "HS256", Typ: "JWT", Ver: "v1", Kid: "v1"}
	if h.Alg != "HS256" {
		t.Error("alg")
	}
	if h.Kid != "v1" {
		t.Error("kid")
	}
}

func TestCurrentKid(t *testing.T) {
	r := newKeyRing()
	if r.CurrentKid() != "v1" {
		t.Errorf("kid: %s", r.CurrentKid())
	}
}

func TestEncrypt_AutoJTI(t *testing.T) {
	c := Claims{UserID: "u1"}
	tok, err := newKeyRing().Encrypt(c)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := newKeyRing().Decrypt(tok)
	if got.JTI == "" {
		t.Error("JTI should be auto-generated")
	}
}

func TestEncrypt_AutoIssuer(t *testing.T) {
	c := Claims{UserID: "u1"}
	tok, _ := newKeyRing().Encrypt(c)
	got, _ := newKeyRing().Decrypt(tok)
	if got.Issuer != "rinco" {
		t.Errorf("Issuer: %s", got.Issuer)
	}
}
