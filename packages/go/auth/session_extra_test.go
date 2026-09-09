// Package auth - extra tests for session store.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// MemorySessionStore
// =============================================================================

func TestExtra_MemorySessionStore_New(t *testing.T) {
	s := NewMemorySessionStore()
	if s == nil {
		t.Fatal("NewMemorySessionStore returned nil")
	}
	if s.store == nil {
		t.Error("store map should be initialized")
	}
}

func TestExtra_MemorySessionStore_Create(t *testing.T) {
	s := NewMemorySessionStore()
	ctx := context.Background()
	d := SessionData{ID: "s1", UserID: "u1"}
	if err := s.Create(ctx, d, 0); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, ok := s.store["s1"]; !ok {
		t.Error("session not stored")
	}
}

func TestExtra_MemorySessionStore_Create_AutoTimestamp(t *testing.T) {
	s := NewMemorySessionStore()
	ctx := context.Background()
	d := SessionData{ID: "s1", UserID: "u1"}
	_ = s.Create(ctx, d, 0)
	got := s.store["s1"]
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should be auto-set")
	}
	if got.LastSeenAt.IsZero() {
		t.Error("LastSeenAt should be auto-set")
	}
}

func TestExtra_MemorySessionStore_Get_NotFound(t *testing.T) {
	s := NewMemorySessionStore()
	_, err := s.Get(context.Background(), "missing")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestExtra_MemorySessionStore_Get_Found(t *testing.T) {
	s := NewMemorySessionStore()
	ctx := context.Background()
	_ = s.Create(ctx, SessionData{ID: "s1", UserID: "u1"}, 0)
	got, err := s.Get(ctx, "s1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.UserID != "u1" {
		t.Errorf("UserID mismatch: %s", got.UserID)
	}
}

func TestExtra_MemorySessionStore_Touch(t *testing.T) {
	s := NewMemorySessionStore()
	ctx := context.Background()
	_ = s.Create(ctx, SessionData{ID: "s1", UserID: "u1"}, 0)
	original := s.store["s1"].LastSeenAt
	time.Sleep(2 * time.Millisecond)
	if err := s.Touch(ctx, "s1", 0); err != nil {
		t.Fatalf("touch: %v", err)
	}
	if !s.store["s1"].LastSeenAt.After(original) {
		t.Error("LastSeenAt should advance after Touch")
	}
}

func TestExtra_MemorySessionStore_Touch_NotFound(t *testing.T) {
	s := NewMemorySessionStore()
	err := s.Touch(context.Background(), "missing", 0)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestExtra_MemorySessionStore_Delete(t *testing.T) {
	s := NewMemorySessionStore()
	ctx := context.Background()
	_ = s.Create(ctx, SessionData{ID: "s1", UserID: "u1"}, 0)
	if err := s.Delete(ctx, "s1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := s.store["s1"]; ok {
		t.Error("session should be deleted")
	}
}

func TestExtra_MemorySessionStore_Delete_NonExistent(t *testing.T) {
	s := NewMemorySessionStore()
	if err := s.Delete(context.Background(), "missing"); err != nil {
		t.Errorf("delete non-existent should be no-op, got %v", err)
	}
}

func TestExtra_MemorySessionStore_DeleteByUser(t *testing.T) {
	s := NewMemorySessionStore()
	ctx := context.Background()
	_ = s.Create(ctx, SessionData{ID: "s1", UserID: "u1"}, 0)
	_ = s.Create(ctx, SessionData{ID: "s2", UserID: "u1"}, 0)
	_ = s.Create(ctx, SessionData{ID: "s3", UserID: "u2"}, 0)
	if err := s.DeleteByUser(ctx, "u1"); err != nil {
		t.Fatalf("deletebyuser: %v", err)
	}
	if _, ok := s.store["s1"]; ok {
		t.Error("s1 should be deleted")
	}
	if _, ok := s.store["s2"]; ok {
		t.Error("s2 should be deleted")
	}
	if _, ok := s.store["s3"]; !ok {
		t.Error("s3 (different user) should remain")
	}
}

func TestExtra_MemorySessionStore_ListByUser(t *testing.T) {
	s := NewMemorySessionStore()
	ctx := context.Background()
	_ = s.Create(ctx, SessionData{ID: "s1", UserID: "u1"}, 0)
	_ = s.Create(ctx, SessionData{ID: "s2", UserID: "u1"}, 0)
	_ = s.Create(ctx, SessionData{ID: "s3", UserID: "u2"}, 0)
	list, err := s.ListByUser(ctx, "u1")
	if err != nil {
		t.Fatalf("listbyuser: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(list))
	}
	for _, sess := range list {
		if sess.UserID != "u1" {
			t.Errorf("found session for wrong user: %s", sess.UserID)
		}
	}
}

func TestExtra_MemorySessionStore_ListByUser_Empty(t *testing.T) {
	s := NewMemorySessionStore()
	list, err := s.ListByUser(context.Background(), "unknown")
	if err != nil {
		t.Fatalf("listbyuser: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}
}

func TestExtra_MemorySessionStore_Concurrent(t *testing.T) {
	s := NewMemorySessionStore()
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := "s" + string(rune('a'+idx%26)) + "_" + string(rune('0'+idx%10))
			_ = s.Create(ctx, SessionData{ID: id, UserID: "u"}, 0)
		}(i)
	}
	wg.Wait()
	if len(s.store) == 0 {
		t.Error("expected at least one session after concurrent creates")
	}
}

// =============================================================================
// SessionData JSON serialization
// =============================================================================

func TestExtra_SessionData_JSON_Roundtrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	d := SessionData{
		ID:          "s1",
		UserID:      "u1",
		TenantID:    "t1",
		Email:       "u@example.com",
		Roles:       []string{"admin"},
		Permissions: []string{"read", "write"},
		IPAddress:   "1.2.3.4",
		UserAgent:   "Mozilla/5.0",
		DeviceFP:    "fp-123",
		MFAVerified: true,
		Metadata:    map[string]interface{}{"k": "v"},
		CreatedAt:   now,
		LastSeenAt:  now,
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got SessionData
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != d.ID || got.UserID != d.UserID || got.TenantID != d.TenantID {
		t.Errorf("core fields mismatch")
	}
	if !got.MFAVerified {
		t.Error("MFAVerified should be true")
	}
	if got.Metadata["k"] != "v" {
		t.Errorf("metadata mismatch: %v", got.Metadata)
	}
}

func TestExtra_SessionData_OptionalFields(t *testing.T) {
	d := SessionData{ID: "s1"}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	// The struct uses ``omitempty`` on a subset of fields.  Specifically,
	// the following fields are omitted when empty: ``permissions``,
	// ``ip``, ``user_agent``, ``device_fp``, ``metadata``.  Others
	// (tenant_id, email, roles) appear as zero values.
	omitted := []string{`"permissions"`, `"ip"`, `"user_agent"`, `"device_fp"`, `"metadata"`}
	for _, key := range omitted {
		if strings.Contains(s, key) {
			t.Errorf("field %s should be omitted: %s", key, s)
		}
	}
	// Required fields must be present.
	if !strings.Contains(s, `"id"`) {
		t.Errorf("missing id: %s", s)
	}
}

func TestExtra_SessionData_UnmarshalInvalid(t *testing.T) {
	var d SessionData
	err := json.Unmarshal([]byte("not-json"), &d)
	if err == nil {
		t.Error("expected unmarshal error")
	}
}

// =============================================================================
// Errors
// =============================================================================

func TestExtra_SessionErrors(t *testing.T) {
	if ErrSessionNotFound == nil {
		t.Error("ErrSessionNotFound should not be nil")
	}
	if ErrSessionExpired == nil {
		t.Error("ErrSessionExpired should not be nil")
	}
	if ErrSessionNotFound.Error() == "" {
		t.Error("ErrSessionNotFound should have a message")
	}
	if ErrSessionExpired.Error() == "" {
		t.Error("ErrSessionExpired should have a message")
	}
	if errors.Is(ErrSessionNotFound, ErrSessionNotFound) != true {
		t.Error("errors.Is should match itself")
	}
}

// =============================================================================
// SessionStore interface conformance
// =============================================================================

func TestExtra_MemorySessionStore_ConformsInterface(t *testing.T) {
	var _ SessionStore = (*MemorySessionStore)(nil)
	// Verify a *MemorySessionStore can be assigned to SessionStore.
	var s SessionStore = NewMemorySessionStore()
	if s == nil {
		t.Error("SessionStore should accept MemorySessionStore")
	}
}

// =============================================================================
// Internal crypto helpers
// =============================================================================

func TestExtra_Sha256SumImpl(t *testing.T) {
	// sha256 of "abc" is ba7816bf...
	got := sha256sumImpl([]byte("abc"))
	if len(got) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(got))
	}
	expected := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	gotHex := ""
	for _, b := range got {
		gotHex += string("0123456789abcdef"[(b>>4)&0xF]) + string("0123456789abcdef"[b&0xF])
	}
	if gotHex != expected {
		t.Errorf("sha256 mismatch:\n got %s\nwant %s", gotHex, expected)
	}
}

func TestExtra_Sha256SumImpl_Empty(t *testing.T) {
	got := sha256sumImpl(nil)
	if len(got) != 32 {
		t.Errorf("expected 32 bytes for nil, got %d", len(got))
	}
	// sha256("") = e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
}

// =============================================================================
// UUID generation helper
// =============================================================================

func TestExtra_UuidGen_ValidUUID(t *testing.T) {
	id := uuidGen()
	if len(id) < 32 {
		t.Errorf("UUID too short: %s", id)
	}
}

func TestExtra_UuidGen_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := uuidGen()
		if seen[id] {
			t.Errorf("duplicate UUID: %s", id)
		}
		seen[id] = true
	}
}
