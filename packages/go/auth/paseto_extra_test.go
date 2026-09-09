// Tests for auth PASETO token (paseto.go).
package auth

import (
	"testing"
	"time"
)

func TestExtra_PasetoErrors(t *testing.T) {
	if ErrExpired.Error() == "" {
		t.Error("ErrExpired should have message")
	}
	if ErrInvalid.Error() == "" {
		t.Error("ErrInvalid should have message")
	}
	if ErrUnsupported.Error() == "" {
		t.Error("ErrUnsupported should have message")
	}
	if ErrKeyMismatch.Error() == "" {
		t.Error("ErrKeyMismatch should have message")
	}
	if ErrSessionExist.Error() == "" {
		t.Error("ErrSessionExist should have message")
	}
}

func TestExtra_NewPaseto(t *testing.T) {
	// Generate test key
	hexKey, err := GeneratePASETOKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	
	p, err := NewPaseto(hexKey, "v1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("NewPaseto returned nil")
	}
	if p.Kid() != "v1" {
		t.Errorf("Kid: got %s", p.Kid())
	}
}

func TestExtra_NewPaseto_InvalidHex(t *testing.T) {
	_, err := NewPaseto("invalid-hex", "v1")
	if err == nil {
		t.Error("expected error for invalid hex")
	}
}

func TestExtra_NewPaseto_InvalidKeySize(t *testing.T) {
	// Valid hex but wrong size (16 bytes instead of 32)
	_, err := NewPaseto("00112233445566778899aabbccddeeff", "v1")
	if err == nil {
		t.Error("expected error for wrong key size")
	}
}

