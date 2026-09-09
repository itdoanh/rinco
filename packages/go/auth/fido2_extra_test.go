// Tests for auth FIDO2 helpers (fido2.go).
package auth

import (
	"context"
	"testing"
	"time"
)

func TestExtra_WebAuthnConfig_Defaults(t *testing.T) {
	cfg := WebAuthnConfig{
		RPDisplayName: "Test",
		RPID:         "test.example.com",
	}
	
	wcfg := cfg.ToWebAuthnConfig()
	if wcfg.RPDisplayName != "Test" {
		t.Error("RPDisplayName")
	}
	if wcfg.RPID != "test.example.com" {
		t.Error("RPID")
	}
}

func TestExtra_WebAuthnConfig_TimeoutDefault(t *testing.T) {
	cfg := WebAuthnConfig{
		RPDisplayName: "Test",
		RPID:         "test.example.com",
	}
	
	wcfg := cfg.ToWebAuthnConfig()
	if wcfg.Timeout == 0 {
		t.Error("Timeout should have default")
	}
}

func TestExtra_WebAuthnConfig_UserVerification(t *testing.T) {
	cfg := WebAuthnConfig{
		RPDisplayName:  "Test",
		RPID:          "test.example.com",
		UserVerification: "required",
	}
	
	wcfg := cfg.ToWebAuthnConfig()
	if wcfg.AuthenticatorSelection.UserVerification == "" {
		t.Error("UserVerification should be set")
	}
}

func TestExtra_WebAuthnUser(t *testing.T) {
	user := &WebAuthnUser{
		ID:          []byte("user-id"),
		Name:        "testuser",
		DisplayName: "Test User",
	}
	
	if string(user.WebAuthnID()) != "user-id" {
		t.Error("WebAuthnID")
	}
	if user.WebAuthnName() != "testuser" {
		t.Error("WebAuthnName")
	}
	if user.WebAuthnDisplayName() != "Test User" {
		t.Error("WebAuthnDisplayName")
	}
}

func TestExtra_WebAuthnUser_AddCredential(t *testing.T) {
	user := &WebAuthnUser{
		ID:          []byte("user-id"),
		Name:        "testuser",
		DisplayName: "Test User",
	}
	
	if len(user.Credentials) != 0 {
		t.Error("should start with 0 credentials")
	}
	
	// Can't add real credential without webauthn import, but we can verify the method exists
	// This tests the interface
	_ = user.WebAuthnCredentials()
}

func TestExtra_MemoryChallengeStore(t *testing.T) {
	store := NewMemoryChallengeStore()
	if store == nil {
		t.Fatal("NewMemoryChallengeStore returned nil")
	}
	if store.data == nil {
		t.Error("data map should be initialized")
	}
}

func TestExtra_MemoryChallengeStore_SaveGet(t *testing.T) {
	store := NewMemoryChallengeStore()
	
	data := ChallengeData{
		Flow:      "registration",
		UserID:    "user-123",
		TenantID:  "tenant-456",
		Challenge: "challenge-abc",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	
	err := store.Save(context.Background(), "key1", data, 5*time.Minute)
	if err != nil {
		t.Errorf("Save error: %v", err)
	}
	
	got, err := store.Get(context.Background(), "key1")
	if err != nil {
		t.Errorf("Get error: %v", err)
	}
	if got.Flow != "registration" {
		t.Errorf("Flow: got %s", got.Flow)
	}
	if got.UserID != "user-123" {
		t.Errorf("UserID: got %s", got.UserID)
	}
}

func TestExtra_MemoryChallengeStore_GetNotFound(t *testing.T) {
	store := NewMemoryChallengeStore()
	
	_, err := store.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent key")
	}
	if err != ErrChallengeNotFound {
		t.Errorf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestExtra_MemoryChallengeStore_Expired(t *testing.T) {
	store := NewMemoryChallengeStore()
	
	// Save with a very short TTL that will expire
	data := ChallengeData{
		Flow: "login",
		ExpiresAt: time.Now().Add(10 * time.Millisecond),
	}
	
	store.Save(context.Background(), "expired-key", data, 10*time.Millisecond)
	
	// Wait for it to expire
	time.Sleep(20 * time.Millisecond)
	
	_, err := store.Get(context.Background(), "expired-key")
	if err == nil {
		t.Error("expected error for expired challenge")
	}
	if err != ErrChallengeExpired {
		t.Errorf("expected ErrChallengeExpired, got %v", err)
	}
}

func TestExtra_MemoryChallengeStore_Delete(t *testing.T) {
	store := NewMemoryChallengeStore()
	
	data := ChallengeData{
		Flow:      "login",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	
	store.Save(context.Background(), "delete-key", data, 5*time.Minute)
	
	err := store.Delete(context.Background(), "delete-key")
	if err != nil {
		t.Errorf("Delete error: %v", err)
	}
	
	_, err = store.Get(context.Background(), "delete-key")
	if err != ErrChallengeNotFound {
		t.Errorf("expected ErrChallengeNotFound after delete, got %v", err)
	}
}

func TestExtra_MemoryChallengeStore_Concurrent(t *testing.T) {
	store := NewMemoryChallengeStore()
	
	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			data := ChallengeData{Flow: "test", ExpiresAt: time.Now().Add(time.Hour)}
			_ = store.Save(context.Background(), "concurrent-key", data, time.Hour)
			done <- true
		}(i)
	}
	
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestExtra_ChallengeData_Fields(t *testing.T) {
	data := ChallengeData{
		Flow:         "registration",
		UserID:       "user-1",
		TenantID:     "tenant-1",
		Challenge:    "challenge-123",
		AllowedCreds: [][]byte{[]byte("cred1"), []byte("cred2")},
		ExpiresAt:    time.Now().Add(time.Hour),
		Extra:        map[string]string{"key": "value"},
	}
	
	if data.Flow != "registration" {
		t.Error("Flow")
	}
	if data.UserID != "user-1" {
		t.Error("UserID")
	}
	if len(data.AllowedCreds) != 2 {
		t.Error("AllowedCreds length")
	}
	if data.Extra["key"] != "value" {
		t.Error("Extra")
	}
}

func TestExtra_newV7String(t *testing.T) {
	s := newV7String()
	if s == "" {
		t.Error("newV7String should not return empty")
	}
}

func TestExtra_newV7String_Unique(t *testing.T) {
	s1 := newV7String()
	s2 := newV7String()
	if s1 == s2 {
		t.Error("newV7String should return unique values")
	}
}

func TestExtra_strBody(t *testing.T) {
	result := strBody("testKey", "testValue")
	if result == "" {
		t.Error("strBody should not return empty")
	}
}

func TestExtra_Errors(t *testing.T) {
	if ErrChallengeNotFound.Error() == "" {
		t.Error("ErrChallengeNotFound should have message")
	}
	if ErrChallengeExpired.Error() == "" {
		t.Error("ErrChallengeExpired should have message")
	}
}

func TestExtra_WebAuthnConfig_Attestation(t *testing.T) {
	cfg := WebAuthnConfig{
		RPDisplayName:          "Test",
		RPID:                  "test.example.com",
		AttestationPreference: "direct",
	}
	
	wcfg := cfg.ToWebAuthnConfig()
	if wcfg.AttestationPreference == "" {
		t.Error("AttestationPreference should be set")
	}
}
