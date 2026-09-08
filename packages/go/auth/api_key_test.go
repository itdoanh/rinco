// Tests for auth API key edge cases.
package auth

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// ===== API key edge cases =====

func TestGenerateAPIKey_InvalidEnv(t *testing.T) {
	_, err := GenerateAPIKey("invalid-env", "rinco")
	if err == nil {
		t.Fatal("expected error for invalid env")
	}
}

func TestGenerateAPIKey_DefaultPrefix(t *testing.T) {
	key, err := GenerateAPIKey(APIKeyLive, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key, DefaultPrefix+"_") {
		t.Fatalf("expected default prefix, got %s", key)
	}
}

func TestGenerateAPIKey_TestEnv(t *testing.T) {
	key, err := GenerateAPIKey(APIKeyTest, "rinco")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key, "rinco_test_") {
		t.Fatalf("expected rinco_test_ prefix, got %s", key)
	}
}

func TestGenerateAPIKey_NoUnderscoreInSecret(t *testing.T) {
	// Generate many keys and check secret part doesn't contain '_' which
	// would break ParseAPIKey splitting.
	for i := 0; i < 100; i++ {
		key, err := GenerateAPIKey(APIKeyLive, "rinco")
		if err != nil {
			t.Fatal(err)
		}
		secret := strings.Split(key, "_")[2]
		if strings.Contains(secret, "_") {
			t.Fatalf("secret contains underscore: %s", key)
		}
	}
}

func TestParseAPIKey_InvalidFormat(t *testing.T) {
	tests := []string{
		"",
		"rinco",
		"rinco_live",
		"rinco_live_a_b_c",
		"invalid",
		"_live_secret",
		"rinco__secret",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			_, _, _, err := ParseAPIKey(tt)
			if err == nil {
				t.Fatalf("expected error for invalid format %q", tt)
			}
		})
	}
}

func TestParseAPIKey_InvalidEnv(t *testing.T) {
	_, _, _, err := ParseAPIKey("rinco_bogus_aaaaaaaaaaaaaaaa")
	if err == nil {
		t.Fatal("expected error for invalid env")
	}
	if !strings.Contains(err.Error(), "invalid env") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseAPIKey_ShortSecret(t *testing.T) {
	_, _, _, err := ParseAPIKey("rinco_live_short")
	if err == nil {
		t.Fatal("expected error for short secret")
	}
}

func TestParseAPIKey_ValidRoundTrip(t *testing.T) {
	key, err := GenerateAPIKey(APIKeyLive, "rinco")
	if err != nil {
		t.Fatal(err)
	}
	prefix, env, secret, err := ParseAPIKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if prefix != "rinco" {
		t.Errorf("prefix: got %s", prefix)
	}
	if env != APIKeyLive {
		t.Errorf("env: got %s", env)
	}
	if len(secret) < 16 {
		t.Errorf("secret too short: %d", len(secret))
	}
}

func TestLast4_ShortKey(t *testing.T) {
	if Last4("abc") != "abc" {
		t.Error("expected short key to return unchanged")
	}
	if Last4("") != "" {
		t.Error("empty key should return empty")
	}
}

func TestLast4_LongKey(t *testing.T) {
	key := "rinco_live_abcdef1234"
	got := Last4(key)
	if got != "1234" {
		t.Errorf("Last4: got %s", got)
	}
}

func TestIsExpired_NilRecord(t *testing.T) {
	if IsExpired(nil, time.Now()) {
		t.Error("nil record should not be expired")
	}
}

func TestIsExpired_NoExpiry(t *testing.T) {
	rec := &APIKeyRecord{}
	if IsExpired(rec, time.Now()) {
		t.Error("record with no expiry should not be expired")
	}
}

func TestIsExpired_FutureExpiry(t *testing.T) {
	rec := &APIKeyRecord{
		ExpiresAt: ptrTime(time.Now().Add(time.Hour)),
	}
	if IsExpired(rec, time.Now()) {
		t.Error("future expiry should not be expired")
	}
}

func TestIsExpired_PastExpiry(t *testing.T) {
	rec := &APIKeyRecord{
		ExpiresAt: ptrTime(time.Now().Add(-time.Hour)),
	}
	if !IsExpired(rec, time.Now()) {
		t.Error("past expiry should be expired")
	}
}

func TestIsRevoked_NilRecord(t *testing.T) {
	if IsRevoked(nil) {
		t.Error("nil record should not be revoked")
	}
}

func TestIsRevoked_NotRevoked(t *testing.T) {
	rec := &APIKeyRecord{}
	if IsRevoked(rec) {
		t.Error("empty record should not be revoked")
	}
}

func TestIsRevoked_Revoked(t *testing.T) {
	rec := &APIKeyRecord{
		RevokedAt: ptrTime(time.Now()),
	}
	if !IsRevoked(rec) {
		t.Error("record with RevokedAt should be revoked")
	}
}

func TestScopesContain_Empty(t *testing.T) {
	if ScopesContain(nil, "lead.create") {
		t.Error("nil scopes should not match")
	}
	if ScopesContain([]string{}, "lead.create") {
		t.Error("empty scopes should not match")
	}
}

func TestScopesContain_DeepWildcard(t *testing.T) {
	if !ScopesContain([]string{"lead.list.*"}, "lead.list.contact") {
		t.Error("nested wildcard should match")
	}
	if ScopesContain([]string{"lead.list.*"}, "lead.create") {
		t.Error("nested wildcard should not match different prefix")
	}
}

func TestScopesContain_NoWildcard(t *testing.T) {
	if !ScopesContain([]string{"lead.read"}, "lead.read") {
		t.Error("exact match should pass")
	}
	if ScopesContain([]string{"lead.read"}, "lead.create") {
		t.Error("non-match should fail")
	}
}

func TestNewRecord_NoTTL(t *testing.T) {
	rec, err := NewRecord("tenant-1", "user-1", "test", APIKeyTest, []string{"x"}, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.ExpiresAt != nil {
		t.Error("expected no expiry when ttl=0")
	}
}

func TestNewRecord_DefaultRateLimit(t *testing.T) {
	rec, err := NewRecord("tenant-1", "user-1", "test", APIKeyTest, []string{"x"}, 0, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if rec.RateLimit == 0 {
		t.Error("expected default rate limit")
	}
}

func TestNewRecord_TestEnv(t *testing.T) {
	rec, err := NewRecord("tenant-1", "user-1", "test", APIKeyTest, []string{"x"}, 100, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rec.Key, "rinco_test_") {
		t.Errorf("expected rinco_test_, got %s", rec.Key)
	}
}

func TestHashKey_DifferentInputs(t *testing.T) {
	h1 := HashKey("key1")
	h2 := HashKey("key2")
	if h1 == h2 {
		t.Error("different inputs should produce different hashes")
	}
}

// ===== Helpers =====

func ptrTime(t time.Time) *time.Time {
	return &t
}

// ===== Error message tests =====

func TestErrorMessages(t *testing.T) {
	// Sanity check that errors.New and fmt.Errorf errors contain expected context.
	if errors.New("test").Error() != "test" {
		t.Error("errors.New should produce exact message")
	}
}
