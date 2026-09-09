// Tests for auth API key helpers (api_key.go).
package auth

import (
	"strings"
	"testing"
	"time"
)

func TestExtra_GenerateAPIKey_Live(t *testing.T) {
	key, err := GenerateAPIKey(APIKeyLive, "testprefix")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if key == "" {
		t.Fatal("key should not be empty")
	}
	parts := strings.Split(key, "_")
	if len(parts) != 3 {
		t.Errorf("expected 3 parts, got %d", len(parts))
	}
	if parts[0] != "testprefix" {
		t.Errorf("prefix: got %s", parts[0])
	}
	if parts[1] != "live" {
		t.Errorf("env: got %s", parts[1])
	}
}

func TestExtra_GenerateAPIKey_Test(t *testing.T) {
	key, err := GenerateAPIKey(APIKeyTest, "mypfx")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	parts := strings.Split(key, "_")
	if parts[1] != "test" {
		t.Errorf("env: got %s", parts[1])
	}
}

func TestExtra_GenerateAPIKey_DefaultPrefix(t *testing.T) {
	key, err := GenerateAPIKey(APIKeyLive, "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	parts := strings.Split(key, "_")
	if parts[0] != DefaultPrefix {
		t.Errorf("prefix: got %s, want %s", parts[0], DefaultPrefix)
	}
}

func TestExtra_GenerateAPIKey_InvalidEnv(t *testing.T) {
	_, err := GenerateAPIKey("invalid", "testprefix")
	if err == nil {
		t.Error("expected error for invalid env")
	}
}

func TestExtra_GenerateAPIKey_Unique(t *testing.T) {
	keys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		key, err := GenerateAPIKey(APIKeyLive, "test")
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if keys[key] {
			t.Error("duplicate key generated")
		}
		keys[key] = true
	}
}

func TestExtra_ParseAPIKey_Valid(t *testing.T) {
	key, _ := GenerateAPIKey(APIKeyLive, "mypfx")
	prefix, env, secret, err := ParseAPIKey(key)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if prefix != "mypfx" {
		t.Errorf("prefix: got %s", prefix)
	}
	if env != APIKeyLive {
		t.Errorf("env: got %s", env)
	}
	if secret == "" {
		t.Error("secret should not be empty")
	}
}