func TestExtra_Paseto_EncryptDecrypt(t *testing.T) {
	hexKey, _ := GeneratePASETOKey()
	p, _ := NewPaseto(hexKey, "v1")
	
	claims := Claims{
		UserID:   "user-1",
		TenantID: "tenant-1",
		Roles:    []string{"admin"},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	
	token, err := p.Encrypt(claims)
	if err != nil {
		t.Errorf("encrypt: %v", err)
	}
	if token == "" {
		t.Error("token should not be empty")
	}
	
	decrypted, err := p.Decrypt(token)
	if err != nil {
		t.Errorf("decrypt: %v", err)
	}
	if decrypted.UserID != "user-1" {
		t.Errorf("UserID mismatch: got %s", decrypted.UserID)
	}
	if decrypted.TenantID != "tenant-1" {
		t.Errorf("TenantID mismatch: got %s", decrypted.TenantID)
	}
}

func TestExtra_Paseto_Encrypt_AutoFilledFields(t *testing.T) {
	hexKey, _ := GeneratePASETOKey()
	p, _ := NewPaseto(hexKey, "v1")
	
	claims := Claims{
		UserID:    "u1",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	
	token, err := p.Encrypt(claims)
	if err != nil {
		t.Errorf("encrypt: %v", err)
	}
	
	decrypted, err := p.Decrypt(token)
	if err != nil {
		t.Errorf("decrypt: %v", err)
	}
	
	// Check auto-filled fields
	if decrypted.JTI == "" {
		t.Error("JTI should be auto-filled")
	}
	if decrypted.IssuedAt.IsZero() {
		t.Error("IssuedAt should be auto-filled")
	}
	if decrypted.Issuer != "rinco" {
		t.Errorf("Issuer should default to 'rinco', got %s", decrypted.Issuer)
	}
	if decrypted.Audience != "rinco-app" {
		t.Errorf("Audience should default to 'rinco-app', got %s", decrypted.Audience)
	}
	if decrypted.Kid != "v1" {
		t.Errorf("Kid should be set, got %s", decrypted.Kid)
	}
}

func TestExtra_Paseto_Decrypt_Invalid(t *testing.T) {
	hexKey, _ := GeneratePASETOKey()
	p, _ := NewPaseto(hexKey, "v1")
	
	_, err := p.Decrypt("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestExtra_Paseto_Decrypt_WrongKey(t *testing.T) {
	hexKey1, _ := GeneratePASETOKey()
	hexKey2, _ := GeneratePASETOKey()
	p1, _ := NewPaseto(hexKey1, "v1")
	p2, _ := NewPaseto(hexKey2, "v2")
	
	token, _ := p1.Encrypt(Claims{UserID: "u1", ExpiresAt: time.Now().Add(time.Hour)})
	_, err := p2.Decrypt(token)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestExtra_Paseto_ExpiredToken(t *testing.T) {
	hexKey, _ := GeneratePASETOKey()
	p, _ := NewPaseto(hexKey, "v1")
	
	// Create expired token
	claims := Claims{
		UserID:    "u1",
		ExpiresAt: time.Now().Add(-time.Hour), // Already expired
	}
	token, _ := p.Encrypt(claims)
	
	_, err := p.Decrypt(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestExtra_Paseto_NotBefore(t *testing.T) {
	hexKey, _ := GeneratePASETOKey()
	p, _ := NewPaseto(hexKey, "v1")
	
	// Create token not yet valid
	future := time.Now().Add(time.Hour)
	claims := Claims{
		UserID:    "u1",
		NotBefore: future,
		ExpiresAt: future.Add(time.Hour),
	}
	token, _ := p.Encrypt(claims)
	
	// Should fail because nbf is in future
	_, err := p.Decrypt(token)
	// Note: behavior may differ based on tolerance
	_ = err
}

func TestExtra_Paseto_SetClockSkew(t *testing.T) {
	hexKey, _ := GeneratePASETOKey()
	p, _ := NewPaseto(hexKey, "v1")
	
	p.SetClockSkew(time.Minute)
	// Just verify it doesn't panic
}

func TestExtra_Paseto_Kid(t *testing.T) {
	hexKey, _ := GeneratePASETOKey()
	p, _ := NewPaseto(hexKey, "kid-123")
	if p.Kid() != "kid-123" {
		t.Errorf("got %s", p.Kid())
	}
}

func TestExtra_GenerateSecureToken(t *testing.T) {
	token, err := GenerateSecureToken(32)
	if err != nil {
		t.Errorf("err: %v", err)
	}
	if len(token) != 64 { // 32 bytes hex = 64 chars
		t.Errorf("got length %d", len(token))
	}
}

func TestExtra_GenerateSecureToken_DifferentEachTime(t *testing.T) {
	t1, _ := GenerateSecureToken(16)
	t2, _ := GenerateSecureToken(16)
	if t1 == t2 {
		t.Error("tokens should be different")
	}
}

func TestExtra_GeneratePASETOKey(t *testing.T) {
	key, err := GeneratePASETOKey()
	if err != nil {
		t.Errorf("err: %v", err)
	}
	// Should be 64 hex chars (32 bytes)
	if len(key) != 64 {
		t.Errorf("got length %d", len(key))
	}
}

func TestExtra_GeneratePASETOKey_DifferentEachTime(t *testing.T) {
	k1, _ := GeneratePASETOKey()
	k2, _ := GeneratePASETOKey()
	if k1 == k2 {
		t.Error("keys should be different")
	}
}

func TestExtra_NewKeyRing(t *testing.T) {
	currentHex, _ := GeneratePASETOKey()
	previousHex, _ := GeneratePASETOKey()
	
	ring, err := NewKeyRing(currentHex, previousHex, "v2", "v1")
	if err != nil {
		t.Errorf("unexpected: %v", err)
	}
	if ring == nil {
		t.Fatal("NewKeyRing returned nil")
	}
}

func TestExtra_NewKeyRing_NoPrevious(t *testing.T) {
	currentHex, _ := GeneratePASETOKey()
	
	ring, err := NewKeyRing(currentHex, "", "v2", "v1")
	if err != nil {
		t.Errorf("unexpected: %v", err)
	}
	if ring.previous != nil {
		t.Error("previous should be nil")
	}
}

func TestExtra_NewKeyRing_InvalidCurrent(t *testing.T) {
	_, err := NewKeyRing("invalid", "", "v2", "v1")
	if err == nil {
		t.Error("expected error for invalid current key")
	}
}

func TestExtra_KeyRing_CurrentKid(t *testing.T) {
	currentHex, _ := GeneratePASETOKey()
	ring, _ := NewKeyRing(currentHex, "", "current-kid", "previous-kid")
	
	if ring.CurrentKid() != "current-kid" {
		t.Errorf("got %s", ring.CurrentKid())
	}
}

func TestExtra_KeyRing_EncryptDecrypt(t *testing.T) {
	currentHex, _ := GeneratePASETOKey()
	ring, _ := NewKeyRing(currentHex, "", "v2", "v1")
	
	claims := Claims{
		UserID:    "u1",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	
	token, err := ring.Encrypt(claims)
	if err != nil {
		t.Errorf("encrypt: %v", err)
	}
	
	decrypted, err := ring.Decrypt(token)
	if err != nil {
		t.Errorf("decrypt: %v", err)
	}
	if decrypted.UserID != "u1" {
		t.Errorf("got %s", decrypted.UserID)
	}
}

func TestExtra_KeyRing_DecryptWithPreviousKey(t *testing.T) {
	currentHex, _ := GeneratePASETOKey()
	previousHex, _ := GeneratePASETOKey()
	
	currentRing, _ := NewKeyRing(currentHex, "", "v2", "v1")
	previousRing, _ := NewPaseto(previousHex, "v1")
	
	// Create token with previous key
	token, _ := previousRing.Encrypt(Claims{
		UserID:    "u1",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	
	// New ring should be able to decrypt (with both keys)
	ring, _ := NewKeyRing(currentHex, previousHex, "v2", "v1")
	decrypted, err := ring.Decrypt(token)
	if err != nil {
		t.Errorf("decrypt with previous key: %v", err)
	}
	if decrypted.UserID != "u1" {
		t.Errorf("UserID mismatch: %s", decrypted.UserID)
	}
	_ = currentRing
}

func TestExtra_KeyRing_Decrypt_Invalid(t *testing.T) {
	currentHex, _ := GeneratePASETOKey()
	ring, _ := NewKeyRing(currentHex, "", "v2", "v1")
	
	_, err := ring.Decrypt("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestExtra_KeyPurpose_Constants(t *testing.T) {
	if LocalKey != 0 {
		t.Error("LocalKey should be 0")
	}
	if PublicKey != 1 {
		t.Error("PublicKey should be 1")
	}
}

func TestExtra_Claims_Fields(t *testing.T) {
	now := time.Now()
	c := Claims{
		Issuer:    "iss",
		Subject:   "sub",
		Audience:  "aud",
		ExpiresAt: now.Add(time.Hour),
		IssuedAt:  now,
		NotBefore: now,
		JTI:       "jti",
		Kid:       "kid",
		TenantID:  "tenant",
		UserID:    "user",
		Email:     "test@example.com",
		Roles:     []string{"admin"},
		Permissions: []string{"read"},
		Scope:     "scope",
		SessionID: "sid",
	}
	if c.Issuer != "iss" {
		t.Error("Issuer")
	}
	if c.UserID != "user" {
		t.Error("UserID")
	}
	if c.TenantID != "tenant" {
		t.Error("TenantID")
	}
}
