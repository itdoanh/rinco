// Package auth - tests.
package auth

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// ===== PASETO tests =====

func TestPasetoRoundTrip(t *testing.T) {
	p, err := NewPaseto("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "k1")
	if err != nil {
		t.Fatal(err)
	}
	claims := Claims{
		Subject:  "user-123",
		TenantID: "tenant-456",
		Roles:    []string{"admin", "user"},
	}
	tok, err := p.Encrypt(claims)
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.Decrypt(tok)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got.Subject != "user-123" || got.TenantID != "tenant-456" {
		t.Fatalf("claims mismatch: %+v", got)
	}
	if got.Kid != "k1" {
		t.Fatalf("kid mismatch: got %q", got.Kid)
	}
}

func TestKeyRingRotation(t *testing.T) {
	currentHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	previousHex := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"

	ring, err := NewKeyRing(currentHex, previousHex, "current", "previous")
	if err != nil {
		t.Fatal(err)
	}

	// Sign with current key
	tok, err := ring.Encrypt(Claims{Subject: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	// Verify with current
	if _, err := ring.Decrypt(tok); err != nil {
		t.Fatalf("verify current: %v", err)
	}

	// Manually create token with previous key
	prevPaseto, _ := NewPaseto(previousHex, "previous")
	oldTok, err := prevPaseto.Encrypt(Claims{Subject: "u1-old"})
	if err != nil {
		t.Fatal(err)
	}
	// Verify with keyring should fallback to previous
	got, err := ring.Decrypt(oldTok)
	if err != nil {
		t.Fatalf("verify previous: %v", err)
	}
	if got.Subject != "u1-old" {
		t.Fatalf("expected u1-old, got %q", got.Subject)
	}
}

func TestExpiredToken(t *testing.T) {
	p, _ := NewPaseto("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "k1")
	tok, err := p.Encrypt(Claims{
		Subject:   "x",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Decrypt(tok); err != ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

// ===== Argon2 tests =====

func TestArgon2HashAndVerify(t *testing.T) {
	hash, err := HashPassword("super-secret-password")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("bad hash format: %s", hash)
	}
	ok, err := VerifyPassword("super-secret-password", hash)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("verify failed")
	}
	ok, _ = VerifyPassword("wrong-password", hash)
	if ok {
		t.Fatal("verify should fail for wrong password")
	}
}

// ===== RBAC tests =====

func TestRBACBuiltins(t *testing.T) {
	e := NewEngine()
	roles := e.ListRoles()
	if len(roles) < 5 {
		t.Fatalf("expected >=5 builtin roles, got %d", len(roles))
	}
	if !e.CheckPermission("super_admin", "anything.you.want") {
		t.Fatal("super_admin should have wildcard")
	}
	if !e.CheckPermission("tenant_admin", "lead.create") {
		t.Fatal("tenant_admin should have lead.*")
	}
	if e.CheckPermission("guest", "lead.create") {
		t.Fatal("guest should NOT have lead.create")
	}
}

func TestRBACWildcard(t *testing.T) {
	if !matchPermission("lead.*", "lead.create") {
		t.Fatal("wildcard prefix should match")
	}
	if matchPermission("lead.*", "contact.create") {
		t.Fatal("wildcard should not match different resource")
	}
	if !matchPermission("*", "anything") {
		t.Fatal("* should match everything")
	}
}

func TestRBACPrincipal(t *testing.T) {
	e := NewEngine()
	// User có cả "user" + "manager" → gộp permissions
	perms := e.PrincipalPermissions([]string{"user", "manager"}, nil)
	if len(perms) == 0 {
		t.Fatal("expected merged permissions")
	}
	hasLeadRead := false
	for _, p := range perms {
		if p == "lead.read" {
			hasLeadRead = true
			break
		}
	}
	if !hasLeadRead {
		t.Fatal("expected lead.read from manager role")
	}
}

func TestRBACAddCustomRole(t *testing.T) {
	e := NewEngine()
	err := e.AddRole(Role{
		Name:        "sales_rep",
		Permissions: []string{"lead.read", "lead.create"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !e.CheckPermission("sales_rep", "lead.read") {
		t.Fatal("custom role should have permission")
	}
	// Cannot override built-in
	if err := e.AddRole(Role{Name: "super_admin", Permissions: []string{"x"}}); err == nil {
		t.Fatal("should not override built-in super_admin role")
	}
}

// ===== API key tests =====

func TestGenerateAndParseAPIKey(t *testing.T) {
	key, err := GenerateAPIKey(APIKeyLive, "rinco")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key, "rinco_live_") {
		t.Fatalf("bad prefix: %s", key)
	}
	prefix, env, secret, err := ParseAPIKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if prefix != "rinco" || env != APIKeyLive || len(secret) < 16 {
		t.Fatalf("parse mismatch: %+v %+v %+v", prefix, env, secret)
	}
}

func TestAPIKeyHash(t *testing.T) {
	key := "rinco_live_abcdef1234567890"
	h1 := HashKey(key)
	h2 := HashKey(key)
	if h1 != h2 {
		t.Fatal("hash should be deterministic")
	}
	if len(h1) != 64 {
		t.Fatalf("sha256 hex should be 64 chars, got %d", len(h1))
	}
}

func TestAPIKeyScopes(t *testing.T) {
	if !ScopesContain([]string{"*"}, "lead.create") {
		t.Fatal("* should match")
	}
	if !ScopesContain([]string{"lead.*"}, "lead.create") {
		t.Fatal("lead.* should match lead.create")
	}
	if ScopesContain([]string{"contact.*"}, "lead.create") {
		t.Fatal("contact.* should not match lead.create")
	}
}

func TestNewAPIKeyRecord(t *testing.T) {
	rec, err := NewRecord("tenant-1", "user-1", "test key", APIKeyTest, []string{"lead.read"}, 100, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rec.Key, "rinco_test_") {
		t.Fatalf("expected rinco_test_, got %s", rec.Key)
	}
	if len(rec.Last4) != 4 {
		t.Fatalf("last4 wrong length: %d", len(rec.Last4))
	}
	if rec.ExpiresAt == nil {
		t.Fatal("expires_at should be set")
	}
}

// ===== Session tests =====

func TestMemorySessionStore(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()
	data := SessionData{
		ID:       "sess-1",
		UserID:   "user-1",
		TenantID: "tenant-1",
		Roles:    []string{"admin"},
	}
	if err := store.Create(ctx, data, time.Minute); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, "sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != "user-1" {
		t.Fatalf("got %+v", got)
	}
	if err := store.Touch(ctx, "sess-1", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteByUser(ctx, "user-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "sess-1"); err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSessionJSON(t *testing.T) {
	data := SessionData{
		ID:         "sess-1",
		UserID:     "user-1",
		Roles:      []string{"a", "b"},
		MFAVerified: true,
		Metadata:   map[string]interface{}{"k": "v"},
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	var got SessionData
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "sess-1" || !got.MFAVerified {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
}

// ===== OAuth state/PKCE helpers =====

func TestStateAndPKCE(t *testing.T) {
	state, err := GenerateState(16)
	if err != nil {
		t.Fatal(err)
	}
	if !ValidateState(state, state) {
		t.Fatal("validate should pass")
	}
	if ValidateState(state, "x") {
		t.Fatal("validate should fail")
	}
	v, c, err := GeneratePKCE()
	if err != nil {
		t.Fatal(err)
	}
	if v == "" || c == "" || v == c {
		t.Fatal("pkce should produce distinct verifier+challenge")
	}
}

func TestEncodeDecodeStateCookie(t *testing.T) {
	raw := EncodeStateToCookie(map[string]string{"state": "abc", "verifier": "xyz"})
	got, err := DecodeStateFromCookie(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got["state"] != "abc" || got["verifier"] != "xyz" {
		t.Fatalf("decode mismatch: %+v", got)
	}
}