func TestExtra_ParseAPIKey_InvalidFormat(t *testing.T) {
	_, _, _, err := ParseAPIKey("invalid-key")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestExtra_ParseAPIKey_InvalidEnv(t *testing.T) {
	_, _, _, err := ParseAPIKey("pfx_wrong_secret")
	if err == nil {
		t.Error("expected error for invalid env")
	}
}

func TestExtra_ParseAPIKey_TooShortSecret(t *testing.T) {
	_, _, _, err := ParseAPIKey("pfx_live_short")
	if err == nil {
		t.Error("expected error for too short secret")
	}
}

func TestExtra_HashKey(t *testing.T) {
	hash1 := HashKey("test-key")
	hash2 := HashKey("test-key")
	if hash1 != hash2 {
		t.Error("same key should produce same hash")
	}
	
	hash3 := HashKey("different-key")
	if hash1 == hash3 {
		t.Error("different keys should produce different hashes")
	}
}

func TestExtra_HashKey_Length(t *testing.T) {
	hash := HashKey("any-key")
	if len(hash) != 64 { // SHA256 hex = 64 chars
		t.Errorf("expected 64 chars, got %d", len(hash))
	}
}

func TestExtra_Last4(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abcdefghij", "ghij"},
		{"ab", "ab"},
		{"a", "a"},
		{"", ""},
		{"abcd", "abcd"},
	}
	for _, tt := range tests {
		got := Last4(tt.input)
		if got != tt.expected {
			t.Errorf("Last4(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestExtra_IsExpired_NilRecord(t *testing.T) {
	if IsExpired(nil, time.Now()) {
		t.Error("nil record should not be expired")
	}
}

func TestExtra_IsExpired_NoExpiry(t *testing.T) {
	record := &APIKeyRecord{}
	if IsExpired(record, time.Now()) {
		t.Error("record without expiry should not be expired")
	}
}

func TestExtra_IsExpired_NotExpired(t *testing.T) {
	future := time.Now().Add(time.Hour)
	record := &APIKeyRecord{ExpiresAt: &future}
	if IsExpired(record, time.Now()) {
		t.Error("future expiry should not be expired")
	}
}

func TestExtra_IsExpired_Expired(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	record := &APIKeyRecord{ExpiresAt: &past}
	if !IsExpired(record, time.Now()) {
		t.Error("past expiry should be expired")
	}
}

func TestExtra_IsRevoked_Nil(t *testing.T) {
	if IsRevoked(nil) {
		t.Error("nil record should not be revoked")
	}
}

func TestExtra_IsRevoked_NotRevoked(t *testing.T) {
	record := &APIKeyRecord{}
	if IsRevoked(record) {
		t.Error("record without revoked_at should not be revoked")
	}
}

func TestExtra_IsRevoked_Revoked(t *testing.T) {
	now := time.Now()
	record := &APIKeyRecord{RevokedAt: &now}
	if !IsRevoked(record) {
		t.Error("record with revoked_at should be revoked")
	}
}

func TestExtra_ScopesContain_Exact(t *testing.T) {
	scopes := []string{"read", "write"}
	if !ScopesContain(scopes, "read") {
		t.Error("should contain read")
	}
	if ScopesContain(scopes, "admin") {
		t.Error("should not contain admin")
	}
}

func TestExtra_ScopesContain_Wildcard(t *testing.T) {
	scopes := []string{"*"}
	if !ScopesContain(scopes, "anything") {
		t.Error("* should match anything")
	}
}

func TestExtra_ScopesContain_PrefixWildcard(t *testing.T) {
	scopes := []string{"users.*"}
	if !ScopesContain(scopes, "users.read") {
		t.Error("users.* should match users.read")
	}
	if !ScopesContain(scopes, "users.write") {
		t.Error("users.* should match users.write")
	}
	if ScopesContain(scopes, "other.read") {
		t.Error("users.* should not match other.read")
	}
}

func TestExtra_ScopesContain_Empty(t *testing.T) {
	if ScopesContain([]string{}, "read") {
		t.Error("empty scopes should not contain anything")
	}
}

func TestExtra_ScopesContain_Nil(t *testing.T) {
	if ScopesContain(nil, "read") {
		t.Error("nil scopes should not contain anything")
	}
}

func TestExtra_NewRecord(t *testing.T) {
	record, err := NewRecord("tenant1", "user1", "Test Key", APIKeyLive, []string{"read"}, 1000, 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if record.TenantID != "tenant1" {
		t.Error("TenantID")
	}
	if record.UserID != "user1" {
		t.Error("UserID")
	}
	if record.Name != "Test Key" {
		t.Error("Name")
	}
	if record.Key == "" {
		t.Error("Key should not be empty")
	}
	if record.Last4 == "" {
		t.Error("Last4 should not be empty")
	}
}

func TestExtra_NewRecord_WithExpiry(t *testing.T) {
	ttl := 24 * time.Hour
	record, err := NewRecord("t", "u", "k", APIKeyLive, nil, 0, ttl)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if record.ExpiresAt == nil {
		t.Error("ExpiresAt should be set")
	}
}

func TestExtra_NewRecord_DefaultRateLimit(t *testing.T) {
	record, err := NewRecord("t", "u", "k", APIKeyLive, nil, 0, 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if record.RateLimit != 1000 {
		t.Errorf("default rate limit: got %d", record.RateLimit)
	}
}

func TestExtra_APIKey_Fields(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)
	key := APIKey{
		ID:        "id-1",
		TenantID:  "tenant-1",
		UserID:    "user-1",
		Name:      "Test Key",
		Prefix:    "rinco_live_abc",
		Scopes:    []string{"read"},
		RateLimit: 100,
		ExpiresAt: &future,
		CreatedAt: now,
	}
	if key.ID != "id-1" {
		t.Error("ID")
	}
	if key.TenantID != "tenant-1" {
		t.Error("TenantID")
	}
}

func TestExtra_APIKeyWithSecret_Fields(t *testing.T) {
	now := time.Now()
	kws := APIKeyWithSecret{
		APIKey: APIKey{
			ID:        "id-1",
			CreatedAt: now,
		},
		Key:   "secret-key",
		Last4: "eyJ0",
	}
	if kws.Key != "secret-key" {
		t.Error("Key")
	}
	if kws.Last4 != "eyJ0" {
		t.Error("Last4")
	}
}

func TestExtra_APIKeyRecord_Fields(t *testing.T) {
	now := time.Now()
	record := APIKeyRecord{
		ID:         "id-1",
		TenantID:   "tenant-1",
		UserID:     "user-1",
		Name:       "Test Key",
		Prefix:     "rinco_live_abc",
		Hash:       "hash123",
		Last4:      "abc1",
		Scopes:     []string{"read"},
		RateLimit:  100,
		CreatedAt:  now,
	}
	if record.ID != "id-1" {
		t.Error("ID")
	}
	if record.Hash != "hash123" {
		t.Error("Hash")
	}
}

func TestExtra_APIKeyEnv_Constants(t *testing.T) {
	if APIKeyLive != "live" {
		t.Errorf("APIKeyLive: got %s", APIKeyLive)
	}
	if APIKeyTest != "test" {
		t.Errorf("APIKeyTest: got %s", APIKeyTest)
	}
}
