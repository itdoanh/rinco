// Tests for auth-service crypto primitives.
package crypto

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Password hashing
// =============================================================================

func TestExtraHashPassword_Format(t *testing.T) {
	hash, err := HashPassword("hunter2")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	// Expected PHC format: $argon2id$v=19$m=...,t=...,p=...$salt$hash
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("prefix missing: %q", hash[:20])
	}
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Errorf("expected 6 parts, got %d", len(parts))
	}
	if parts[2] != "v=19" {
		t.Errorf("version: %q", parts[2])
	}
}

func TestExtraVerifyPassword_RoundTrip(t *testing.T) {
	hash, err := HashPassword("hunter2")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	ok, err := VerifyPassword("hunter2", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Error("expected password to match")
	}
}

func TestExtraVerifyPassword_WrongPassword(t *testing.T) {
	hash, err := HashPassword("hunter2")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	ok, err := VerifyPassword("hunter3", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if ok {
		t.Error("expected wrong password to fail")
	}
}

func TestExtraVerifyPassword_BadEncoding(t *testing.T) {
	// Not argon2id prefix should return an error.
	ok, err := VerifyPassword("x", "$bcrypt$v=1$1$2$3")
	if err == nil {
		t.Error("expected error for non-argon2id prefix")
	}
	if ok {
		t.Error("expected ok=false")
	}
}

func TestExtraVerifyPassword_MalformedHash(t *testing.T) {
	ok, err := VerifyPassword("x", "$argon2id$bad")
	if err == nil {
		t.Error("expected error for malformed hash")
	}
	if ok {
		t.Error("expected ok=false")
	}
}

func TestExtraHashPassword_SaltsAreUnique(t *testing.T) {
	// Two hashes of the same password should produce different output
	// (each one uses fresh randomness).
	h1, _ := HashPassword("same")
	h2, _ := HashPassword("same")
	if h1 == h2 {
		t.Error("two hashes of same password should differ (salt randomness)")
	}
}

func TestExtraHashPasswordCustom_RespectsParams(t *testing.T) {
	// Time cost of 1 vs 3 should produce measurably different output.
	h1, _ := HashPasswordCustom("p", 32*1024, 1, 1, 8, 16)
	h2, _ := HashPasswordCustom("p", 32*1024, 3, 1, 8, 16)
	if h1 == h2 {
		t.Error("hashes with different params should differ")
	}
}

// =============================================================================
// KeyRing setup
// =============================================================================

func TestExtraNewKeyRing_TooShort(t *testing.T) {
	_, err := NewKeyRing([]byte("short"), nil, "k1", "k2")
	if err == nil {
		t.Error("expected error for short key")
	}
}

func TestExtraNewKeyRing_OK(t *testing.T) {
	r, err := NewKeyRing(make([]byte, 32), make([]byte, 32), "current", "prev")
	if err != nil {
		t.Fatalf("NewKeyRing: %v", err)
	}
	if r.CurrentKid() != "current" {
		t.Errorf("CurrentKid: %s", r.CurrentKid())
	}
}

func TestExtraNewKeyRing_NoPrevious(t *testing.T) {
	r, err := NewKeyRing(make([]byte, 32), nil, "k1", "")
	if err != nil {
		t.Fatalf("NewKeyRing: %v", err)
	}
	if r.CurrentKid() != "k1" {
		t.Errorf("CurrentKid: %s", r.CurrentKid())
	}
}

func TestExtraNewKeyRingFromHex_OK(t *testing.T) {
	curr := strings.Repeat("a", 64)
	prev := strings.Repeat("b", 64)
	r, err := NewKeyRingFromHex(curr, prev, "c", "p")
	if err != nil {
		t.Fatalf("NewKeyRingFromHex: %v", err)
	}
	if r.CurrentKid() != "c" {
		t.Errorf("CurrentKid: %s", r.CurrentKid())
	}
}

func TestExtraNewKeyRingFromHex_BadHex(t *testing.T) {
	_, err := NewKeyRingFromHex("not-hex", "", "c", "p")
	if err == nil {
		t.Error("expected error for bad hex")
	}
}

func TestExtraNewKeyRingFromHex_ZeroPreviousTreatedAsNone(t *testing.T) {
	// 64 zeros = "all zeros" which our code treats as "no previous key".
	curr := strings.Repeat("a", 64)
	zero := strings.Repeat("0", 64)
	r, err := NewKeyRingFromHex(curr, zero, "c", "p")
	if err != nil {
		t.Fatalf("NewKeyRingFromHex: %v", err)
	}
	if r.previous != nil {
		t.Error("zero hex should leave previous key nil")
	}
}

// =============================================================================
// Encrypt / Decrypt
// =============================================================================

func roundTripHelper(t *testing.T) *KeyRing {
	t.Helper()
	currKey, _ := GenerateKey()
	prevKey, _ := GenerateKey()
	curr, _ := hex.DecodeString(currKey)
	prev, _ := hex.DecodeString(prevKey)
	r, err := NewKeyRing(curr, prev, "k1", "k2")
	if err != nil {
		t.Fatalf("NewKeyRing: %v", err)
	}
	return r
}

func TestExtraEncryptDecrypt_RoundTrip(t *testing.T) {
	r := roundTripHelper(t)
	tok, err := r.Encrypt(Claims{
		Subject:   "user-1",
		UserID:    "u1",
		TenantID:  "t1",
		Roles:     []string{"admin"},
		Scope:     "read write",
		Audience:  "rinco-app",
		NotBefore: time.Now().Add(-time.Minute),
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	c, err := r.Decrypt(tok)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if c.Subject != "user-1" || c.UserID != "u1" || c.TenantID != "t1" {
		t.Errorf("claims mismatch: %+v", c)
	}
	if len(c.Roles) != 1 || c.Roles[0] != "admin" {
		t.Errorf("roles: %v", c.Roles)
	}
}

func TestExtraEncrypt_AutoFillDefaults(t *testing.T) {
	r := roundTripHelper(t)
	tok, err := r.Encrypt(Claims{Subject: "x"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := r.Decrypt(tok)
	if err != nil {
		t.Fatal(err)
	}
	if c.Issuer != "rinco" {
		t.Errorf("default issuer: got %q", c.Issuer)
	}
	if c.Audience != "rinco-app" {
		t.Errorf("default audience: got %q", c.Audience)
	}
	if c.JTI == "" {
		t.Error("JTI should be auto-generated")
	}
	if c.IssuedAt.IsZero() {
		t.Error("IssuedAt should be set automatically")
	}
}

func TestExtraDecrypt_TooFewParts(t *testing.T) {
	r := roundTripHelper(t)
	if _, err := r.Decrypt("a.b"); err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestExtraDecrypt_BadBase64Header(t *testing.T) {
	r := roundTripHelper(t)
	if _, err := r.Decrypt("!@#$.c.d"); err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for bad header, got %v", err)
	}
}

func TestExtraDecrypt_BadSignature(t *testing.T) {
	r1 := roundTripHelper(t)
	r2 := roundTripHelper(t) // different keys
	tok, _ := r1.Encrypt(Claims{Subject: "x", ExpiresAt: time.Now().Add(time.Hour)})
	// r2 should not be able to verify a token signed by r1 (different key, unknown kid).
	if _, err := r2.Decrypt(tok); err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for unknown kid, got %v", err)
	}
}

func TestExtraDecrypt_ExpiredToken(t *testing.T) {
	r := roundTripHelper(t)
	tok, _ := r.Encrypt(Claims{
		Subject:   "x",
		ExpiresAt: time.Now().Add(-time.Minute), // already expired
	})
	if _, err := r.Decrypt(tok); err != ErrExpiredToken {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestExtraDecrypt_NotYetValid(t *testing.T) {
	r := roundTripHelper(t)
	future := time.Now().Add(time.Hour)
	tok, _ := r.Encrypt(Claims{
		Subject:   "x",
		NotBefore: future,
		ExpiresAt: future.Add(time.Hour),
	})
	if _, err := r.Decrypt(tok); err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for not-yet-valid, got %v", err)
	}
}

func TestExtraDecrypt_PreviousKey(t *testing.T) {
	currKey, _ := GenerateKey()
	prevKey, _ := GenerateKey()
	curr, _ := hex.DecodeString(currKey)
	prev, _ := hex.DecodeString(prevKey)
	r, err := NewKeyRing(curr, prev, "now", "then")
	if err != nil {
		t.Fatal(err)
	}

	// Manually craft a token using the previous key + previous kid.
	tok, err := signWithKeyAndKid(r, Claims{Subject: "old"}, prev, "then")
	if err != nil {
		t.Fatalf("signWithKeyAndKid: %v", err)
	}
	c, err := r.Decrypt(tok)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if c.Subject != "old" {
		t.Errorf("subject: %s", c.Subject)
	}
}

// signWithKeyAndKid is a helper that builds a token signed with the supplied
// key + kid pair (used to test the key rotation path in Decrypt).
func signWithKeyAndKid(r *KeyRing, c Claims, key []byte, kid string) (string, error) {
	c.IssuedAt = time.Now()
	c.ExpiresAt = c.IssuedAt.Add(time.Hour)
	hdr := Header{Alg: "HS256", Typ: "RINCO-JWT", Ver: "v1", Kid: kid}
	headerJSON, err := json.Marshal(hdr)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	h64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	p64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signed := h64 + "." + p64
	sig := r.sign([]byte(signed), key)
	s64 := base64.RawURLEncoding.EncodeToString(sig)
	return signed + "." + s64, nil
}

// =============================================================================
// Random helpers
// =============================================================================

func TestExtraGenerateRandomToken_Length(t *testing.T) {
	tok, err := GenerateRandomToken(32)
	if err != nil {
		t.Fatal(err)
	}
	// 32 bytes → 64 hex chars.
	if len(tok) != 64 {
		t.Errorf("expected 64 chars, got %d", len(tok))
	}
}

func TestExtraGenerateRandomToken_Unique(t *testing.T) {
	a, _ := GenerateRandomToken(16)
	b, _ := GenerateRandomToken(16)
	if a == b {
		t.Error("two random tokens should not collide")
	}
}

func TestExtraGenerateKey_HexShape(t *testing.T) {
	k, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	// 32 bytes → 64 hex chars.
	if len(k) != 64 {
		t.Errorf("expected 64 chars, got %d", len(k))
	}
	if _, err := hex.DecodeString(k); err != nil {
		t.Errorf("not valid hex: %v", err)
	}
}

func TestExtraSHA256Hex_KnownVector(t *testing.T) {
	// SHA-256 of "" = e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	got := SHA256Hex([]byte(""))
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Errorf("SHA256Hex empty: got %q", got)
	}
}

func TestExtraSHA256Hex_DifferentInputsProduceDifferentHashes(t *testing.T) {
	a := SHA256Hex([]byte("a"))
	b := SHA256Hex([]byte("b"))
	if a == b {
		t.Error("different inputs should hash differently")
	}
}

func TestExtraTokenLifetimeBytes_Length(t *testing.T) {
	b := TokenLifetimeBytes(time.Second)
	if len(b) != 8 {
		t.Errorf("expected 8 bytes, got %d", len(b))
	}
}
